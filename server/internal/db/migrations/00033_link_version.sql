-- +goose Up
-- Optional pinned version for a relationship (used by "Uses" links): NULL tracks
-- the target's latest revision; a number pins the reference to that version.
ALTER TABLE link ADD COLUMN version INTEGER;

-- +goose Down
ALTER TABLE link DROP COLUMN version;
