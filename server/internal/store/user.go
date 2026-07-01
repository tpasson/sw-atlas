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

// Account (site) roles. A "user" owns and edits their own workspace(s); an
// "admin" additionally manages user accounts and the explore page. These are the
// account roles — distinct from the per-workspace roles owner/editor/viewer.
const (
	RoleAdmin = "admin"
	RoleUser  = "user"
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
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Role         string    `json:"role"`
	WorkspaceID  string    `json:"workspaceId"`
	CreatedAt    time.Time `json:"createdAt"`
	Email        string    `json:"email"`
	FirstName    string    `json:"firstName"`
	LastName     string    `json:"lastName"`
	TokenVersion int       `json:"-"` // session-revocation counter (never exposed)
}

func normUsername(u string) string { return strings.ToLower(strings.TrimSpace(u)) }

func normRole(r string) string {
	if r == RoleAdmin {
		return RoleAdmin
	}
	return RoleUser
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
		`SELECT id, username, role, workspace_id, created_at, email, first_name, last_name, token_version, password_hash
		 FROM app_user WHERE username = $1`, normUsername(username)).
		Scan(&u.ID, &u.Username, &u.Role, &u.WorkspaceID, &u.CreatedAt, &u.Email, &u.FirstName, &u.LastName, &u.TokenVersion, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, "", ErrNotFound
	}
	return u, hash, err
}

// GetUserByID returns a single account's public profile (no password hash).
func (s *Store) GetUserByID(ctx context.Context, id string) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx,
		`SELECT id, username, role, workspace_id, created_at, email, first_name, last_name
		 FROM app_user WHERE id = $1`, id).
		Scan(&u.ID, &u.Username, &u.Role, &u.WorkspaceID, &u.CreatedAt, &u.Email, &u.FirstName, &u.LastName)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

// UpdateUserProfile sets a user's real name and email (all optional).
func (s *Store) UpdateUserProfile(ctx context.Context, id, firstName, lastName, email string) error {
	ct, err := s.pool.Exec(ctx,
		`UPDATE app_user SET first_name = $2, last_name = $3, email = $4 WHERE id = $1`,
		id, strings.TrimSpace(firstName), strings.TrimSpace(lastName), strings.TrimSpace(email))
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UserTokenVersion returns a user's current session-revocation counter (used to
// invalidate stale JWTs). ErrNotFound if the account no longer exists.
func (s *Store) UserTokenVersion(ctx context.Context, id string) (int, error) {
	var v int
	err := s.pool.QueryRow(ctx, `SELECT token_version FROM app_user WHERE id = $1`, id).Scan(&v)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	return v, err
}

// ListUsers returns all accounts, oldest first.
func (s *Store) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, username, role, workspace_id, created_at, email, first_name, last_name
		   FROM app_user ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []User{}
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.WorkspaceID, &u.CreatedAt, &u.Email, &u.FirstName, &u.LastName); err != nil {
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
	// Bump token_version so the new role takes effect immediately (the user's old
	// token, which carries the previous role, is invalidated).
	if _, err := tx.Exec(ctx, `UPDATE app_user SET role = $2, token_version = token_version + 1 WHERE id = $1`, id, role); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// SetUserPassword updates the hash and bumps token_version (revoking all existing
// sessions). Returns the new token_version so the caller can re-issue the current
// user's own cookie and keep them logged in.
func (s *Store) SetUserPassword(ctx context.Context, id, passwordHash string) (int, error) {
	if passwordHash == "" {
		return 0, errors.New("password is required")
	}
	var tv int
	err := s.pool.QueryRow(ctx,
		`UPDATE app_user SET password_hash = $2, token_version = token_version + 1
		 WHERE id = $1 RETURNING token_version`, id, passwordHash).Scan(&tv)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	return tv, err
}

// RenameUser changes an account's login username AND its personal workspace slug
// (so the /{slug} URL follows the name). Bumps token_version. Returns the new slug
// (= normalized username) and token_version. ErrConflict if the name/slug is taken;
// rejects reserved slugs. Only the user's HOME workspace changes — projects are
// untouched.
func (s *Store) RenameUser(ctx context.Context, id, username string) (string, int, error) {
	username = normUsername(username)
	if username == "" {
		return "", 0, errors.New("username is required")
	}
	if isReservedSlug(username) {
		return "", 0, errors.New("that name is reserved")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", 0, err
	}
	defer tx.Rollback(ctx)

	var oldName, homeWs string
	if err := tx.QueryRow(ctx, `SELECT username, workspace_id FROM app_user WHERE id = $1`, id).
		Scan(&oldName, &homeWs); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", 0, ErrNotFound
		}
		return "", 0, err
	}
	// Home workspace: always move the slug; move the display name too only if it
	// still equals the old username (i.e. was never customized).
	if _, err := tx.Exec(ctx,
		`UPDATE workspace SET slug = $2, name = CASE WHEN name = $3 THEN $2 ELSE name END
		 WHERE id = $1`, homeWs, username, oldName); err != nil {
		if isUniqueViolation(err) {
			return "", 0, ErrConflict
		}
		return "", 0, err
	}
	var tv int
	if err := tx.QueryRow(ctx,
		`UPDATE app_user SET username = $2, token_version = token_version + 1
		 WHERE id = $1 RETURNING token_version`, id, username).Scan(&tv); err != nil {
		if isUniqueViolation(err) {
			return "", 0, ErrConflict
		}
		return "", 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", 0, err
	}
	return username, tv, nil
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
