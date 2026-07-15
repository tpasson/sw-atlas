package store

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
)

// Workflow is a named, reusable status set (statuses + transitions) plus an
// optional saved status-flow layout. An ItemType can reference one by key
// instead of embedding its own statuses, so a workflow shared by many types is
// edited in exactly one place. Stored as JSON in app_setting (key 'workflows').
type Workflow struct {
	Key      string          `json:"key"`
	Label    string          `json:"label"`
	Statuses []ItemStatus    `json:"statuses"`
	Layout   json.RawMessage `json:"layout,omitempty"`
}

// GetWorkflows returns the workspace's shared workflows.
func (s *Store) GetWorkflows(ctx context.Context, ws string) ([]Workflow, error) {
	var v string
	err := s.pool.QueryRow(ctx, `SELECT value FROM app_setting WHERE key = 'workflows' AND workspace_id = $1`, ws).Scan(&v)
	if errors.Is(err, pgx.ErrNoRows) {
		return []Workflow{}, nil
	}
	if err != nil {
		return nil, err
	}
	var wfs []Workflow
	if err := json.Unmarshal([]byte(v), &wfs); err != nil {
		return []Workflow{}, nil // tolerate a malformed blob rather than break reads
	}
	return wfs, nil
}

// SetWorkflows replaces the workspace's shared workflows (keyed, deduped).
func (s *Store) SetWorkflows(ctx context.Context, ws string, wfs []Workflow) error {
	clean := []Workflow{}
	seen := map[string]bool{}
	for _, w := range wfs {
		if w.Key == "" || seen[w.Key] {
			continue
		}
		seen[w.Key] = true
		if w.Statuses == nil {
			w.Statuses = []ItemStatus{}
		}
		clean = append(clean, w)
	}
	b, err := json.Marshal(clean)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO app_setting (workspace_id, key, value) VALUES ($1, 'workflows', $2)
		 ON CONFLICT (workspace_id, key) DO UPDATE SET value = EXCLUDED.value`, ws, string(b))
	return err
}
