-- +goose Up
-- 00028 renamed the account role 'editor' -> 'user' but left the original CHECK
-- from 00013 (role IN ('admin','editor')), so inserting a new 'user' account fails
-- with a constraint violation (surfaced as a 500 on "Add user"). Update the check.
ALTER TABLE app_user DROP CONSTRAINT IF EXISTS app_user_role_check;
ALTER TABLE app_user ADD CONSTRAINT app_user_role_check CHECK (role IN ('admin', 'user'));

-- +goose Down
ALTER TABLE app_user DROP CONSTRAINT IF EXISTS app_user_role_check;
