-- Federated / inbound mentions: a single table for Webmentions (W3C) and
-- minimal ActivityPub `Create(Note)` replies. Both behave the same from the
-- thread's point of view — somebody on another site pointed at us with a
-- snippet of text and we render it next to the native comments.
--
-- status: pending until the verifier confirms the source links to the target
--         (webmention) or signature/inReplyTo validates (activitypub).
-- kind:   determines verifier behaviour and how the source is displayed.
CREATE TABLE IF NOT EXISTS mentions (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id     INTEGER NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    thread_id   INTEGER NOT NULL REFERENCES threads(id) ON DELETE CASCADE,
    source      TEXT    NOT NULL,                            -- URL or AP actor URI
    target      TEXT    NOT NULL,                            -- our URL that was referenced
    author_name TEXT    NOT NULL DEFAULT '',
    author_url  TEXT    NOT NULL DEFAULT '',
    snippet     TEXT    NOT NULL DEFAULT '',
    kind        TEXT    NOT NULL DEFAULT 'webmention',       -- 'webmention' | 'activitypub'
    status      TEXT    NOT NULL DEFAULT 'pending',          -- 'pending' | 'verified' | 'rejected'
    reason      TEXT    NOT NULL DEFAULT '',                 -- failure cause, when rejected
    created_at  INTEGER NOT NULL,
    verified_at INTEGER NOT NULL DEFAULT 0,
    UNIQUE(thread_id, source)
);
CREATE INDEX IF NOT EXISTS idx_mentions_thread_status ON mentions(thread_id, status);
CREATE INDEX IF NOT EXISTS idx_mentions_site         ON mentions(site_id);
