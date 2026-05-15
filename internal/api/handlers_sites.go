package api

import (
	"net/http"
	"strings"
)

type createSiteReq struct {
	Slug           string `json:"slug"`
	Name           string `json:"name"`
	AllowedOrigins string `json:"allowed_origins"`
	RequireAuth    bool   `json:"require_auth"`
}

func (s *Server) handleListSites(w http.ResponseWriter, r *http.Request) {
	if _, err := requireAdmin(r); err != nil {
		writeAPIError(w, err)
		return
	}
	sites, err := s.Store.ListSites()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list failed")
		return
	}
	writeJSON(w, http.StatusOK, sites)
}

func (s *Server) handleCreateSite(w http.ResponseWriter, r *http.Request) {
	admin, err := requireAdmin(r)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	var req createSiteReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	req.Slug = strings.TrimSpace(req.Slug)
	req.Name = strings.TrimSpace(req.Name)
	if req.Slug == "" || req.Name == "" {
		writeError(w, http.StatusBadRequest, "slug and name required")
		return
	}
	if req.AllowedOrigins == "" {
		req.AllowedOrigins = "*"
	}
	site, err := s.Store.CreateSite(req.Slug, req.Name, req.AllowedOrigins, req.RequireAuth)
	if err != nil {
		writeError(w, http.StatusConflict, "could not create site (slug may exist)")
		return
	}
	s.Store.AddAudit(&admin.ID, admin.Name, "site.create", "site", site.ID, "")
	writeJSON(w, http.StatusCreated, site)
}
