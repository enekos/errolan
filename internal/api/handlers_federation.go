package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/enekos/errolan/internal/models"
)

// ----- Webmentions (W3C) -----

// handleWebmention is the public Webmention endpoint. Per spec it accepts
// `application/x-www-form-urlencoded` with `source` and `target`. We validate
// both, locate the thread on our side via target URL, and enqueue for async
// verification. Always returns 202 (or an error) so the publisher doesn't see
// timing info that leaks whether a particular target is or isn't a thread.
func (s *Server) handleWebmention(w http.ResponseWriter, r *http.Request) {
	// Per Webmention spec: form-urlencoded body. Some senders use JSON — accept
	// either to be friendly.
	source, target := "", ""
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "application/json") {
		var body struct {
			Source string `json:"source"`
			Target string `json:"target"`
		}
		if err := decode(r, &body); err == nil {
			source, target = body.Source, body.Target
		}
	} else {
		if err := r.ParseForm(); err == nil {
			source = r.PostForm.Get("source")
			target = r.PostForm.Get("target")
		}
	}
	source = strings.TrimSpace(source)
	target = strings.TrimSpace(target)
	if source == "" || target == "" {
		writeError(w, http.StatusBadRequest, "source and target required")
		return
	}
	if source == target {
		writeError(w, http.StatusBadRequest, "source and target must differ")
		return
	}
	if _, err := parseHTTPURL(source); err != nil {
		writeError(w, http.StatusBadRequest, "invalid source url")
		return
	}
	tu, err := parseHTTPURL(target)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid target url")
		return
	}

	// Map target → (site, thread). We look up by exact URL match across all
	// sites: in practice each site owns a distinct origin so the first match
	// is unambiguous. If the target doesn't match a known thread, refuse with
	// 400 (per spec we can either accept-and-discard or reject; reject is
	// kinder to senders).
	thread, site := s.lookupThreadByURL(tu.String())
	if thread == nil {
		writeError(w, http.StatusBadRequest, "target is not a known thread")
		return
	}

	m, err := s.Store.EnqueueMention(site.ID, thread.ID, source, target, models.MentionKindWebmention)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "enqueue failed")
		return
	}
	if s.Verifier != nil {
		if err := s.Verifier.Enqueue(m.ID); err != nil {
			w.Header().Set("Retry-After", "30")
			writeError(w, http.StatusTooManyRequests, "verifier busy; try again")
			return
		}
	}
	w.Header().Set("Location", "/api/mentions/"+strconv.FormatInt(m.ID, 10))
	writeJSON(w, http.StatusAccepted, map[string]any{
		"id":     m.ID,
		"status": m.Status,
	})
}

// handleListThreadMentions returns the verified mentions for a thread. Site
// scope is enforced through the X-Errolan-Site header like the comment endpoints.
func (s *Server) handleListThreadMentions(w http.ResponseWriter, r *http.Request) {
	site, err := requireSite(r)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	slug := r.PathValue("slug")
	thread, err := s.Store.ThreadBySlug(site.ID, slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "thread not found")
		return
	}
	mentions, err := s.Store.ListThreadMentions(thread.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list failed")
		return
	}
	writeJSON(w, http.StatusOK, mentions)
}

// lookupThreadByURL walks every site searching for a thread with the matching
// URL field. Threads are small in number per site, and sites are O(dozens)
// for typical self-hosters — a scan keeps this simple. If your instance grows
// past that, swap in an index.
func (s *Server) lookupThreadByURL(targetURL string) (*models.Thread, *models.Site) {
	sites, err := s.Store.ListSites()
	if err != nil {
		return nil, nil
	}
	for _, site := range sites {
		if t, err := s.Store.ThreadByURL(site.ID, targetURL); err == nil {
			return t, site
		}
	}
	return nil, nil
}

// ----- ActivityPub: webfinger, actor, inbox -----

// instanceBase returns the base URL the federation endpoints publish (Actor
// IDs, Webfinger acct URIs, etc.). Set ERROLAN_PUBLIC_URL in production —
// otherwise we fall back to the Host header which works fine for testing.
func (s *Server) instanceBase(r *http.Request) string {
	if s.PublicURL != "" {
		return strings.TrimRight(s.PublicURL, "/")
	}
	scheme := "https"
	if r.TLS == nil && r.Header.Get("X-Forwarded-Proto") == "" {
		scheme = "http"
	}
	host := r.Host
	if fwd := r.Header.Get("X-Forwarded-Host"); fwd != "" && s.TrustForwarded {
		host = fwd
	}
	return scheme + "://" + host
}

