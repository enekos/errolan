package api

import (
	"net/http"
	"strings"

	"github.com/enekos/errolan/internal/auth"
	"github.com/enekos/errolan/internal/store"
)

type registerReq struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

type authResp struct {
	Token string `json:"token"`
	User  struct {
		ID      int64  `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		IsAdmin bool   `json:"is_admin"`
	} `json:"user"`
}

// validPassword rejects trivially weak passwords. We're not chasing NIST
// compliance, but at least disallow short, blank, or email-matching passwords.
func validPassword(pw, email string) bool {
	pw = strings.TrimSpace(pw)
	if len(pw) < 8 || len(pw) > 256 {
		return false
	}
	if strings.EqualFold(pw, email) {
		return false
	}
	return true
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if !s.AuthLimiter.Allow("ip:" + clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, "slow down")
		return
	}
	var req registerReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Name = strings.TrimSpace(req.Name)
	if !strings.Contains(req.Email, "@") || len(req.Email) > 254 || req.Name == "" || len(req.Name) > 80 {
		writeError(w, http.StatusBadRequest, "valid email and name (<=80 chars) required")
		return
	}
	if !validPassword(req.Password, req.Email) {
		writeError(w, http.StatusBadRequest, "password must be 8-256 chars and not equal to your email")
		return
	}
	if _, err := s.Store.UserByEmail(req.Email); err == nil {
		writeError(w, http.StatusConflict, "email already registered")
		return
	} else if err != store.ErrNotFound {
		writeError(w, http.StatusInternalServerError, "lookup failed")
		return
	}
	hash, err := auth.Hash(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "hash failed")
		return
	}
	u, err := s.Store.CreateUser(req.Email, req.Name, hash, false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create failed")
		return
	}
	tok, err := s.Auth.Issue(u.ID, u.IsAdmin)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token issue failed")
		return
	}
	resp := authResp{Token: tok}
	resp.User.ID = u.ID
	resp.User.Email = u.Email
	resp.User.Name = u.Name
	resp.User.IsAdmin = u.IsAdmin
	writeJSON(w, http.StatusCreated, resp)
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if !s.AuthLimiter.Allow("ip:" + ip) {
		writeError(w, http.StatusTooManyRequests, "slow down")
		return
	}
	var req loginReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	// Lockout key is per-account; we don't expose whether the email exists.
	lockKey := "email:" + req.Email
	if !s.Lockout.Allowed(lockKey) {
		writeError(w, http.StatusTooManyRequests, "too many attempts; try again later")
		return
	}

	u, err := s.Store.UserByEmail(req.Email)
	if err != nil || !auth.Verify(u.PasswordHash, req.Password) {
		if locked := s.Lockout.Failure(lockKey); locked {
			writeError(w, http.StatusTooManyRequests, "account temporarily locked")
			return
		}
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if u.IsBanned {
		writeError(w, http.StatusForbidden, "account banned")
		return
	}
	s.Lockout.Success(lockKey)
	tok, err := s.Auth.Issue(u.ID, u.IsAdmin)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token issue failed")
		return
	}
	resp := authResp{Token: tok}
	resp.User.ID = u.ID
	resp.User.Email = u.Email
	resp.User.Name = u.Name
	resp.User.IsAdmin = u.IsAdmin
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	if u == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":       u.ID,
		"email":    u.Email,
		"name":     u.Name,
		"is_admin": u.IsAdmin,
	})
}
