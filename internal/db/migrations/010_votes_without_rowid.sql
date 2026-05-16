-- Convert votes to WITHOUT ROWID so the PRIMARY KEY (user_id, comment_id)
-- becomes the actual storage key, eliminating the separate rowid → table
-- indirection on every vote lookup. The PK B-tree now stores value and
-- created_at inline, so the LEFT JOIN in comment queries needs only one
-- index traversal instead of two (index + table lookup).
CREATE TABLE votes_new (
    user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    comment_id  INTEGER NOT NULL REFERENCES comments(id) ON DELETE CASCADE,
    value       INTEGER NOT NULL,
    created_at  INTEGER NOT NULL,
    PRIMARY KEY (user_id, comment_id)
) WITHOUT ROWID;

INSERT INTO votes_new SELECT * FROM votes;
DROP TABLE votes;
ALTER TABLE votes_new RENAME TO votes;

-- Recreate the covering index for direct value lookups.
CREATE INDEX IF NOT EXISTS idx_votes_user_comment_value ON votes(user_id, comment_id, value);
