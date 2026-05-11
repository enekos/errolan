-- Optimise reply fetching when listing thread comments.
-- The reply query filters by parent_id IN (...) and orders by created_at ASC.
CREATE INDEX IF NOT EXISTS idx_comments_parent_created ON comments(parent_id, created_at);
