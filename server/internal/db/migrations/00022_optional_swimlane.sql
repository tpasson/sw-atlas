-- +goose Up
-- Off-timeline artifacts (V1): work-item / container types live in the Explorer
-- without a lane, so an item's swimlane becomes optional. Timeline types still
-- set it; the timeline simply doesn't render lane-less items.
ALTER TABLE item ALTER COLUMN swimlane_id DROP NOT NULL;

-- +goose Down
ALTER TABLE item ALTER COLUMN swimlane_id SET NOT NULL;
