-- Comment moderation/quality fields
ALTER TABLE comments ADD COLUMN pinned INTEGER NOT NULL DEFAULT 0;
ALTER TABLE comments ADD COLUMN edit_count INTEGER NOT NULL DEFAULT 0;

-- Thread denormalized counters (avoid COUNT(*) on every load)
ALTER TABLE threads ADD COLUMN comment_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE threads ADD COLUMN last_comment_at INTEGER NOT NULL DEFAULT 0;

-- Backfill the counters for any pre-existing data
UPDATE threads SET
  comment_count   = (SELECT COUNT(*) FROM comments WHERE thread_id = threads.id AND status != 'deleted'),
  last_comment_at = COALESCE((SELECT MAX(created_at) FROM comments WHERE thread_id = threads.id), 0);

-- Composite indexes for common ordered scans
CREATE INDEX IF NOT EXISTS idx_comments_thread_created ON comments(thread_id, created_at);
CREATE INDEX IF NOT EXISTS idx_comments_thread_score   ON comments(thread_id, score DESC);

-- One flag per (authenticated user, comment); anonymous flags rely on
-- the IP-based rate limiter rather than a unique constraint.
CREATE UNIQUE INDEX IF NOT EXISTS idx_flags_unique_user
    ON flags(comment_id, user_id) WHERE user_id IS NOT NULL;

-- Audit log for moderation actions
CREATE TABLE IF NOT EXISTS audit_log (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    actor_id    INTEGER REFERENCES users(id) ON DELETE SET NULL,
    actor_name  TEXT NOT NULL DEFAULT '',
    action      TEXT NOT NULL,
    target_kind TEXT NOT NULL,
    target_id   INTEGER NOT NULL,
    metadata    TEXT NOT NULL DEFAULT '',
    created_at  INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_audit_created ON audit_log(created_at);
