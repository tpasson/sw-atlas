-- +goose Up
-- Per-item link to a source-control resource (GitHub/GitLab release, PR, branch,
-- commit, issue, …). Free-form URL; the client parses it for a pretty badge. Null = unset.
ALTER TABLE item ADD COLUMN scm_url TEXT;

-- +goose Down
ALTER TABLE item DROP COLUMN scm_url;
