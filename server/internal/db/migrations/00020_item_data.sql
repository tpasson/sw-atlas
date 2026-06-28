-- +goose Up
-- Per-type custom fields: an item carries a JSONB bag of values whose shape is
-- declared by its type's field schema (validated client-side; stored opaque).
ALTER TABLE item ADD COLUMN data JSONB NOT NULL DEFAULT '{}'::jsonb;

-- +goose Down
ALTER TABLE item DROP COLUMN IF EXISTS data;
