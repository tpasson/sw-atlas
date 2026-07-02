package store

import (
	"context"
	"encoding/json"
	"errors"
)

// ErrLimitReached is returned when a create would exceed an instance quota.
var ErrLimitReached = errors.New("limit reached — ask an admin to raise it")

// Limits are instance-wide, admin-configurable caps. A value of 0 means
// unlimited / off, so an admin can disable any single limit.
type Limits struct {
	WritesPerMinute    int `json:"writesPerMinute"`    // per-user write rate limit
	MaxItemsPerPlan    int `json:"maxItemsPerPlan"`    // native items per workspace/plan
	MaxProjectsPerUser int `json:"maxProjectsPerUser"` // owned collaborative projects (excludes the home plan)
}

// DefaultLimits are the recommended out-of-the-box caps: generous enough never to
// bother normal use, but low enough to bound a runaway/abusive client. They apply
// until an admin saves a config (which may set any value to 0 = unlimited).
var DefaultLimits = Limits{WritesPerMinute: 240, MaxItemsPerPlan: 2000, MaxProjectsPerUser: 50}

const limitsKey = "limits"

// GetLimits returns the configured limits. An unset config yields DefaultLimits;
// once saved, the stored values are used verbatim (0 = unlimited is honoured).
func (s *Store) GetLimits(ctx context.Context) (Limits, error) {
	raw, err := s.GetInstanceSetting(ctx, limitsKey)
	if err != nil {
		return DefaultLimits, err
	}
	if raw == nil {
		return DefaultLimits, nil
	}
	lim := Limits{}
	if err := json.Unmarshal(raw, &lim); err != nil {
		return DefaultLimits, nil
	}
	return lim, nil
}

// SetLimits validates and persists the instance limits.
func (s *Store) SetLimits(ctx context.Context, lim Limits) error {
	if lim.WritesPerMinute < 0 || lim.MaxItemsPerPlan < 0 || lim.MaxProjectsPerUser < 0 {
		return errors.New("limits must be zero or positive")
	}
	raw, err := json.Marshal(lim)
	if err != nil {
		return err
	}
	return s.SetInstanceSetting(ctx, limitsKey, raw)
}

// countNativeItems counts the items a workspace owns itself (excludes mirrored
// content, which is bounded by its source and not created by the user).
func (s *Store) countNativeItems(ctx context.Context, ws string) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM item WHERE workspace_id = $1 AND source_system IS NULL`, ws).Scan(&n)
	return n, err
}

// CountOwnedProjects counts the collaborative projects a user owns, excluding
// their personal home plan (which every user always has).
func (s *Store) CountOwnedProjects(ctx context.Context, userID string) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM workspace
		 WHERE owner_user_id = $1
		   AND id <> COALESCE((SELECT workspace_id FROM app_user WHERE id = $1), '')`, userID).Scan(&n)
	return n, err
}
