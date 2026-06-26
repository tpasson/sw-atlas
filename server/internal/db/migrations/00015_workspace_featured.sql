-- +goose Up
-- Discovery landing page (#15): the admin can mark public workspaces as
-- "featured" so they're pinned on the explore page.
ALTER TABLE workspace ADD COLUMN featured BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE workspace DROP COLUMN featured;
