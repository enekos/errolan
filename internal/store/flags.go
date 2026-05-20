package store

import (
	"strings"
	"time"

	"github.com/enekos/errolan/internal/models"
)

type FlaggedComment struct {
	Comment   *models.Comment `json:"comment"`
	FlagCount int             `json:"flag_count"`
}

// Flag records a flag against a comment. Returns ErrConflict if the same
// (user, comment) pair has already flagged (the SQL UNIQUE index enforces it).
func (s *Store) Flag(commentID int64, userID *int64, reason string) error {
	_, err := s.DB.Exec(
		`INSERT INTO flags (comment_id, user_id, reason, created_at) VALUES (?, ?, ?, ?)`,
		commentID, userID, reason, time.Now().Unix(),
	)
	if err != nil && strings.Contains(err.Error(), "UNIQUE") {
		return ErrConflict
	}
	return err
}

// CountFlags returns the number of distinct flags on a comment, used by the
// auto-hide-on-flag threshold.
func (s *Store) CountFlags(commentID int64) (int, error) {
	var n int
	err := s.DB.QueryRow(`SELECT COUNT(*) FROM flags WHERE comment_id = ?`, commentID).Scan(&n)
	return n, err
}

func (s *Store) ListFlagged(limit int) ([]*FlaggedComment, error) {
	rows, err := s.DB.Query(
		`SELECT `+commentColsPrefixed+`, COUNT(f.id) AS flag_count
		   FROM comments c
		   JOIN flags f ON f.comment_id = c.id
		  GROUP BY c.id
		  ORDER BY flag_count DESC, c.created_at DESC
		  LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*FlaggedComment
	for rows.Next() {
		c := &models.Comment{}
		var pinned, fc int
		if err := rows.Scan(
			&c.ID, &c.ThreadID, &c.ParentID, &c.UserID, &c.AuthorName, &c.Body,
			&c.Status, &c.Score, &pinned, &c.EditCount, &c.Anchor,
			&c.RangeQuote, &c.RangePrefix, &c.RangeSuffix, &c.RangeStart, &c.RangeEnd,
			&c.ModerationReason, &c.CreatedAt, &c.UpdatedAt, &fc,
		); err != nil {
			return nil, err
		}
		c.Pinned = pinned != 0
		out = append(out, &FlaggedComment{Comment: c, FlagCount: fc})
	}
	return out, rows.Err()
}
