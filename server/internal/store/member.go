package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// ErrLastOwner guards against removing/demoting a project's final owner.
var ErrLastOwner = errors.New("a project must keep at least one owner")

// Member is one roster entry for the Members UI.
type Member struct {
	UserID    string    `json:"userId"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	Email     string    `json:"email,omitempty"` // omitted (blanked) for anonymous requesters
}

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

// ListMembers returns a workspace's members (owners first, then by username).
func (s *Store) ListMembers(ctx context.Context, wsID string) ([]Member, error) {
	out := []Member{}
	rows, err := s.pool.Query(ctx,
		`SELECT u.id, u.username, m.role, m.created_at, u.first_name, u.last_name, u.email
		   FROM workspace_member m JOIN app_user u ON u.id = m.user_id
		  WHERE m.workspace_id = $1
		  ORDER BY (m.role = 'owner') DESC, u.username`, wsID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.UserID, &m.Username, &m.Role, &m.CreatedAt, &m.FirstName, &m.LastName, &m.Email); err != nil {
			return out, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) countOwners(ctx context.Context, wsID string) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM workspace_member WHERE workspace_id = $1 AND role = 'owner'`, wsID).Scan(&n)
	return n, err
}

func (s *Store) memberRole(ctx context.Context, wsID, userID string) (string, error) {
	var role string
	err := s.pool.QueryRow(ctx,
		`SELECT role FROM workspace_member WHERE workspace_id = $1 AND user_id = $2`, wsID, userID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return role, err
}

// AddMember invites a user (by username) into a workspace as editor or viewer
// (upserting the role). Returns ErrNotFound if the username doesn't exist.
func (s *Store) AddMember(ctx context.Context, wsID, username, role string) (Member, error) {
	if role != WSRoleEditor && role != WSRoleViewer && role != WSRoleOwner {
		role = WSRoleViewer
	}
	u, _, err := s.GetUserByUsername(ctx, username)
	if err != nil {
		return Member{}, err
	}
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO workspace_member (workspace_id, user_id, role) VALUES ($1, $2, $3)
		 ON CONFLICT (workspace_id, user_id) DO UPDATE SET role = $3`, wsID, u.ID, role); err != nil {
		return Member{}, err
	}
	return Member{UserID: u.ID, Username: u.Username, Role: role}, nil
}

// SetMemberRole changes a member's role, refusing to demote the last owner.
func (s *Store) SetMemberRole(ctx context.Context, wsID, userID, role string) error {
	if role != WSRoleOwner && role != WSRoleEditor && role != WSRoleViewer {
		return ErrConflict
	}
	cur, err := s.memberRole(ctx, wsID, userID)
	if err != nil {
		return err
	}
	if cur == WSRoleOwner && role != WSRoleOwner {
		if n, err := s.countOwners(ctx, wsID); err != nil {
			return err
		} else if n <= 1 {
			return ErrLastOwner
		}
	}
	_, err = s.pool.Exec(ctx, `UPDATE workspace_member SET role = $3 WHERE workspace_id = $1 AND user_id = $2`, wsID, userID, role)
	return err
}

// RemoveMember removes a member (or a user leaving), refusing the last owner.
func (s *Store) RemoveMember(ctx context.Context, wsID, userID string) error {
	cur, err := s.memberRole(ctx, wsID, userID)
	if err != nil {
		return err
	}
	if cur == WSRoleOwner {
		if n, err := s.countOwners(ctx, wsID); err != nil {
			return err
		} else if n <= 1 {
			return ErrLastOwner
		}
	}
	_, err = s.pool.Exec(ctx, `DELETE FROM workspace_member WHERE workspace_id = $1 AND user_id = $2`, wsID, userID)
	return err
}
