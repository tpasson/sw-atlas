-- +goose Up
-- Item-type registry foundation: every item references a type by key. Built-in
-- keys equal the legacy `kind` values (milestone/event/point), so existing items
-- map 1:1 and nothing changes for users. Custom types arrive later (T3).
ALTER TABLE item ADD COLUMN type_key TEXT NOT NULL DEFAULT 'milestone';
UPDATE item SET type_key = kind;

-- +goose Down
ALTER TABLE item DROP COLUMN IF EXISTS type_key;
