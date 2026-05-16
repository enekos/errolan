package store

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/enekos/errolan/internal/models"
)

// attachReactions fills c.Reactions (code → count) for every comment in byID,
// plus c.MyReacts for the viewer (if any). Uses GROUP_CONCAT to return one row
// per comment instead of one row per reaction type, reducing Rows.Next overhead.
func (s *Store) attachReactions(byID map[int64]*models.Comment, viewerID *int64) {
	if len(byID) == 0 {
		return
	}
	args := make([]any, 0, len(byID))
	for id := range byID {
		args = append(args, id)
	}
	q := fmt.Sprintf(`SELECT comment_id, GROUP_CONCAT(code || ':' || count, ',')
		FROM reaction_counts WHERE comment_id IN (%s) GROUP BY comment_id`, inPlaceholders(len(args)))
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var cid int64
		var raw string
		if err := rows.Scan(&cid, &raw); err == nil {
			if c, ok := byID[cid]; ok {
				if c.Reactions == nil {
					c.Reactions = make(map[string]int)
				}
				for _, part := range strings.Split(raw, ",") {
					if kv := strings.SplitN(part, ":", 2); len(kv) == 2 {
						if n, err := strconv.Atoi(kv[1]); err == nil {
							c.Reactions[kv[0]] = n
						}
					}
				}
			}
		}
	}
	if viewerID == nil {
		return
	}
	mArgs := append([]any{*viewerID}, args...)
	mq := fmt.Sprintf(`SELECT comment_id, GROUP_CONCAT(code, ',')
		FROM reactions WHERE user_id = ? AND comment_id IN (%s) GROUP BY comment_id`, inPlaceholders(len(args)))
	mrows, err := s.DB.Query(mq, mArgs...)
	if err != nil {
		return
	}
	defer mrows.Close()
	for mrows.Next() {
		var cid int64
		var raw string
		if err := mrows.Scan(&cid, &raw); err == nil {
			if c, ok := byID[cid]; ok {
				for _, code := range strings.Split(raw, ",") {
					if code != "" {
						c.MyReacts = append(c.MyReacts, code)
					}
				}
			}
		}
	}
}

// ToggleReaction adds or removes a (user, comment, code) reaction. Returns the
// new count for that code and whether the viewer is now reacting with it.
func (s *Store) ToggleReaction(userID, commentID int64, code string) (count int, active bool, err error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return 0, false, err
	}
	defer tx.Rollback()

	var existed int
	err = tx.QueryRow(
		`SELECT 1 FROM reactions WHERE user_id = ? AND comment_id = ? AND code = ?`,
		userID, commentID, code,
	).Scan(&existed)
	if err != nil && err != sql.ErrNoRows {
		return 0, false, err
	}

	delta := 1
	if err == nil {
		// Already exists → remove
		if _, err := tx.Exec(`DELETE FROM reactions WHERE user_id = ? AND comment_id = ? AND code = ?`, userID, commentID, code); err != nil {
			return 0, false, err
		}
		delta = -1
		active = false
	} else {
		if _, err := tx.Exec(
			`INSERT INTO reactions (user_id, comment_id, code, created_at) VALUES (?, ?, ?, ?)`,
			userID, commentID, code, time.Now().Unix(),
		); err != nil {
			return 0, false, err
		}
		active = true
	}

	if delta > 0 {
		if _, err := tx.Exec(
			`INSERT INTO reaction_counts (comment_id, code, count) VALUES (?, ?, 1)
			 ON CONFLICT(comment_id, code) DO UPDATE SET count = count + 1`,
			commentID, code,
		); err != nil {
			return 0, false, err
		}
	} else {
		if _, err := tx.Exec(
			`UPDATE reaction_counts SET count = MAX(0, count - 1) WHERE comment_id = ? AND code = ?`,
			commentID, code,
		); err != nil {
			return 0, false, err
		}
		// Clean up zero rows so the count map stays compact.
		if _, err := tx.Exec(
			`DELETE FROM reaction_counts WHERE comment_id = ? AND code = ? AND count <= 0`,
			commentID, code,
		); err != nil {
			return 0, false, err
		}
	}

	// Keep comments.score in sync as "total reactions" for the best-sort.
	if _, err := tx.Exec(`UPDATE comments SET score = score + ? WHERE id = ?`, delta, commentID); err != nil {
		return 0, false, err
	}

	if err := tx.QueryRow(
		`SELECT COALESCE(count, 0) FROM reaction_counts WHERE comment_id = ? AND code = ?`,
		commentID, code,
	).Scan(&count); err != nil && err != sql.ErrNoRows {
		return 0, false, err
	}
	if err := tx.Commit(); err != nil {
		return 0, false, err
	}
	return count, active, nil
}
