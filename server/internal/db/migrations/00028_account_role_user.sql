-- +goose Up
-- Rename the account (site) role 'editor' → 'user' so it no longer collides with
-- the per-workspace 'editor' role. Account roles are now: admin | user.
UPDATE app_user SET role = 'user' WHERE role = 'editor';

-- +goose Down
UPDATE app_user SET role = 'editor' WHERE role = 'user';
