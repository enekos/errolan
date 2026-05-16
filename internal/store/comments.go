package store

import (
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/enekos/errolan/internal/models"
)

const commentCols = `id, thread_id, parent_id, user_id, author_name, body, status, score, pinned, edit_count, anchor, moderation_reason, created_at, updated_at`

// commentColsPrefixed is `c.id, c.thread_id, …` for queries that join the
// comments table to others where a bare column name would be ambiguous.
const commentColsPrefixed = `c.id, c.thread_id, c.parent_id, c.user_id, c.author_name, c.body, c.status, c.score, c.pinned, c.edit_count, c.anchor, c.moderation_reason, c.created_at, c.updated_at`

func scanCommentInto(c *models.Comment, row scanner) error {
	var pinned int
	if err := row.Scan(
		&c.ID, &c.ThreadID, &c.ParentID, &c.UserID, &c.AuthorName, &c.Body,
		&c.Status, &c.Score, &pinned, &c.EditCount, &c.Anchor,
		&c.ModerationReason, &c.CreatedAt, &c.UpdatedAt,
	); err != nil {
		return err
	}
	c.Pinned = pinned != 0
	return nil
}

func scanCommentWithVote(c *models.Comment, row scanner) error {
	var pinned int
	if err := row.Scan(
		&c.ID, &c.ThreadID, &c.ParentID, &c.UserID, &c.AuthorName, &c.Body,
		&c.Status, &c.Score, &pinned, &c.EditCount, &c.Anchor,
		&c.ModerationReason, &c.CreatedAt, &c.UpdatedAt, &c.MyVote,
	); err != nil {
		return err
	}
	c.Pinned = pinned != 0
	return nil
}

func scanComment(row scanner) (*models.Comment, error) {
	c := &models.Comment{}
	if err := scanCommentInto(c, row); err != nil {
		return nil, err
	}
	return c, nil
}

// SortOrder controls top-level comment ordering. Replies always sort by created_at.
type SortOrder string

const (
	SortBest   SortOrder = "best"
	SortNewest SortOrder = "newest"
	SortOldest SortOrder = "oldest"
)

func (s SortOrder) clause() string {
	switch s {
	case SortNewest:
		return "c.pinned DESC, c.created_at DESC"
	case SortOldest:
		return "c.pinned DESC, c.created_at ASC"
	default:
		return "c.pinned DESC, c.score DESC, c.created_at ASC"
	}
}

type ListCommentsOpts struct {
	Sort           SortOrder
	Limit          int // top-level pagination; 0 = all
	BeforeID       int64
	ViewerID       *int64
	IncludePending bool // admins see comments still waiting in the queue
}

