// Package config loads runtime configuration from environment variables.
// Centralising the parse step means main.go is short and tests can construct
// a Config directly without setting env vars.
package config

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

// Config is the fully-resolved runtime configuration. Every field has a sensible
// default — main.go only needs to call Load() and check the returned warnings.
type Config struct {
	Addr           string
	DBPath         string
	JWTSecret      string
	TokenTTL       time.Duration
	AdminEmail     string
	AdminPassword  string
	AdminCORS      string
	SDKDir         string
	TrustForwarded bool
	WebhookURL     string
	PublicURL      string

	// GitHub OAuth (optional). Set both to enable the GitHub login flow.
	GitHubClientID     string
	GitHubClientSecret string

	// JWTSecretGenerated is true when no secret was provided and Load()
	// generated an ephemeral one. main.go logs a warning so operators notice.
	JWTSecretGenerated bool
}

// Load reads the ERROLAN_* environment variables. It only returns an error for
// conditions that are clearly unsafe (e.g. failed to generate a random
// fallback secret); anything else falls back to a sensible default.
func Load() (*Config, error) {
	cfg := &Config{
		Addr:           envOr("ERROLAN_ADDR", ":8080"),
		DBPath:         envOr("ERROLAN_DB", "errolan.db"),
		JWTSecret:      os.Getenv("ERROLAN_JWT_SECRET"),
		TokenTTL:       7 * 24 * time.Hour,
		AdminEmail:     os.Getenv("ERROLAN_ADMIN_EMAIL"),
		AdminPassword:  os.Getenv("ERROLAN_ADMIN_PASSWORD"),
		AdminCORS:      envOr("ERROLAN_ADMIN_CORS", "*"),
		SDKDir:         envOr("ERROLAN_SDK_DIR", "sdk"),
		TrustForwarded:     strings.EqualFold(os.Getenv("ERROLAN_TRUST_FORWARDED"), "true"),
		WebhookURL:         os.Getenv("ERROLAN_WEBHOOK_URL"),
		PublicURL:          os.Getenv("ERROLAN_PUBLIC_URL"),
		GitHubClientID:     os.Getenv("ERROLAN_GITHUB_CLIENT_ID"),
		GitHubClientSecret: os.Getenv("ERROLAN_GITHUB_CLIENT_SECRET"),
	}
	if cfg.JWTSecret == "" {
		secret, err := generateSecret(32)
		if err != nil {
			return nil, fmt.Errorf("generate jwt secret: %w", err)
		}
		cfg.JWTSecret = secret
		cfg.JWTSecretGenerated = true
	}
	return cfg, nil
}

// ResolveSDKDir checks the configured SDK directory exists and returns the
// path to serve. If missing, it returns "" so the server can disable /sdk/.
// The caller is expected to log the situation.
func (c *Config) ResolveSDKDir() (path string, missing bool) {
	if c.SDKDir == "" {
		return "", false
	}
	if _, err := os.Stat(c.SDKDir); errors.Is(err, os.ErrNotExist) {
		return "", true
	}
	return c.SDKDir, false
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func generateSecret(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
