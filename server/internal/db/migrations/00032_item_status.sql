-- +goose Up
-- Per-item workflow status. The set of statuses (and allowed transitions) is
-- configured per item-type in the item_types JSON; this column holds the current
-- status key. Empty means "no status".
ALTER TABLE item ADD COLUMN status TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE item DROP COLUMN status;
