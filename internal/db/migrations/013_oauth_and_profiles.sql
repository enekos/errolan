-- OAuth identities: one row per (provider, subject) pointing at a local user.
-- Subject is the provider-side stable user id (NEVER email — users can change
-- email but the subject is permanent), kept opaque so adding new providers
-- doesn't require schema changes.
CREATE TABLE IF NOT EXISTS oauth_identities (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider    TEXT    NOT NULL,                 -- short name: 'github', 'gitlab', ...
    subject     TEXT    NOT NULL,                 -- provider-side stable user id
    email       TEXT    NOT NULL DEFAULT '',      -- snapshot at link time (informational)
    avatar_url  TEXT    NOT NULL DEFAULT '',      -- snapshot at link time
    created_at  INTEGER NOT NULL,
    UNIQUE(provider, subject)
);
CREATE INDEX IF NOT EXISTS idx_oauth_identities_user ON oauth_identities(user_id);

-- Profile metadata for the public reader profile page. Optional — a row exists
-- iff the user has edited their profile or signed in via OAuth (which pre-fills
-- the avatar URL from the provider).
CREATE TABLE IF NOT EXISTS user_profiles (
    user_id     INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    bio         TEXT    NOT NULL DEFAULT '',
    website     TEXT    NOT NULL DEFAULT '',
    avatar_url  TEXT    NOT NULL DEFAULT '',      -- non-Gravatar override (e.g. GitHub avatar)
    updated_at  INTEGER NOT NULL DEFAULT 0
);
