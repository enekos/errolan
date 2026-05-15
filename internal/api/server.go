package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/enekos/errolan/internal/auth"
	"github.com/enekos/errolan/internal/cache"
	"github.com/enekos/errolan/internal/hub"
	"github.com/enekos/errolan/internal/lockout"
	"github.com/enekos/errolan/internal/ratelimit"
	"github.com/enekos/errolan/internal/store"
	"github.com/enekos/errolan/internal/webhook"
)

// Server wires HTTP handlers to the store, auth, hub, caches, and limiters.
// Construction lives in NewServer; fields are exposed so tests / main can
// override defaults (e.g. swap in deterministic limiters).
type Server struct {
	Store     *store.Store
	Auth      *auth.Service
	Logger    *slog.Logger
	Webhook   *webhook.Notifier
	AdminCORS string
	SDKDir    string

	// TrustForwarded says X-Forwarded-For / X-Real-IP may be trusted (i.e. we
	// know we're behind a reverse proxy). Off by default — only enable when
	// the proxy strips spoofed headers.
	TrustForwarded bool

	// Caches and limiters.
	SiteCache     *cache.TTL
	GlobalLimiter *ratelimit.Limiter // per-IP, all requests
	AuthLimiter   *ratelimit.Limiter // per-IP, /api/auth/*
	WriteLimiter  *ratelimit.Limiter // per-IP, comment writes
	Lockout       *lockout.Tracker   // per-account login lockout

	Hub *hub.Hub
}

// ServerOptions bundles the cross-cutting dependencies that come from main.go
// or a test. Anything not set falls back to safe defaults inside NewServer.
type ServerOptions struct {
	AdminCORS      string
	SDKDir         string
	TrustForwarded bool
	WebhookURL     string
	Logger         *slog.Logger
}

func NewServer(st *store.Store, a *auth.Service, opts ServerOptions) *Server {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.AdminCORS == "" {
		opts.AdminCORS = "*"
	}
	return &Server{
		Store:          st,
		Auth:           a,
		Logger:         opts.Logger,
		Webhook:        webhook.New(opts.WebhookURL, opts.Logger),
		AdminCORS:      opts.AdminCORS,
		SDKDir:         opts.SDKDir,
		TrustForwarded: opts.TrustForwarded,
		SiteCache:      cache.New(256, 60*time.Second),
		GlobalLimiter:  ratelimit.New(20, 60),  // 20 req/s, burst 60
		AuthLimiter:    ratelimit.New(0.2, 5),  // 1 every 5s, burst 5
		WriteLimiter:   ratelimit.New(0.5, 10), // 1 every 2s, burst 10
		Lockout:        lockout.New(5, 15*time.Minute, 15*time.Minute),
		Hub:            hub.New(),
	}
}

// notifyWebhook is a thin shim so handler code can keep the old call site.
// All real work — fire-and-forget POST, timeout, logging — lives in the
// webhook package.
func (s *Server) notifyWebhook(payload map[string]any) {
	s.Webhook.Send(payload)
}

// Handler builds the request-handling pipeline. Order: cors first (handles
// OPTIONS), then recovery / security / gzip / realIP / rate limit / body limit
// / site & user resolution / mux.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	s.registerRoutes(mux)

	return s.cors(
		recoverer(s.Logger)(
			securityHeaders(
				compress(
					s.realIP(
						s.rateLimit(
							limitBody(
								s.resolveSite(s.resolveUser(mux)),
							),
						),
					),
				),
			),
		),
	)
}

// registerRoutes mounts every API endpoint on the supplied mux. Grouping the
// declarations in one place makes the public surface obvious at a glance.
func (s *Server) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Auth
	mux.HandleFunc("POST /api/auth/register", s.handleRegister)
	mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	mux.HandleFunc("GET /api/auth/me", s.handleMe)

	// Sites (admin)
	mux.HandleFunc("GET /api/sites", s.handleListSites)
	mux.HandleFunc("POST /api/sites", s.handleCreateSite)

	// Threads
	mux.HandleFunc("GET /api/threads/{slug}", s.handleGetThread)
	mux.HandleFunc("POST /api/threads/{slug}/lock", s.handleLockThread)
	mux.HandleFunc("POST /api/threads/{slug}/comments", s.handleCreateComment)
	mux.HandleFunc("GET /api/threads/{slug}/events", s.handleThreadEvents) // SSE

	// Comments
	mux.HandleFunc("PATCH /api/comments/{id}", s.handleEditComment)
	mux.HandleFunc("DELETE /api/comments/{id}", s.handleDeleteComment)
	mux.HandleFunc("POST /api/comments/{id}/vote", s.handleVote)
	mux.HandleFunc("POST /api/comments/{id}/flag", s.handleFlag)
	mux.HandleFunc("POST /api/comments/{id}/pin", s.handlePinComment)  // admin
	mux.HandleFunc("POST /api/comments/{id}/reactions", s.handleReact) // toggle emoji reaction

	// Emoji pack (per site)
	mux.HandleFunc("GET /api/emojis", s.handleListEmojis)
	mux.HandleFunc("POST /api/emojis", s.handleUpsertEmoji)          // admin
	mux.HandleFunc("DELETE /api/emojis/{code}", s.handleDeleteEmoji) // admin

	// Admin: users + audit
	mux.HandleFunc("GET /api/admin/users", s.handleListUsers)
	mux.HandleFunc("POST /api/admin/users/{id}/ban", s.handleBanUser)
	mux.HandleFunc("GET /api/mod/flagged", s.handleListFlagged)
	mux.HandleFunc("GET /api/mod/audit", s.handleAuditLog)

	// Moderation: per-site policy, blocklist, hold queue
	mux.HandleFunc("GET /api/sites/{slug}/moderation", s.handleGetModeration)
	mux.HandleFunc("PATCH /api/sites/{slug}/moderation", s.handleUpdateModeration)
	mux.HandleFunc("GET /api/sites/{slug}/moderation/blocklist", s.handleListBlocklist)
	mux.HandleFunc("POST /api/sites/{slug}/moderation/blocklist", s.handleAddBlocklist)
	mux.HandleFunc("DELETE /api/sites/{slug}/moderation/blocklist/{id}", s.handleDeleteBlocklist)
	mux.HandleFunc("GET /api/mod/queue", s.handleModQueue)
	mux.HandleFunc("POST /api/comments/{id}/approve", s.handleApproveComment)
	mux.HandleFunc("POST /api/comments/{id}/reject", s.handleRejectComment)

	// SDK static
	if s.SDKDir != "" {
		fs := http.FileServer(http.Dir(s.SDKDir))
		mux.Handle("GET /sdk/", http.StripPrefix("/sdk/", fs))
	}
}

// ----- response helpers -----

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decode(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}
