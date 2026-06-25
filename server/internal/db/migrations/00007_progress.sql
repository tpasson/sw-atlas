-- +goose Up
-- Per-item progress 0..100 (% complete), independent of the maturity stage. Null = unset.
ALTER TABLE item ADD COLUMN progress INT;

-- +goose Down
ALTER TABLE item DROP COLUMN progress;
