package store

import (
	"time"

	"github.com/enekos/errolan/internal/models"
)

func (s *Store) ListEmojis(siteID int64) ([]*models.Emoji, error) {
	rows, err := s.DB.Query(
		`SELECT id, site_id, code, label, svg, sort, created_at FROM emojis WHERE site_id = ? ORDER BY sort, id`,
		siteID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.Emoji
	for rows.Next() {
		e := &models.Emoji{}
		if err := rows.Scan(&e.ID, &e.SiteID, &e.Code, &e.Label, &e.SVG, &e.Sort, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) UpsertEmoji(siteID int64, code, label, svg string, sortOrder int) (*models.Emoji, error) {
	now := time.Now().Unix()
	_, err := s.DB.Exec(
		`INSERT INTO emojis (site_id, code, label, svg, sort, created_at) VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(site_id, code) DO UPDATE SET label = excluded.label, svg = excluded.svg, sort = excluded.sort`,
		siteID, code, label, svg, sortOrder, now,
	)
	if err != nil {
		return nil, err
	}
	e := &models.Emoji{}
	err = s.DB.QueryRow(
		`SELECT id, site_id, code, label, svg, sort, created_at FROM emojis WHERE site_id = ? AND code = ?`,
		siteID, code,
	).Scan(&e.ID, &e.SiteID, &e.Code, &e.Label, &e.SVG, &e.Sort, &e.CreatedAt)
	return e, err
}

func (s *Store) DeleteEmoji(siteID int64, code string) error {
	res, err := s.DB.Exec(`DELETE FROM emojis WHERE site_id = ? AND code = ?`, siteID, code)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// CodesForSite returns the set of valid emoji codes for a site. Used to gate
// reactions so users can't react with arbitrary strings.
func (s *Store) CodesForSite(siteID int64) (map[string]struct{}, error) {
	rows, err := s.DB.Query(`SELECT code FROM emojis WHERE site_id = ?`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]struct{})
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err == nil {
			out[code] = struct{}{}
		}
	}
	return out, rows.Err()
}
