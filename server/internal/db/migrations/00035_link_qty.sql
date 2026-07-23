-- +goose Up
-- Quantity on a relationship edge (used by "uses" links): how many of the target
-- the source consumes — the BOM multiplicity ("this assembly uses 6× screw M4").
-- NULL means 1; the quantity lives on the edge, never as duplicate links (the
-- (a_item_id, b_item_id) pair stays unique).
ALTER TABLE link ADD COLUMN qty INTEGER;

-- +goose Down
ALTER TABLE link DROP COLUMN qty;
