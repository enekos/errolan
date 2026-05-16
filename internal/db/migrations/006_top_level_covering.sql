-- Covering index for top-level comment listing queries.
-- Supports the common paths: best (pinned+score), oldest (pinned+created_asc),
-- and newest (pinned+created_desc) by allowing SQLite to seek directly to
-- the relevant thread + NULL parent and read rows in the correct order.
CREATE INDEX IF NOT EXISTS idx_comments_thread_parent_pinned_score ON comments(thread_id, parent_id, pinned DESC, score DESC, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_comments_thread_parent_pinned_created ON comments(thread_id, parent_id, pinned DESC, created_at DESC);
