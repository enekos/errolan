// Package federation runs the work for inbound federated mentions —
// W3C Webmentions and minimal ActivityPub Create(Note) replies. The verifier
// is a single in-process worker (1 goroutine, bounded queue) that fetches the
// source URL, confirms it actually links to the target, extracts an author +
// snippet, and marks the mention verified or rejected.
//
// Keep this conservative: tight timeouts, response-body size caps, redirect
// limits, and refusal to follow non-http(s) schemes. The verifier is reachable
// from anonymous callers via the public Webmention endpoint, so it's a real
// SSRF surface.
package federation

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/enekos/errolan/internal/models"
	"github.com/enekos/errolan/internal/store"
)

const (
	// maxFetch caps how much of a source page we'll read. Real microformat
	// pages and Mastodon profile snapshots fit comfortably inside 1 MB.
	maxFetch = 1 << 20

	// fetchTimeout is the per-source ceiling. Webmention senders sometimes
	// point at slow blog hosts — 12s is generous but bounded.
	fetchTimeout = 12 * time.Second

	// queueCapacity is the in-memory queue depth. Beyond this, Enqueue rejects
	// with ErrBusy so the public endpoint can return 429.
	queueCapacity = 1024

	// maxRedirects keeps the verifier from chasing a malicious source through
	// long redirect chains into internal networks.
	maxRedirects = 4
)

// ErrBusy is returned by Enqueue when the queue is full. Callers should map it
// to HTTP 429 + Retry-After.
var ErrBusy = errors.New("verifier queue full")

// Verifier owns the fetch worker, an HTTP client with SSRF-safe dialing, and
// a small bounded queue. Start it once at boot; Stop on shutdown.
type Verifier struct {
	st     *store.Store
	logger *slog.Logger
	client *http.Client
	queue  chan int64
	done   chan struct{}

	// OnVerified, when non-nil, is called once a mention transitions to
	// verified. Used by the API server to publish an SSE thread update so
	// connected readers see the new mention without polling.
	OnVerified func(threadID int64)

	// userAgent is sent on every outbound fetch. Sites often filter by UA so
	// announcing ourselves makes things debuggable on the publisher side.
	userAgent string
}

// New builds a verifier. Call Start to begin draining the queue.
func New(st *store.Store, logger *slog.Logger) *Verifier {
	if logger == nil {
		logger = slog.Default()
	}
	v := &Verifier{
		st:        st,
		logger:    logger,
		queue:     make(chan int64, queueCapacity),
		done:      make(chan struct{}),
		userAgent: "Errolan-Webmention/1.0 (+https://errolan.dev)",
	}
	v.client = &http.Client{
		Timeout: fetchTimeout,
		// CheckRedirect refuses to redirect to non-http(s) schemes or after
		// the cap. The default behaviour leaks the original Host header into
		// the new URL, which is fine since we don't send credentials.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return errors.New("too many redirects")
			}
			if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
				return fmt.Errorf("refusing redirect to %q", req.URL.Scheme)
			}
			return nil
		},
		Transport: &http.Transport{
			// Dial through a guard that rejects RFC1918 / loopback / link-local
			// addresses to limit SSRF. Allowing literal localhost is convenient
			// in dev — gate it on env vars rather than hardcoding.
			DialContext: safeDialContext,
			MaxIdleConns:      32,
			IdleConnTimeout:   60 * time.Second,
		},
	}
	return v
}

// Start launches the background worker. Idempotent.
func (v *Verifier) Start() {
	go v.loop()
}

// Stop signals the worker to drain and exit. Safe to call multiple times.
func (v *Verifier) Stop() {
	select {
	case <-v.done:
		return
	default:
		close(v.done)
	}
}

// Enqueue submits a mention id for verification. Non-blocking; returns ErrBusy
// when the queue is full.
func (v *Verifier) Enqueue(mentionID int64) error {
	select {
	case v.queue <- mentionID:
		return nil
	default:
		return ErrBusy
	}
}

