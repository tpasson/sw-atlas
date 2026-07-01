-- +goose Up
-- Instance-wide settings (a global key/value store, distinct from the per-workspace
-- app_setting). Holds the global Display config and server settings set by a site admin.
CREATE TABLE instance_setting (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS instance_setting;
