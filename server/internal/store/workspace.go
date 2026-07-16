package store

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
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

// UserWorkspace is one switcher entry: a workspace the user belongs to + my role.
type UserWorkspace struct {
	Slug       string `json:"slug"`
	Name       string `json:"name"`
	Role       string `json:"role"`
	Visibility string `json:"visibility"`
}

// ListWorkspacesForUser returns every workspace the user is a member of with the
// user's role in each (powers the project switcher).
func (s *Store) ListWorkspacesForUser(ctx context.Context, userID string) ([]UserWorkspace, error) {
	out := []UserWorkspace{}
	rows, err := s.pool.Query(ctx,
		`SELECT w.slug, w.name, m.role, w.visibility
		   FROM workspace_member m JOIN workspace w ON w.id = m.workspace_id
		  WHERE m.user_id = $1
		  ORDER BY w.name`, userID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var uw UserWorkspace
		if err := rows.Scan(&uw.Slug, &uw.Name, &uw.Role, &uw.Visibility); err != nil {
			return out, err
		}
		out = append(out, uw)
	}
	return out, rows.Err()
}

// ListAllWorkspaces returns every workspace (for site admins). Role is the admin's
// own membership role where they have one, otherwise empty (read-only access).
func (s *Store) ListAllWorkspaces(ctx context.Context, adminUserID string) ([]UserWorkspace, error) {
	out := []UserWorkspace{}
	rows, err := s.pool.Query(ctx,
		`SELECT w.slug, w.name, COALESCE(m.role, ''), w.visibility
		   FROM workspace w
		   LEFT JOIN workspace_member m ON m.workspace_id = w.id AND m.user_id = $1
		  ORDER BY w.name`, adminUserID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var uw UserWorkspace
		if err := rows.Scan(&uw.Slug, &uw.Name, &uw.Role, &uw.Visibility); err != nil {
			return out, err
		}
		out = append(out, uw)
	}
	return out, rows.Err()
}

var slugNonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(name string) string {
	return strings.Trim(slugNonAlnum.ReplaceAllString(strings.ToLower(strings.TrimSpace(name)), "-"), "-")
}

// reservedSlugs can never be workspace slugs — they collide with app routes.
// Keep in sync with RESERVED_SLUGS in the frontend store.
var reservedSlugs = map[string]bool{
	"api": true, "assets": true, "health": true, "explore": true, "shared": true,
	"w": true, "default": true, "index.html": true, "favicon.svg": true, "projects": true,
}

func isReservedSlug(slug string) bool { return reservedSlugs[slug] }

// CreateWorkspace creates a project owned by ownerUserID, seeds the owner
// membership, and derives a unique slug from the name (numeric suffix on clash).
func (s *Store) CreateWorkspace(ctx context.Context, ownerUserID, name string) (Workspace, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Workspace{}, errors.New("name is required")
	}
	base := slugify(name)
	if base == "" || isReservedSlug(base) {
		base = "project"
	}
	id := "ws-" + uuid.NewString()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Workspace{}, err
	}
	defer tx.Rollback(ctx)

	slug := base
	for i := 2; ; i++ {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspace WHERE slug = $1)`, slug).Scan(&exists); err != nil {
			return Workspace{}, err
		}
		if !exists && !isReservedSlug(slug) {
			break
		}
		if i > 99 {
			return Workspace{}, ErrConflict
		}
		slug = fmt.Sprintf("%s-%d", base, i)
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO workspace (id, slug, name, owner_user_id, visibility) VALUES ($1, $2, $3, $4, 'private')`,
		id, slug, name, ownerUserID); err != nil {
		return Workspace{}, err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO app_setting (workspace_id, key, value) VALUES ($1, 'public_read_enabled', 'false')`, id); err != nil {
		return Workspace{}, err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO workspace_member (workspace_id, user_id, role) VALUES ($1, $2, 'owner')`, id, ownerUserID); err != nil {
		return Workspace{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Workspace{}, err
	}
	return Workspace{ID: id, Slug: slug, Name: name, OwnerUserID: &ownerUserID, Visibility: "private"}, nil
}

