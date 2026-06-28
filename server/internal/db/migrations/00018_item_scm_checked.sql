-- +goose Up
-- Throttle column for the live-SCM-status refresh of native milestones: records
-- when an item's scm_url was last polled so the background sweep can back off.
ALTER TABLE item ADD COLUMN scm_checked_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE item DROP COLUMN IF EXISTS scm_checked_at;
