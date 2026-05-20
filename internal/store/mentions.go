package store

import (
	"database/sql"
	"strings"
	"time"

	"github.com/enekos/errolan/internal/models"
)

const mentionCols = `id, site_id, thread_id, source, target, author_name, author_url, snippet, kind, status, reason, created_at, verified_at`

func scanMention(row scanner) (*models.Mention, error) {
	m := &models.Mention{}
	if err := row.Scan(
		&m.ID, &m.SiteID, &m.ThreadID, &m.Source, &m.Target,
		&m.AuthorName, &m.AuthorURL, &m.Snippet,
		&m.Kind, &m.Status, &m.Reason,
		&m.CreatedAt, &m.VerifiedAt,
	); err != nil {
		return nil, err
	}
	return m, nil
}

// EnqueueMention records (or replaces) a pending mention. Re-submitting the
// same source for the same thread restarts verification with a fresh row —
// this is the Webmention spec behaviour (a publisher can edit a post and the
// receiver should re-check).
func (s *Store) EnqueueMention(siteID, threadID int64, source, target, kind string) (*models.Mention, error) {
	now := time.Now().Unix()
	// Try insert; on UNIQUE collision, reset the existing row to pending.
	res, err := s.DB.Exec(
		`INSERT INTO mentions (site_id, thread_id, source, target, kind, status, created_at)
		 VALUES (?, ?, ?, ?, ?, 'pending', ?)`,
		siteID, threadID, source, target, kind, now,
	)
	if err != nil {
		if !strings.Contains(err.Error(), "UNIQUE") {
			return nil, err
		}
		if _, err := s.DB.Exec(
			`UPDATE mentions SET status='pending', reason='', verified_at=0, created_at=?, kind=? WHERE thread_id=? AND source=?`,
			now, kind, threadID, source,
		); err != nil {
			return nil, err
		}
		row := s.DB.QueryRow(
			`SELECT `+mentionCols+` FROM mentions WHERE thread_id=? AND source=?`,
			threadID, source,
		)
		return scanMention(row)
	}
	id, _ := res.LastInsertId()
	return s.MentionByID(id)
}

func (s *Store) MentionByID(id int64) (*models.Mention, error) {
	row := s.DB.QueryRow(`SELECT `+mentionCols+` FROM mentions WHERE id = ?`, id)
	m, err := scanMention(row)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return m, err
}

// MarkMentionVerified finalises a row with the extracted author + snippet.
func (s *Store) MarkMentionVerified(id int64, authorName, authorURL, snippet string) error {
	_, err := s.DB.Exec(
		`UPDATE mentions SET status='verified', author_name=?, author_url=?, snippet=?, reason='', verified_at=? WHERE id=?`,
		authorName, authorURL, snippet, time.Now().Unix(), id,
	)
	return err
}

// MarkMentionVerifiedDraft stashes the snippet + author *without* flipping the
// row to verified. Used by the ActivityPub inbox handler — we already have the
// note content but want the verifier worker to confirm the actor is reachable
// before we surface the mention to readers.
func (s *Store) MarkMentionVerifiedDraft(id int64, authorName, authorURL, snippet string) error {
	_, err := s.DB.Exec(
		`UPDATE mentions SET author_name=?, author_url=?, snippet=? WHERE id=? AND status='pending'`,
		authorName, authorURL, snippet, id,
	)
	return err
}

func (s *Store) MarkMentionRejected(id int64, reason string) error {
	_, err := s.DB.Exec(
		`UPDATE mentions SET status='rejected', reason=? WHERE id=?`,
		reason, id,
	)
	return err
}

// ListThreadMentions returns the visible (verified) mentions for a thread,
// most-recent-first. Pending and rejected mentions are kept for audit/moderation
// but not surfaced in the public listing.
func (s *Store) ListThreadMentions(threadID int64) ([]*models.Mention, error) {
	rows, err := s.DB.Query(
		`SELECT `+mentionCols+` FROM mentions WHERE thread_id=? AND status='verified' ORDER BY created_at DESC`,
		threadID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.Mention
	for rows.Next() {
		m, err := scanMention(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ListPendingMentions returns the verifier work queue (pending only).
func (s *Store) ListPendingMentions(limit int) ([]*models.Mention, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.DB.Query(
		`SELECT `+mentionCols+` FROM mentions WHERE status='pending' ORDER BY created_at ASC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.Mention
	for rows.Next() {
		m, err := scanMention(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ThreadByURL looks a thread up by its stored URL. This is what lets us match
// a Webmention's `target` URL back to a thread on our side.
func (s *Store) ThreadByURL(siteID int64, url string) (*models.Thread, error) {
	row := s.DB.QueryRow(
		`SELECT `+threadCols+` FROM threads WHERE site_id = ? AND url = ? LIMIT 1`,
		siteID, url,
	)
	t, err := scanThread(row)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return t, err
}
