package store

import (
	"database/sql"
	"testing"

	"github.com/enekos/errolan/internal/db"
)

// setupTestStore returns a fresh in-memory DB and a Store, plus a seeded
// site/thread/user/comment so callers can exercise the vote/reaction paths.
func setupTestStore(t *testing.T) (*Store, int64 /*commentID*/, int64 /*userID*/) {
	t.Helper()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	if err := db.Migrate(conn); err != nil {
		t.Fatal(err)
	}

	mustExec := func(q string, args ...any) sql.Result {
		t.Helper()
		res, err := conn.Exec(q, args...)
		if err != nil {
			t.Fatalf("exec %q: %v", q, err)
		}
		return res
	}

	mustExec(`INSERT INTO sites (slug, name, api_key, allowed_origins, require_auth, created_at) VALUES ('s', 'S', 'erl_x', '*', 0, 1)`)
	r := mustExec(`INSERT INTO threads (site_id, slug, title, url, locked, comment_count, last_comment_at, created_at) VALUES (1, 't', '', '', 0, 0, 0, 1)`)
	threadID, _ := r.LastInsertId()
	r = mustExec(`INSERT INTO users (email, name, password_hash, is_admin, is_banned, created_at) VALUES ('a@b', 'A', '', 0, 0, 1)`)
	userID, _ := r.LastInsertId()
	r = mustExec(`INSERT INTO comments (thread_id, parent_id, user_id, author_name, body, status, score, pinned, edit_count, anchor, moderation_reason, created_at, updated_at)
	              VALUES (?, NULL, ?, 'A', 'hi', 'visible', 0, 0, 0, '', '', 1, 1)`, threadID, userID)
	commentID, _ := r.LastInsertId()

	return New(conn), commentID, userID
}

// commentScore is a tiny helper so tests don't have to remember the column name.
func commentScore(t *testing.T, s *Store, id int64) int {
	t.Helper()
	var n int
	if err := s.DB.QueryRow(`SELECT score FROM comments WHERE id = ?`, id).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestVoteUpDownClear(t *testing.T) {
	s, cid, uid := setupTestStore(t)

	if got, err := s.Vote(uid, cid, 1); err != nil || got != 1 {
		t.Fatalf("upvote: score=%d err=%v", got, err)
	}
	if got, err := s.Vote(uid, cid, -1); err != nil || got != -1 {
		t.Fatalf("flip to downvote: score=%d err=%v", got, err)
	}
	if got, err := s.Vote(uid, cid, 0); err != nil || got != 0 {
		t.Fatalf("clear: score=%d err=%v", got, err)
	}
}

// TestVoteNoOpShortCircuit verifies that re-voting with the same value does not
// touch the votes row. We snapshot the votes table state and ensure it didn't
// move after the no-op call.
func TestVoteNoOpShortCircuit(t *testing.T) {
	s, cid, uid := setupTestStore(t)

	if _, err := s.Vote(uid, cid, 1); err != nil {
		t.Fatal(err)
	}
	var beforeCreated int64
	if err := s.DB.QueryRow(`SELECT created_at FROM votes WHERE user_id = ? AND comment_id = ?`, uid, cid).Scan(&beforeCreated); err != nil {
		t.Fatal(err)
	}
	beforeScore := commentScore(t, s, cid)

	got, err := s.Vote(uid, cid, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got != beforeScore {
		t.Fatalf("no-op vote changed score: before=%d after=%d", beforeScore, got)
	}

	var afterCreated int64
	if err := s.DB.QueryRow(`SELECT created_at FROM votes WHERE user_id = ? AND comment_id = ?`, uid, cid).Scan(&afterCreated); err != nil {
		t.Fatal(err)
	}
	if afterCreated != beforeCreated {
		t.Fatalf("no-op vote rewrote the row (created_at moved %d → %d)", beforeCreated, afterCreated)
	}
}
