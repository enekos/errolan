-- Support CountUserComments and other user-centric lookups.
CREATE INDEX IF NOT EXISTS idx_comments_user ON comments(user_id);

-- Speed up pending-comment listings (very selective filter).
CREATE INDEX IF NOT EXISTS idx_comments_status_created ON comments(status, created_at DESC);
