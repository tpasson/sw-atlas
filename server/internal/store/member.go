package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// Per-workspace roles (distinct from the global app_user.role admin/editor, which
// governs site-level account + explore administration).
const (
	WSRoleOwner  = "owner"  // manage members, rename/delete, set visibility, + edit
	WSRoleEditor = "editor" // full edit of the plan
	WSRoleViewer = "viewer" // read-only, even when the workspace is private
)

// RoleInWorkspace returns the user's role in a workspace, or "" if not a member.
// This is the authorization source for read/edit access.
func (s *Store) RoleInWorkspace(ctx context.Context, userID, wsID string) (string, error) {
	if userID == "" || wsID == "" {
		return "", nil
	}
	var role string
	err := s.pool.QueryRow(ctx,
		`SELECT role FROM workspace_member WHERE user_id = $1 AND workspace_id = $2`,
		userID, wsID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return role, err
}
