-- +goose Up
CREATE TABLE baseline (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    note       TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Snapshot of each item at capture time. item_id is intentionally NOT a foreign
-- key: a baseline must keep items that were later deleted from the live plan.
CREATE TABLE baseline_item (
    baseline_id TEXT NOT NULL REFERENCES baseline(id) ON DELETE CASCADE,
    item_id     TEXT NOT NULL,
    swimlane_id TEXT NOT NULL,
    sub_lane_id TEXT,
    title       TEXT NOT NULL,
    year        INT  NOT NULL,
    month       INT  NOT NULL,
    when_date   DATE,
    start_date  DATE,
    end_date    DATE,
    kind        TEXT NOT NULL DEFAULT 'milestone',
    marker      TEXT NOT NULL DEFAULT 'diamond',
    PRIMARY KEY (baseline_id, item_id)
);
CREATE INDEX idx_baseline_item_baseline ON baseline_item(baseline_id);

-- +goose Down
DROP TABLE IF EXISTS baseline_item;
DROP TABLE IF EXISTS baseline;
