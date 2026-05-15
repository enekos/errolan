package store

import (
	"database/sql"
	"time"

	"github.com/enekos/errolan/internal/models"
)

const threadCols = `id, site_id, slug, title, url, locked, comment_count, last_comment_at, created_at`

func scanThread(row scanner) (*models.Thread, error) {
	var t models.Thread
	var locked int
	if err := row.Scan(&t.ID, &t.SiteID, &t.Slug, &t.Title, &t.URL, &locked, &t.CommentCount, &t.LastCommentAt, &t.CreatedAt); err != nil {
		return nil, err
	}
	t.Locked = locked != 0
	return &t, nil
}

func (s *Store) GetOrCreateThread(siteID int64, slug, title, url string) (*models.Thread, error) {
	t, err := s.ThreadBySlug(siteID, slug)
	if err == nil {
		// Threads are auto-created on first GET, so refreshing title/url here
		// keeps them up to date if the host page changed them.
		if title != "" || url != "" {
			if _, e := s.DB.Exec(
				`UPDATE threads SET title = COALESCE(NULLIF(?, ''), title), url = COALESCE(NULLIF(?, ''), url) WHERE id = ?`,
				title, url, t.ID,
			); e == nil {
				if title != "" {
					t.Title = title
				}
				if url != "" {
					t.URL = url
				}
			}
		}
		return t, nil
	}
	if err != ErrNotFound {
		return nil, err
	}
	t = &models.Thread{
		SiteID:    siteID,
		Slug:      slug,
		Title:     title,
		URL:       url,
		CreatedAt: time.Now().Unix(),
	}
	res, err := s.DB.Exec(
		`INSERT INTO threads (site_id, slug, title, url, created_at) VALUES (?, ?, ?, ?, ?)`,
		t.SiteID, t.Slug, t.Title, t.URL, t.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	t.ID = id
	return t, nil
}

func (s *Store) ThreadBySlug(siteID int64, slug string) (*models.Thread, error) {
	row := s.DB.QueryRow(`SELECT `+threadCols+` FROM threads WHERE site_id = ? AND slug = ?`, siteID, slug)
	t, err := scanThread(row)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return t, err
}

func (s *Store) ThreadByID(id int64) (*models.Thread, error) {
	row := s.DB.QueryRow(`SELECT `+threadCols+` FROM threads WHERE id = ?`, id)
	t, err := scanThread(row)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return t, err
}

func (s *Store) SetThreadLocked(id int64, locked bool) error {
	_, err := s.DB.Exec(`UPDATE threads SET locked = ? WHERE id = ?`, boolInt(locked), id)
	return err
}
