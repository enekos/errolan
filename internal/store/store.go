package store

import (
	"crypto/md5"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/enekosarasola/errolan/internal/models"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
)

type Store struct {
	DB *sql.DB
}

func New(db *sql.DB) *Store { return &Store{DB: db} }

func newAPIKey() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return "erl_" + hex.EncodeToString(b)
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ---------- Sites ----------

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
	rows, err := s.DB.Query(`SELECT id, slug, name, api_key, allowed_origins, require_auth, created_at FROM sites ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.Site
	for rows.Next() {
		var st models.Site
		var ra int
		if err := rows.Scan(&st.ID, &st.Slug, &st.Name, &st.APIKey, &st.AllowedOrigins, &ra, &st.CreatedAt); err != nil {
			return nil, err
		}
		st.RequireAuth = ra != 0
		out = append(out, &st)
	}
	return out, rows.Err()
}

func (s *Store) SiteByAPIKey(key string) (*models.Site, error) {
	var st models.Site
	var ra int
	err := s.DB.QueryRow(
		`SELECT id, slug, name, api_key, allowed_origins, require_auth, created_at FROM sites WHERE api_key = ?`, key,
	).Scan(&st.ID, &st.Slug, &st.Name, &st.APIKey, &st.AllowedOrigins, &ra, &st.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	st.RequireAuth = ra != 0
	return &st, nil
}

func (s *Store) SiteBySlug(slug string) (*models.Site, error) {
	var st models.Site
	var ra int
	err := s.DB.QueryRow(
		`SELECT id, slug, name, api_key, allowed_origins, require_auth, created_at FROM sites WHERE slug = ?`, slug,
	).Scan(&st.ID, &st.Slug, &st.Name, &st.APIKey, &st.AllowedOrigins, &ra, &st.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	st.RequireAuth = ra != 0
	return &st, nil
}

// ---------- Users ----------

func (s *Store) CreateUser(email, name, passwordHash string, isAdmin bool) (*models.User, error) {
	u := &models.User{
		Email:        email,
		Name:         name,
		PasswordHash: passwordHash,
		IsAdmin:      isAdmin,
		CreatedAt:    time.Now().Unix(),
	}
	res, err := s.DB.Exec(
		`INSERT INTO users (email, name, password_hash, is_admin, created_at) VALUES (?, ?, ?, ?, ?)`,
		u.Email, u.Name, u.PasswordHash, boolInt(u.IsAdmin), u.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	u.ID = id
	return u, nil
}

func scanUser(row interface{ Scan(...any) error }) (*models.User, error) {
	var u models.User
	var admin, banned int
	if err := row.Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &admin, &banned, &u.CreatedAt); err != nil {
		return nil, err
	}
	u.IsAdmin = admin != 0
	u.IsBanned = banned != 0
	return &u, nil
}

func (s *Store) UserByEmail(email string) (*models.User, error) {
	row := s.DB.QueryRow(`SELECT id, email, name, password_hash, is_admin, is_banned, created_at FROM users WHERE email = ?`, email)
	u, err := scanUser(row)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return u, err
}

func (s *Store) UserByID(id int64) (*models.User, error) {
	row := s.DB.QueryRow(`SELECT id, email, name, password_hash, is_admin, is_banned, created_at FROM users WHERE id = ?`, id)
	u, err := scanUser(row)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return u, err
}

func (s *Store) ListUsers(limit, offset int) ([]*models.User, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.DB.Query(
		`SELECT id, email, name, password_hash, is_admin, is_banned, created_at FROM users ORDER BY id DESC LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *Store) SetUserBanned(id int64, banned bool) error {
	_, err := s.DB.Exec(`UPDATE users SET is_banned = ? WHERE id = ?`, boolInt(banned), id)
	return err
}

