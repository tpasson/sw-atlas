package store

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Roles. Editors may fully edit their own workspace; admins additionally manage
// user accounts.
const (
	RoleAdmin  = "admin"
	RoleEditor = "editor"
)

var (
	// ErrConflict is returned when a unique constraint (e.g. username) is violated.
	ErrConflict = errors.New("username already exists")
	// ErrLastAdmin guards against demoting or deleting the final admin account.
	ErrLastAdmin = errors.New("cannot remove the last admin")
	// ErrProtected guards the bootstrap admin / default workspace from deletion.
	ErrProtected = errors.New("this account is protected and cannot be deleted")
)

// User is an account. PasswordHash is never part of this struct — it is read and
// written separately so it never leaks into a JSON response.
type User struct {
	ID          string    `json:"id"`
	Username    string    `json:"username"`
	Role        string    `json:"role"`
	WorkspaceID string    `json:"workspaceId"`
	CreatedAt   time.Time `json:"createdAt"`
}

func normUsername(u string) string { return strings.ToLower(strings.TrimSpace(u)) }

func normRole(r string) string {
	if r == RoleAdmin {
		return RoleAdmin
	}
	return RoleEditor
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// CreateUser inserts a user together with a fresh personal workspace they own.
// The workspace slug is the username (usernames are unique). The new workspace
// starts private (public-read off) to match its 'private' visibility.
func (s *Store) CreateUser(ctx context.Context, username, passwordHash, role string) (User, error) {
	username = normUsername(username)
	if username == "" || passwordHash == "" {
		return User{}, errors.New("username and password are required")
	}
	id := uuid.NewString()
	wsID := "ws-" + id

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`INSERT INTO workspace (id, slug, name, owner_user_id, visibility)
		 VALUES ($1, $2, $3, $4, 'private')`, wsID, username, username, id); err != nil {
		if isUniqueViolation(err) {
			return User{}, ErrConflict
		}
		return User{}, err
	}
	// New personal workspace is private: turn the legacy public-read flag off too.
	if _, err := tx.Exec(ctx,
		`INSERT INTO app_setting (workspace_id, key, value) VALUES ($1, 'public_read_enabled', 'false')`,
		wsID); err != nil {
		return User{}, err
	}

	var u User
	err = tx.QueryRow(ctx,
		`INSERT INTO app_user (id, username, password_hash, role, workspace_id)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, username, role, workspace_id, created_at`,
		id, username, passwordHash, normRole(role), wsID).
		Scan(&u.ID, &u.Username, &u.Role, &u.WorkspaceID, &u.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return User{}, ErrConflict
		}
		return User{}, err
	}
	// The creator owns their home workspace.
	if _, err := tx.Exec(ctx,
		`INSERT INTO workspace_member (workspace_id, user_id, role) VALUES ($1, $2, 'owner')`,
		wsID, id); err != nil {
		return User{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return User{}, err
	}
	return u, nil
}

// GetUserByUsername returns the user plus their stored password hash, for login.
func (s *Store) GetUserByUsername(ctx context.Context, username string) (User, string, error) {
	var u User
	var hash string
	err := s.pool.QueryRow(ctx,
		`SELECT id, username, role, workspace_id, created_at, password_hash
		 FROM app_user WHERE username = $1`, normUsername(username)).
		Scan(&u.ID, &u.Username, &u.Role, &u.WorkspaceID, &u.CreatedAt, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, "", ErrNotFound
	}
	return u, hash, err
}

// ListUsers returns all accounts, oldest first.
func (s *Store) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, username, role, workspace_id, created_at FROM app_user ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []User{}
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.WorkspaceID, &u.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// CountUsers reports how many accounts exist (used for bootstrap + warnings).
func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM app_user`).Scan(&n)
	return n, err
}

// SetUserRole changes a user's role, refusing to demote the last admin.
func (s *Store) SetUserRole(ctx context.Context, id, role string) error {
	role = normRole(role)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var cur string
	if err := tx.QueryRow(ctx, `SELECT role FROM app_user WHERE id = $1`, id).Scan(&cur); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if cur == RoleAdmin && role != RoleAdmin {
		if err := assertNotLastAdmin(ctx, tx); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE app_user SET role = $2 WHERE id = $1`, id, role); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// SetUserPassword replaces a user's password hash.
func (s *Store) SetUserPassword(ctx context.Context, id, passwordHash string) error {
	if passwordHash == "" {
		return errors.New("password is required")
	}
	ct, err := s.pool.Exec(ctx, `UPDATE app_user SET password_hash = $2 WHERE id = $1`, id, passwordHash)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteUser removes a user and their personal workspace, cascading all of that
// workspace's plan data. It refuses to delete the last admin or the bootstrap
// admin that owns the primary 'default' workspace.
func (s *Store) DeleteUser(ctx context.Context, id string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var role, wsID string
	if err := tx.QueryRow(ctx, `SELECT role, workspace_id FROM app_user WHERE id = $1`, id).Scan(&role, &wsID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if wsID == DefaultWorkspaceID {
		return ErrProtected
	}
	if role == RoleAdmin {
		if err := assertNotLastAdmin(ctx, tx); err != nil {
			return err
		}
	}
	// Deleting the personal workspace cascades to app_user (workspace_id FK is
	// ON DELETE CASCADE) and every plan-scoped table.
	if _, err := tx.Exec(ctx, `DELETE FROM workspace WHERE id = $1`, wsID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// EnsureBootstrapAdmin creates the first admin from the env editor credentials
// when no accounts exist yet, binding them to the pre-existing default
// workspace. Idempotent: a no-op once any user exists or when no hash is set.
func (s *Store) EnsureBootstrapAdmin(ctx context.Context, username, passwordHash string) error {
	username = normUsername(username)
	if username == "" || passwordHash == "" {
		return nil
	}
	n, err := s.CountUsers(ctx)
	if err != nil || n > 0 {
		return err
	}
	id := uuid.NewString()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`INSERT INTO app_user (id, username, password_hash, role, workspace_id)
		 VALUES ($1, $2, $3, 'admin', $4)`, id, username, passwordHash, DefaultWorkspaceID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`UPDATE workspace SET owner_user_id = $1 WHERE id = $2 AND owner_user_id IS NULL`,
		id, DefaultWorkspaceID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO workspace_member (workspace_id, user_id, role) VALUES ($1, $2, 'owner')
		 ON CONFLICT DO NOTHING`,
		DefaultWorkspaceID, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func assertNotLastAdmin(ctx context.Context, tx pgx.Tx) error {
	var admins int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM app_user WHERE role = 'admin'`).Scan(&admins); err != nil {
		return err
	}
	if admins <= 1 {
		return ErrLastAdmin
	}
	return nil
}