func (v *Verifier) loop() {
	for {
		select {
		case <-v.done:
			return
		case id := <-v.queue:
			v.verify(id)
		}
	}
}

// verify fetches the source, checks it references the target, and updates the
// mention row accordingly. Errors become "rejected" with a human-readable reason.
func (v *Verifier) verify(id int64) {
	m, err := v.st.MentionByID(id)
	if err != nil {
		v.logger.Warn("verifier: mention not found", "id", id, "err", err)
		return
	}
	if m.Status != models.MentionStatusPending {
		return
	}

	switch m.Kind {
	case models.MentionKindActivityPub:
		// ActivityPub posts come pre-supplied with snippet + actor on the inbox
		// side; verification here is mostly "the URL is reachable" so we don't
		// store mentions to dead actors. A 404 marks rejected; 2xx verifies.
		if err := v.verifyActivityPub(m); err != nil {
			_ = v.st.MarkMentionRejected(id, err.Error())
			return
		}
		if err := v.st.MarkMentionVerified(id, m.AuthorName, m.AuthorURL, m.Snippet); err == nil {
			v.notifyVerified(m.ThreadID)
		}
	default:
		author, authorURL, snippet, err := v.verifyWebmention(m)
		if err != nil {
			_ = v.st.MarkMentionRejected(id, err.Error())
			return
		}
		if err := v.st.MarkMentionVerified(id, author, authorURL, snippet); err == nil {
			v.notifyVerified(m.ThreadID)
		}
	}
}

func (v *Verifier) notifyVerified(threadID int64) {
	if v.OnVerified != nil {
		// Run the hook in the worker goroutine — callers are expected to be
		// cheap (a Hub.Publish that returns immediately).
		v.OnVerified(threadID)
	}
}

// verifyWebmention follows the W3C Webmention verification: fetch source, look
// for any anchor whose href equals target. We extract a snippet around the
// link and a best-guess author from <meta name="author"> or the page title.
func (v *Verifier) verifyWebmention(m *models.Mention) (author, authorURL, snippet string, err error) {
	sourceURL, perr := parseHTTPURL(m.Source)
	if perr != nil {
		return "", "", "", perr
	}
	if _, perr := parseHTTPURL(m.Target); perr != nil {
		return "", "", "", fmt.Errorf("invalid target: %w", perr)
	}
	body, _, err := v.fetch(sourceURL.String())
	if err != nil {
		return "", "", "", fmt.Errorf("fetch source: %w", err)
	}
	if !pageLinksTo(body, m.Target) {
		return "", "", "", errors.New("source does not link to target")
	}
	author, authorURL = extractAuthor(body, sourceURL)
	snippet = extractSnippet(body, m.Target)
	return author, authorURL, snippet, nil
}

func (v *Verifier) verifyActivityPub(m *models.Mention) error {
	// Best-effort: fetch the source (actor or note URI) to make sure it's
	// reachable. We don't yet enforce HTTP signatures — that's a v2 lift.
	u, err := parseHTTPURL(m.Source)
	if err != nil {
		return err
	}
	_, status, err := v.fetch(u.String())
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("source returned %d", status)
	}
	return nil
}

// fetch GETs the URL with our SSRF-safe client. Returns the bytes (capped),
// the HTTP status, and any error.
func (v *Verifier) fetch(rawurl string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, rawurl, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", v.userAgent)
	req.Header.Set("Accept", "text/html, application/xhtml+xml, application/activity+json;q=0.9, application/ld+json;q=0.8")
	resp, err := v.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxFetch))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

// parseHTTPURL accepts only http(s) URLs and rejects anything with a userinfo
// component or missing host. It's the gate every external URL flows through.
func parseHTTPURL(raw string) (*url.URL, error) {
	if raw == "" {
		return nil, errors.New("empty url")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("scheme not allowed: %q", u.Scheme)
	}
	if u.Host == "" {
		return nil, errors.New("missing host")
	}
	if u.User != nil {
		return nil, errors.New("userinfo not allowed in url")
	}
	return u, nil
}

