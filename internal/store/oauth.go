package store

import (
	"database/sql"
	"strings"
	"time"

	"github.com/enekos/errolan/internal/models"
)

// IdentityByProviderSubject looks up the local user behind a provider+subject
// pair, returning ErrNotFound when there's no link yet (signal to the caller
// to provision a new user).
func (s *Store) IdentityByProviderSubject(provider, subject string) (*models.OAuthIdentity, error) {
	row := s.DB.QueryRow(
		`SELECT id, user_id, provider, subject, email, avatar_url, created_at
		   FROM oauth_identities
		  WHERE provider = ? AND subject = ?`,
		provider, subject,
	)
	id := &models.OAuthIdentity{}
	if err := row.Scan(&id.ID, &id.UserID, &id.Provider, &id.Subject, &id.Email, &id.AvatarURL, &id.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return id, nil
}

// LinkIdentity creates (or updates) an OAuth identity row for the given user.
// Idempotent on (provider, subject) — second call with the same pair just
// refreshes the snapshot fields. Used both at first login and when re-linking
// after a user logs in with an email that already has a local account.
func (s *Store) LinkIdentity(userID int64, provider, subject, email, avatarURL string) (*models.OAuthIdentity, error) {
	now := time.Now().Unix()
	_, err := s.DB.Exec(
		`INSERT INTO oauth_identities (user_id, provider, subject, email, avatar_url, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		userID, provider, subject, email, avatarURL, now,
	)
	if err != nil && strings.Contains(err.Error(), "UNIQUE") {
		// Existing row — refresh the snapshot fields. The user_id MUST stay
		// stable (we keyed the lookup on provider+subject, so any link points
		// at the original local user).
		if _, e := s.DB.Exec(
			`UPDATE oauth_identities SET email = ?, avatar_url = ? WHERE provider = ? AND subject = ?`,
			email, avatarURL, provider, subject,
		); e != nil {
			return nil, e
		}
		return s.IdentityByProviderSubject(provider, subject)
	}
	if err != nil {
		return nil, err
	}
	return s.IdentityByProviderSubject(provider, subject)
}

// IdentitiesForUser returns every linked provider for a user. Used by /api/me/export.
func (s *Store) IdentitiesForUser(userID int64) ([]*models.OAuthIdentity, error) {
	rows, err := s.DB.Query(
		`SELECT id, user_id, provider, subject, email, avatar_url, created_at
		   FROM oauth_identities WHERE user_id = ? ORDER BY created_at ASC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.OAuthIdentity
	for rows.Next() {
		id := &models.OAuthIdentity{}
		if err := rows.Scan(&id.ID, &id.UserID, &id.Provider, &id.Subject, &id.Email, &id.AvatarURL, &id.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// CreateUserForOAuth is the "no existing user, no existing identity" path. We
// store a NULL-marker password hash because OAuth users never set a password —
// validPassword in the auth handler will refuse to ever match it.
const oauthPasswordSentinel = "!oauth-only!"

// CreateUserForOAuth provisions a new local user from an OAuth identity and
// links the identity in a single transaction. Email may be empty if the
// provider refused to share one; we fall back to a synthetic placeholder.
func (s *Store) CreateUserForOAuth(provider, subject, email, name, avatarURL string) (*models.User, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	now := time.Now().Unix()

	// Synthesize a placeholder email when the provider didn't supply one.
	// Required because the users.email column is UNIQUE NOT NULL — without
	// a value, the second OAuth-only user would collide on empty string.
	if email == "" {
		email = provider + "+" + subject + "@oauth.errolan.invalid"
	}
	if name == "" {
		name = provider + ":" + subject
	}

	res, err := tx.Exec(
		`INSERT INTO users (email, name, password_hash, is_admin, created_at) VALUES (?, ?, ?, 0, ?)`,
		strings.ToLower(email), name, oauthPasswordSentinel, now,
	)
	if err != nil {
		return nil, err
	}
	userID, _ := res.LastInsertId()

	if _, err := tx.Exec(
		`INSERT INTO oauth_identities (user_id, provider, subject, email, avatar_url, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		userID, provider, subject, email, avatarURL, now,
	); err != nil {
		return nil, err
	}

	// Seed a profile row with the avatar URL so the reader profile page picks
	// up the GitHub-style avatar instead of falling back to Gravatar.
	if avatarURL != "" {
		if _, err := tx.Exec(
			`INSERT OR IGNORE INTO user_profiles (user_id, avatar_url, updated_at) VALUES (?, ?, ?)`,
			userID, avatarURL, now,
		); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.UserByID(userID)
}
