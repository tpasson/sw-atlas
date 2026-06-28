package store

import "context"

// ItemType is a registry entry: a kind of artifact bound to a coded behavior
// family that decides how it renders and which fields apply. Built-ins are
// seeded and non-deletable; per-workspace custom types merge on top later (T3).
type ItemType struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Family  string `json:"family"` // see the Family* constants
	Icon    string `json:"icon"`   // lucide marker id (e.g. "l:Diamond")
	Color   string `json:"color"`  // "" = inherit the lane colour
	Builtin bool   `json:"builtin"`
}

// Behavior families are CODED and finite — they decide rendering + field set.
// The set of types WITHIN a family is data (so new types need no code).
const (
	FamilyTimelinePoint = "timeline-point" // dated point on a lane
	FamilyTimelineRange = "timeline-range" // start→end bar on a lane
	FamilyWorkItem      = "work-item"      // backlog/board card (status, assignee, parent)
	FamilyContainer     = "container"      // grouping node
)

// DefaultItemTypes are the seeded built-ins. Their keys equal the legacy item
// kinds (milestone/event/point) so existing items map 1:1 — see migration 00019.
func DefaultItemTypes() []ItemType {
	return []ItemType{
		{Key: "milestone", Label: "Milestone", Family: FamilyTimelinePoint, Icon: "l:Diamond", Builtin: true},
		{Key: "event", Label: "Event", Family: FamilyTimelineRange, Icon: "l:Flag", Builtin: true},
		{Key: "point", Label: "Point", Family: FamilyTimelinePoint, Icon: "l:Circle", Builtin: true},
	}
}

// ListItemTypes returns the item-type catalog for a workspace. For now it is
// just the built-ins; T3 will merge per-workspace custom types from settings.
func (s *Store) ListItemTypes(ctx context.Context, ws string) ([]ItemType, error) {
	return DefaultItemTypes(), nil
}
