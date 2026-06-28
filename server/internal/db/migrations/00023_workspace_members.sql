-- +goose Up
-- Multi-member workspaces (P1): a user↔workspace membership with a per-workspace
-- role. This becomes the authorization source (replacing "you can only touch your
-- own workspace"). Backfill makes every existing user the OWNER of their current
-- home workspace, so behavior is unchanged for today's single-owner workspaces.
CREATE TABLE workspace_member (
    workspace_id TEXT NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    user_id      TEXT NOT NULL REFERENCES app_user(id)  ON DELETE CASCADE,
    role         TEXT NOT NULL DEFAULT 'viewer'
                   CHECK (role IN ('owner', 'editor', 'viewer')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (workspace_id, user_id)
);
CREATE INDEX idx_workspace_member_user ON workspace_member(user_id);

INSERT INTO workspace_member (workspace_id, user_id, role)
SELECT workspace_id, id, 'owner' FROM app_user
ON CONFLICT DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS workspace_member;
