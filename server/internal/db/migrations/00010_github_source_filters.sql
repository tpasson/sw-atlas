-- +goose Up
-- Per-source import filters to keep dense repos manageable.
ALTER TABLE github_source ADD COLUMN stable_only  BOOLEAN NOT NULL DEFAULT false; -- releases: skip pre-releases
ALTER TABLE github_source ADD COLUMN state_filter TEXT    NOT NULL DEFAULT 'all';  -- issues/PRs: all | open | closed
ALTER TABLE github_source ADD COLUMN since_date   TEXT    NOT NULL DEFAULT '';      -- only items dated on/after (YYYY-MM-DD)
ALTER TABLE github_source ADD COLUMN max_per_type INT     NOT NULL DEFAULT 0;       -- keep only the N most recent per type (0 = all)

-- +goose Down
ALTER TABLE github_source DROP COLUMN stable_only;
ALTER TABLE github_source DROP COLUMN state_filter;
ALTER TABLE github_source DROP COLUMN since_date;
ALTER TABLE github_source DROP COLUMN max_per_type;
