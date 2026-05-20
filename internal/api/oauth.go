package api

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/enekos/errolan/internal/oauth"
)

// OAuthProvider is re-exported here so callers (main.go, tests) don't have to
// import internal/oauth alongside internal/api. Functionally identical.
type OAuthProvider = oauth.Provider

// oauthRegistry indexes the configured providers by their short name and
// holds the CSRF state map. One instance per Server.
//
// State design:
//
//   - We mint a fresh token on every /api/auth/oauth/{name} redirect, stash
//     the post-login redirect URL with it, and verify-then-delete on callback.
//   - Tokens expire after stateTTL. We sweep lazily on each touch so we don't
//     run a background goroutine.
//
// For multi-replica deployments, swap this for a shared store (signed cookies
// work well — they keep the state with the user's browser rather than us).
type oauthRegistry struct {
	providers map[string]OAuthProvider

	stateMu sync.Mutex
	states  map[string]oauthState
}

type oauthState struct {
	createdAt time.Time
	redirect  string
}

const oauthStateTTL = 10 * time.Minute

// newOAuthRegistry builds a registry from the given providers slice. Nil and
// duplicates are silently dropped — the boot-time error is logged by main.go
// when it builds the provider list.
func newOAuthRegistry(providers []OAuthProvider) *oauthRegistry {
	r := &oauthRegistry{
		providers: make(map[string]OAuthProvider),
		states:    make(map[string]oauthState),
	}
	for _, p := range providers {
		if p == nil {
			continue
		}
		r.providers[p.Name()] = p
	}
	return r
}

// Has returns true when any provider is configured. Handlers use this to 404
// the OAuth endpoints when the operator hasn't set up an identity provider.
func (r *oauthRegistry) Has() bool { return len(r.providers) > 0 }

// Provider returns the named provider or nil.
func (r *oauthRegistry) Provider(name string) OAuthProvider {
	if r == nil {
		return nil
	}
	return r.providers[name]
}

// List returns metadata about every configured provider in stable order
// (alphabetical by name). Used by /api/auth/oauth to advertise what's
// available to the SDK so it can render a row of login buttons.
type oauthProviderInfo struct {
	Name  string `json:"name"`
	Label string `json:"label"`
}

func (r *oauthRegistry) List() []oauthProviderInfo {
	out := make([]oauthProviderInfo, 0, len(r.providers))
	for _, p := range r.providers {
		out = append(out, oauthProviderInfo{Name: p.Name(), Label: p.Label()})
	}
	// Stable order: alphabetical, no allocation beyond what's already there.
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].Name < out[i].Name {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

// IssueState mints a fresh state token bound to the post-login redirect.
func (r *oauthRegistry) IssueState(redirect string) string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Random source failure is a system-level problem; fall back to a
		// timestamp so we still produce *some* token. ConsumeState verifies
		// the value was issued, so a weak token can't be replayed.
		return "ts-" + time.Now().Format("20060102150405.000000")
	}
	tok := hex.EncodeToString(b)
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	r.sweepLocked()
	r.states[tok] = oauthState{createdAt: time.Now(), redirect: redirect}
	return tok
}

// ConsumeState verifies and deletes a state token, returning the redirect URL
// that was stashed at issue time. Unknown / expired tokens return ("", false).
func (r *oauthRegistry) ConsumeState(tok string) (string, bool) {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	r.sweepLocked()
	s, ok := r.states[tok]
	if !ok {
		return "", false
	}
	delete(r.states, tok)
	if time.Since(s.createdAt) > oauthStateTTL {
		return "", false
	}
	return s.redirect, true
}

func (r *oauthRegistry) sweepLocked() {
	cutoff := time.Now().Add(-oauthStateTTL)
	for k, v := range r.states {
		if v.createdAt.Before(cutoff) {
			delete(r.states, k)
		}
	}
}
