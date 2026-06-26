package store

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Workspace is a tenant: one plan, owned by a user (or the bootstrap admin for
// the default workspace). The slug is what appears in the URL (/{slug}).
type Workspace struct {
	ID          string  `json:"id"`
	Slug        string  `json:"slug"`
	Name        string  `json:"name"`
	OwnerUserID *string `json:"ownerUserId,omitempty"`
	Visibility  string  `json:"visibility"`
}

// GetWorkspace looks a workspace up by id.
func (s *Store) GetWorkspace(ctx context.Context, id string) (Workspace, error) {
	return s.scanWorkspace(ctx, `WHERE id = $1`, id)
}

// GetWorkspaceBySlug resolves the URL slug (case-insensitive) to a workspace.
func (s *Store) GetWorkspaceBySlug(ctx context.Context, slug string) (Workspace, error) {
	return s.scanWorkspace(ctx, `WHERE slug = $1`, strings.ToLower(strings.TrimSpace(slug)))
}

func (s *Store) scanWorkspace(ctx context.Context, where string, arg string) (Workspace, error) {
	var w Workspace
	err := s.pool.QueryRow(ctx,
		`SELECT id, slug, name, owner_user_id, visibility FROM workspace `+where, arg).
		Scan(&w.ID, &w.Slug, &w.Name, &w.OwnerUserID, &w.Visibility)
	if errors.Is(err, pgx.ErrNoRows) {
		return Workspace{}, ErrNotFound
	}
	return w, err
}
