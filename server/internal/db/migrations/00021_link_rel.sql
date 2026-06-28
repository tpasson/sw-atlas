-- +goose Up
-- Typed relationships: a link carries a relationship kind. Existing links (and
-- any link inserted without an explicit rel — e.g. mirrored/baseline links) take
-- 'depends-on', preserving today's dependency semantics.
ALTER TABLE link ADD COLUMN rel TEXT NOT NULL DEFAULT 'depends-on';

-- +goose Down
ALTER TABLE link DROP COLUMN IF EXISTS rel;
