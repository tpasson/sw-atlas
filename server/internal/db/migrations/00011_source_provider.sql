-- +goose Up
-- Generalise the GitHub source to other providers (self-hosted Gitea/Forgejo, …).
ALTER TABLE github_source ADD COLUMN provider TEXT NOT NULL DEFAULT 'github'; -- github | gitea
ALTER TABLE github_source ADD COLUMN api_base TEXT NOT NULL DEFAULT '';       -- explicit REST base for self-hosted (e.g. https://host:port/api/v1)

-- +goose Down
ALTER TABLE github_source DROP COLUMN provider;
ALTER TABLE github_source DROP COLUMN api_base;
