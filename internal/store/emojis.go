package store

import (
	"time"

	"github.com/enekos/errolan/internal/models"
)

func (s *Store) ListEmojis(siteID int64) ([]*models.Emoji, error) {
	s.emojiMu.RLock()
	if emojis, ok := s.emojiCache[siteID]; ok {
		s.emojiMu.RUnlock()
		return emojis, nil
	}
	s.emojiMu.RUnlock()

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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	s.emojiMu.Lock()
	s.emojiCache[siteID] = out
	s.emojiMu.Unlock()
	return out, nil
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
	if err != nil {
		return nil, err
	}
	s.emojiMu.Lock()
	delete(s.emojiCache, siteID)
	s.emojiMu.Unlock()
	return e, nil
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
	s.emojiMu.Lock()
	delete(s.emojiCache, siteID)
	s.emojiMu.Unlock()
	return nil
}

// CodesForSite returns the set of valid emoji codes for a site. Used to gate
// reactions so users can't react with arbitrary strings. Reuses the ListEmojis
// cache to avoid a separate round-trip.
func (s *Store) CodesForSite(siteID int64) (map[string]struct{}, error) {
	emojis, err := s.ListEmojis(siteID)
	if err != nil {
		return nil, err
	}
	out := make(map[string]struct{}, len(emojis))
	for _, e := range emojis {
		out[e.Code] = struct{}{}
	}
	return out, nil
}
