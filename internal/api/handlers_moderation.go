package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/enekos/errolan/internal/models"
	"github.com/enekos/errolan/internal/moderation"
	"github.com/enekos/errolan/internal/store"
)

// resolveAdminSite returns the site identified by the {slug} path segment and
// requires the caller to be admin. Moderation config is a server-side admin
// operation, so we deliberately don't use the public `X-Errolan-Site` header.
func (s *Server) resolveAdminSite(r *http.Request) (*models.Site, *models.User, error) {
	admin, err := requireAdmin(r)
	if err != nil {
		return nil, nil, err
	}
	slug := r.PathValue("slug")
	if slug == "" {
		return nil, admin, errSiteSlugRequired
	}
	site, err := s.Store.SiteBySlug(slug)
	if err != nil {
		return nil, admin, errSiteNotFound
	}
	return site, admin, nil
}

var (
	errSiteSlugRequired = httpError(http.StatusBadRequest, "site slug required")
	errSiteNotFound     = httpError(http.StatusNotFound, "site not found")
)

type apiError struct {
	status int
	msg    string
}

func (e *apiError) Error() string { return e.msg }
func httpError(s int, m string) error {
	return &apiError{status: s, msg: m}
}

func writeAPIError(w http.ResponseWriter, err error) {
	if ae, ok := err.(*apiError); ok {
		writeError(w, ae.status, ae.msg)
		return
	}
	writeError(w, http.StatusInternalServerError, err.Error())
}

// ----- Site moderation settings -----

func (s *Server) handleGetModeration(w http.ResponseWriter, r *http.Request) {
	site, _, err := s.resolveAdminSite(r)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	settings, err := s.Store.ModerationSettings(site.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load failed")
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func (s *Server) handleUpdateModeration(w http.ResponseWriter, r *http.Request) {
	site, admin, err := s.resolveAdminSite(r)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	var req models.ModerationSettings
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	req.SiteID = site.ID
	if err := moderation.ValidateSettings(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.Store.UpdateModerationSettings(&req); err != nil {
		writeError(w, http.StatusInternalServerError, "save failed")
		return
	}
	s.Store.AddAudit(&admin.ID, admin.Name, "moderation.update", "site", site.ID, "")
	writeJSON(w, http.StatusOK, req)
}

// ----- Blocklist CRUD -----

type blocklistReq struct {
	Kind    string `json:"kind"`
	Pattern string `json:"pattern"`
	Action  string `json:"action"`
}

func (s *Server) handleListBlocklist(w http.ResponseWriter, r *http.Request) {
	site, _, err := s.resolveAdminSite(r)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	entries, err := s.Store.ListBlocklist(site.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list failed")
		return
	}
	if entries == nil {
		entries = []*models.BlocklistEntry{}
	}
	writeJSON(w, http.StatusOK, entries)
}

func (s *Server) handleAddBlocklist(w http.ResponseWriter, r *http.Request) {
	site, admin, err := s.resolveAdminSite(r)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	var req blocklistReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	req.Kind = strings.TrimSpace(strings.ToLower(req.Kind))
	req.Action = strings.TrimSpace(strings.ToLower(req.Action))
	if req.Action == "" {
		req.Action = "hold"
	}
	req.Pattern = strings.TrimSpace(req.Pattern)
	if err := moderation.ValidateRule(req.Kind, req.Pattern, req.Action); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	entry, err := s.Store.AddBlocklistEntry(site.ID, req.Kind, req.Pattern, req.Action)
	if err != nil {
		if err == store.ErrConflict {
			writeError(w, http.StatusConflict, "rule already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "save failed")
		return
	}
	s.Store.AddAudit(&admin.ID, admin.Name, "moderation.block_add", "site", site.ID, req.Kind+":"+req.Pattern)
	writeJSON(w, http.StatusCreated, entry)
}

func (s *Server) handleDeleteBlocklist(w http.ResponseWriter, r *http.Request) {
	site, admin, err := s.resolveAdminSite(r)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	id, err := parseInt64Path(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.Store.DeleteBlocklistEntry(site.ID, id); err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "rule not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	s.Store.AddAudit(&admin.ID, admin.Name, "moderation.block_delete", "blocklist", id, "")
	w.WriteHeader(http.StatusNoContent)
}

// ----- Queue / approve / reject -----

func (s *Server) handleModQueue(w http.ResponseWriter, r *http.Request) {
	if _, err := requireAdmin(r); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	var siteID int64
	if slug := strings.TrimSpace(r.URL.Query().Get("site")); slug != "" {
		site, err := s.Store.SiteBySlug(slug)
		if err != nil {
			writeError(w, http.StatusNotFound, "site not found")
			return
		}
		siteID = site.ID
	}
	pending, err := s.Store.ListPendingComments(siteID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list failed")
		return
	}
	if pending == nil {
		pending = []*models.Comment{}
	}
	writeJSON(w, http.StatusOK, pending)
}

func (s *Server) handleApproveComment(w http.ResponseWriter, r *http.Request) {
	admin, err := requireAdmin(r)
	if err != nil {
		writeError(w, http.StatusForbidden, err.Error())
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
	if c.Status != models.CommentStatusPending {
		writeError(w, http.StatusConflict, "comment is not pending")
		return
	}
	if err := s.Store.ApproveComment(id); err != nil {
		writeError(w, http.StatusInternalServerError, "approve failed")
		return
	}
	_ = s.Store.SetModerationReason(id, "")
	s.Store.AddAudit(&admin.ID, admin.Name, "comment.approve", "comment", id, "")
	s.Hub.Publish(c.ThreadID, "comment")
	s.notifyWebhook(map[string]any{"event": "comment.approved", "comment_id": id, "actor": admin.Email})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRejectComment(w http.ResponseWriter, r *http.Request) {
	admin, err := requireAdmin(r)
	if err != nil {
		writeError(w, http.StatusForbidden, err.Error())
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
	if c.Status != models.CommentStatusPending {
		writeError(w, http.StatusConflict, "comment is not pending")
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	_ = decode(r, &body)
	if err := s.Store.RejectComment(id); err != nil {
		writeError(w, http.StatusInternalServerError, "reject failed")
		return
	}
	reason := strings.TrimSpace(body.Reason)
	if reason != "" {
		_ = s.Store.SetModerationReason(id, reason)
	}
	s.Store.AddAudit(&admin.ID, admin.Name, "comment.reject", "comment", id, reason)
	s.notifyWebhook(map[string]any{"event": "comment.rejected", "comment_id": id, "reason": reason, "actor": admin.Email})
	w.WriteHeader(http.StatusNoContent)
}
