-- +goose Up
-- Background auto-refresh for GitHub/Gitea sources: a poll interval and a small
-- map of per-resource ETags so the poller can short-circuit on 304 (unchanged).
ALTER TABLE github_source
    ADD COLUMN interval_seconds INT NOT NULL DEFAULT 900,
    ADD COLUMN etags            JSONB NOT NULL DEFAULT '{}'::jsonb;

-- +goose Down
ALTER TABLE github_source
    DROP COLUMN IF EXISTS interval_seconds,
    DROP COLUMN IF EXISTS etags;
