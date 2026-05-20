package store

import (
	"database/sql"
	"time"

	"github.com/enekos/errolan/internal/models"
)

// ProfileByUserID returns the editable profile row for a user, or ErrNotFound
// if the user hasn't set one. The handler layer falls back to defaults derived
// from the User record (Gravatar from email, name as-is) when this returns
// ErrNotFound — there's no need to materialize an empty row.
func (s *Store) ProfileByUserID(userID int64) (*models.UserProfile, error) {
	row := s.DB.QueryRow(
		`SELECT user_id, bio, website, avatar_url, updated_at FROM user_profiles WHERE user_id = ?`,
		userID,
	)
	p := &models.UserProfile{}
	if err := row.Scan(&p.UserID, &p.Bio, &p.Website, &p.AvatarURL, &p.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return p, nil
}

// UpsertProfile saves or replaces a profile row. SQLite UPSERT keeps this
// as one round trip.
func (s *Store) UpsertProfile(p *models.UserProfile) error {
	p.UpdatedAt = time.Now().Unix()
	_, err := s.DB.Exec(
		`INSERT INTO user_profiles (user_id, bio, website, avatar_url, updated_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(user_id) DO UPDATE SET
		   bio = excluded.bio,
		   website = excluded.website,
		   avatar_url = excluded.avatar_url,
		   updated_at = excluded.updated_at`,
		p.UserID, p.Bio, p.Website, p.AvatarURL, p.UpdatedAt,
	)
	return err
}

// CommentsByUser returns the user's visible comments, most-recent-first, for
// the public profile page. Soft-deleted / hidden / pending comments are
// excluded — these are read by anyone.
func (s *Store) CommentsByUser(userID int64, limit int) ([]*models.Comment, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.DB.Query(
		`SELECT `+commentCols+` FROM comments
		  WHERE user_id = ? AND status = 'visible'
		  ORDER BY created_at DESC LIMIT ?`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.Comment
	for rows.Next() {
		c, err := scanComment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// CountCommentsByUser returns the total of the same set the profile page
// shows (visible only). Used in the profile header ("142 comments").
func (s *Store) CountCommentsByUser(userID int64) (int, error) {
	var n int
	err := s.DB.QueryRow(
		`SELECT COUNT(*) FROM comments WHERE user_id = ? AND status = 'visible'`,
		userID,
	).Scan(&n)
	return n, err
}
