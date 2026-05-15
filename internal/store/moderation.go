package store

import (
	"database/sql"
	"strings"
	"time"

	"github.com/enekos/errolan/internal/models"
)

// ModerationSettings returns the policy for a site, or default values when no
// row exists yet (sites pre-feature-rollout behave exactly like before).
func (s *Store) ModerationSettings(siteID int64) (*models.ModerationSettings, error) {
	m := models.DefaultModerationSettings(siteID)
	row := s.DB.QueryRow(
		`SELECT site_id, mode, hold_new_users, min_account_age_seconds, min_body_length,
		        max_links, link_policy, anonymous_link_policy, auto_hide_flag_count, updated_at
		   FROM site_moderation WHERE site_id = ?`, siteID,
	)
	err := row.Scan(
		&m.SiteID, &m.Mode, &m.HoldNewUsers, &m.MinAccountAgeSeconds, &m.MinBodyLength,
		&m.MaxLinks, &m.LinkPolicy, &m.AnonymousLinkPolicy, &m.AutoHideFlagCount, &m.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return &m, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *Store) UpdateModerationSettings(m *models.ModerationSettings) error {
	m.UpdatedAt = time.Now().Unix()
	_, err := s.DB.Exec(
		`INSERT INTO site_moderation (site_id, mode, hold_new_users, min_account_age_seconds, min_body_length,
		     max_links, link_policy, anonymous_link_policy, auto_hide_flag_count, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(site_id) DO UPDATE SET
		   mode = excluded.mode,
		   hold_new_users = excluded.hold_new_users,
		   min_account_age_seconds = excluded.min_account_age_seconds,
		   min_body_length = excluded.min_body_length,
		   max_links = excluded.max_links,
		   link_policy = excluded.link_policy,
		   anonymous_link_policy = excluded.anonymous_link_policy,
		   auto_hide_flag_count = excluded.auto_hide_flag_count,
		   updated_at = excluded.updated_at`,
		m.SiteID, m.Mode, m.HoldNewUsers, m.MinAccountAgeSeconds, m.MinBodyLength,
		m.MaxLinks, m.LinkPolicy, m.AnonymousLinkPolicy, m.AutoHideFlagCount, m.UpdatedAt,
	)
	return err
}

func (s *Store) ListBlocklist(siteID int64) ([]*models.BlocklistEntry, error) {
	rows, err := s.DB.Query(
		`SELECT id, site_id, kind, pattern, action, created_at FROM moderation_blocklist WHERE site_id = ? ORDER BY id`,
		siteID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.BlocklistEntry
	for rows.Next() {
		e := &models.BlocklistEntry{}
		if err := rows.Scan(&e.ID, &e.SiteID, &e.Kind, &e.Pattern, &e.Action, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) AddBlocklistEntry(siteID int64, kind, pattern, action string) (*models.BlocklistEntry, error) {
	now := time.Now().Unix()
	res, err := s.DB.Exec(
		`INSERT INTO moderation_blocklist (site_id, kind, pattern, action, created_at) VALUES (?, ?, ?, ?, ?)`,
		siteID, kind, pattern, action, now,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return nil, ErrConflict
		}
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &models.BlocklistEntry{
		ID:        id,
		SiteID:    siteID,
		Kind:      kind,
		Pattern:   pattern,
		Action:    action,
		CreatedAt: now,
	}, nil
}

func (s *Store) DeleteBlocklistEntry(siteID, id int64) error {
	res, err := s.DB.Exec(`DELETE FROM moderation_blocklist WHERE site_id = ? AND id = ?`, siteID, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
