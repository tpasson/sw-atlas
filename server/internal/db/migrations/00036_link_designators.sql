-- +goose Up
-- Reference designators on a "uses" edge (electronics BOM): free text naming the
-- individual usage positions, e.g. "C1, C2, C10-C17". Like qty, this describes
-- the USAGE (edge), not the part — the part stays a single item. NULL = none.
ALTER TABLE link ADD COLUMN designators TEXT;

-- +goose Down
ALTER TABLE link DROP COLUMN designators;
