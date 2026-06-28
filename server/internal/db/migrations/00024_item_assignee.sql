-- +goose Up
-- Assignees (X1): an artifact can be assigned to a user (a project member). On
-- the user's deletion the assignment clears rather than blocking the delete.
ALTER TABLE item ADD COLUMN assignee_id TEXT REFERENCES app_user(id) ON DELETE SET NULL;
CREATE INDEX idx_item_assignee ON item(assignee_id);

-- +goose Down
ALTER TABLE item DROP COLUMN IF EXISTS assignee_id;
