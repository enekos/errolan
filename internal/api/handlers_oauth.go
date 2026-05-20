package api

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/enekos/errolan/internal/models"
	"github.com/enekos/errolan/internal/oauth"
	"github.com/enekos/errolan/internal/store"
)

// ----- /api/auth/oauth/{provider}: provider-agnostic OAuth flow -----
//
// Everything here is provider-agnostic. The Server doesn't import any concrete
// provider (GitHub, GitLab, …) — it just dispatches by short name through the
// oauthRegistry. Adding a provider requires no changes to this file.

// handleListOAuthProviders returns the configured providers so the SDK can
// render a row of "Sign in with X" buttons.
func (s *Server) handleListOAuthProviders(w http.ResponseWriter, r *http.Request) {
	if !s.OAuth.Has() {
		writeJSON(w, http.StatusOK, []any{})
		return
	}
	writeJSON(w, http.StatusOK, s.OAuth.List())
}

// handleOAuthStart redirects the browser to the provider's authorization
// endpoint. The `redirect` query param is the post-login URL the SDK wants the
// user to land on (typically the page they were reading); we stash it
// alongside the state token and replay it from the callback.
func (s *Server) handleOAuthStart(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("provider")
	p := s.OAuth.Provider(name)
	if p == nil {
		writeError(w, http.StatusNotFound, "provider not configured")
		return
	}
	redirect := strings.TrimSpace(r.URL.Query().Get("redirect"))
	if !validReturnURL(redirect) {
		redirect = ""
	}
	state := s.OAuth.IssueState(redirect)
	http.Redirect(w, r, p.AuthURL(state, s.oauthCallbackURL(r, p)), http.StatusFound)
}

// handleOAuthCallback completes the OAuth dance: verify state, exchange code,
// look up (or create) the local user, issue a JWT, redirect back to the SDK.
func (s *Server) handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("provider")
	p := s.OAuth.Provider(name)
	if p == nil {
		writeError(w, http.StatusNotFound, "provider not configured")
		return
	}
	q := r.URL.Query()
	if errCode := q.Get("error"); errCode != "" {
		writeError(w, http.StatusBadRequest, "oauth error: "+errCode)
		return
	}
	code := q.Get("code")
	state := q.Get("state")
	if code == "" || state == "" {
		writeError(w, http.StatusBadRequest, "missing code or state")
		return
	}
	redirect, ok := s.OAuth.ConsumeState(state)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid or expired state")
		return
	}

	identity, err := p.Exchange(context.Background(), code, s.oauthCallbackURL(r, p))
	if err != nil {
		s.Logger.Warn("oauth exchange failed", "provider", name, "err", err)
		writeError(w, http.StatusBadGateway, "oauth provider rejected the code")
		return
	}

	user, err := s.linkOrProvisionUser(identity)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "user provisioning failed")
		return
	}

	token, err := s.Auth.Issue(user.ID, user.IsAdmin)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token issue failed")
		return
	}

	// Two delivery options, depending on whether the SDK gave us a redirect:
	//   (a) redirect=<post-login url> → 302 back with #token=… in the fragment.
	//       Fragment, not query, so the token never lands in server logs.
	//   (b) no redirect → return JSON so a programmatic client (curl, an SPA
	//       opening a popup) can pluck the token out of the response body.
	if redirect != "" {
		dest := redirect
		sep := "#"
		if strings.Contains(dest, "#") {
			sep = "&"
		}
		http.Redirect(w, r, dest+sep+"errolan_token="+url.QueryEscape(token), http.StatusFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"token": token,
		"user": map[string]any{
			"id": user.ID, "email": user.Email, "name": user.Name, "is_admin": user.IsAdmin,
		},
		"provider": identity.Provider,
	})
}

// linkOrProvisionUser is the "do whatever it takes to land at a local User"
// step. Order of operations:
//
//   1. (provider, subject) already linked → return that user.
//   2. Provider supplied a verified email matching an existing local user →
//      link this identity to that user (e.g. existing password account adds
//      "Sign in with GitHub" later).
//   3. Otherwise → provision a brand-new local user.
func (s *Server) linkOrProvisionUser(id *oauth.Identity) (*models.User, error) {
	// Already linked?
	if rec, err := s.Store.IdentityByProviderSubject(id.Provider, id.Subject); err == nil {
		u, err := s.Store.UserByID(rec.UserID)
		if err != nil {
			return nil, err
		}
		// Refresh snapshot fields on every login so a renamed/avatar-updated
		// GitHub profile propagates to the export and the profile sidebar.
		_, _ = s.Store.LinkIdentity(u.ID, id.Provider, id.Subject, id.Email, id.AvatarURL)
		return u, nil
	} else if err != store.ErrNotFound {
		return nil, err
	}

	// Existing local user with the same email?
	if id.Email != "" {
		if u, err := s.Store.UserByEmail(id.Email); err == nil {
			if _, err := s.Store.LinkIdentity(u.ID, id.Provider, id.Subject, id.Email, id.AvatarURL); err != nil {
				return nil, err
			}
			return u, nil
		} else if err != store.ErrNotFound {
			return nil, err
		}
	}

	// Fresh user.
	return s.Store.CreateUserForOAuth(id.Provider, id.Subject, id.Email, id.Name, id.AvatarURL)
}

// oauthCallbackURL builds the registered callback URL for a provider. The
// provider's OAuth app must list this URL exactly. We respect PublicURL so
// production callbacks don't depend on the Host header (which a reverse proxy
// might rewrite).
func (s *Server) oauthCallbackURL(r *http.Request, p oauth.Provider) string {
	return s.instanceBase(r) + "/api/auth/oauth/" + p.Name() + "/callback"
}

// validReturnURL refuses anything that isn't an http(s) URL with a host, so
// the OAuth flow can't be turned into an open-redirect gadget. We don't
// constrain *which* host on purpose — the SDK is hosted on many origins and
// the operator's CORS allowlist is the real gate for downstream concerns.
func validReturnURL(raw string) bool {
	if raw == "" {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}
