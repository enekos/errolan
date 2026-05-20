package api

import (
	"net/http"
	"strconv"
	"time"
)

// handleExportMe returns every record the system holds for the authenticated
// user as a single JSON download. The response is intended to be saved
// locally — we set a Content-Disposition so the browser does the right thing.
func (s *Server) handleExportMe(w http.ResponseWriter, r *http.Request) {
	user, err := requireUser(r)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	export, err := s.Store.ExportUser(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "export failed")
		return
	}
	// Filename includes the user id and current date — useful for archives
	// but no PII (email/name) lands in the filename.
	filename := "errolan-export-user-" + strconv.FormatInt(user.ID, 10) +
		"-" + time.Now().UTC().Format("2006-01-02") + ".json"
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	writeJSON(w, http.StatusOK, export)
}

// handleDeleteMe is the GDPR "right to erasure" endpoint. We require the user
// to confirm by passing {"confirm": true} so an accidental DELETE /api/me
// can't trigger irreversible anonymisation. After success the JWT is no longer
// usable (the user is flagged banned), so the client should drop the token.
func (s *Server) handleDeleteMe(w http.ResponseWriter, r *http.Request) {
	user, err := requireUser(r)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	if user.IsAdmin {
		// Admins must demote themselves first. Otherwise self-delete could
		// leave the instance with no admin at all (especially on bootstrap).
		writeError(w, http.StatusForbidden, "admins must demote before deleting their account")
		return
	}
	var req struct {
		Confirm bool `json:"confirm"`
	}
	_ = decode(r, &req)
	if !req.Confirm {
		writeError(w, http.StatusBadRequest, "set {\"confirm\":true} to proceed")
		return
	}
	if err := s.Store.SoftDeleteUser(user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	s.Store.AddAudit(&user.ID, "[deleted]", "user.self_delete", "user", user.ID, "")
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "deleted",
		"note":   "Your comments now show '[deleted]' and your profile is removed. Replies to your comments are preserved.",
	})
}
