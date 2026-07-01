-- +goose Up
-- Optional user profile: email + real name, shown when you click a user. Email is
-- only exposed to authenticated requesters (never on the public explore page).
ALTER TABLE app_user ADD COLUMN email TEXT NOT NULL DEFAULT '';
ALTER TABLE app_user ADD COLUMN first_name TEXT NOT NULL DEFAULT '';
ALTER TABLE app_user ADD COLUMN last_name TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE app_user DROP COLUMN IF EXISTS last_name;
ALTER TABLE app_user DROP COLUMN IF EXISTS first_name;
ALTER TABLE app_user DROP COLUMN IF EXISTS email;
