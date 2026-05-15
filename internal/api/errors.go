package api

import (
	"errors"
	"net/http"
)

// apiError pairs an HTTP status with a user-facing message. Handlers return it
// when validation/auth fails so the dispatch helper (writeAPIError) can pick
// the right status without each handler repeating the writeError + return dance.
type apiError struct {
	Status  int
	Message string
}

func (e *apiError) Error() string { return e.Message }

// httpError builds an *apiError. It returns the error interface so callers can
// `return nil, httpError(...)` ergonomically.
func httpError(status int, msg string) error {
	return &apiError{Status: status, Message: msg}
}

// Common errors reused across handlers. Defining them once keeps phrasing
// consistent and lets callers compare with errors.Is in tests.
var (
	errAuthRequired     = httpError(http.StatusUnauthorized, "authentication required")
	errAdminOnly        = httpError(http.StatusForbidden, "admin only")
	errSiteHeader       = httpError(http.StatusBadRequest, "X-Errolan-Site header required")
	errSiteSlugRequired = httpError(http.StatusBadRequest, "site slug required")
	errSiteNotFound     = httpError(http.StatusNotFound, "site not found")
	errInvalidID        = httpError(http.StatusBadRequest, "invalid id")
	errInvalidBody      = httpError(http.StatusBadRequest, "invalid body")
)

// writeAPIError emits the error to the response writer. *apiError carries its
// own status; anything else falls back to 500.
func writeAPIError(w http.ResponseWriter, err error) {
	var ae *apiError
	if errors.As(err, &ae) {
		writeError(w, ae.Status, ae.Message)
		return
	}
	writeError(w, http.StatusInternalServerError, err.Error())
}
