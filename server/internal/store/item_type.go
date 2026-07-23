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
	Key      string   `json:"key"`
	Label    string   `json:"label"`
	Type     string   `json:"type"`              // text | number | select | multiselect | date | reference
	Options  []string `json:"options,omitempty"` // choices for select / multiselect
	Required bool     `json:"required,omitempty"`
	RefType  string   `json:"refType,omitempty"`  // type=reference: target item-type key
	RefMulti bool     `json:"refMulti,omitempty"` // type=reference: allow multiple references
}

// ItemType is a registry entry: a kind of artifact bound to a coded behavior
// family that decides how it renders, plus a field schema for its extra data.
// Built-ins are seeded and non-deletable; per-workspace custom types merge on top.
type ItemType struct {
	Key         string          `json:"key"`
	Label       string          `json:"label"`
	Family      string          `json:"family"`         // see the Family* constants
	Icon        string          `json:"icon"`           // lucide marker id (e.g. "l:Diamond")
	Color       string          `json:"color"`          // "" = inherit the lane colour
	Fill        *bool           `json:"fill,omitempty"` // nil/true = filled icon, false = outline
	Fields      []ItemField     `json:"fields"`
	WorkflowKey string          `json:"workflowKey,omitempty"` // reference a shared Workflow; when set, its statuses+layout are used
	Statuses    []ItemStatus    `json:"statuses,omitempty"`    // configurable workflow states + transitions (inline, when no workflowKey)
	Layout      json.RawMessage `json:"layout,omitempty"`      // optional saved status-flow layout: { nodes:{...}, edges:{...} }
	Builtin     bool            `json:"builtin"`
}