// CreateComment inserts a comment with the supplied moderation status. Pending
// comments do NOT bump the denormalised thread counter — only public, visible
// comments count toward what readers will eventually see.
func (s *Store) CreateComment(threadID int64, parentID *int64, userID *int64, authorName, body, email, anchor, status, modReason string) (*models.Comment, error) {
	if status == "" {
		status = models.CommentStatusVisible
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	now := time.Now().Unix()
	res, err := tx.Exec(
		`INSERT INTO comments (thread_id, parent_id, user_id, author_name, body, status, score, pinned, edit_count, anchor, moderation_reason, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, 0, 0, 0, ?, ?, ?, ?)`,
		threadID, parentID, userID, authorName, body, status, anchor, modReason, now, now,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()

	if status == models.CommentStatusVisible {
		if _, err := tx.Exec(
			`UPDATE threads SET comment_count = comment_count + 1, last_comment_at = ? WHERE id = ?`,
			now, threadID,
		); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	c, err := s.CommentByID(id, nil)
	if err != nil {
		return nil, err
	}
	if email != "" {
		c.AvatarURL = gravatarURL(email)
	}
	return c, nil
}

// CountUserComments is used by the moderation engine to decide whether a
// registered user is still "new" for the hold-new-users policy.
func (s *Store) CountUserComments(userID int64) (int, error) {
	var n int
	err := s.DB.QueryRow(`SELECT COUNT(*) FROM comments WHERE user_id = ?`, userID).Scan(&n)
	return n, err
}

func (s *Store) CommentByID(id int64, viewerID *int64) (*models.Comment, error) {
	viewerIDArg := int64(-1)
	if viewerID != nil {
		viewerIDArg = *viewerID
	}
	row := s.DB.QueryRow(
		`SELECT `+commentCols+`, COALESCE(v.value, 0) FROM comments c LEFT JOIN votes v ON v.comment_id = c.id AND v.user_id = ? WHERE c.id = ?`,
		viewerIDArg, id,
	)
	c := &models.Comment{}
	if err := scanCommentWithVote(c, row); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return c, nil
}

func (s *Store) ListThreadComments(threadID int64, opts ListCommentsOpts) (roots []*models.Comment, hasMore bool, err error) {
	if opts.Sort == "" {
		opts.Sort = SortBest
	}

	top, hasMore, err := s.listTopLevelComments(threadID, opts)
	if err != nil {
		return nil, false, err
	}
	if len(top) == 0 {
		return nil, hasMore, nil
	}

	byID := make(map[int64]*models.Comment, len(top)*6)
	for _, c := range top {
		byID[c.ID] = c
	}
	replies, err := s.listRepliesFor(byID, opts.IncludePending, opts.ViewerID)
	if err != nil {
		return nil, false, err
	}
	for _, c := range replies {
		byID[c.ID] = c
	}

	s.attachAvatars(byID)
	s.attachReactions(byID, opts.ViewerID)

	// Soft-deleted comments hide author + body in public listings.
	for _, c := range byID {
		if c.Status == models.CommentStatusDeleted {
			c.Body = "[deleted]"
			c.AuthorName = "[deleted]"
			c.AvatarURL = ""
		}
	}

	// Stitch replies onto their parents. Orphans stay in byID but unrendered.
	for _, c := range replies {
		if c.ParentID == nil {
			continue
		}
		if p, ok := byID[*c.ParentID]; ok {
			p.Replies = append(p.Replies, c)
		}
	}
	return top, hasMore, nil
}

// listTopLevelComments fetches the paginated parent_id IS NULL set.
func (s *Store) listTopLevelComments(threadID int64, opts ListCommentsOpts) ([]*models.Comment, bool, error) {
	args := []any{threadID}
	where := "c.thread_id = ? AND c.parent_id IS NULL"
	if !opts.IncludePending {
		where += " AND c.status != 'pending'"
	}
	if opts.BeforeID > 0 {
		where += " AND c.id < ?"
		args = append(args, opts.BeforeID)
	}
	limitClause := ""
	if opts.Limit > 0 {
		limitClause = fmt.Sprintf(" LIMIT %d", opts.Limit+1)
	}

	viewerIDArg := int64(-1)
	if opts.ViewerID != nil {
		viewerIDArg = *opts.ViewerID
	}
	args = append([]any{viewerIDArg}, args...)
	q := fmt.Sprintf(`SELECT %s, COALESCE(v.value, 0) FROM comments c LEFT JOIN votes v ON v.comment_id = c.id AND v.user_id = ? WHERE %s ORDER BY %s%s`, commentColsPrefixed, where, opts.Sort.clause(), limitClause)

	rows, err := s.DB.Query(q, args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	topBuf := make([]models.Comment, opts.Limit+1)
	top := make([]*models.Comment, 0, opts.Limit+1)
	i := 0
	for rows.Next() {
		if err := scanCommentWithVote(&topBuf[i], rows); err != nil {
			return nil, false, err
		}
		top = append(top, &topBuf[i])
		i++
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	hasMore := false
	if opts.Limit > 0 && len(top) > opts.Limit {
		top = top[:opts.Limit]
		hasMore = true
	}
	return top, hasMore, nil
}

// listRepliesFor fetches every reply whose parent_id is one of the top-level
// comment IDs already in byID, in a single round-trip.
func (s *Store) listRepliesFor(byID map[int64]*models.Comment, includePending bool, viewerID *int64) ([]*models.Comment, error) {
	ids := make([]int64, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	statusFilter := ""
	if !includePending {
		statusFilter = " AND c.status != 'pending'"
	}

	viewerIDArg := int64(-1)
	if viewerID != nil {
		viewerIDArg = *viewerID
	}
	args = append([]any{viewerIDArg}, args...)
	placeholders := inPlaceholders(len(byID))
	q := fmt.Sprintf(`SELECT %s, COALESCE(v.value, 0) FROM comments c LEFT JOIN votes v ON v.comment_id = c.id AND v.user_id = ? WHERE c.parent_id IN (%s)%s ORDER BY c.created_at ASC`,
		commentColsPrefixed, placeholders, statusFilter)
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	est := len(byID) * 5
	if est < 64 {
		est = 64
	}
	replyBuf := make([]models.Comment, est)
	out := make([]*models.Comment, 0, est)
	i := 0
	for rows.Next() {
		var c *models.Comment
		if i < len(replyBuf) {
			c = &replyBuf[i]
			if err := scanCommentWithVote(c, rows); err != nil {
				return nil, err
			}
		} else {
			c = &models.Comment{}
			if err := scanCommentWithVote(c, rows); err != nil {
				return nil, err
			}
		}
		out = append(out, c)
		i++
	}
	return out, rows.Err()
}

// attachAvatars batches a single SELECT for the registered-user emails referenced
// by the comments and populates AvatarURL from the Gravatar hash.
func (s *Store) attachAvatars(byID map[int64]*models.Comment) {
	userIDs := make(map[int64]struct{})
	for _, c := range byID {
		if c.UserID != nil {
			userIDs[*c.UserID] = struct{}{}
		}
	}
	if len(userIDs) == 0 {
		return
	}
	ids := make([]int64, 0, len(userIDs))
	for id := range userIDs {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	q := "SELECT id, email FROM users WHERE id IN (" + inPlaceholders(len(userIDs)) + ")"
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		return
	}
	defer rows.Close()
	emails := make(map[int64]string, len(userIDs))
	for rows.Next() {
		var id int64
		var email string
		if err := rows.Scan(&id, &email); err == nil {
			emails[id] = email
		}
	}
	for _, c := range byID {
		if c.UserID != nil {
			if e, ok := emails[*c.UserID]; ok {
				c.AvatarURL = gravatarURL(e)
			}
		}
	}
}

func (s *Store) UpdateCommentBody(id int64, body string) error {
	_, err := s.DB.Exec(
		`UPDATE comments SET body = ?, edit_count = edit_count + 1, updated_at = ? WHERE id = ?`,
		body, time.Now().Unix(), id,
	)
	return err
}

// SetCommentStatus updates a comment's status, keeping the denormalised thread
// counter in sync so only publicly visible comments count toward comment_count.
func (s *Store) SetCommentStatus(id int64, status string) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var threadID int64
	var prev string
	if err := tx.QueryRow(`SELECT thread_id, status FROM comments WHERE id = ?`, id).Scan(&threadID, &prev); err != nil {
		return err
	}
	if prev == status {
		return tx.Commit()
	}
	if _, err := tx.Exec(`UPDATE comments SET status = ?, updated_at = ? WHERE id = ?`, status, time.Now().Unix(), id); err != nil {
		return err
	}
	wasVisible := prev == models.CommentStatusVisible
	nowVisible := status == models.CommentStatusVisible
	if wasVisible != nowVisible {
		delta := -1
		if nowVisible {
			delta = 1
		}
		if _, err := tx.Exec(`UPDATE threads SET comment_count = MAX(0, comment_count + ?) WHERE id = ?`, delta, threadID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) ApproveComment(id int64) error {
	return s.SetCommentStatus(id, models.CommentStatusVisible)
}

func (s *Store) RejectComment(id int64) error {
	return s.SetCommentStatus(id, models.CommentStatusHidden)
}

func (s *Store) SetModerationReason(id int64, reason string) error {
	_, err := s.DB.Exec(`UPDATE comments SET moderation_reason = ? WHERE id = ?`, reason, id)
	return err
}

func (s *Store) SetCommentPinned(id int64, pinned bool) error {
	_, err := s.DB.Exec(`UPDATE comments SET pinned = ? WHERE id = ?`, boolInt(pinned), id)
	return err
}

// ListPendingComments returns comments awaiting review across every site,
// most-recent-first. Optional siteID filter when an admin wants a single feed.
func (s *Store) ListPendingComments(siteID int64, limit int) ([]*models.Comment, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	q := `SELECT ` + commentCols + ` FROM comments c WHERE c.status = 'pending'`
	args := []any{}
	if siteID > 0 {
		q += ` AND c.thread_id IN (SELECT id FROM threads WHERE site_id = ?)`
		args = append(args, siteID)
	}
	q += ` ORDER BY c.created_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.DB.Query(q, args...)
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
