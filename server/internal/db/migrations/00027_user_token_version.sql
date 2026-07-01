-- +goose Up
-- Session revocation: a per-user token version stamped into the session JWT and
-- checked on every authenticated request. Bumping it (on password change, role
-- change, or account deletion) instantly invalidates all existing tokens.
ALTER TABLE app_user ADD COLUMN token_version INT NOT NULL DEFAULT 1;

-- +goose Down
ALTER TABLE app_user DROP COLUMN IF EXISTS token_version;
