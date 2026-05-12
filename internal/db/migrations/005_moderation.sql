-- Per-site moderation policy. One row per site; defaults preserve the prior
-- "open" behaviour so existing sites are unaffected.
CREATE TABLE IF NOT EXISTS site_moderation (
    site_id                 INTEGER PRIMARY KEY REFERENCES sites(id) ON DELETE CASCADE,
    mode                    TEXT    NOT NULL DEFAULT 'open',     -- open | pre_moderation
    hold_new_users          INTEGER NOT NULL DEFAULT 0,          -- hold the first N comments from each registered user
    min_account_age_seconds INTEGER NOT NULL DEFAULT 0,
    min_body_length         INTEGER NOT NULL DEFAULT 0,
    max_links               INTEGER NOT NULL DEFAULT -1,         -- -1 = unlimited
    link_policy             TEXT    NOT NULL DEFAULT 'allow',    -- allow | hold | reject when any link present
    anonymous_link_policy   TEXT    NOT NULL DEFAULT 'allow',    -- as above, but only for anonymous posters
    auto_hide_flag_count    INTEGER NOT NULL DEFAULT 0,          -- 0 = off; otherwise auto-hide once N distinct flags land
    updated_at              INTEGER NOT NULL DEFAULT 0
);

-- Per-site keyword/regex blocklist. Decoupled from the settings row so the
-- list can grow without bloating each settings read.
CREATE TABLE IF NOT EXISTS moderation_blocklist (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id     INTEGER NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    kind        TEXT    NOT NULL,                                -- 'keyword' | 'regex'
    pattern     TEXT    NOT NULL,
    action      TEXT    NOT NULL DEFAULT 'hold',                 -- 'hold' | 'reject'
    created_at  INTEGER NOT NULL,
    UNIQUE(site_id, kind, pattern)
);
CREATE INDEX IF NOT EXISTS idx_blocklist_site ON moderation_blocklist(site_id);

-- Human-readable reason attached to held/hidden comments. Empty when nothing
-- in the policy fired.
ALTER TABLE comments ADD COLUMN moderation_reason TEXT NOT NULL DEFAULT '';

-- Queue scans filter by status; we also want a usable (status, thread_id)
-- ordering for the per-site listing.
CREATE INDEX IF NOT EXISTS idx_comments_status_thread ON comments(status, thread_id);
