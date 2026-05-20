package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// githubProvider implements Provider against GitHub's OAuth app flow. Public
// constructor below; the type is unexported on purpose — callers should only
// touch it through the Provider interface.
type githubProvider struct {
	clientID     string
	clientSecret string
	http         *http.Client
}

// NewGitHub builds a GitHub OAuth provider. The two arguments come from the
// "OAuth Apps" page in GitHub developer settings. Returns nil if either is
// empty so callers can pass through "maybe enabled" config without branching.
func NewGitHub(clientID, clientSecret string) Provider {
	if clientID == "" || clientSecret == "" {
		return nil
	}
	return &githubProvider{
		clientID:     clientID,
		clientSecret: clientSecret,
		http:         &http.Client{Timeout: 10 * time.Second},
	}
}

func (g *githubProvider) Name() string  { return "github" }
func (g *githubProvider) Label() string { return "GitHub" }

func (g *githubProvider) AuthURL(state, redirectURI string) string {
	q := url.Values{}
	q.Set("client_id", g.clientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("scope", "read:user user:email")
	q.Set("state", state)
	q.Set("allow_signup", "true")
	return "https://github.com/login/oauth/authorize?" + q.Encode()
}

// Exchange runs GitHub's two-step dance: code → access token, then access
// token → user profile + primary email. Both responses are JSON because we
// pass `Accept: application/json` (the legacy URL-encoded form is also valid
// but harder to parse and not worth supporting).
func (g *githubProvider) Exchange(ctx context.Context, code, redirectURI string) (*Identity, error) {
	tok, err := g.exchangeToken(ctx, code, redirectURI)
	if err != nil {
		return nil, err
	}

	user, err := g.fetchUser(ctx, tok)
	if err != nil {
		return nil, err
	}

	// GitHub's public user payload only includes a primary email when the
	// user made it public. /user/emails works for any user with the
	// `user:email` scope and gives us the verified primary even when private.
	if user.Email == "" {
		if email, e := g.fetchPrimaryEmail(ctx, tok); e == nil {
			user.Email = email
		}
	}

	name := user.Name
	if name == "" {
		name = user.Login
	}
	return &Identity{
		Provider:  g.Name(),
		Subject:   fmt.Sprintf("%d", user.ID),
		Email:     strings.ToLower(strings.TrimSpace(user.Email)),
		Name:      name,
		AvatarURL: user.AvatarURL,
	}, nil
}

type githubUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

type githubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

func (g *githubProvider) exchangeToken(ctx context.Context, code, redirectURI string) (string, error) {
	form := url.Values{}
	form.Set("client_id", g.clientID)
	form.Set("client_secret", g.clientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://github.com/login/oauth/access_token",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := g.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var body struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}
	if body.Error != "" {
		return "", fmt.Errorf("github oauth: %s", body.ErrorDesc)
	}
	if body.AccessToken == "" {
		return "", errors.New("github oauth: empty access token")
	}
	return body.AccessToken, nil
}

func (g *githubProvider) fetchUser(ctx context.Context, token string) (*githubUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := g.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github user fetch: status %d", resp.StatusCode)
	}
	var u githubUser
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return nil, fmt.Errorf("decode user: %w", err)
	}
	return &u, nil
}

func (g *githubProvider) fetchPrimaryEmail(ctx context.Context, token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := g.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github emails fetch: status %d", resp.StatusCode)
	}
	var emails []githubEmail
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", err
	}
	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}
	return "", errors.New("no verified primary email")
}
