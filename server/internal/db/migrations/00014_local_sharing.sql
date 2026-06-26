-- +goose Up
-- Slice D: intra-server publish & subscribe. A share scope can be "published"
-- (opt-in, server-wide discoverable by other users); a subscription can point at
-- a local workspace's published scope instead of an external URL+token. Local
-- subscriptions store empty remote_url/token and are recognised by a non-null
-- source_workspace_id.
ALTER TABLE share_scope  ADD COLUMN published BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE subscription ADD COLUMN source_workspace_id TEXT REFERENCES workspace(id) ON DELETE CASCADE;
ALTER TABLE subscription ADD COLUMN source_scope_id     TEXT;
CREATE INDEX idx_share_scope_published ON share_scope(published) WHERE published;

-- +goose Down
DROP INDEX IF EXISTS idx_share_scope_published;
ALTER TABLE subscription DROP COLUMN source_scope_id;
ALTER TABLE subscription DROP COLUMN source_workspace_id;
ALTER TABLE share_scope  DROP COLUMN published;
