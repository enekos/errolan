// Package oauth abstracts third-party identity providers behind a single
// Provider interface so the rest of Errolan never depends on a specific
// vendor's quirks. Each concrete provider (GitHub, GitLab, Google, …) lives
// in its own file in this package and is opaque to callers.
//
// Adding a new provider is three steps:
//
//   1. Implement Provider in `<name>.go` (Name, AuthURL, Exchange).
//   2. Surface a constructor (e.g. NewGitHub) that takes the client id/secret
//      and returns Provider.
//   3. Wire it in main.go from config — the API server discovers providers
//      via the slice you pass in ServerOptions.OAuthProviders.
//
// The HTTP plumbing (state CSRF tokens, callback dispatch, identity → user
// linking) lives in the api package and is provider-agnostic.
package oauth

import (
	"context"
)

// Identity is the normalized result of a successful OAuth login. Different
// providers expose different fields — we surface the smallest set the comment
// system actually needs and lose the rest.
type Identity struct {
	// Provider is the same short name returned by Provider.Name(). Stored in
	// the oauth_identities table so we can re-link the user on next login.
	Provider string

	// Subject is the provider-side stable user id. NEVER use email here:
	// users can change email but their provider-side id is permanent.
	Subject string

	// Email is best-effort: optional public profile email when the provider
	// returns it. Empty when the user hides their primary email.
	Email string

	// Name is the human-readable display name. Falls back to the username
	// when no display name is set.
	Name string

	// AvatarURL is the provider's profile picture. Optional.
	AvatarURL string
}

// Provider is the contract every identity provider satisfies. Implementations
// must be safe for concurrent use — the registry holds one Provider per name
// for the lifetime of the process.
type Provider interface {
	// Name returns a stable short identifier — used as the URL segment in
	// /api/auth/oauth/{name}/* and as the discriminator in the identities
	// table. Lowercase ASCII, no spaces, no slashes.
	Name() string

	// Label returns the user-facing display name ("GitHub", "GitLab", …).
	// The /api/auth/oauth endpoint lists available providers using this.
	Label() string

	// AuthURL builds the provider's authorization endpoint URL the SDK
	// redirects the user to. State is opaque CSRF protection — the caller
	// generated it and will verify it on the callback.
	AuthURL(state, redirectURI string) string

	// Exchange swaps an authorization code for an Identity. The caller has
	// already validated the state token; implementations should focus on
	// the code-for-token-for-profile dance.
	Exchange(ctx context.Context, code, redirectURI string) (*Identity, error)
}
