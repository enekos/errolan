package api

import (
	"compress/gzip"
	"context"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/enekosarasola/errolan/internal/models"
)

type ctxKey int

const (
	ctxKeyUser ctxKey = iota
	ctxKeySite
	ctxKeyIP
)

// Maximum request body size for non-streaming endpoints. Comment bodies are
// capped at 8 KB at the handler layer; this is a safety net for everything else.
const maxRequestBody = 64 * 1024

func recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic: %v\n%s", rec, debug.Stack())
				writeError(w, http.StatusInternalServerError, "internal error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// realIP extracts the trusted client IP. If trustForwarded is true, we honor
// X-Forwarded-For / X-Real-IP from a proxy — otherwise we use RemoteAddr.
// The result is stashed in context for downstream rate limiters.
func (s *Server) realIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := remoteHost(r.RemoteAddr)
		if s.TrustForwarded {
			if h := r.Header.Get("X-Forwarded-For"); h != "" {
				// take the first (left-most) entry — the original client
				parts := strings.Split(h, ",")
				ip = strings.TrimSpace(parts[0])
			} else if h := r.Header.Get("X-Real-IP"); h != "" {
				ip = strings.TrimSpace(h)
			}
		}
		ctx := context.WithValue(r.Context(), ctxKeyIP, ip)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func remoteHost(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}

func clientIP(r *http.Request) string {
	if v, ok := r.Context().Value(ctxKeyIP).(string); ok {
		return v
	}
	return remoteHost(r.RemoteAddr)
}

// securityHeaders is a small set of conservative defaults. The SDK is loaded
// cross-origin, so we don't set X-Frame-Options on /sdk/.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer-when-downgrade")
		// Disable client features the API never legitimately needs.
		h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		if !strings.HasPrefix(r.URL.Path, "/sdk/") {
			h.Set("X-Frame-Options", "DENY")
		}
		next.ServeHTTP(w, r)
	})
}

// limitBody wraps the request body in a MaxBytesReader. Streaming endpoints
// (SSE) bypass this by being mounted before the middleware composition.
func limitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil && r.ContentLength > maxRequestBody {
			writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
		next.ServeHTTP(w, r)
	})
}

// ----- gzip -----

type gzipResponseWriter struct {
	http.ResponseWriter
	gz *gzip.Writer
}

func (g *gzipResponseWriter) Write(p []byte) (int, error) { return g.gz.Write(p) }

var gzipPool = sync.Pool{
	New: func() any {
		w, _ := gzip.NewWriterLevel(io.Discard, gzip.DefaultCompression)
		return w
	},
}

func compress(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// SSE must not be gzipped — chunked event delivery would buffer.
		if strings.Contains(r.URL.Path, "/events") {
			next.ServeHTTP(w, r)
			return
		}
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		gz := gzipPool.Get().(*gzip.Writer)
		gz.Reset(w)
		defer func() {
			gz.Close()
			gzipPool.Put(gz)
		}()
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Add("Vary", "Accept-Encoding")
		w.Header().Del("Content-Length")
		next.ServeHTTP(&gzipResponseWriter{ResponseWriter: w, gz: gz}, r)
	})
}

// ----- rate limiting -----

// rateLimit applies one bucket to all requests on this composition. Per-route
// limits (auth, comment posting) are applied at the handler level.
func (s *Server) rateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := "ip:" + clientIP(r)
		if !s.GlobalLimiter.Allow(key) {
			w.Header().Set("Retry-After", "30")
			writeError(w, http.StatusTooManyRequests, "too many requests")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := s.originAllowed(r, origin)
		if allowed != "" {
			w.Header().Set("Access-Control-Allow-Origin", allowed)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Errolan-Site, If-None-Match")
		w.Header().Set("Access-Control-Expose-Headers", "ETag")
		w.Header().Set("Access-Control-Max-Age", "86400")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) originAllowed(r *http.Request, origin string) string {
	if origin == "" {
		return ""
	}
	siteKey := r.Header.Get("X-Errolan-Site")
	if siteKey != "" {
		site := s.siteByKey(siteKey)
		if site != nil {
			if site.AllowedOrigins == "*" {
				return origin
			}
			for _, o := range strings.Split(site.AllowedOrigins, ",") {
				if strings.TrimSpace(o) == origin {
					return origin
				}
			}
			return ""
		}
	}
	if s.AdminCORS == "*" {
		return origin
	}
	for _, o := range strings.Split(s.AdminCORS, ",") {
		if strings.TrimSpace(o) == origin {
			return origin
		}
	}
	return ""
}

// siteByKey is the cached lookup used by CORS and resolveSite. Sites are stable
// for the lifetime of a request, so a short TTL cache is safe.
func (s *Server) siteByKey(key string) *models.Site {
	if v, ok := s.SiteCache.Get(key); ok {
		return v.(*models.Site)
	}
	site, err := s.Store.SiteByAPIKey(key)
	if err != nil {
		return nil
	}
	s.SiteCache.Set(key, site)
	return site
}

func (s *Server) resolveUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if strings.HasPrefix(header, "Bearer ") {
			raw := strings.TrimPrefix(header, "Bearer ")
			claims, err := s.Auth.Parse(raw)
			if err == nil {
				u, err := s.Store.UserByID(claims.UserID)
				if err == nil && !u.IsBanned {
					r = r.WithContext(context.WithValue(r.Context(), ctxKeyUser, u))
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) resolveSite(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-Errolan-Site")
		// EventSource cannot set headers cross-origin, so accept the site key
		// from the query string for safe (GET) requests. The site key is
		// considered public (like a Disqus shortname), so this leaks nothing.
		if key == "" && r.Method == http.MethodGet {
			key = r.URL.Query().Get("site")
		}
		if key != "" {
			if site := s.siteByKey(key); site != nil {
				r = r.WithContext(context.WithValue(r.Context(), ctxKeySite, site))
			}
		}
		next.ServeHTTP(w, r)
	})
}

func userFrom(r *http.Request) *models.User {
	u, _ := r.Context().Value(ctxKeyUser).(*models.User)
	return u
}

func siteFrom(r *http.Request) *models.Site {
	s, _ := r.Context().Value(ctxKeySite).(*models.Site)
	return s
}

func requireUser(r *http.Request) (*models.User, error) {
	u := userFrom(r)
	if u == nil {
		return nil, errors.New("authentication required")
	}
	return u, nil
}

func requireSite(r *http.Request) (*models.Site, error) {
	s := siteFrom(r)
	if s == nil {
		return nil, errors.New("X-Errolan-Site header required")
	}
	return s, nil
}

func requireAdmin(r *http.Request) (*models.User, error) {
	u, err := requireUser(r)
	if err != nil {
		return nil, err
	}
	if !u.IsAdmin {
		return nil, errors.New("admin only")
	}
	return u, nil
}
