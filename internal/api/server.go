package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/enekos/errolan/internal/auth"
	"github.com/enekos/errolan/internal/cache"
	"github.com/enekos/errolan/internal/hub"
	"github.com/enekos/errolan/internal/lockout"
	"github.com/enekos/errolan/internal/ratelimit"
	"github.com/enekos/errolan/internal/store"
)

type Server struct {
	Store     *store.Store
	Auth      *auth.Service
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

	// WebhookURL, when set, receives moderation events as POST JSON.
	WebhookURL string
}

func NewServer(st *store.Store, a *auth.Service) *Server {
	return &Server{
		Store:         st,
		Auth:          a,
		AdminCORS:     "*",
		SiteCache:     cache.New(256, 60*time.Second),
		GlobalLimiter: ratelimit.New(20, 60),       // 20 req/s, burst 60
		AuthLimiter:   ratelimit.New(0.2, 5),       // 1 every 5s, burst 5
		WriteLimiter:  ratelimit.New(0.5, 10),      // 1 every 2s, burst 10
		Lockout:       lockout.New(5, 15*time.Minute, 15*time.Minute),
		Hub:           hub.New(),
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

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
	mux.HandleFunc("POST /api/comments/{id}/pin", s.handlePinComment)       // admin
	mux.HandleFunc("POST /api/comments/{id}/reactions", s.handleReact)      // toggle emoji reaction

	// Emoji pack (per site)
	mux.HandleFunc("GET /api/emojis", s.handleListEmojis)
	mux.HandleFunc("POST /api/emojis", s.handleUpsertEmoji)                 // admin
	mux.HandleFunc("DELETE /api/emojis/{code}", s.handleDeleteEmoji)        // admin

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

	// Order: cors first (handles OPTIONS), then recovery / security / gzip /
	// realIP / rate limit / body limit / site & user resolution.
	return s.cors(
		recoverer(
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

// ----- helpers -----

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
