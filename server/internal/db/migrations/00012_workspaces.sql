-- +goose Up
-- Multi-tenant support (Slice A): every plan-scoped table gains a workspace_id.
-- A single "default" workspace holds all existing data. The DEFAULT 'default' on
-- each column is a permanent safety net for any INSERT that forgets to set it.
CREATE TABLE workspace (
    id            TEXT PRIMARY KEY,
    slug          TEXT NOT NULL UNIQUE,
    name          TEXT NOT NULL,
    owner_user_id TEXT,
    visibility    TEXT NOT NULL DEFAULT 'private',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Seed the default workspace BEFORE the FK ALTERs so the DEFAULT 'default' on the
-- new columns satisfies the foreign key for all existing rows.
INSERT INTO workspace (id, slug, name) VALUES ('default', 'default', 'Default') ON CONFLICT (id) DO NOTHING;

ALTER TABLE swimlane      ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'default' REFERENCES workspace(id) ON DELETE CASCADE;
ALTER TABLE item          ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'default' REFERENCES workspace(id) ON DELETE CASCADE;
ALTER TABLE sub_lane      ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'default' REFERENCES workspace(id) ON DELETE CASCADE;
ALTER TABLE link          ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'default' REFERENCES workspace(id) ON DELETE CASCADE;
ALTER TABLE baseline      ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'default' REFERENCES workspace(id) ON DELETE CASCADE;
ALTER TABLE share_scope   ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'default' REFERENCES workspace(id) ON DELETE CASCADE;
ALTER TABLE share_token   ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'default' REFERENCES workspace(id) ON DELETE CASCADE;
ALTER TABLE subscription  ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'default' REFERENCES workspace(id) ON DELETE CASCADE;
ALTER TABLE github_source ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'default' REFERENCES workspace(id) ON DELETE CASCADE;

CREATE INDEX idx_swimlane_ws      ON swimlane(workspace_id);
CREATE INDEX idx_item_ws          ON item(workspace_id);
CREATE INDEX idx_sub_lane_ws      ON sub_lane(workspace_id);
CREATE INDEX idx_baseline_ws      ON baseline(workspace_id);
CREATE INDEX idx_share_scope_ws   ON share_scope(workspace_id);
CREATE INDEX idx_subscription_ws  ON subscription(workspace_id);
CREATE INDEX idx_github_source_ws ON github_source(workspace_id);

-- app_setting is keyed by (key); re-key it by (workspace_id, key) so each
-- workspace has its own settings (public_read, palette, groups).
ALTER TABLE app_setting ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'default' REFERENCES workspace(id) ON DELETE CASCADE;
ALTER TABLE app_setting DROP CONSTRAINT app_setting_pkey;
ALTER TABLE app_setting ADD PRIMARY KEY (workspace_id, key);

-- +goose Down
ALTER TABLE app_setting DROP CONSTRAINT app_setting_pkey;
ALTER TABLE app_setting ADD PRIMARY KEY (key);
ALTER TABLE app_setting DROP COLUMN workspace_id;

DROP INDEX IF EXISTS idx_github_source_ws;
DROP INDEX IF EXISTS idx_subscription_ws;
DROP INDEX IF EXISTS idx_share_scope_ws;
DROP INDEX IF EXISTS idx_baseline_ws;
DROP INDEX IF EXISTS idx_sub_lane_ws;
DROP INDEX IF EXISTS idx_item_ws;
DROP INDEX IF EXISTS idx_swimlane_ws;

ALTER TABLE github_source DROP COLUMN workspace_id;
ALTER TABLE subscription  DROP COLUMN workspace_id;
ALTER TABLE share_token   DROP COLUMN workspace_id;
ALTER TABLE share_scope   DROP COLUMN workspace_id;
ALTER TABLE baseline      DROP COLUMN workspace_id;
ALTER TABLE link          DROP COLUMN workspace_id;
ALTER TABLE sub_lane      DROP COLUMN workspace_id;
ALTER TABLE item          DROP COLUMN workspace_id;
ALTER TABLE swimlane      DROP COLUMN workspace_id;

DROP TABLE IF EXISTS workspace;
