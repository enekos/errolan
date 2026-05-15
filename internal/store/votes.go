package store

import (
	"database/sql"
	"time"
)

// Vote sets a user's vote on a comment to value ∈ {-1, 0, 1} and returns the
// resulting score. Re-voting with the same value is a no-op for storage.
func (s *Store) Vote(userID, commentID int64, value int) (int, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var existing int
	err = tx.QueryRow(`SELECT value FROM votes WHERE user_id = ? AND comment_id = ?`, userID, commentID).Scan(&existing)
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}
	delta := value - existing

	// Re-voting with the same value is a no-op: just return current score.
	if delta == 0 {
		var score int
		if err := tx.QueryRow(`SELECT score FROM comments WHERE id = ?`, commentID).Scan(&score); err != nil {
			return 0, err
		}
		return score, tx.Commit()
	}

	switch {
	case value == 0:
		if _, err := tx.Exec(`DELETE FROM votes WHERE user_id = ? AND comment_id = ?`, userID, commentID); err != nil {
			return 0, err
		}
	case err == sql.ErrNoRows:
		if _, err := tx.Exec(
			`INSERT INTO votes (user_id, comment_id, value, created_at) VALUES (?, ?, ?, ?)`,
			userID, commentID, value, time.Now().Unix(),
		); err != nil {
			return 0, err
		}
	default:
		if _, err := tx.Exec(`UPDATE votes SET value = ? WHERE user_id = ? AND comment_id = ?`, value, userID, commentID); err != nil {
			return 0, err
		}
	}

	if _, err := tx.Exec(`UPDATE comments SET score = score + ? WHERE id = ?`, delta, commentID); err != nil {
		return 0, err
	}
	var score int
	if err := tx.QueryRow(`SELECT score FROM comments WHERE id = ?`, commentID).Scan(&score); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return score, nil
}
