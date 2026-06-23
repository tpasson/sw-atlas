-- +goose Up
CREATE TABLE swimlane (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    color      TEXT NOT NULL DEFAULT '#0A84FF',
    sort_order INT  NOT NULL DEFAULT 0
);

CREATE TABLE sub_lane (
    id          TEXT PRIMARY KEY,
    swimlane_id TEXT NOT NULL REFERENCES swimlane(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    sort_order  INT  NOT NULL DEFAULT 0
);
CREATE INDEX idx_sub_lane_swimlane ON sub_lane(swimlane_id);

CREATE TABLE item (
    id             TEXT PRIMARY KEY,
    swimlane_id    TEXT NOT NULL REFERENCES swimlane(id) ON DELETE CASCADE,
    sub_lane_id    TEXT REFERENCES sub_lane(id) ON DELETE CASCADE,
    year           INT  NOT NULL,
    month          INT  NOT NULL,
    title          TEXT NOT NULL,
    what           TEXT NOT NULL DEFAULT '',
    why            TEXT NOT NULL DEFAULT '',
    how            TEXT NOT NULL DEFAULT '',
    who            TEXT NOT NULL DEFAULT '',
    when_date      DATE,
    kind           TEXT NOT NULL DEFAULT 'milestone',
    marker         TEXT NOT NULL DEFAULT 'diamond',
    start_date     DATE,
    end_date       DATE,
    color          TEXT,
    -- provenance: when set, the item is mirrored from an external source and is
    -- read-only in ATLAS (the source system is always master).
    source_system  TEXT,
    external_id    TEXT,
    external_url   TEXT,
    last_synced_at TIMESTAMPTZ
);
CREATE INDEX idx_item_swimlane ON item(swimlane_id);
CREATE INDEX idx_item_year ON item(year);

CREATE TABLE link (
    a_item_id TEXT NOT NULL REFERENCES item(id) ON DELETE CASCADE,
    b_item_id TEXT NOT NULL REFERENCES item(id) ON DELETE CASCADE,
    PRIMARY KEY (a_item_id, b_item_id)
);

CREATE TABLE app_setting (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
INSERT INTO app_setting (key, value) VALUES ('public_read_enabled', 'true');

-- +goose Down
DROP TABLE IF EXISTS app_setting;
DROP TABLE IF EXISTS link;
DROP TABLE IF EXISTS item;
DROP TABLE IF EXISTS sub_lane;
DROP TABLE IF EXISTS swimlane;
