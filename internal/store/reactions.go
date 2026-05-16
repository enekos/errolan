package store

import (
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/enekos/errolan/internal/models"
)

// parseReactions parses a GROUP_CONCAT result like "like:5,heart:3" into the
// provided map without allocating intermediate slices.
func parseReactions(raw string, m map[string]int) {
	start := 0
	for i := 0; i <= len(raw); i++ {
		if i == len(raw) || raw[i] == ',' {
			part := raw[start:i]
			if j := strings.IndexByte(part, ':'); j != -1 {
				if n, err := strconv.Atoi(part[j+1:]); err == nil {
					m[part[:j]] = n
				}
			}
			start = i + 1
		}
	}
}

// parseMyReacts parses a GROUP_CONCAT result like "like,heart" and appends
// each non-empty code to out without allocating intermediate slices.
func parseMyReacts(raw string, out []string) []string {
	start := 0
	for i := 0; i <= len(raw); i++ {
		if i == len(raw) || raw[i] == ',' {
			if code := raw[start:i]; code != "" {
				out = append(out, code)
			}
			start = i + 1
		}
	}
	return out
}

// attachReactions fills c.Reactions (code → count) for every comment in byID,
// plus c.MyReacts for the viewer (if any). Uses GROUP_CONCAT to return one row
// per comment instead of one row per reaction type, reducing Rows.Next overhead.
// When a viewer is present, a correlated subquery fetches the viewer's own
// reactions in the same round-trip without a JOIN.
func (s *Store) attachReactions(byID map[int64]*models.Comment, viewerID *int64) {
	if len(byID) == 0 {
		return
	}
	ids := make([]int64, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	// Sort IDs so SQLite can do sequential index scans instead of random lookups.
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	// Try cache first.
	missing := make([]int64, 0, len(ids))
	s.reactMu.RLock()
	for _, id := range ids {
		if entry, ok := s.reactCache[id]; ok {
			if c := byID[id]; c != nil {
				if entry.reactions != nil {
					c.Reactions = make(map[string]int, len(entry.reactions))
					for k, v := range entry.reactions {
						c.Reactions[k] = v
					}
				}
				if viewerID != nil {
					if codes, ok := entry.myReacts[*viewerID]; ok {
						c.MyReacts = make([]string, len(codes))
						copy(c.MyReacts, codes)
					}
				}
			}
		} else {
			missing = append(missing, id)
		}
	}
	s.reactMu.RUnlock()
	if len(missing) == 0 {
		return
	}

	args := make([]any, 0, len(missing))
	for _, id := range missing {
		args = append(args, id)
	}

	if viewerID != nil {
		q := fmt.Sprintf(`SELECT rc.comment_id, GROUP_CONCAT(rc.code || ':' || rc.count, ','),
			(SELECT GROUP_CONCAT(code, ',') FROM reactions WHERE user_id = ? AND comment_id = rc.comment_id)
		FROM reaction_counts rc WHERE rc.comment_id IN (%s) GROUP BY rc.comment_id`, inPlaceholders(len(args)))
		rows, err := s.DB.Query(q, append([]any{*viewerID}, args...)...)
		if err != nil {
			return
		}
		defer rows.Close()
		for rows.Next() {
			var cid int64
			var reactionsRaw string
			var myReactsRaw sql.NullString
			if err := rows.Scan(&cid, &reactionsRaw, &myReactsRaw); err == nil {
				if c, ok := byID[cid]; ok {
					if c.Reactions == nil {
						c.Reactions = make(map[string]int)
					}
					parseReactions(reactionsRaw, c.Reactions)
					if myReactsRaw.Valid {
						c.MyReacts = parseMyReacts(myReactsRaw.String, c.MyReacts)
					}
				}
			}
		}
		// Populate cache for queried comments.
		s.reactMu.Lock()
		for _, id := range missing {
			if c := byID[id]; c != nil {
				entry := reactCacheEntry{
					reactions: make(map[string]int, len(c.Reactions)),
					myReacts:  make(map[int64][]string),
				}
				for k, v := range c.Reactions {
					entry.reactions[k] = v
				}
				entry.myReacts[*viewerID] = make([]string, len(c.MyReacts))
				copy(entry.myReacts[*viewerID], c.MyReacts)
				s.reactCache[id] = entry
			}
		}
		s.reactMu.Unlock()
		return
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
				parseReactions(raw, c.Reactions)
			}
		}
	}
	// Populate cache for queried comments.
	s.reactMu.Lock()
	for _, id := range missing {
		if c := byID[id]; c != nil {
			entry := reactCacheEntry{
				reactions: make(map[string]int, len(c.Reactions)),
				myReacts:  make(map[int64][]string),
			}
			for k, v := range c.Reactions {
				entry.reactions[k] = v
			}
			s.reactCache[id] = entry
		}
	}
	s.reactMu.Unlock()
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
	s.reactMu.Lock()
	delete(s.reactCache, commentID)
	s.reactMu.Unlock()
	return count, active, nil
}
