package store

import (
	"database/sql"
	"time"

	"github.com/enekos/errolan/internal/models"
)

const siteCols = `id, slug, name, api_key, allowed_origins, require_auth, created_at`

func scanSite(row scanner) (*models.Site, error) {
	var st models.Site
	var ra int
	if err := row.Scan(&st.ID, &st.Slug, &st.Name, &st.APIKey, &st.AllowedOrigins, &ra, &st.CreatedAt); err != nil {
		return nil, err
	}
	st.RequireAuth = ra != 0
	return &st, nil
}

func (s *Store) CreateSite(slug, name, allowedOrigins string, requireAuth bool) (*models.Site, error) {
	site := &models.Site{
		Slug:           slug,
		Name:           name,
		APIKey:         newAPIKey(),
		AllowedOrigins: allowedOrigins,
		RequireAuth:    requireAuth,
		CreatedAt:      time.Now().Unix(),
	}
	res, err := s.DB.Exec(
		`INSERT INTO sites (slug, name, api_key, allowed_origins, require_auth, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		site.Slug, site.Name, site.APIKey, site.AllowedOrigins, boolInt(site.RequireAuth), site.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	site.ID = id
	return site, nil
}

func (s *Store) ListSites() ([]*models.Site, error) {
	rows, err := s.DB.Query(`SELECT ` + siteCols + ` FROM sites ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.Site
	for rows.Next() {
		st, err := scanSite(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	return out, rows.Err()
}

func (s *Store) SiteByAPIKey(key string) (*models.Site, error) {
	row := s.DB.QueryRow(`SELECT `+siteCols+` FROM sites WHERE api_key = ?`, key)
	st, err := scanSite(row)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return st, err
}

func (s *Store) SiteBySlug(slug string) (*models.Site, error) {
	row := s.DB.QueryRow(`SELECT `+siteCols+` FROM sites WHERE slug = ?`, slug)
	st, err := scanSite(row)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return st, err
}
