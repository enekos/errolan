package api

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/enekos/errolan/internal/models"
	"github.com/enekos/errolan/internal/moderation"
	"github.com/enekos/errolan/internal/store"
)

// threadSiteView is the trimmed-down site payload returned alongside a thread.
// The API key never leaks here — that's an admin-only field.
type threadSiteView struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	RequireAuth bool   `json:"require_auth"`
}

// threadViewerView is the trimmed-down viewer payload for the current request.
// Password hash and ban state are deliberately omitted.
type threadViewerView struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	IsAdmin bool   `json:"is_admin"`
}

type threadResp struct {
	Thread   *models.Thread    `json:"thread"`
	Site     threadSiteView    `json:"site"`
	Comments []*models.Comment `json:"comments"`
	HasMore  bool              `json:"has_more"`
	NextID   *int64            `json:"next_id,omitempty"`
	Sort     string            `json:"sort"`
	Viewer   *threadViewerView `json:"viewer,omitempty"`
	Emojis   []*models.Emoji   `json:"emojis"`
	Mentions []*models.Mention `json:"mentions,omitempty"`
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

// threadETag mixes in the mention count so freshly-verified Webmentions /
// ActivityPub replies invalidate the cache too — comment_count alone wouldn't
// move when a mention arrives.
func threadETag(t *models.Thread, mentionCount int) string {
	raw := fmt.Sprintf("%d:%d:%d:%d", t.ID, t.CommentCount, t.LastCommentAt, mentionCount)
	sum := sha1.Sum([]byte(raw))
	return `W/"` + hex.EncodeToString(sum[:8]) + `"`
}

func (s *Server) handleGetThread(w http.ResponseWriter, r *http.Request) {
	site, err := requireSite(r)
	if err != nil {
		writeAPIError(w, err)
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

	// Pre-fetch the mention list now so it folds into the ETag. The list is
	// also re-used in the response body below — one query, two consumers.
	mentions, _ := s.Store.ListThreadMentions(thread.ID)

	// Viewer scopes the ETag because my_vote varies by user.
	viewer := userFrom(r)
	tag := threadETag(thread, len(mentions))
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
		Sort:           parseSort(sortParam),
		Limit:          limit,
		BeforeID:       beforeID,
		ViewerID:       viewerID,
		IncludePending: viewer != nil && viewer.IsAdmin,
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
		Site: threadSiteView{
			Slug:        site.Slug,
			Name:        site.Name,
			RequireAuth: site.RequireAuth,
		},
		Emojis:   emojis,
		Mentions: mentions,
	}
	if viewer != nil {
		resp.Viewer = &threadViewerView{
			ID:      viewer.ID,
			Name:    viewer.Name,
			Email:   viewer.Email,
			IsAdmin: viewer.IsAdmin,
		}
	}
	w.Header().Set("ETag", tag)
	w.Header().Set("Cache-Control", "private, no-cache")
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleLockThread(w http.ResponseWriter, r *http.Request) {
	admin, err := requireAdmin(r)
	if err != nil {
		writeAPIError(w, err)
		return
	}
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

	// Text-range selectors (W3C Annotation style) — optional. When present they
	// pin the comment to a specific passage inside the anchor element.
	RangeQuote  string `json:"range_quote"`
	RangePrefix string `json:"range_prefix"`
	RangeSuffix string `json:"range_suffix"`
	RangeStart  int    `json:"range_start"`
	RangeEnd    int    `json:"range_end"`
}

// Bounds for selector fields. The quote is what we actually display and re-anchor
// against; prefix/suffix are short context lifelines for fuzzy lookup.
const (
	maxRangeQuote   = 1000
	maxRangeContext = 64
)

// validateTextRange normalises and bounds a user-supplied text selector. It
// returns the cleaned-up range and an error message; a zero range with no
// error means "no selector supplied" which is a valid legacy comment.
func validateTextRange(req createCommentReq) (store.TextRange, string) {
	q := strings.TrimSpace(req.RangeQuote)
	if q == "" && req.RangeStart == 0 && req.RangeEnd == 0 {
		return store.TextRange{}, ""
	}
	if q == "" {
		return store.TextRange{}, "range_quote required when range is specified"
	}
	if len(q) > maxRangeQuote {
		return store.TextRange{}, "range_quote too long"
	}
	if req.RangeStart < 0 || req.RangeEnd < 0 || req.RangeEnd < req.RangeStart {
		return store.TextRange{}, "invalid range offsets"
	}
	prefix := req.RangePrefix
	if len(prefix) > maxRangeContext {
		prefix = prefix[len(prefix)-maxRangeContext:]
	}
	suffix := req.RangeSuffix
	if len(suffix) > maxRangeContext {
		suffix = suffix[:maxRangeContext]
	}
	return store.TextRange{
		Quote:  q,
		Prefix: prefix,
		Suffix: suffix,
		Start:  req.RangeStart,
		End:    req.RangeEnd,
	}, ""
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
		writeAPIError(w, err)
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

	rng, rngErr := validateTextRange(req)
	if rngErr != "" {
		writeError(w, http.StatusBadRequest, rngErr)
		return
	}
	// A text range without an anchor is meaningless — we need to know which
	// element the offsets/quote refer to.
	if rng.Quote != "" && anchor == "" {
		writeError(w, http.StatusBadRequest, "anchor required when range_quote is set")
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
		// Replies inherit the parent's paragraph anchor and text range — the
		// whole sub-thread stays pinned to the same passage.
		anchor = parent.Anchor
		rng = store.TextRange{
			Quote:  parent.RangeQuote,
			Prefix: parent.RangePrefix,
			Suffix: parent.RangeSuffix,
			Start:  parent.RangeStart,
			End:    parent.RangeEnd,
		}
	}

	// Run the moderation engine. Admins skip the queue: an admin posting is
	// implicitly trusted and a pre-moderation site shouldn't force its own
	// owner to approve themselves.
	decision := s.evaluateComment(site.ID, user, user == nil, body)

	if decision.Action == moderation.ActionReject {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
			"error":  "comment rejected by moderation",
			"reason": decision.Reason,
		})
		return
	}

	status := models.CommentStatusVisible
	if decision.Action == moderation.ActionHold {
		status = models.CommentStatusPending
	}

	c, err := s.Store.CreateComment(thread.ID, req.ParentID, userID, authorName, body, email, anchor, status, decision.Reason, rng)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create failed")
		return
	}

	if status == models.CommentStatusPending {
		// Don't fanout to the thread's SSE subscribers — the comment isn't
		// public yet. Surface the hold to ops via webhook + audit instead.
		s.Store.AddAudit(userID, authorName, "comment.hold", "comment", c.ID, decision.Reason)
		s.notifyWebhook(map[string]any{
			"event":     "comment.pending",
			"site":      site.Slug,
			"thread":    thread.Slug,
			"comment":   c,
			"reason":    decision.Reason,
		})
		writeJSON(w, http.StatusAccepted, map[string]any{
			"status":  "pending",
			"reason":  decision.Reason,
			"comment": c,
		})
		return
	}

	s.Hub.Publish(thread.ID, "comment")
	s.notifyWebhook(map[string]any{"event": "comment.created", "site": site.Slug, "thread": thread.Slug, "comment": c})
	writeJSON(w, http.StatusCreated, c)
}

func parseInt64Path(r *http.Request, name string) (int64, error) {
	return strconv.ParseInt(r.PathValue(name), 10, 64)
}
