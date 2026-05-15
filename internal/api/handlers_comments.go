package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/enekos/errolan/internal/models"
	"github.com/enekos/errolan/internal/moderation"
	"github.com/enekos/errolan/internal/store"
)

func (s *Server) handleEditComment(w http.ResponseWriter, r *http.Request) {
	user, err := requireUser(r)
	if err != nil {
		writeAPIError(w, err)
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
	if !user.IsAdmin && (c.UserID == nil || *c.UserID != user.ID) {
		writeError(w, http.StatusForbidden, "not your comment")
		return
	}
	var req struct {
		Body string `json:"body"`
	}
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	body := strings.TrimSpace(req.Body)
	if body == "" || len(body) > 8000 {
		writeError(w, http.StatusBadRequest, "invalid body length")
		return
	}

	// Re-evaluate the edited body so a user can't post clean text then edit in
	// blocked keywords / links to bypass the engine. Admins are exempt from the
	// engine the same way they are on create.
	thread, err := s.Store.ThreadByID(c.ThreadID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "thread lookup failed")
		return
	}
	decision := s.evaluateComment(thread.SiteID, user, c.UserID == nil, body)

	if decision.Action == moderation.ActionReject {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
			"error":  "edit rejected by moderation",
			"reason": decision.Reason,
		})
		return
	}

	if err := s.Store.UpdateCommentBody(id, body); err != nil {
		writeError(w, http.StatusInternalServerError, "update failed")
		return
	}

	// Hold-on-edit: move a previously visible comment back into the queue. The
	// denormalised thread counter is corrected by SetCommentStatus. Pending or
	// hidden comments stay where they are.
	if decision.Action == moderation.ActionHold && c.Status == models.CommentStatusVisible {
		if err := s.Store.SetCommentStatus(id, models.CommentStatusPending); err == nil {
			_ = s.Store.SetModerationReason(id, decision.Reason)
			s.Store.AddAudit(&user.ID, user.Name, "comment.hold_edit", "comment", id, decision.Reason)
			s.Hub.Publish(c.ThreadID, "edit")
			s.notifyWebhook(map[string]any{
				"event":      "comment.pending",
				"comment_id": id,
				"reason":     decision.Reason,
				"trigger":    "edit",
			})
			writeJSON(w, http.StatusAccepted, map[string]any{
				"status": "pending",
				"reason": decision.Reason,
			})
			return
		}
	}

	if user.IsAdmin && (c.UserID == nil || *c.UserID != user.ID) {
		s.Store.AddAudit(&user.ID, user.Name, "comment.edit", "comment", id, "")
	}
	s.Hub.Publish(c.ThreadID, "edit")
	updated, _ := s.Store.CommentByID(id, &user.ID)
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleDeleteComment(w http.ResponseWriter, r *http.Request) {
	user, err := requireUser(r)
	if err != nil {
		writeAPIError(w, err)
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
	if !user.IsAdmin && (c.UserID == nil || *c.UserID != user.ID) {
		writeError(w, http.StatusForbidden, "not your comment")
		return
	}
	if err := s.Store.SetCommentStatus(id, models.CommentStatusDeleted); err != nil {
		writeError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	if user.IsAdmin && (c.UserID == nil || *c.UserID != user.ID) {
		s.Store.AddAudit(&user.ID, user.Name, "comment.delete", "comment", id, "")
		s.notifyWebhook(map[string]any{"event": "comment.delete", "comment_id": id, "actor": user.Email})
	}
	s.Hub.Publish(c.ThreadID, "delete")
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleVote(w http.ResponseWriter, r *http.Request) {
	user, err := requireUser(r)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	id, err := parseInt64Path(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req struct {
		Value int `json:"value"`
	}
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.Value < -1 || req.Value > 1 {
		writeError(w, http.StatusBadRequest, "value must be -1, 0, or 1")
		return
	}
	c, err := s.Store.CommentByID(id, nil)
	if err != nil {
		writeError(w, http.StatusNotFound, "comment not found")
		return
	}
	score, err := s.Store.Vote(user.ID, id, req.Value)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "vote failed")
		return
	}
	s.Hub.Publish(c.ThreadID, "vote")
	writeJSON(w, http.StatusOK, map[string]any{"score": score, "my_vote": req.Value})
}

func (s *Server) handleFlag(w http.ResponseWriter, r *http.Request) {
	if !s.WriteLimiter.Allow("flag:" + clientIP(r)) {
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
	var req struct {
		Reason string `json:"reason"`
	}
	_ = decode(r, &req)
	reason := strings.TrimSpace(req.Reason)
	if len(reason) > 500 {
		reason = reason[:500]
	}
	var userID *int64
	if u := userFrom(r); u != nil {
		userID = &u.ID
	}
	if err := s.Store.Flag(id, userID, reason); err != nil {
		if err == store.ErrConflict {
			// Idempotent: already flagged by this user.
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeError(w, http.StatusInternalServerError, "flag failed")
		return
	}
	s.notifyWebhook(map[string]any{"event": "comment.flag", "comment_id": id, "thread_id": c.ThreadID})

	// Auto-hide-on-flag-threshold: when the site policy sets a non-zero
	// auto_hide_flag_count, hide a still-visible comment once enough distinct
	// flags have landed. Best-effort — failures don't block the flag response.
	if c.Status == models.CommentStatusVisible {
		if thread, err := s.Store.ThreadByID(c.ThreadID); err == nil {
			if settings, err := s.Store.ModerationSettings(thread.SiteID); err == nil && settings.AutoHideFlagCount > 0 {
				if count, err := s.Store.CountFlags(id); err == nil && count >= settings.AutoHideFlagCount {
					if err := s.Store.SetCommentStatus(id, models.CommentStatusHidden); err == nil {
						s.Store.AddAudit(nil, "system", "comment.auto_hide", "comment", id, "flag threshold ("+strconv.Itoa(count)+")")
						s.Hub.Publish(c.ThreadID, "hide")
					}
				}
			}
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handlePinComment(w http.ResponseWriter, r *http.Request) {
	admin, err := requireAdmin(r)
	if err != nil {
		writeAPIError(w, err)
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
	var req struct {
		Pinned bool `json:"pinned"`
	}
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := s.Store.SetCommentPinned(id, req.Pinned); err != nil {
		writeError(w, http.StatusInternalServerError, "update failed")
		return
	}
	s.Store.AddAudit(&admin.ID, admin.Name, "comment.pin", "comment", id, fmt.Sprintf(`{"pinned":%t}`, req.Pinned))
	s.Hub.Publish(c.ThreadID, "pin")
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListFlagged(w http.ResponseWriter, r *http.Request) {
	if _, err := requireAdmin(r); err != nil {
		writeAPIError(w, err)
		return
	}
	flagged, err := s.Store.ListFlagged(100)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list failed")
		return
	}
	writeJSON(w, http.StatusOK, flagged)
}