// RenameWorkspace changes a project's display name (owner-gated at the API).
func (s *Store) RenameWorkspace(ctx context.Context, wsID, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("name is required")
	}
	ct, err := s.pool.Exec(ctx, `UPDATE workspace SET name = $2 WHERE id = $1`, wsID, name)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteWorkspace deletes a project and everything it owns (FK ON DELETE CASCADE).
// A user's personal home workspace can't be deleted this way — that would cascade
// to the user account itself.
func (s *Store) DeleteWorkspace(ctx context.Context, wsID string) error {
	var isHome bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app_user WHERE workspace_id = $1)`, wsID).Scan(&isHome); err != nil {
		return err
	}
	if isHome {
		return fmt.Errorf("a personal home workspace can't be deleted as a project: %w", ErrProtected)
	}
	ct, err := s.pool.Exec(ctx, `DELETE FROM workspace WHERE id = $1`, wsID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// PublicWorkspace is a discovery-directory entry: a public plan plus a small
// summary so the landing page can render a card and link to /{slug}.
type PublicWorkspace struct {
	Slug           string  `json:"slug"`
	Name           string  `json:"name"`
	OwnerName      string  `json:"ownerName"`
	OwnerFirstName string  `json:"ownerFirstName,omitempty"`
	OwnerLastName  string  `json:"ownerLastName,omitempty"`
	OwnerEmail     string  `json:"ownerEmail,omitempty"` // blanked for anonymous requesters
	Personal       bool    `json:"personal"`             // true = a user's private home workspace; false = a project
	ItemCount      int     `json:"itemCount"`
	LateCount      int     `json:"lateCount"` // overdue, incomplete, non-event items
	NextTitle      *string `json:"nextTitle,omitempty"`
	NextDate       *string `json:"nextDate,omitempty"`   // YYYY-MM-DD of the next upcoming dated item
	LastChange     *string `json:"lastChange,omitempty"` // RFC3339 timestamp of the most recent edit
	Featured       bool    `json:"featured"`
}

// ListPublicWorkspaces returns every publicly-readable workspace for the explore
// page. A workspace counts as public when its public_read flag is true (or unset,
// matching GetPublicRead). Featured plans sort first, then by soonest milestone.
func (s *Store) ListPublicWorkspaces(ctx context.Context) ([]PublicWorkspace, error) {
	out := []PublicWorkspace{}
	rows, err := s.pool.Query(ctx,
		`SELECT w.slug, w.name, COALESCE(u.username, w.name), w.featured,
		        EXISTS(SELECT 1 FROM app_user au WHERE au.workspace_id = w.id),
		        (SELECT count(*) FROM item i WHERE i.workspace_id = w.id AND i.source_system IS NULL),
		        (SELECT count(*) FROM item i WHERE i.workspace_id = w.id
		             AND i.source_system IS NULL
		             AND i.progress IS NOT NULL AND i.progress < 100
		             AND COALESCE(i.end_date, i.when_date) < CURRENT_DATE),
		        (SELECT max(i.updated_at) FROM item i WHERE i.workspace_id = w.id),
		        n.title, n.when_date,
		        COALESCE(u.first_name, ''), COALESCE(u.last_name, ''), COALESCE(u.email, '')
		 FROM workspace w
		 LEFT JOIN app_user u ON u.id = w.owner_user_id
		 LEFT JOIN app_setting s ON s.workspace_id = w.id AND s.key = 'public_read_enabled'
		 LEFT JOIN LATERAL (
		     SELECT title, when_date FROM item
		     WHERE workspace_id = w.id AND source_system IS NULL AND when_date >= CURRENT_DATE
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
		var nextDate, lastChange *time.Time
		if err := rows.Scan(&p.Slug, &p.Name, &p.OwnerName, &p.Featured, &p.Personal, &p.ItemCount, &p.LateCount, &lastChange, &p.NextTitle, &nextDate, &p.OwnerFirstName, &p.OwnerLastName, &p.OwnerEmail); err != nil {
			return out, err
		}
		if nextDate != nil {
			d := nextDate.Format("2006-01-02")
			p.NextDate = &d
		}
		if lastChange != nil {
			d := lastChange.Format(time.RFC3339)
			p.LastChange = &d
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
