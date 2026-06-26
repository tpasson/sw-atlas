-- +goose Up
-- Multi-user (Slice B): named accounts with a role. Each user owns one personal
-- workspace (workspace.owner_user_id), so logging in lands you on your own plan.
-- The env-configured editor is bootstrapped as the first admin against the
-- pre-existing 'default' workspace (see store.EnsureBootstrapAdmin), preserving
-- all existing data and behaviour.
CREATE TABLE app_user (
    id            TEXT PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'editor' CHECK (role IN ('admin', 'editor')),
    workspace_id  TEXT NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_app_user_workspace ON app_user(workspace_id);

-- +goose Down
DROP TABLE IF EXISTS app_user;
