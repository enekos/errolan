-- Paragraph anchor on a comment. Empty string == thread-level (Disqus-style).
-- A non-empty value names a paragraph on the host page (e.g. "p2") and turns
-- the comment into marginalia.
ALTER TABLE comments ADD COLUMN anchor TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_comments_thread_anchor ON comments(thread_id, anchor);

-- Emoji reactions replace the simple +1/-1 vote model. A user can apply
-- multiple distinct reaction codes to a single comment, but only one of each.
CREATE TABLE IF NOT EXISTS reactions (
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    comment_id INTEGER NOT NULL REFERENCES comments(id) ON DELETE CASCADE,
    code       TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    PRIMARY KEY (user_id, comment_id, code)
);
CREATE INDEX IF NOT EXISTS idx_reactions_comment ON reactions(comment_id);

-- Denormalized per-comment, per-code counts so listing comments doesn't have
-- to GROUP BY across the reactions table.
CREATE TABLE IF NOT EXISTS reaction_counts (
    comment_id INTEGER NOT NULL REFERENCES comments(id) ON DELETE CASCADE,
    code       TEXT NOT NULL,
    count      INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (comment_id, code)
);
CREATE INDEX IF NOT EXISTS idx_reaction_counts_comment ON reaction_counts(comment_id);

-- Per-site custom emoji pack. `svg` holds inline SVG markup (a few KB per
-- emoji) or a https URL — the SDK accepts either.
CREATE TABLE IF NOT EXISTS emojis (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id    INTEGER NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    code       TEXT NOT NULL,
    label      TEXT NOT NULL DEFAULT '',
    svg        TEXT NOT NULL,
    sort       INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,
    UNIQUE(site_id, code)
);
CREATE INDEX IF NOT EXISTS idx_emojis_site ON emojis(site_id, sort);
