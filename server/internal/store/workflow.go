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
	Builtin  bool            `json:"builtin,omitempty"` // seeded default, always present
}

// DefaultWorkflows are the shipped defaults, always present (like built-in item
// types). "standard" carries a hand-arranged status-flow layout so the diagram
// looks right out of the box; a workspace can override its statuses/layout.
func DefaultWorkflows() []Workflow {
	standardLayout := json.RawMessage(`{"nodes":{"todo":{"x":49,"y":28},"in-progress":{"x":270,"y":27},"blocked":{"x":270,"y":127},"done":{"x":270,"y":-76},"cancelled":{"x":653,"y":31}},"edges":{"todo|cancelled":{"x":352,"y":-147,"a":"T","b":"T"},"in-progress|done":{"a":"T","b":"B"},"in-progress|cancelled":{"b":"L"},"blocked|cancelled":{"x":526,"y":128,"a":"R","b":"B"}}}`)
	return []Workflow{{
		Key: "standard", Label: "Standard", Builtin: true,
		Statuses: []ItemStatus{
			{Key: "todo", Label: "To Do", Tone: "neutral", To: []string{"in-progress", "cancelled"}},
			{Key: "in-progress", Label: "In Progress", Tone: "progress", To: []string{"blocked", "done", "cancelled"}},
			{Key: "blocked", Label: "Blocked", Tone: "warning", To: []string{"in-progress", "cancelled"}},
			{Key: "done", Label: "Done", Tone: "positive", To: []string{"in-progress"}},
			{Key: "cancelled", Label: "Cancelled", Tone: "negative", To: []string{"todo"}},
		},
		Layout: standardLayout,
	}}
}

// mergeWorkflows returns the built-in defaults (overridden by a stored workflow
// of the same key), then any stored custom workflows.
func mergeWorkflows(stored []Workflow) []Workflow {
	byKey := map[string]Workflow{}
	for _, w := range stored {
		byKey[w.Key] = w
	}
	out := []Workflow{}
	isDefault := map[string]bool{}
	for _, d := range DefaultWorkflows() {
		isDefault[d.Key] = true
		if ov, ok := byKey[d.Key]; ok {
			ov.Builtin = true
			out = append(out, ov)
		} else {
			out = append(out, d)
		}
	}
	for _, w := range stored {
		if isDefault[w.Key] {
			continue
		}
		w.Builtin = false
		out = append(out, w)
	}
	return out
}

// GetWorkflows returns the workspace's shared workflows: built-in defaults merged
// with any stored overrides + custom workflows.
func (s *Store) GetWorkflows(ctx context.Context, ws string) ([]Workflow, error) {
	var stored []Workflow
	var v string
	err := s.pool.QueryRow(ctx, `SELECT value FROM app_setting WHERE key = 'workflows' AND workspace_id = $1`, ws).Scan(&v)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if err == nil {
		_ = json.Unmarshal([]byte(v), &stored) // tolerate a malformed blob → treated as none
	}
	return mergeWorkflows(stored), nil
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
