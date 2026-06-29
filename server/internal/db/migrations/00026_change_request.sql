-- +goose Up
-- Change requests: a member proposes a change to the plan; the workspace owner
-- approves (applied to the live plan) or rejects it. The payload is the proposed
-- item shape (same JSON as an item); kind 'edit' targets an existing item, kind
-- 'create' proposes a new one.
CREATE TABLE change_request (
  id             TEXT        PRIMARY KEY,
  workspace_id   TEXT        NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
  author_id      TEXT        REFERENCES app_user(id) ON DELETE SET NULL,
  kind           TEXT        NOT NULL DEFAULT 'edit',     -- edit | create
  target_item_id TEXT,                                    -- the item to change (edit)
  payload        JSONB       NOT NULL,                    -- proposed item fields
  note           TEXT        NOT NULL DEFAULT '',         -- proposer's rationale
  status         TEXT        NOT NULL DEFAULT 'pending',  -- pending | approved | rejected
  decided_by     TEXT        REFERENCES app_user(id) ON DELETE SET NULL,
  decided_at     TIMESTAMPTZ,
  decision_note  TEXT        NOT NULL DEFAULT '',
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_change_request_ws ON change_request(workspace_id, status);

-- +goose Down
DROP TABLE IF EXISTS change_request;
