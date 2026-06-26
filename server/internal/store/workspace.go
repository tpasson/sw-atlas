package store

import (
	"context"
	"errors"
	"strings"
	"time"

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

// PublicWorkspace is a discovery-directory entry: a public plan plus a small
// summary so the landing page can render a card and link to /{slug}.
type PublicWorkspace struct {
	Slug      string  `json:"slug"`
	Name      string  `json:"name"`
	OwnerName string  `json:"ownerName"`
	ItemCount int     `json:"itemCount"`
	NextTitle *string `json:"nextTitle,omitempty"`
	NextDate  *string `json:"nextDate,omitempty"` // YYYY-MM-DD of the next upcoming milestone
	Featured  bool    `json:"featured"`
}

// ListPublicWorkspaces returns every publicly-readable workspace for the explore
// page. A workspace counts as public when its public_read flag is true (or unset,
// matching GetPublicRead). Featured plans sort first, then by soonest milestone.
func (s *Store) ListPublicWorkspaces(ctx context.Context) ([]PublicWorkspace, error) {
	out := []PublicWorkspace{}
	rows, err := s.pool.Query(ctx,
		`SELECT w.slug, w.name, COALESCE(u.username, w.name), w.featured,
		        (SELECT count(*) FROM item i WHERE i.workspace_id = w.id),
		        n.title, n.when_date
		 FROM workspace w
		 LEFT JOIN app_user u ON u.id = w.owner_user_id
		 LEFT JOIN app_setting s ON s.workspace_id = w.id AND s.key = 'public_read_enabled'
		 LEFT JOIN LATERAL (
		     SELECT title, when_date FROM item
		     WHERE workspace_id = w.id AND when_date >= CURRENT_DATE
		     ORDER BY when_date LIMIT 1
		 ) n ON true
		 WHERE COALESCE(s.value, 'true') = 'true'
		 ORDER BY w.featured DESC, n.when_date NULLS LAST, w.slug`)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var p PublicWorkspace
		var nextDate *time.Time
		if err := rows.Scan(&p.Slug, &p.Name, &p.OwnerName, &p.Featured, &p.ItemCount, &p.NextTitle, &nextDate); err != nil {
			return out, err
		}
		if nextDate != nil {
			d := nextDate.Format("2006-01-02")
			p.NextDate = &d
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// SetWorkspaceFeatured pins/unpins a workspace on the explore page (admin only).
func (s *Store) SetWorkspaceFeatured(ctx context.Context, slug string, featured bool) error {
	ct, err := s.pool.Exec(ctx,
		`UPDATE workspace SET featured = $2 WHERE slug = $1`, strings.ToLower(strings.TrimSpace(slug)), featured)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