// ItemStatus is one workflow state a type can be in. The first status in a type's
// list is the initial one for new items; To lists the status keys it may move to
// (empty = any). Tone maps to a fixed semantic colour in the UI (so "approved"
// can't be red): neutral | info | progress | positive | warning | negative.
type ItemStatus struct {
	Key   string   `json:"key"`
	Label string   `json:"label"`
	Tone  string   `json:"tone"`
	Color string   `json:"color,omitempty"` // optional hex override of the tone's colour
	To    []string `json:"to,omitempty"`
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

// DescriptionFields are the standard prose fields every type ships with by
// default (the former What/Why/Where "W-questions"). They are ordinary type
// fields now — editable and removable per type like any other field.
func DescriptionFields() []ItemField {
	return []ItemField{
		{Key: "what", Label: "What", Type: "textarea"},
		{Key: "why", Label: "Why", Type: "textarea"},
		{Key: "how", Label: "Where", Type: "textarea"},
	}
}

// DefaultItemTypes are the seeded built-ins. Their keys equal the legacy item
// kinds (milestone/event/point) so existing items map 1:1 — see migration 00019.
func DefaultItemTypes() []ItemType {
	return []ItemType{
		{Key: "milestone", Label: "Milestones", Family: FamilyTimelinePoint, Icon: "l:Diamond", Fields: DescriptionFields(), WorkflowKey: "standard", Builtin: true},
		{Key: "event", Label: "Events", Family: FamilyTimelineRange, Icon: "l:Flag", Fields: DescriptionFields(), WorkflowKey: "standard", Builtin: true},
	}
}

// legacyBuiltinLabel are the singular default names built-ins shipped with before
// they were pluralised. A stored override still equal to one of these is a phantom
// from older saves (which persisted the resolved label even when unchanged) and is
// ignored on read, so the current default wins. A genuinely different name is kept.
var legacyBuiltinLabel = map[string]string{"milestone": "Milestone", "event": "Event"}

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

// ListItemTypes returns the workspace's item-type catalog: built-ins first (with
// any icon/colour/label/fill overrides applied), then the user-defined types.
func (s *Store) ListItemTypes(ctx context.Context, ws string) ([]ItemType, error) {
	stored, err := s.customItemTypes(ctx, ws)
	if err != nil {
		return DefaultItemTypes(), err
	}
	builtin := builtinTypeKeys()
	overrides := map[string]ItemType{}
	custom := []ItemType{}
	for _, t := range stored {
		if t.Key == "" {
			continue
		}
		if builtin[t.Key] {
			overrides[t.Key] = t
		} else {
			t.Builtin = false
			custom = append(custom, t)
		}
	}
	out := []ItemType{}
	for _, d := range DefaultItemTypes() {
		if ov, ok := overrides[d.Key]; ok {
			if ov.Label != "" && ov.Label != legacyBuiltinLabel[d.Key] {
				d.Label = ov.Label
			}
			if ov.Icon != "" {
				d.Icon = ov.Icon
			}
			d.Color = ov.Color
			d.Fill = ov.Fill
			if ov.Fields != nil {
				d.Fields = ov.Fields
			}
			d.WorkflowKey = ov.WorkflowKey
			d.Statuses = ov.Statuses
			d.Layout = ov.Layout
		}
		out = append(out, d)
	}
	out = append(out, custom...)
	// Every type must reference a workflow — default any without one (including
	// stored overrides that predate this rule) to the built-in "standard", so all
	// items get statuses (empty-status items then resolve to the start status).
	for i := range out {
		if out[i].WorkflowKey == "" {
			out[i].WorkflowKey = "standard"
		}
	}
	// Resolve shared-workflow references: a type with a WorkflowKey inherits that
	// workflow's statuses + layout (single source of truth for reused flows).
	if wfs, werr := s.GetWorkflows(ctx, ws); werr == nil && len(wfs) > 0 {
		byKey := make(map[string]Workflow, len(wfs))
		for _, w := range wfs {
			byKey[w.Key] = w
		}
		for i := range out {
			if k := out[i].WorkflowKey; k != "" {
				if w, ok := byKey[k]; ok {
					out[i].Statuses = w.Statuses
					out[i].Layout = w.Layout
				}
			}
		}
	}
	return out, nil
}

// SetItemTypes persists the type catalog for a workspace. Custom types are stored
// in full; for built-ins only icon/colour/label/fill are overridable (their key,
// family and field set stay authoritative from DefaultItemTypes).
func (s *Store) SetItemTypes(ctx context.Context, ws string, types []ItemType) error {
	builtin := builtinTypeKeys()
	defaults := map[string]ItemType{}
	for _, d := range DefaultItemTypes() {
		defaults[d.Key] = d
	}
	clean := []ItemType{}
	seen := map[string]bool{}
	for _, t := range types {
		if t.Key == "" || seen[t.Key] {
			continue
		}
		seen[t.Key] = true
		if builtin[t.Key] {
			d := defaults[t.Key]
			label := t.Label
			if label == "" {
				label = d.Label
			}
			icon := t.Icon
			if icon == "" {
				icon = d.Icon
			}
			fields := t.Fields
			if fields == nil {
				fields = []ItemField{}
			}
			wfKey := t.WorkflowKey
			if wfKey == "" { // every type must reference a workflow — default to the built-in "standard"
				wfKey = "standard"
			}
			// Referencing a shared workflow → don't persist a stale inline copy.
			bt := ItemType{Key: d.Key, Label: label, Family: d.Family, Icon: icon, Color: t.Color, Fill: t.Fill, Fields: fields, WorkflowKey: wfKey, Builtin: true}
			clean = append(clean, bt)
			continue
		}
		if !validFamily(t.Family) {
			t.Family = FamilyTimelinePoint
		}
		t.Builtin = false
		if t.Fields == nil {
			t.Fields = []ItemField{}
		}
		if t.WorkflowKey == "" { // every type must reference a workflow — default to the built-in "standard"
			t.WorkflowKey = "standard"
		}
		t.Statuses = nil // types don't carry inline statuses — the workflow supplies them
		t.Layout = nil
		clean = append(clean, t)
	}
	b, _ := json.Marshal(clean)
	_, err := s.pool.Exec(ctx,
		`INSERT INTO app_setting (workspace_id, key, value) VALUES ($1, 'item_types', $2)
		 ON CONFLICT (workspace_id, key) DO UPDATE SET value = $2`, ws, b)
	return err
}

// resolveStatus fills an empty status with the type's start status (its first),
// so a status-typed item always has a valid status stored — not just displayed.
func (s *Store) resolveStatus(ctx context.Context, ws string, it *Item) {
	if it.Status != "" {
		return
	}
	types, err := s.ListItemTypes(ctx, ws)
	if err != nil {
		return
	}
	key := typeKeyOf(*it)
	for _, t := range types {
		if t.Key == key && len(t.Statuses) > 0 {
			it.Status = t.Statuses[0].Key
			return
		}
	}
}
