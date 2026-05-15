package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	if _, err := requireAdmin(r); err != nil {
		writeAPIError(w, err)
		return
	}
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	users, err := s.Store.ListUsers(limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list failed")
		return
	}
	// Redact password hashes (already excluded by JSON tag, but be explicit).
	for _, u := range users {
		u.PasswordHash = ""
	}
	writeJSON(w, http.StatusOK, users)
}

func (s *Server) handleBanUser(w http.ResponseWriter, r *http.Request) {
	admin, err := requireAdmin(r)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	id, err := parseInt64Path(r, "id")
	if err != nil {
		writeAPIError(w, errInvalidID)
		return
	}
	if id == admin.ID {
		writeError(w, http.StatusBadRequest, "cannot ban yourself")
		return
	}
	target, err := s.Store.UserByID(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	if target.IsAdmin {
		writeError(w, http.StatusForbidden, "cannot ban another admin")
		return
	}
	var req struct {
		Banned bool `json:"banned"`
	}
	if err := decode(r, &req); err != nil {
		writeAPIError(w, errInvalidBody)
		return
	}
	if err := s.Store.SetUserBanned(id, req.Banned); err != nil {
		writeError(w, http.StatusInternalServerError, "update failed")
		return
	}
	s.Store.AddAudit(&admin.ID, admin.Name, "user.ban", "user", id, fmt.Sprintf(`{"banned":%t}`, req.Banned))
	s.notifyWebhook(map[string]any{"event": "user.ban", "user_id": id, "banned": req.Banned, "actor": admin.Email})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAuditLog(w http.ResponseWriter, r *http.Request) {
	if _, err := requireAdmin(r); err != nil {
		writeAPIError(w, err)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	entries, err := s.Store.ListAudit(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list failed")
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

// handleThreadEvents serves a Server-Sent Events stream of update notifications
// for one thread. Clients should re-fetch the thread on each event.
func (s *Server) handleThreadEvents(w http.ResponseWriter, r *http.Request) {
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
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache, no-store, no-transform")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")

	events, cancel := s.Hub.Subscribe(thread.ID)
	defer cancel()

	// Initial hello so the client knows the stream is live.
	fmt.Fprintf(w, "event: ready\ndata: {}\n\n")
	flusher.Flush()

	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case ev, ok := <-events:
			if !ok {
				return
			}
			fmt.Fprintf(w, "event: update\ndata: {\"kind\":%q}\n\n", ev)
			flusher.Flush()
		case <-heartbeat.C:
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}