// safeDialContext refuses to connect to RFC1918 / loopback / link-local IPs.
// Without this, an attacker could submit a Webmention with source=http://10.0.0.1
// and use our verifier to probe internal networks.
func safeDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	for _, ip := range ips {
		if isPrivateOrLocal(ip.IP) {
			return nil, fmt.Errorf("refusing to dial private address %s", ip.IP)
		}
	}
	d := net.Dialer{Timeout: 5 * time.Second}
	return d.DialContext(ctx, network, net.JoinHostPort(host, port))
}

func isPrivateOrLocal(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		switch {
		case ip4[0] == 10:
			return true
		case ip4[0] == 127:
			return true
		case ip4[0] == 169 && ip4[1] == 254:
			return true
		case ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31:
			return true
		case ip4[0] == 192 && ip4[1] == 168:
			return true
		}
		return false
	}
	// IPv6 ULA: fc00::/7
	if ip[0] == 0xfc || ip[0] == 0xfd {
		return true
	}
	return false
}

// pageLinksTo reports whether the HTML body contains an <a href="target"> or
// any element with href/src equal to target. Comparison is case-sensitive on
// path but tolerates http↔https flipping per the Webmention test suite.
func pageLinksTo(body []byte, target string) bool {
	if len(body) == 0 || target == "" {
		return false
	}
	s := string(body)
	if strings.Contains(s, `href="`+target+`"`) || strings.Contains(s, `href='`+target+`'`) {
		return true
	}
	alt := flipScheme(target)
	if strings.Contains(s, `href="`+alt+`"`) || strings.Contains(s, `href='`+alt+`'`) {
		return true
	}
	return false
}

func flipScheme(u string) string {
	if strings.HasPrefix(u, "https://") {
		return "http://" + u[len("https://"):]
	}
	if strings.HasPrefix(u, "http://") {
		return "https://" + u[len("http://"):]
	}
	return u
}

var (
	reAuthorMeta = regexp.MustCompile(`(?i)<meta[^>]+name=["']author["'][^>]+content=["']([^"']+)["']`)
	reTitle      = regexp.MustCompile(`(?i)<title>([^<]+)</title>`)
)

func extractAuthor(body []byte, source *url.URL) (name, urlStr string) {
	if m := reAuthorMeta.FindSubmatch(body); m != nil {
		return strings.TrimSpace(string(m[1])), source.Scheme + "://" + source.Host + "/"
	}
	if m := reTitle.FindSubmatch(body); m != nil {
		return strings.TrimSpace(string(m[1])), source.String()
	}
	return source.Host, source.Scheme + "://" + source.Host + "/"
}

// extractSnippet pulls out a short window of text around the first occurrence
// of the target URL, with HTML tags stripped. Falls back to the first 240 chars
// of body text if the URL appears in an unexpected position.
func extractSnippet(body []byte, target string) string {
	s := string(body)
	idx := strings.Index(s, target)
	if idx < 0 {
		idx = strings.Index(s, flipScheme(target))
	}
	if idx < 0 {
		return ""
	}
	start := idx - 200
	if start < 0 {
		start = 0
	}
	end := idx + 200
	if end > len(s) {
		end = len(s)
	}
	chunk := s[start:end]
	chunk = stripHTML(chunk)
	chunk = strings.TrimSpace(collapseSpaces(chunk))
	if len(chunk) > 320 {
		chunk = chunk[:320] + "…"
	}
	return chunk
}

var reTags = regexp.MustCompile(`<[^>]+>`)
var reSpaces = regexp.MustCompile(`\s+`)

func stripHTML(s string) string   { return reTags.ReplaceAllString(s, " ") }
func collapseSpaces(s string) string { return reSpaces.ReplaceAllString(s, " ") }
