-- Covering indexes for metadata queries so SQLite can satisfy the common
-- listing lookups entirely from the index without rowid lookups.
CREATE INDEX IF NOT EXISTS idx_votes_user_comment_value ON votes(user_id, comment_id, value);
CREATE INDEX IF NOT EXISTS idx_reaction_counts_comment_code_count ON reaction_counts(comment_id, code, count);