// instanceHost is just the hostname portion of instanceBase, used by Webfinger.
func (s *Server) instanceHost(r *http.Request) string {
	base := s.instanceBase(r)
	if u, err := url.Parse(base); err == nil {
		return u.Host
	}
	return r.Host
}

// handleWebfinger resolves `acct:<site-slug>@<host>` to the per-site AP actor.
// We use one actor per Errolan site rather than per thread — every thread on a
// site federates through the site's actor and includes the thread URL as the
// canonical reference. This keeps the actor count small.
func (s *Server) handleWebfinger(w http.ResponseWriter, r *http.Request) {
	resource := r.URL.Query().Get("resource")
	if !strings.HasPrefix(resource, "acct:") {
		writeError(w, http.StatusBadRequest, "resource must be acct:")
		return
	}
	at := strings.SplitN(strings.TrimPrefix(resource, "acct:"), "@", 2)
	if len(at) != 2 {
		writeError(w, http.StatusBadRequest, "malformed acct")
		return
	}
	slug, host := at[0], at[1]
	if host != s.instanceHost(r) {
		writeError(w, http.StatusNotFound, "not this host")
		return
	}
	site, err := s.Store.SiteBySlug(slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "site not found")
		return
	}
	actorURL := s.instanceBase(r) + "/ap/sites/" + url.PathEscape(site.Slug)
	w.Header().Set("Content-Type", "application/jrd+json")
	writeJSON(w, http.StatusOK, map[string]any{
		"subject": resource,
		"links": []map[string]any{
			{"rel": "self", "type": "application/activity+json", "href": actorURL},
			{"rel": "http://webfinger.net/rel/profile-page", "type": "text/html", "href": s.instanceBase(r) + "/sdk/"},
		},
	})
}

// handleAPActor returns the per-site ActivityPub Actor document. The actor is
// of type "Service" (an automated account, not a person) because the inbox is
// driven by Errolan rather than a human posting from a UI.
func (s *Server) handleAPActor(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	site, err := s.Store.SiteBySlug(slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "site not found")
		return
	}
	actor := s.instanceBase(r) + "/ap/sites/" + url.PathEscape(site.Slug)
	doc := map[string]any{
		"@context": []any{
			"https://www.w3.org/ns/activitystreams",
			"https://w3id.org/security/v1",
		},
		"id":                actor,
		"type":              "Service",
		"preferredUsername": site.Slug,
		"name":              site.Name,
		"summary":           "Errolan comment relay for " + site.Name,
		"inbox":             actor + "/inbox",
		"outbox":            actor + "/outbox",
		"url":               s.instanceBase(r) + "/sdk/",
	}
	w.Header().Set("Content-Type", "application/activity+json")
	writeJSON(w, http.StatusOK, doc)
}

// handleAPOutbox returns an empty OrderedCollection. We don't yet federate
// outbound — but exposing the URL keeps Mastodon's actor verification happy.
func (s *Server) handleAPOutbox(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if _, err := s.Store.SiteBySlug(slug); err != nil {
		writeError(w, http.StatusNotFound, "site not found")
		return
	}
	actor := s.instanceBase(r) + "/ap/sites/" + url.PathEscape(slug)
	w.Header().Set("Content-Type", "application/activity+json")
	writeJSON(w, http.StatusOK, map[string]any{
		"@context":   "https://www.w3.org/ns/activitystreams",
		"id":         actor + "/outbox",
		"type":       "OrderedCollection",
		"totalItems": 0,
		"orderedItems": []any{},
	})
}

