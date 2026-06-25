-- +goose Up
-- A GitHub repository bound as a read-only source: its releases/tags/issues/PRs are
-- mirrored into a swimlane (source_system = github_source.id), reusing the same
-- provenance + locking machinery as federation subscriptions.
CREATE TABLE github_source (
    id               TEXT PRIMARY KEY,
    owner            TEXT NOT NULL,
    repo             TEXT NOT NULL,
    html_url         TEXT NOT NULL,
    token            TEXT NOT NULL DEFAULT '',      -- optional PAT (private repos / higher rate limit); never exposed
    include_releases BOOLEAN NOT NULL DEFAULT true,
    include_tags     BOOLEAN NOT NULL DEFAULT false,
    include_issues   BOOLEAN NOT NULL DEFAULT false,
    include_prs      BOOLEAN NOT NULL DEFAULT false,
    last_synced_at   TIMESTAMPTZ,
    last_status      TEXT NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE github_source;
