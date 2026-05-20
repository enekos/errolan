package store

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/enekos/errolan/internal/models"
)

// UserExport is the full data payload returned by /api/me/export. JSON-shaped
// so a user can drop it straight into a vault / archive without us inventing a
// proprietary format. Pointers stay nil when there's no profile row.
type UserExport struct {
	User       *models.User             `json:"user"`
	Profile    *models.UserProfile      `json:"profile,omitempty"`
	Identities []*models.OAuthIdentity  `json:"oauth_identities,omitempty"`
	Comments   []*models.Comment        `json:"comments"`
	Reactions  []ReactionExport         `json:"reactions"`
	ExportedAt int64                    `json:"exported_at"`
}

// ReactionExport carries enough context that someone reading the export can
// see *what* a reaction was applied to (comment id) without needing to cross-
// reference. We keep the comment_id rather than copying the body because the
// body already appears in the Comments slice if it was the user's own.
type ReactionExport struct {
	CommentID int64  `json:"comment_id"`
	Code      string `json:"code"`
	CreatedAt int64  `json:"created_at"`
}

// ExportUser collects every record we hold for a user into a single payload.
// Returns ErrNotFound if the user doesn't exist.
func (s *Store) ExportUser(userID int64) (*UserExport, error) {
	user, err := s.UserByID(userID)
	if err != nil {
		return nil, err
	}
	out := &UserExport{User: user, ExportedAt: time.Now().Unix()}

	// Profile (optional).
	if p, err := s.ProfileByUserID(userID); err == nil {
		out.Profile = p
	} else if err != ErrNotFound {
		return nil, err
	}

	// OAuth identities — strip Subject from the export? No: the user already
	// knows their own provider IDs, and including them lets them prove the
	// link if they migrate to another instance.
	if ids, err := s.IdentitiesForUser(userID); err == nil {
		out.Identities = ids
	} else {
		return nil, err
	}

	// All comments (any status — the user owns this data so they get the
	// hidden / deleted rows too, with the soft-delete state preserved).
	rows, err := s.DB.Query(
		`SELECT `+commentCols+` FROM comments WHERE user_id = ? ORDER BY created_at ASC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		c, err := scanComment(rows)
		if err != nil {
			return nil, err
		}
		out.Comments = append(out.Comments, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Reactions placed by the user. We don't need to attach the target comment
	// body — that would balloon the export and isn't owned by them anyway.
	rxRows, err := s.DB.Query(
		`SELECT comment_id, code, created_at FROM reactions WHERE user_id = ? ORDER BY created_at ASC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rxRows.Close()
	for rxRows.Next() {
		var rx ReactionExport
		if err := rxRows.Scan(&rx.CommentID, &rx.Code, &rx.CreatedAt); err != nil {
			return nil, err
		}
		out.Reactions = append(out.Reactions, rx)
	}
	return out, rxRows.Err()
}

// SoftDeleteUser anonymises a user and tombstones their content. The user row
// is kept (with scrambled email + name + banned=1) rather than deleted so the
// foreign keys on comments / reactions / audit_log stay valid. Comments are
// flipped to status='deleted' which already hides body + author_name in the
// public listing path.
//
// We deliberately keep:
//   - comment rows (so reply threading isn't shattered),
//   - the user row (so foreign keys hold; reads anonymize),
//   - audit log entries (operators still need to know who took what action).
//
// We deliberately destroy:
//   - email, name, password hash, OAuth identity rows, profile row.
//
// This is the GDPR-recommended path — anonymization rather than hard delete —
// because hard-deleting a top-level comment with 50 replies would orphan them.
func (s *Store) SoftDeleteUser(userID int64) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Make sure we don't accidentally collide with an existing email by
	// scrambling rather than fixing the value. The random suffix is enough
	// entropy to avoid collisions for the lifetime of the instance.
	scrambled, err := scrambledEmail(userID)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(
		`UPDATE users SET email = ?, name = '[deleted]', password_hash = '!deleted!', is_banned = 1 WHERE id = ?`,
		scrambled, userID,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM oauth_identities WHERE user_id = ?`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM user_profiles WHERE user_id = ?`, userID); err != nil {
		return err
	}
	// Replace every comment body so personal content is gone even though the
	// row stays. We update visible counters down to zero by sending each one
	// through the soft-delete path.
	if _, err := tx.Exec(
		`UPDATE comments SET body = '[deleted]', author_name = '[deleted]' WHERE user_id = ?`,
		userID,
	); err != nil {
		return err
	}
	// Recompute thread comment_count for any thread the user posted in. A
	// targeted UPDATE per-thread keeps this O(threads touched) rather than
	// rescanning every thread on the instance.
	if _, err := tx.Exec(
		`UPDATE threads
		   SET comment_count = (
		     SELECT COUNT(*) FROM comments c
		      WHERE c.thread_id = threads.id AND c.status = 'visible'
		   )
		 WHERE id IN (SELECT DISTINCT thread_id FROM comments WHERE user_id = ?)`,
		userID,
	); err != nil {
		return err
	}
	// Flip every still-visible comment to deleted in a separate pass — body
	// has already been redacted above, so this just gates the listing path.
	if _, err := tx.Exec(
		`UPDATE comments SET status = 'deleted', updated_at = ? WHERE user_id = ? AND status = 'visible'`,
		time.Now().Unix(), userID,
	); err != nil {
		return err
	}
	// Reactions: drop them. The denormalised reaction_counts table would
	// drift, so recompute the touched rows.
	if _, err := tx.Exec(
		`UPDATE reaction_counts
		   SET count = (
		     SELECT COUNT(*) FROM reactions r
		      WHERE r.comment_id = reaction_counts.comment_id AND r.code = reaction_counts.code
		   )
		 WHERE comment_id IN (SELECT comment_id FROM reactions WHERE user_id = ?)`,
		userID,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM reactions WHERE user_id = ?`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM votes WHERE user_id = ?`, userID); err != nil {
		return err
	}
	return tx.Commit()
}

// scrambledEmail returns a deterministic-shaped but unguessable replacement
// email for an anonymised user. The shape keeps the UNIQUE constraint happy
// without leaking the original address.
func scrambledEmail(userID int64) (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("deleted-%d-%s@deleted.invalid", userID, hex.EncodeToString(b)), nil
}
