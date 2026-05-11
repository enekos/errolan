package api

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/enekosarasola/errolan/internal/models"
	"github.com/enekosarasola/errolan/internal/store"
)

type threadResp struct {
	Thread   *models.Thread    `json:"thread"`
	Site     map[string]any    `json:"site"`
	Comments []*models.Comment `json:"comments"`
	HasMore  bool              `json:"has_more"`
	NextID   *int64            `json:"next_id,omitempty"`
	Sort     string            `json:"sort"`
	Viewer   map[string]any    `json:"viewer,omitempty"`
	Emojis   []*models.Emoji   `json:"emojis"`
}

func parseSort(s string) store.SortOrder {
	switch strings.ToLower(s) {
	case "newest":
		return store.SortNewest
	case "oldest":
		return store.SortOldest
	default:
		return store.SortBest
	}
}

func threadETag(t *models.Thread) string {
	raw := fmt.Sprintf("%d:%d:%d", t.ID, t.CommentCount, t.LastCommentAt)
	sum := sha1.Sum([]byte(raw))
	return `W/"` + hex.EncodeToString(sum[:8]) + `"`
}

func (s *Server) handleGetThread(w http.ResponseWriter, r *http.Request) {
	site, err := requireSite(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	slug := r.PathValue("slug")
	if slug == "" {
		writeError(w, http.StatusBadRequest, "thread slug required")
		return
	}
	q := r.URL.Query()
	title := q.Get("title")
	urlStr := q.Get("url")

	thread, err := s.Store.GetOrCreateThread(site.ID, slug, title, urlStr)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "thread load failed")
		return
	}

	// ETag short-circuit. Only valid for the default sort/page; we include
	// the sort+cursor in the response payload anyway so this still saves work
	// for the common case (auto-mounted widget polling).
	sortParam := q.Get("sort")
	beforeID, _ := strconv.ParseInt(q.Get("before_id"), 10, 64)
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	// Viewer scopes the ETag because my_vote varies by user.
	viewer := userFrom(r)
	tag := threadETag(thread)
	if viewer != nil {
		tag = `W/"u` + strconv.FormatInt(viewer.ID, 10) + ":" + strings.Trim(tag, `W/"`) + `"`
	}
	if sortParam == "" && beforeID == 0 && r.Header.Get("If-None-Match") == tag {
		w.Header().Set("ETag", tag)
		w.WriteHeader(http.StatusNotModified)
		return
	}

	var viewerID *int64
	if viewer != nil {
		id := viewer.ID
		viewerID = &id
	}
	comments, hasMore, err := s.Store.ListThreadComments(thread.ID, store.ListCommentsOpts{
		Sort:     parseSort(sortParam),
		Limit:    limit,
		BeforeID: beforeID,
		ViewerID: viewerID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "comment load failed")
		return
	}

	var nextID *int64
	if hasMore && len(comments) > 0 {
		id := comments[len(comments)-1].ID
		nextID = &id
	}

	emojis, _ := s.Store.ListEmojis(site.ID)
	resp := threadResp{
		Thread:   thread,
		Comments: comments,
		HasMore:  hasMore,
		NextID:   nextID,
		Sort:     string(parseSort(sortParam)),
		Site: map[string]any{
			"slug":         site.Slug,
			"name":         site.Name,
			"require_auth": site.RequireAuth,
		},
		Emojis: emojis,
	}
	if viewer != nil {
		resp.Viewer = map[string]any{
			"id":       viewer.ID,
			"name":     viewer.Name,
			"email":    viewer.Email,
			"is_admin": viewer.IsAdmin,
		}
	}
	w.Header().Set("ETag", tag)
	w.Header().Set("Cache-Control", "private, no-cache")
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleLockThread(w http.ResponseWriter, r *http.Request) {
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
	slug := r.PathValue("slug")
	thread, err := s.Store.ThreadBySlug(site.ID, slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "thread not found")
		return
	}
	var body struct {
		Locked bool `json:"locked"`
	}
	if err := decode(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := s.Store.SetThreadLocked(thread.ID, body.Locked); err != nil {
		writeError(w, http.StatusInternalServerError, "update failed")
		return
	}
	s.Store.AddAudit(&admin.ID, admin.Name, "thread.lock", "thread", thread.ID, fmt.Sprintf(`{"locked":%t}`, body.Locked))
	s.Hub.Publish(thread.ID, "lock")
	s.notifyWebhook(map[string]any{"event": "thread.lock", "thread_id": thread.ID, "locked": body.Locked})
	w.WriteHeader(http.StatusNoContent)
}

