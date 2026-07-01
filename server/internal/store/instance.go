package store

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
)

// GetInstanceSetting returns a global (instance-wide) setting as an opaque JSON
// blob, or nil if unset. Distinct from the per-workspace app_setting.
func (s *Store) GetInstanceSetting(ctx context.Context, key string) (json.RawMessage, error) {
	var v string
	err := s.pool.QueryRow(ctx, `SELECT value FROM instance_setting WHERE key = $1`, key).Scan(&v)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return json.RawMessage(v), nil
}

// SetInstanceSetting stores a global setting (an opaque JSON value).
func (s *Store) SetInstanceSetting(ctx context.Context, key string, raw json.RawMessage) error {
	if len(raw) == 0 {
		raw = json.RawMessage("{}")
	}
	if !json.Valid(raw) {
		return errors.New("value must be valid JSON")
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO instance_setting (key, value) VALUES ($1, $2)
		 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, key, string(raw))
	return err
}

// InstanceStats are read-only counts for the admin server panel.
type InstanceStats struct {
	Users      int `json:"users"`
	Workspaces int `json:"workspaces"`
	Items      int `json:"items"`
}

// GetInstanceStats returns instance-wide counts (users, workspaces, items).
func (s *Store) GetInstanceStats(ctx context.Context) (InstanceStats, error) {
	var st InstanceStats
	err := s.pool.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM app_user),
		       (SELECT count(*) FROM workspace),
		       (SELECT count(*) FROM item)`).Scan(&st.Users, &st.Workspaces, &st.Items)
	return st, err
}
