package api

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/enekos/errolan/internal/models"
	"github.com/enekos/errolan/internal/store"
)

// profileResponse is the public reader-profile payload. Email is deliberately
// NOT exposed; we derive the avatar from either the profile override or a
// Gravatar-style MD5 hash so the actual email never leaves the server.
type profileResponse struct {
	UserID       int64             `json:"user_id"`
	Name         string            `json:"name"`
	AvatarURL    string            `json:"avatar_url,omitempty"`
	Bio          string            `json:"bio,omitempty"`
	Website      string            `json:"website,omitempty"`
	JoinedAt     int64             `json:"joined_at"`
	IsBanned     bool              `json:"is_banned,omitempty"`
	IsAdmin      bool              `json:"is_admin,omitempty"`
	CommentCount int               `json:"comment_count"`
	Comments     []*models.Comment `json:"comments,omitempty"`
}

// handleGetProfile returns the public-facing profile for a user. Anonymous
// (no user_id on comments) commenters can't be profiled — there's no shared
// identity to attach to. Banned / soft-deleted users return a stripped-down
// "[deleted]" payload rather than 404 so existing comment links keep working.
func (s *Server) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	id, err := parseInt64Path(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	u, err := s.Store.UserByID(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	resp := profileResponse{
		UserID:   u.ID,
		Name:     u.Name,
		JoinedAt: u.CreatedAt,
		IsBanned: u.IsBanned,
		IsAdmin:  u.IsAdmin,
	}

	// Banned-or-deleted: redact and return early. We keep the row so links
	// resolve, but a strangers shouldn't see history.
	if u.IsBanned {
		resp.Name = "[deleted]"
		resp.IsAdmin = false
		writeJSON(w, http.StatusOK, resp)
		return
	}

	// Profile overrides (bio, custom avatar). Missing row = use defaults.
	if p, err := s.Store.ProfileByUserID(u.ID); err == nil {
		resp.Bio = p.Bio
		resp.Website = p.Website
		if p.AvatarURL != "" {
			resp.AvatarURL = p.AvatarURL
		}
	} else if err != store.ErrNotFound {
		writeError(w, http.StatusInternalServerError, "profile load failed")
		return
	}
	if resp.AvatarURL == "" {
		resp.AvatarURL = gravatarFromEmail(u.Email)
	}

	if n, err := s.Store.CountCommentsByUser(u.ID); err == nil {
		resp.CommentCount = n
	}
	if cs, err := s.Store.CommentsByUser(u.ID, 50); err == nil {
		resp.Comments = cs
	}
	writeJSON(w, http.StatusOK, resp)
}

// gravatarFromEmail is the same hash the store uses for inline comment
// avatars. Duplicating it here (rather than exporting from store) keeps the
// avatar policy in one place per concern — comment list vs profile.
func gravatarFromEmail(email string) string {
	if email == "" {
		return ""
	}
	sum := md5.Sum([]byte(strings.ToLower(strings.TrimSpace(email))))
	return fmt.Sprintf("https://www.gravatar.com/avatar/%s?s=160&d=mp", hex.EncodeToString(sum[:]))
}

// handleUpdateMyProfile lets the authenticated user edit their bio/website.
// We never let the user change their name here — that's tied to the auth
// record and would shift attribution on existing comments.
func (s *Server) handleUpdateMyProfile(w http.ResponseWriter, r *http.Request) {
	user, err := requireUser(r)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	var req struct {
		Bio       string `json:"bio"`
		Website   string `json:"website"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	req.Bio = strings.TrimSpace(req.Bio)
	req.Website = strings.TrimSpace(req.Website)
	req.AvatarURL = strings.TrimSpace(req.AvatarURL)
	if len(req.Bio) > 500 {
		writeError(w, http.StatusBadRequest, "bio too long (max 500)")
		return
	}
	if req.Website != "" {
		if _, perr := parseHTTPURL(req.Website); perr != nil {
			writeError(w, http.StatusBadRequest, "website must be a http(s) URL")
			return
		}
	}
	if req.AvatarURL != "" {
		if _, perr := parseHTTPURL(req.AvatarURL); perr != nil {
			writeError(w, http.StatusBadRequest, "avatar_url must be a http(s) URL")
			return
		}
	}
	p := &models.UserProfile{
		UserID:    user.ID,
		Bio:       req.Bio,
		Website:   req.Website,
		AvatarURL: req.AvatarURL,
	}
	if err := s.Store.UpsertProfile(p); err != nil {
		writeError(w, http.StatusInternalServerError, "save failed")
		return
	}
	writeJSON(w, http.StatusOK, p)
}
