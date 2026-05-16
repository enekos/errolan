package store

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/enekos/errolan/internal/db"
)

func setupBenchDB(b *testing.B) (*sql.DB, int64, func()) {
	conn, err := db.Open(":memory:")
	if err != nil {
		b.Fatal(err)
	}
	if err := db.Migrate(conn); err != nil {
		b.Fatal(err)
	}

	// Insert a site
	res, err := conn.Exec(`INSERT INTO sites (slug, name, api_key, allowed_origins, require_auth, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"bench", "Bench Site", "key_bench", "*", 0, 1)
	if err != nil {
		b.Fatal(err)
	}
	siteID, _ := res.LastInsertId()

	// Insert a thread
	res, err = conn.Exec(`INSERT INTO threads (site_id, slug, title, url, locked, comment_count, last_comment_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		siteID, "thread", "Thread", "", 0, 0, 0, 1)
	if err != nil {
		b.Fatal(err)
	}
	threadID, _ := res.LastInsertId()

	cleanup := func() {
		conn.Close()
	}
	return conn, threadID, cleanup
}

func seedComments(b *testing.B, conn *sql.DB, threadID int64, numTopLevel, repliesPerTop int) {
	b.Helper()
	tx, err := conn.Begin()
	if err != nil {
		b.Fatal(err)
	}
	defer tx.Commit()
	now := int64(1000000)
	// Insert top-level comments individually to get parent IDs
	parentIDs := make([]int64, numTopLevel)
	for i := 0; i < numTopLevel; i++ {
		res, err := tx.Exec(
			`INSERT INTO comments (thread_id, parent_id, user_id, author_name, body, status, score, pinned, edit_count, anchor, created_at, updated_at)
			 VALUES (?, NULL, NULL, ?, ?, 'visible', ?, 0, 0, '', ?, ?)`,
			threadID, fmt.Sprintf("author_%d", i), fmt.Sprintf("body_%d", i), i%100, now+int64(i), now+int64(i))
		if err != nil {
			b.Fatal(err)
		}
		parentIDs[i], _ = res.LastInsertId()
	}
	// Batch insert replies
	const batchSize = 200
	values := make([]string, 0, batchSize)
	args := make([]any, 0, batchSize*12)
	for i := 0; i < numTopLevel; i++ {
		for j := 0; j < repliesPerTop; j++ {
			values = append(values, "(?,?,?,?,?,?,?,?,?,?,?,?)")
			args = append(args, threadID, parentIDs[i], nil, fmt.Sprintf("reply_author_%d_%d", i, j), fmt.Sprintf("reply_body_%d_%d", i, j), "visible", j%50, 0, 0, "", now+int64(i*1000+j), now+int64(i*1000+j))
			if len(values) >= batchSize {
				_, err := tx.Exec(`INSERT INTO comments (thread_id, parent_id, user_id, author_name, body, status, score, pinned, edit_count, anchor, created_at, updated_at) VALUES `+strings.Join(values, ","), args...)
				if err != nil {
					b.Fatal(err)
				}
				values = values[:0]
				args = args[:0]
			}
		}
	}
	if len(values) > 0 {
		_, err := tx.Exec(`INSERT INTO comments (thread_id, parent_id, user_id, author_name, body, status, score, pinned, edit_count, anchor, created_at, updated_at) VALUES `+strings.Join(values, ","), args...)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func seedVotes(b *testing.B, conn *sql.DB, threadID int64, numUsers int) {
	b.Helper()
	tx, err := conn.Begin()
	if err != nil {
		b.Fatal(err)
	}
	defer tx.Commit()
	// Insert users
	values := make([]string, 0, 200)
	args := make([]any, 0, 200*4)
	for i := 0; i < numUsers; i++ {
		values = append(values, "(?,?,'',0,0,?)")
		args = append(args, fmt.Sprintf("user%d@example.com", i), fmt.Sprintf("User%d", i), int64(i))
		if len(values) >= 200 {
			_, err := tx.Exec(`INSERT INTO users (email, name, password_hash, is_admin, is_banned, created_at) VALUES `+strings.Join(values, ","), args...)
			if err != nil {
				b.Fatal(err)
			}
			values = values[:0]
			args = args[:0]
		}
	}
	if len(values) > 0 {
		_, err := tx.Exec(`INSERT INTO users (email, name, password_hash, is_admin, is_banned, created_at) VALUES `+strings.Join(values, ","), args...)
		if err != nil {
			b.Fatal(err)
		}
	}
	// Vote on comments in the thread
	rows, err := tx.Query(`SELECT id FROM comments WHERE thread_id = ?`, threadID)
	if err != nil {
		b.Fatal(err)
	}
	var cids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			b.Fatal(err)
		}
		cids = append(cids, id)
	}
	rows.Close()
	// Batch insert votes
	const batchSize = 200
	voteValues := make([]string, 0, batchSize)
	voteArgs := make([]any, 0, batchSize*4)
	for ui := 0; ui < numUsers; ui++ {
		for _, cid := range cids {
			if (ui+int(cid))%3 == 0 {
				continue
			}
			val := 1
			if (ui+int(cid))%5 == 0 {
				val = -1
			}
			voteValues = append(voteValues, "(?,?,?,?)")
			voteArgs = append(voteArgs, ui+1, cid, val, int64(ui)+cid)
			if len(voteValues) >= batchSize {
				_, err := tx.Exec(`INSERT INTO votes (user_id, comment_id, value, created_at) VALUES `+strings.Join(voteValues, ","), voteArgs...)
				if err != nil {
					b.Fatal(err)
				}
				voteValues = voteValues[:0]
				voteArgs = voteArgs[:0]
			}
		}
	}
	if len(voteValues) > 0 {
		_, err := tx.Exec(`INSERT INTO votes (user_id, comment_id, value, created_at) VALUES `+strings.Join(voteValues, ","), voteArgs...)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func seedReactions(b *testing.B, conn *sql.DB, threadID int64, numUsers int) {
	b.Helper()
	tx, err := conn.Begin()
	if err != nil {
		b.Fatal(err)
	}
	defer tx.Commit()
	// Insert emojis
	codes := []string{"like", "heart", "fire", "rocket", "eyes"}
	for _, code := range codes {
		_, err := tx.Exec(`INSERT INTO emojis (site_id, code, label, svg, sort, created_at) VALUES (1, ?, ?, '', 0, 1)`, code, code)
		if err != nil {
			b.Fatal(err)
		}
	}
	// Insert reaction_counts and reactions
	rows, err := tx.Query(`SELECT id FROM comments WHERE thread_id = ?`, threadID)
	if err != nil {
		b.Fatal(err)
	}
	var cids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			b.Fatal(err)
		}
		cids = append(cids, id)
	}
	rows.Close()

	// Batch insert reactions
	const batchSize = 200
	reactionValues := make([]string, 0, batchSize)
	reactionArgs := make([]any, 0, batchSize*4)
	// Count reactions per (comment_id, code) in Go
	counts := make(map[int64]map[string]int, len(cids))
	for ui := 0; ui < numUsers; ui++ {
		for _, cid := range cids {
			code := codes[(ui+int(cid))%len(codes)]
			reactionValues = append(reactionValues, "(?,?,?,?)")
			reactionArgs = append(reactionArgs, ui+1, cid, code, int64(ui)+cid)
			if len(reactionValues) >= batchSize {
				_, err := tx.Exec(`INSERT INTO reactions (user_id, comment_id, code, created_at) VALUES `+strings.Join(reactionValues, ","), reactionArgs...)
				if err != nil {
					b.Fatal(err)
				}
				reactionValues = reactionValues[:0]
				reactionArgs = reactionArgs[:0]
			}
			if counts[cid] == nil {
				counts[cid] = make(map[string]int)
			}
			counts[cid][code]++
		}
	}
	if len(reactionValues) > 0 {
		_, err := tx.Exec(`INSERT INTO reactions (user_id, comment_id, code, created_at) VALUES `+strings.Join(reactionValues, ","), reactionArgs...)
		if err != nil {
			b.Fatal(err)
		}
	}

	// Batch insert final reaction_counts
	countValues := make([]string, 0, batchSize)
	countArgs := make([]any, 0, batchSize*3)
	for cid, codeMap := range counts {
		for code, cnt := range codeMap {
			countValues = append(countValues, "(?,?,?)")
			countArgs = append(countArgs, cid, code, cnt)
			if len(countValues) >= batchSize {
				_, err := tx.Exec(`INSERT INTO reaction_counts (comment_id, code, count) VALUES `+strings.Join(countValues, ","), countArgs...)
				if err != nil {
					b.Fatal(err)
				}
				countValues = countValues[:0]
				countArgs = countArgs[:0]
			}
		}
	}
	if len(countValues) > 0 {
		_, err := tx.Exec(`INSERT INTO reaction_counts (comment_id, code, count) VALUES `+strings.Join(countValues, ","), countArgs...)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkListThreadComments_SmallNoViewer — small thread, no viewer context.
func BenchmarkListThreadComments_SmallNoViewer(b *testing.B) {
	conn, threadID, cleanup := setupBenchDB(b)
	defer cleanup()
	seedComments(b, conn, threadID, 10, 5)
	s := New(conn)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := s.ListThreadComments(threadID, ListCommentsOpts{Sort: SortBest, Limit: 50})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkListThreadComments_SmallWithViewer — small thread, viewer with votes/reactions.
func BenchmarkListThreadComments_SmallWithViewer(b *testing.B) {
	conn, threadID, cleanup := setupBenchDB(b)
	defer cleanup()
	seedComments(b, conn, threadID, 10, 5)
	seedVotes(b, conn, threadID, 5)
	seedReactions(b, conn, threadID, 5)
	viewerID := int64(1)
	s := New(conn)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := s.ListThreadComments(threadID, ListCommentsOpts{Sort: SortBest, Limit: 50, ViewerID: &viewerID})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkListThreadComments_MediumNoViewer — medium thread (~100 top-level, 500 replies).
func BenchmarkListThreadComments_MediumNoViewer(b *testing.B) {
	conn, threadID, cleanup := setupBenchDB(b)
	defer cleanup()
	seedComments(b, conn, threadID, 100, 5)
	s := New(conn)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := s.ListThreadComments(threadID, ListCommentsOpts{Sort: SortBest, Limit: 50})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkListThreadComments_MediumWithViewer — medium thread with votes/reactions.
func BenchmarkListThreadComments_MediumWithViewer(b *testing.B) {
	conn, threadID, cleanup := setupBenchDB(b)
	defer cleanup()
	seedComments(b, conn, threadID, 100, 5)
	seedVotes(b, conn, threadID, 50)
	seedReactions(b, conn, threadID, 50)
	viewerID := int64(1)
	s := New(conn)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := s.ListThreadComments(threadID, ListCommentsOpts{Sort: SortBest, Limit: 50, ViewerID: &viewerID})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkListThreadComments_LargeNoViewer — large thread (500 top-level, 2500 replies).
func BenchmarkListThreadComments_LargeNoViewer(b *testing.B) {
	conn, threadID, cleanup := setupBenchDB(b)
	defer cleanup()
	seedComments(b, conn, threadID, 500, 5)
	s := New(conn)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := s.ListThreadComments(threadID, ListCommentsOpts{Sort: SortBest, Limit: 50})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkListThreadComments_LargeWithViewer — large thread with votes/reactions.
func BenchmarkListThreadComments_LargeWithViewer(b *testing.B) {
	conn, threadID, cleanup := setupBenchDB(b)
	defer cleanup()
	seedComments(b, conn, threadID, 500, 5)
	seedVotes(b, conn, threadID, 200)
	seedReactions(b, conn, threadID, 200)
	viewerID := int64(1)
	s := New(conn)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := s.ListThreadComments(threadID, ListCommentsOpts{Sort: SortBest, Limit: 50, ViewerID: &viewerID})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkListThreadComments_DeepReplies — many replies per top-level comment.
func BenchmarkListThreadComments_DeepReplies(b *testing.B) {
	conn, threadID, cleanup := setupBenchDB(b)
	defer cleanup()
	seedComments(b, conn, threadID, 50, 50)
	viewerID := int64(1)
	s := New(conn)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := s.ListThreadComments(threadID, ListCommentsOpts{Sort: SortBest, Limit: 50, ViewerID: &viewerID})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkListThreadComments_NewestSort — stress the newest sort path.
func BenchmarkListThreadComments_NewestSort(b *testing.B) {
	conn, threadID, cleanup := setupBenchDB(b)
	defer cleanup()
	seedComments(b, conn, threadID, 100, 5)
	seedVotes(b, conn, threadID, 50)
	viewerID := int64(1)
	s := New(conn)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := s.ListThreadComments(threadID, ListCommentsOpts{Sort: SortNewest, Limit: 50, ViewerID: &viewerID})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkListThreadComments_OldestSort — stress the oldest sort path.
func BenchmarkListThreadComments_OldestSort(b *testing.B) {
	conn, threadID, cleanup := setupBenchDB(b)
	defer cleanup()
	seedComments(b, conn, threadID, 100, 5)
	seedVotes(b, conn, threadID, 50)
	viewerID := int64(1)
	s := New(conn)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := s.ListThreadComments(threadID, ListCommentsOpts{Sort: SortOldest, Limit: 50, ViewerID: &viewerID})
		if err != nil {
			b.Fatal(err)
		}
	}
}
