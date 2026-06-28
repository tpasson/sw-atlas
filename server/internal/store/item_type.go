package store

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
)

// ItemField declares one type-specific field. Values live in item.data keyed by
// Key; the field set is rendered dynamically by the client.
type ItemField struct {
	Key     string   `json:"key"`
	Label   string   `json:"label"`
	Type    string   `json:"type"`              // text | number | select | date
	Options []string `json:"options,omitempty"` // choices for type=select
}

// ItemType is a registry entry: a kind of artifact bound to a coded behavior
// family that decides how it renders, plus a field schema for its extra data.
// Built-ins are seeded and non-deletable; per-workspace custom types merge on top.
type ItemType struct {
	Key     string      `json:"key"`
	Label   string      `json:"label"`
	Family  string      `json:"family"` // see the Family* constants
	Icon    string      `json:"icon"`   // lucide marker id (e.g. "l:Diamond")
	Color   string      `json:"color"`  // "" = inherit the lane colour
	Fields  []ItemField `json:"fields"`
	Builtin bool        `json:"builtin"`
}

// Behavior families are CODED and finite — they decide rendering + field set.
// The set of types WITHIN a family is data (so new types need no code).
const (
	FamilyTimelinePoint = "timeline-point" // dated point on a lane
	FamilyTimelineRange = "timeline-range" // start→end bar on a lane
	FamilyWorkItem      = "work-item"      // backlog/board card (status, assignee, parent)
	FamilyContainer     = "container"      // grouping node
)

// validFamily reports whether f is one of the coded behavior families.
func validFamily(f string) bool {
	switch f {
	case FamilyTimelinePoint, FamilyTimelineRange, FamilyWorkItem, FamilyContainer:
		return true
	}
	return false
}

// DefaultItemTypes are the seeded built-ins. Their keys equal the legacy item
// kinds (milestone/event/point) so existing items map 1:1 — see migration 00019.
func DefaultItemTypes() []ItemType {
	return []ItemType{
		{Key: "milestone", Label: "Milestone", Family: FamilyTimelinePoint, Icon: "l:Diamond", Fields: []ItemField{}, Builtin: true},
		{Key: "event", Label: "Event", Family: FamilyTimelineRange, Icon: "l:Flag", Fields: []ItemField{}, Builtin: true},
		{Key: "point", Label: "Point", Family: FamilyTimelinePoint, Icon: "l:Circle", Fields: []ItemField{}, Builtin: true},
	}
}

func builtinTypeKeys() map[string]bool {
	m := map[string]bool{}
	for _, b := range DefaultItemTypes() {
		m[b.Key] = true
	}
	return m
}

// customItemTypes loads the workspace's user-defined types from app_setting.
func (s *Store) customItemTypes(ctx context.Context, ws string) ([]ItemType, error) {
	var raw []byte
	err := s.pool.QueryRow(ctx, `SELECT value FROM app_setting WHERE workspace_id = $1 AND key = 'item_types'`, ws).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var types []ItemType
	if err := json.Unmarshal(raw, &types); err != nil {
		return nil, nil // tolerate a malformed blob rather than break the plan read
	}
	return types, nil
}

// ListItemTypes returns the workspace's item-type catalog: built-ins first, then
// the user-defined custom types.
func (s *Store) ListItemTypes(ctx context.Context, ws string) ([]ItemType, error) {
	out := DefaultItemTypes()
	custom, err := s.customItemTypes(ctx, ws)
	if err != nil {
		return out, err
	}
	for _, t := range custom {
		t.Builtin = false
		out = append(out, t)
	}
	return out, nil
}

// SetItemTypes persists the custom (non-built-in) types for a workspace. Built-in
// keys and entries with an empty key or unknown family are dropped.
func (s *Store) SetItemTypes(ctx context.Context, ws string, types []ItemType) error {
	builtin := builtinTypeKeys()
	clean := []ItemType{}
	seen := map[string]bool{}
	for _, t := range types {
		if t.Key == "" || builtin[t.Key] || seen[t.Key] {
			continue
		}
		if !validFamily(t.Family) {
			t.Family = FamilyTimelinePoint
		}
		t.Builtin = false
		if t.Fields == nil {
			t.Fields = []ItemField{}
		}
		seen[t.Key] = true
		clean = append(clean, t)
	}
	b, _ := json.Marshal(clean)
	_, err := s.pool.Exec(ctx,
		`INSERT INTO app_setting (workspace_id, key, value) VALUES ($1, 'item_types', $2)
		 ON CONFLICT (workspace_id, key) DO UPDATE SET value = $2`, ws, b)
	return err
}
