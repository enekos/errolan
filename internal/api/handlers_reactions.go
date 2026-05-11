package api

import (
	"net/http"
	"strings"

	"github.com/enekosarasola/errolan/internal/store"
)

// validEmojiCode keeps codes to lowercase ASCII slugs so they're safe as path
// segments and JSON keys, and so users can't sneak HTML in via `:code:`.
func validEmojiCode(code string) bool {
	if len(code) < 1 || len(code) > 40 {
		return false
	}
	for _, r := range code {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_'
		if !ok {
			return false
		}
	}
	return true
}

// ----- Reactions -----

func (s *Server) handleReact(w http.ResponseWriter, r *http.Request) {
	user, err := requireUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	if !s.WriteLimiter.Allow("react:" + clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, "slow down")
		return
	}
	id, err := parseInt64Path(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	c, err := s.Store.CommentByID(id, nil)
	if err != nil {
		writeError(w, http.StatusNotFound, "comment not found")
		return
	}
	thread, err := s.Store.ThreadByID(c.ThreadID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "thread lookup failed")
		return
	}
	var req struct {
		Code string `json:"code"`
	}
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	code := strings.TrimSpace(req.Code)
	if !validEmojiCode(code) {
		writeError(w, http.StatusBadRequest, "invalid emoji code")
		return
	}
	// The code must be in this site's pack — prevents users from inventing
	// arbitrary reaction names that don't render in the SDK.
	allowed, err := s.Store.CodesForSite(thread.SiteID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "pack lookup failed")
		return
	}
	if _, ok := allowed[code]; !ok {
		writeError(w, http.StatusBadRequest, "emoji not in this site's pack")
		return
	}
	count, active, err := s.Store.ToggleReaction(user.ID, id, code)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "react failed")
		return
	}
	s.Hub.Publish(c.ThreadID, "react")
	writeJSON(w, http.StatusOK, map[string]any{"code": code, "count": count, "active": active})
}

// ----- Emoji pack -----

func (s *Server) handleListEmojis(w http.ResponseWriter, r *http.Request) {
	site, err := requireSite(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	emojis, err := s.Store.ListEmojis(site.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list failed")
		return
	}
	writeJSON(w, http.StatusOK, emojis)
}

type emojiReq struct {
	Code  string `json:"code"`
	Label string `json:"label"`
	SVG   string `json:"svg"`
	Sort  int    `json:"sort"`
}

// handleUpsertEmoji adds or updates an emoji on the site identified by
// `X-Errolan-Site`. Requires admin auth.
func (s *Server) handleUpsertEmoji(w http.ResponseWriter, r *http.Request) {
	admin, err := requireAdmin(r)
	if err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	site, err := requireSite(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var req emojiReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if !validEmojiCode(req.Code) {
		writeError(w, http.StatusBadRequest, "invalid code (lowercase a-z, 0-9, - or _, ≤40 chars)")
		return
	}
	if len(req.Label) > 80 {
		writeError(w, http.StatusBadRequest, "label too long")
		return
	}
	// SVG: either a https URL or inline SVG markup. Cap at 32 KB so a rogue
	// admin can't fill the DB with megabyte-sized blobs.
	svg := strings.TrimSpace(req.SVG)
	if svg == "" {
		writeError(w, http.StatusBadRequest, "svg required")
		return
	}
	if len(svg) > 32*1024 {
		writeError(w, http.StatusBadRequest, "svg too large (max 32 KB)")
		return
	}
	if !strings.HasPrefix(svg, "<svg") && !strings.HasPrefix(svg, "https://") {
		writeError(w, http.StatusBadRequest, "svg must be inline <svg…> markup or an https:// URL")
		return
	}
	e, err := s.Store.UpsertEmoji(site.ID, req.Code, req.Label, svg, req.Sort)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "save failed")
		return
	}
	s.Store.AddAudit(&admin.ID, admin.Name, "emoji.upsert", "emoji", e.ID, e.Code)
	writeJSON(w, http.StatusOK, e)
}

func (s *Server) handleDeleteEmoji(w http.ResponseWriter, r *http.Request) {
	admin, err := requireAdmin(r)
	if err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	site, err := requireSite(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	code := r.PathValue("code")
	if !validEmojiCode(code) {
		writeError(w, http.StatusBadRequest, "invalid code")
		return
	}
	if err := s.Store.DeleteEmoji(site.ID, code); err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	s.Store.AddAudit(&admin.ID, admin.Name, "emoji.delete", "emoji", 0, code)
	w.WriteHeader(http.StatusNoContent)
}
