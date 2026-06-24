-- +goose Up
-- Share scopes: a named, reusable selection of what may be shared, plus a detail
-- level. The same scope powers the file export of a subset and the live "abo" feed.
CREATE TABLE share_scope (
    id           TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    detail_level TEXT NOT NULL DEFAULT 'timing', -- 'timing' (titles/dates only) | 'full'
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Whole-lane includes (dynamic: items added to the lane later are shared too).
CREATE TABLE share_scope_lane (
    scope_id    TEXT NOT NULL REFERENCES share_scope(id) ON DELETE CASCADE,
    swimlane_id TEXT NOT NULL REFERENCES swimlane(id) ON DELETE CASCADE,
    PRIMARY KEY (scope_id, swimlane_id)
);

-- Explicit single-item includes (e.g. one milestone from any lane).
CREATE TABLE share_scope_item (
    scope_id TEXT NOT NULL REFERENCES share_scope(id) ON DELETE CASCADE,
    item_id  TEXT NOT NULL REFERENCES item(id) ON DELETE CASCADE,
    PRIMARY KEY (scope_id, item_id)
);

-- Item excludes (e.g. share a whole lane except one milestone — the "9 of 10" case).
CREATE TABLE share_scope_exclude (
    scope_id TEXT NOT NULL REFERENCES share_scope(id) ON DELETE CASCADE,
    item_id  TEXT NOT NULL REFERENCES item(id) ON DELETE CASCADE,
    PRIMARY KEY (scope_id, item_id)
);

-- Subscribe tokens: a bearer secret (stored hashed) granting read-only access to
-- exactly one scope. Multiple tokens per scope allow per-recipient revocation.
CREATE TABLE share_token (
    id               TEXT PRIMARY KEY,
    scope_id         TEXT NOT NULL REFERENCES share_scope(id) ON DELETE CASCADE,
    token_hash       TEXT NOT NULL UNIQUE,
    label            TEXT NOT NULL DEFAULT '',
    expires_at       TIMESTAMPTZ,
    revoked          BOOLEAN NOT NULL DEFAULT false,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_accessed_at TIMESTAMPTZ
);
CREATE INDEX idx_share_token_scope ON share_token(scope_id);

-- +goose Down
DROP TABLE IF EXISTS share_token;
DROP TABLE IF EXISTS share_scope_exclude;
DROP TABLE IF EXISTS share_scope_item;
DROP TABLE IF EXISTS share_scope_lane;
DROP TABLE IF EXISTS share_scope;
