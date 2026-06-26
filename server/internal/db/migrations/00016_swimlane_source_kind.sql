-- +goose Up
-- Mirrored swimlanes record what KIND of source they came from (github, gitea,
-- subscription, …) so the UI can label them precisely instead of a generic
-- "synced" badge.
ALTER TABLE swimlane ADD COLUMN source_kind TEXT;

-- +goose Down
ALTER TABLE swimlane DROP COLUMN source_kind;