func (s *Store) CountUsers() (int, error) {
	var n int
	err := s.DB.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

// ---------- Threads ----------

func scanThread(row interface{ Scan(...any) error }) (*models.Thread, error) {
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
		if title != "" || url != "" {
			if _, e := s.DB.Exec(`UPDATE threads SET title = COALESCE(NULLIF(?, ''), title), url = COALESCE(NULLIF(?, ''), url) WHERE id = ?`, title, url, t.ID); e == nil {
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
	row := s.DB.QueryRow(
		`SELECT id, site_id, slug, title, url, locked, comment_count, last_comment_at, created_at FROM threads WHERE site_id = ? AND slug = ?`,
		siteID, slug,
	)
	t, err := scanThread(row)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return t, err
}

func (s *Store) ThreadByID(id int64) (*models.Thread, error) {
	row := s.DB.QueryRow(
		`SELECT id, site_id, slug, title, url, locked, comment_count, last_comment_at, created_at FROM threads WHERE id = ?`,
		id,
	)
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

// ---------- Comments ----------

// attachAvatars looks up the email for each unique registered user_id referenced
// by the comments and populates AvatarURL from the gravatar hash.
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
	args := make([]any, 0, len(userIDs))
	for id := range userIDs {
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

// inPlaceholders builds a "?,?,?,..." string with n placeholders efficiently.
func inPlaceholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat("?,", n-1) + "?"
}

func gravatarURL(email string) string {
	if email == "" {
		return ""
	}
	sum := md5.Sum([]byte(strings.ToLower(strings.TrimSpace(email))))
	return fmt.Sprintf("https://www.gravatar.com/avatar/%s?s=64&d=mp", hex.EncodeToString(sum[:]))
}

func scanComment(row interface{ Scan(...any) error }) (*models.Comment, error) {
	c := &models.Comment{}
	var pinned int
	if err := row.Scan(&c.ID, &c.ThreadID, &c.ParentID, &c.UserID, &c.AuthorName, &c.Body, &c.Status, &c.Score, &pinned, &c.EditCount, &c.Anchor, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, err
	}
	c.Pinned = pinned != 0
	return c, nil
}

const commentCols = `id, thread_id, parent_id, user_id, author_name, body, status, score, pinned, edit_count, anchor, created_at, updated_at`

func (s *Store) CreateComment(threadID int64, parentID *int64, userID *int64, authorName, body, email, anchor string) (*models.Comment, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	now := time.Now().Unix()
	res, err := tx.Exec(
		`INSERT INTO comments (thread_id, parent_id, user_id, author_name, body, status, score, pinned, edit_count, anchor, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, 'visible', 0, 0, 0, ?, ?, ?)`,
		threadID, parentID, userID, authorName, body, anchor, now, now,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()

	if _, err := tx.Exec(`UPDATE threads SET comment_count = comment_count + 1, last_comment_at = ? WHERE id = ?`, now, threadID); err != nil {
		return nil, err
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

func (s *Store) CommentByID(id int64, viewerID *int64) (*models.Comment, error) {
	row := s.DB.QueryRow(`SELECT `+commentCols+` FROM comments WHERE id = ?`, id)
	c, err := scanComment(row)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if viewerID != nil {
		s.DB.QueryRow(`SELECT value FROM votes WHERE user_id = ? AND comment_id = ?`, *viewerID, c.ID).Scan(&c.MyVote)
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
		return "pinned DESC, created_at DESC"
	case SortOldest:
		return "pinned DESC, created_at ASC"
	default:
		return "pinned DESC, score DESC, created_at ASC"
	}
}

type ListCommentsOpts struct {
	Sort     SortOrder
	Limit    int // top-level pagination; 0 = all
	BeforeID int64
	ViewerID *int64
}

func (s *Store) ListThreadComments(threadID int64, opts ListCommentsOpts) (roots []*models.Comment, hasMore bool, err error) {
	if opts.Sort == "" {
		opts.Sort = SortBest
	}

	// First fetch the paginated top-level (parent_id IS NULL) set.
	var args []any
	args = append(args, threadID)
	where := "thread_id = ? AND parent_id IS NULL"
	if opts.BeforeID > 0 {
		where += " AND id < ?"
		args = append(args, opts.BeforeID)
	}
	limitClause := ""
	if opts.Limit > 0 {
		limitClause = fmt.Sprintf(" LIMIT %d", opts.Limit+1)
	}
	q := fmt.Sprintf(`SELECT %s FROM comments WHERE %s ORDER BY %s%s`, commentCols, where, opts.Sort.clause(), limitClause)
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	var top []*models.Comment
	for rows.Next() {
		c, err := scanComment(rows)
		if err != nil {
			return nil, false, err
		}
		top = append(top, c)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	if opts.Limit > 0 && len(top) > opts.Limit {
		top = top[:opts.Limit]
		hasMore = true
	}

	if len(top) == 0 {
		return nil, hasMore, nil
	}

	// Then fetch every reply for these top-level comments in a single scan.
	byID := make(map[int64]*models.Comment, len(top)*6)
	for _, c := range top {
		byID[c.ID] = c
	}

	// Only fetch replies whose parent_id is one of the returned top-level comments.
	replyArgs := make([]any, 0, len(top))
	for id := range byID {
		replyArgs = append(replyArgs, id)
	}
	q2 := fmt.Sprintf(`SELECT `+commentCols+` FROM comments WHERE parent_id IN (%s) ORDER BY created_at ASC`, inPlaceholders(len(replyArgs)))
	rrows, err := s.DB.Query(q2, replyArgs...)
	if err != nil {
		return nil, false, err
	}
	defer rrows.Close()
	var replies []*models.Comment
	for rrows.Next() {
		c, err := scanComment(rrows)
		if err != nil {
			return nil, false, err
		}
		replies = append(replies, c)
		byID[c.ID] = c
	}
	if err := rrows.Err(); err != nil {
		return nil, false, err
	}

	// Apply viewer votes in one shot.
	if opts.ViewerID != nil {
		voteArgs := make([]any, 0, len(byID)+1)
		voteArgs = append(voteArgs, *opts.ViewerID)
		for id := range byID {
			voteArgs = append(voteArgs, id)
		}
		vq := fmt.Sprintf(`SELECT comment_id, value FROM votes WHERE user_id = ? AND comment_id IN (%s)`, inPlaceholders(len(byID)))
		vrows, err := s.DB.Query(vq, voteArgs...)
		if err == nil {
			defer vrows.Close()
			votes := make(map[int64]int, len(byID))
			for vrows.Next() {
				var cid int64
				var v int
				if err := vrows.Scan(&cid, &v); err == nil {
					votes[cid] = v
				}
			}
			for _, c := range byID {
				if v, ok := votes[c.ID]; ok {
					c.MyVote = v
				}
			}
		}
	}

	// Batch-attach gravatar URLs for all visible registered authors.
	s.attachAvatars(byID)

	// Batch-attach reaction counts + the viewer's own reactions.
	s.attachReactions(byID, opts.ViewerID)

	// Soft-deleted display.
	for _, c := range byID {
		if c.Status == models.CommentStatusDeleted {
			c.Body = "[deleted]"
			c.AuthorName = "[deleted]"
			c.AvatarURL = ""
		}
	}

	// Attach replies to their parents; drop orphans (parent not in top set is fine — it stays in byID).
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

func (s *Store) UpdateCommentBody(id int64, body string) error {
	_, err := s.DB.Exec(`UPDATE comments SET body = ?, edit_count = edit_count + 1, updated_at = ? WHERE id = ?`, body, time.Now().Unix(), id)
	return err
}

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
	wasVisible := prev != models.CommentStatusDeleted
	nowVisible := status != models.CommentStatusDeleted
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

func (s *Store) SetCommentPinned(id int64, pinned bool) error {
	_, err := s.DB.Exec(`UPDATE comments SET pinned = ? WHERE id = ?`, boolInt(pinned), id)
	return err
}

// ---------- Votes ----------

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

	if value == 0 {
		if _, err := tx.Exec(`DELETE FROM votes WHERE user_id = ? AND comment_id = ?`, userID, commentID); err != nil {
			return 0, err
		}
	} else if err == sql.ErrNoRows {
		if _, err := tx.Exec(`INSERT INTO votes (user_id, comment_id, value, created_at) VALUES (?, ?, ?, ?)`,
			userID, commentID, value, time.Now().Unix()); err != nil {
			return 0, err
		}
	} else {
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

// ---------- Flags ----------

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

type FlaggedComment struct {
	Comment   *models.Comment `json:"comment"`
	FlagCount int             `json:"flag_count"`
}

func (s *Store) ListFlagged(limit int) ([]*FlaggedComment, error) {
	rows, err := s.DB.Query(
		`SELECT c.id, c.thread_id, c.parent_id, c.user_id, c.author_name, c.body, c.status,
		        c.score, c.pinned, c.edit_count, c.anchor, c.created_at, c.updated_at, COUNT(f.id) AS flag_count
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
		var pinned int
		var fc int
		if err := rows.Scan(&c.ID, &c.ThreadID, &c.ParentID, &c.UserID, &c.AuthorName, &c.Body, &c.Status, &c.Score, &pinned, &c.EditCount, &c.Anchor, &c.CreatedAt, &c.UpdatedAt, &fc); err != nil {
			return nil, err
		}
		c.Pinned = pinned != 0
		out = append(out, &FlaggedComment{Comment: c, FlagCount: fc})
	}
	return out, rows.Err()
}

// ---------- Audit log ----------

func (s *Store) AddAudit(actorID *int64, actorName, action, kind string, targetID int64, metadata string) {
	_, _ = s.DB.Exec(
		`INSERT INTO audit_log (actor_id, actor_name, action, target_kind, target_id, metadata, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		actorID, actorName, action, kind, targetID, metadata, time.Now().Unix(),
	)
}

func (s *Store) ListAudit(limit int) ([]*models.AuditEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.DB.Query(
		`SELECT id, actor_id, actor_name, action, target_kind, target_id, metadata, created_at FROM audit_log ORDER BY id DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.AuditEntry
	for rows.Next() {
		e := &models.AuditEntry{}
		if err := rows.Scan(&e.ID, &e.ActorID, &e.ActorName, &e.Action, &e.TargetKind, &e.TargetID, &e.Metadata, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ---------- Reactions ----------

// attachReactions fills c.Reactions (code → count) for every comment in byID,
// plus c.MyReacts for the viewer (if any), in two queries.
func (s *Store) attachReactions(byID map[int64]*models.Comment, viewerID *int64) {
	if len(byID) == 0 {
		return
	}
	args := make([]any, 0, len(byID))
	for id := range byID {
		args = append(args, id)
	}
	q := "SELECT comment_id, code, count FROM reaction_counts WHERE comment_id IN (" + inPlaceholders(len(byID)) + ")"
	rows, err := s.DB.Query(q, args...)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var cid int64
			var code string
			var count int
			if err := rows.Scan(&cid, &code, &count); err == nil {
				if c, ok := byID[cid]; ok {
					if c.Reactions == nil {
						c.Reactions = make(map[string]int)
					}
					c.Reactions[code] = count
				}
			}
		}
	}
	if viewerID == nil {
		return
	}
	margs := append([]any{*viewerID}, args...)
	mq := "SELECT comment_id, code FROM reactions WHERE user_id = ? AND comment_id IN (" + inPlaceholders(len(byID)) + ")"
	mrows, err := s.DB.Query(mq, margs...)
	if err != nil {
		return
	}
	defer mrows.Close()
	for mrows.Next() {
		var cid int64
		var code string
		if err := mrows.Scan(&cid, &code); err == nil {
			if c, ok := byID[cid]; ok {
				c.MyReacts = append(c.MyReacts, code)
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
	err = tx.QueryRow(`SELECT 1 FROM reactions WHERE user_id = ? AND comment_id = ? AND code = ?`, userID, commentID, code).Scan(&existed)
	if err != nil && err != sql.ErrNoRows {
		return 0, false, err
	}
	delta := 1
	if err == nil {
		// already exists → remove
		if _, err := tx.Exec(`DELETE FROM reactions WHERE user_id = ? AND comment_id = ? AND code = ?`, userID, commentID, code); err != nil {
			return 0, false, err
		}
		delta = -1
		active = false
	} else {
		if _, err := tx.Exec(`INSERT INTO reactions (user_id, comment_id, code, created_at) VALUES (?, ?, ?, ?)`, userID, commentID, code, time.Now().Unix()); err != nil {
			return 0, false, err
		}
		active = true
	}

	// Upsert the denormalized count.
	if delta > 0 {
		if _, err := tx.Exec(
			`INSERT INTO reaction_counts (comment_id, code, count) VALUES (?, ?, 1)
			 ON CONFLICT(comment_id, code) DO UPDATE SET count = count + 1`,
			commentID, code,
		); err != nil {
			return 0, false, err
		}
	} else {
		if _, err := tx.Exec(`UPDATE reaction_counts SET count = MAX(0, count - 1) WHERE comment_id = ? AND code = ?`, commentID, code); err != nil {
			return 0, false, err
		}
		// Clean up zero rows so the count map stays compact.
		if _, err := tx.Exec(`DELETE FROM reaction_counts WHERE comment_id = ? AND code = ? AND count <= 0`, commentID, code); err != nil {
			return 0, false, err
		}
	}

	// Keep comments.score in sync as "total reactions" for the best-sort.
	if _, err := tx.Exec(`UPDATE comments SET score = score + ? WHERE id = ?`, delta, commentID); err != nil {
		return 0, false, err
	}

	if err := tx.QueryRow(`SELECT COALESCE(count, 0) FROM reaction_counts WHERE comment_id = ? AND code = ?`, commentID, code).Scan(&count); err != nil && err != sql.ErrNoRows {
		return 0, false, err
	}
	if err := tx.Commit(); err != nil {
		return 0, false, err
	}
	return count, active, nil
}

// ---------- Emoji pack ----------

func (s *Store) ListEmojis(siteID int64) ([]*models.Emoji, error) {
	rows, err := s.DB.Query(
		`SELECT id, site_id, code, label, svg, sort, created_at FROM emojis WHERE site_id = ? ORDER BY sort, id`,
		siteID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.Emoji
	for rows.Next() {
		e := &models.Emoji{}
		if err := rows.Scan(&e.ID, &e.SiteID, &e.Code, &e.Label, &e.SVG, &e.Sort, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) UpsertEmoji(siteID int64, code, label, svg string, sortOrder int) (*models.Emoji, error) {
	now := time.Now().Unix()
	_, err := s.DB.Exec(
		`INSERT INTO emojis (site_id, code, label, svg, sort, created_at) VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(site_id, code) DO UPDATE SET label = excluded.label, svg = excluded.svg, sort = excluded.sort`,
		siteID, code, label, svg, sortOrder, now,
	)
	if err != nil {
		return nil, err
	}
	e := &models.Emoji{}
	err = s.DB.QueryRow(
		`SELECT id, site_id, code, label, svg, sort, created_at FROM emojis WHERE site_id = ? AND code = ?`,
		siteID, code,
	).Scan(&e.ID, &e.SiteID, &e.Code, &e.Label, &e.SVG, &e.Sort, &e.CreatedAt)
	return e, err
}

func (s *Store) DeleteEmoji(siteID int64, code string) error {
	res, err := s.DB.Exec(`DELETE FROM emojis WHERE site_id = ? AND code = ?`, siteID, code)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// CodesForSite returns the set of valid emoji codes for a site. Used to gate
// reactions so users can't react with arbitrary strings.
func (s *Store) CodesForSite(siteID int64) (map[string]struct{}, error) {
	rows, err := s.DB.Query(`SELECT code FROM emojis WHERE site_id = ?`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]struct{})
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err == nil {
			out[code] = struct{}{}
		}
	}
	return out, rows.Err()
}