type createCommentReq struct {
	Body       string `json:"body"`
	ParentID   *int64 `json:"parent_id"`
	AuthorName string `json:"author_name"` // anonymous only
	Honeypot   string `json:"website"`     // must be empty
	Anchor     string `json:"anchor"`      // optional paragraph id (marginalia)
}

// validAnchor caps the size and shape of paragraph anchors. The SDK reads
// these from `data-errolan-anchor="..."` on the host page, so they should be
// short stable slugs — not arbitrary user input. 64 chars, [A-Za-z0-9._:-].
func validAnchor(a string) bool {
	if len(a) > 64 {
		return false
	}
	for _, r := range a {
		ok := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
			r == '-' || r == '_' || r == '.' || r == ':'
		if !ok {
			return false
		}
	}
	return true
}

func (s *Server) handleCreateComment(w http.ResponseWriter, r *http.Request) {
	if !s.WriteLimiter.Allow("ip:" + clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, "slow down")
		return
	}
	site, err := requireSite(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	slug := r.PathValue("slug")
	thread, err := s.Store.GetOrCreateThread(site.ID, slug, "", "")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "thread load failed")
		return
	}
	if thread.Locked {
		writeError(w, http.StatusForbidden, "thread locked")
		return
	}
	var req createCommentReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	// Honeypot: legitimate users never fill the hidden "website" field.
	// Pretend success so bots don't probe further; just don't persist.
	if strings.TrimSpace(req.Honeypot) != "" {
		writeJSON(w, http.StatusCreated, map[string]any{"id": 0, "status": "ok"})
		return
	}

	body := strings.TrimSpace(req.Body)
	if body == "" {
		writeError(w, http.StatusBadRequest, "body required")
		return
	}
	if len(body) > 8000 {
		writeError(w, http.StatusBadRequest, "comment too long")
		return
	}

	user := userFrom(r)
	if site.RequireAuth && user == nil {
		writeError(w, http.StatusUnauthorized, "authentication required for this site")
		return
	}

	var userID *int64
	var authorName, email string
	if user != nil {
		userID = &user.ID
		authorName = user.Name
		email = user.Email
	} else {
		authorName = strings.TrimSpace(req.AuthorName)
		if authorName == "" {
			authorName = "Anonymous"
		}
		if len(authorName) > 80 {
			authorName = authorName[:80]
		}
	}

	anchor := strings.TrimSpace(req.Anchor)
	if !validAnchor(anchor) {
		writeError(w, http.StatusBadRequest, "invalid anchor")
		return
	}

	// Bound reply depth to prevent pathological nesting.
	if req.ParentID != nil {
		parent, err := s.Store.CommentByID(*req.ParentID, nil)
		if err != nil || parent.ThreadID != thread.ID {
			writeError(w, http.StatusBadRequest, "invalid parent_id")
			return
		}
		// Limit to one level of nesting (top-level + replies). Replies of replies
		// re-parent to the original top-level — keeps UI flat and avoids attack
		// vectors via deep recursion.
		if parent.ParentID != nil {
			req.ParentID = parent.ParentID
		}
		// Replies inherit the parent's paragraph anchor — keeps the whole
		// conversation pinned next to the same paragraph in marginalia mode.
		anchor = parent.Anchor
	}

	c, err := s.Store.CreateComment(thread.ID, req.ParentID, userID, authorName, body, email, anchor)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create failed")
		return
	}
	s.Hub.Publish(thread.ID, "comment")
	s.notifyWebhook(map[string]any{"event": "comment.created", "site": site.Slug, "thread": thread.Slug, "comment": c})
	writeJSON(w, http.StatusCreated, c)
}

func parseInt64Path(r *http.Request, name string) (int64, error) {
	return strconv.ParseInt(r.PathValue(name), 10, 64)
}
