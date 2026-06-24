-- +goose Up
-- A subscription mirrors a remote share scope (someone else's published schedule)
-- into this plan. The token is the bearer secret used to poll the remote feed.
CREATE TABLE subscription (
    id               TEXT PRIMARY KEY,
    source_label     TEXT NOT NULL,
    remote_url       TEXT NOT NULL,                 -- producer origin, e.g. https://atlas.example.com
    token            TEXT NOT NULL,
    etag             TEXT NOT NULL DEFAULT '',
    interval_seconds INT  NOT NULL DEFAULT 300,
    paused           BOOLEAN NOT NULL DEFAULT false,
    last_synced_at   TIMESTAMPTZ,
    last_status      TEXT NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Mirrored swimlanes carry provenance (source_system = subscription id) and a
-- consumer-local hide toggle. Native lanes leave source_system NULL.
ALTER TABLE swimlane ADD COLUMN source_system TEXT;
ALTER TABLE swimlane ADD COLUMN external_id   TEXT;
ALTER TABLE swimlane ADD COLUMN hidden        BOOLEAN NOT NULL DEFAULT false;
CREATE INDEX idx_swimlane_source ON swimlane(source_system);

-- +goose Down
DROP INDEX IF EXISTS idx_swimlane_source;
ALTER TABLE swimlane DROP COLUMN hidden;
ALTER TABLE swimlane DROP COLUMN external_id;
ALTER TABLE swimlane DROP COLUMN source_system;
DROP TABLE IF EXISTS subscription;