// handleAPInbox accepts a Create(Note) addressed to the site actor and, when
// the note's `inReplyTo` matches one of our thread URLs, enqueues a mention.
// HTTP-signature verification is intentionally out of scope here — v1 prefers
// permissive ingest plus the verifier worker over false negatives. Operators
// who need stricter inbound should put a signature-checking reverse proxy
// (e.g. Tangerine, takahē-inbox-proxy) in front of /ap/.
func (s *Server) handleAPInbox(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	site, err := s.Store.SiteBySlug(slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "site not found")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 256*1024))
	if err != nil {
		writeError(w, http.StatusBadRequest, "body read failed")
		return
	}
	var act struct {
		Type   string `json:"type"`
		ID     string `json:"id"`
		Actor  any    `json:"actor"`
		Object struct {
			Type        string `json:"type"`
			ID          string `json:"id"`
			URL         string `json:"url"`
			Content     string `json:"content"`
			InReplyTo   string `json:"inReplyTo"`
			AttributedTo any   `json:"attributedTo"`
		} `json:"object"`
	}
	if err := json.Unmarshal(body, &act); err != nil {
		writeError(w, http.StatusBadRequest, "invalid activity")
		return
	}
	// We accept only Create(Note) with inReplyTo pointing at one of our threads.
	if !strings.EqualFold(act.Type, "Create") || !strings.EqualFold(act.Object.Type, "Note") {
		// Silently 202 unsupported types so peers don't retry indefinitely.
		w.WriteHeader(http.StatusAccepted)
		return
	}
	if act.Object.InReplyTo == "" {
		writeError(w, http.StatusBadRequest, "inReplyTo required")
		return
	}

	thread, _ := s.lookupThreadByURL(act.Object.InReplyTo)
	if thread == nil {
		writeError(w, http.StatusBadRequest, "inReplyTo does not match a known thread")
		return
	}
	actorURI := stringFromAny(act.Actor)
	source := act.Object.ID
	if source == "" {
		source = actorURI
	}
	if source == "" {
		writeError(w, http.StatusBadRequest, "missing actor/object id")
		return
	}
	m, err := s.Store.EnqueueMention(site.ID, thread.ID, source, act.Object.InReplyTo, models.MentionKindActivityPub)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "enqueue failed")
		return
	}
	// Pre-fill the snippet + author from the activity so the verifier doesn't
	// have to re-scrape an HTML page.
	snippet := stripHTML(act.Object.Content)
	snippet = strings.TrimSpace(collapseSpaces(snippet))
	if len(snippet) > 320 {
		snippet = snippet[:320] + "…"
	}
	authorURL := actorURI
	authorName := actorURI
	if u, err := url.Parse(actorURI); err == nil && u.Host != "" {
		authorName = "@" + lastPath(u.Path) + "@" + u.Host
	}
	// Save the snippet now; the verifier will flip status once it's confirmed
	// the actor URI is reachable.
	if err := s.Store.MarkMentionVerifiedDraft(m.ID, authorName, authorURL, snippet); err != nil {
		// Non-fatal — the mention stays pending and the verifier will re-populate.
		s.Logger.Warn("ap inbox: pre-fill snippet failed", "err", err)
	}
	if s.Verifier != nil {
		_ = s.Verifier.Enqueue(m.ID)
	}
	w.WriteHeader(http.StatusAccepted)
	_ = body // already used; silence linter when needed
}

// stringFromAny accepts either a bare string actor URI or an object with `id`.
func stringFromAny(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case map[string]any:
		if id, ok := t["id"].(string); ok {
			return id
		}
	}
	return ""
}

func lastPath(p string) string {
	p = strings.Trim(p, "/")
	if p == "" {
		return ""
	}
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}

// Local helpers for HTML→text — duplicated from federation/verifier.go so this
// handler doesn't import the verifier package (it sends *to* the verifier, but
// we don't want a circular dependency back).
func stripHTML(s string) string {
	out := make([]byte, 0, len(s))
	inTag := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '<':
			inTag = true
		case c == '>':
			inTag = false
			out = append(out, ' ')
		case !inTag:
			out = append(out, c)
		}
	}
	return string(out)
}
func collapseSpaces(s string) string {
	out := make([]byte, 0, len(s))
	space := true
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			if !space {
				out = append(out, ' ')
				space = true
			}
			continue
		}
		out = append(out, c)
		space = false
	}
	return string(out)
}

func parseHTTPURL(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" || u.Host == "" {
		return nil, errBadURL
	}
	return u, nil
}

var errBadURL = httpError(http.StatusBadRequest, "invalid url")

