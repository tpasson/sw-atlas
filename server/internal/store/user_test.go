package store

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tpasson/sw-atlas/server/internal/db"
)

// TestUserAccounts verifies Slice-B account management: bootstrap, per-user
// workspace isolation, role guards and protected-account deletion. Needs a
// throwaway Postgres — set ATLAS_TEST_DB (the it15.sh wrapper does this).
func TestUserAccounts(t *testing.T) {
	dsn := os.Getenv("ATLAS_TEST_DB")
	if dsn == "" {
		t.Skip("set ATLAS_TEST_DB to run the user accounts test")
	}
	if err := db.Up(dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()
	s := &Store{pool: pool}
	ctx := context.Background()
	must := func(label string, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("%s: %v", label, err)
		}
	}

	// ── bootstrap admin binds to the default workspace, and is idempotent ────
	must("bootstrap", s.EnsureBootstrapAdmin(ctx, "Editor", "hash-admin"))
	must("bootstrap again", s.EnsureBootstrapAdmin(ctx, "editor", "hash-admin"))
	admin, hash, err := s.GetUserByUsername(ctx, "editor") // username normalised to lowercase
	must("get admin", err)
	if admin.Role != RoleAdmin || admin.WorkspaceID != DefaultWorkspaceID {
		t.Fatalf("bootstrap admin wrong: role=%q ws=%q", admin.Role, admin.WorkspaceID)
	}
	if hash != "hash-admin" {
		t.Fatalf("login hash not returned: %q", hash)
	}
	if n, _ := s.CountUsers(ctx); n != 1 {
		t.Fatalf("bootstrap should create exactly one user, got %d", n)
	}

	// ── a new user gets their own private, isolated workspace ───────────────
	// Seed some data in the admin's (default) workspace first.
	_, err = s.CreateSwimlane(ctx, DefaultWorkspaceID, "sw-admin", "Admin lane", "#0A84FF")
	must("admin swimlane", err)

	bob, err := s.CreateUser(ctx, "Bob", "hash-bob", RoleUser)
	must("create bob", err)
	if bob.Role != RoleUser || bob.WorkspaceID == DefaultWorkspaceID || bob.Username != "bob" {
		t.Fatalf("bob wrong: %+v", bob)
	}
	// Bob's fresh workspace is empty (isolated from the admin's default plan).
	bp, err := s.GetPlan(ctx, bob.WorkspaceID)
	must("bob plan", err)
	if len(bp.Swimlanes) != 0 {
		t.Fatalf("new user workspace should be empty, got %d swimlanes", len(bp.Swimlanes))
	}
	// And his workspace starts private (public-read off).
	if pr, _ := s.GetPublicRead(ctx, bob.WorkspaceID); pr {
		t.Fatalf("new user workspace should be private (public-read off)")
	}

	// ── duplicate username conflicts ────────────────────────────────────────
	if _, err := s.CreateUser(ctx, "bob", "x", RoleUser); err != ErrConflict {
		t.Fatalf("duplicate username: want ErrConflict, got %v", err)
	}

	// ── role + last-admin guards ────────────────────────────────────────────
	if err := s.SetUserRole(ctx, admin.ID, RoleUser); err != ErrLastAdmin {
		t.Fatalf("demoting sole admin: want ErrLastAdmin, got %v", err)
	}
	must("promote bob", s.SetUserRole(ctx, bob.ID, RoleAdmin)) // now two admins
	must("demote admin ok", s.SetUserRole(ctx, admin.ID, RoleUser))
	must("restore admin", s.SetUserRole(ctx, admin.ID, RoleAdmin))

	// ── rename: username + home workspace slug follow; guards on taken/reserved ─
	carol, err := s.CreateUser(ctx, "carol", "h", RoleUser)
	must("create carol", err)
	newSlug, _, err := s.RenameUser(ctx, carol.ID, "Caroline")
	must("rename carol", err)
	if newSlug != "caroline" {
		t.Fatalf("rename slug: want caroline, got %q", newSlug)
	}
	if _, _, err := s.GetUserByUsername(ctx, "caroline"); err != nil {
		t.Fatalf("renamed user not found by new name: %v", err)
	}
	if ws, err := s.GetWorkspaceBySlug(ctx, "caroline"); err != nil || ws.ID != carol.WorkspaceID {
		t.Fatalf("home slug should follow the rename: err=%v ws=%+v", err, ws)
	}
	if _, _, err := s.RenameUser(ctx, carol.ID, "editor"); err != ErrConflict {
		t.Fatalf("rename to a taken name: want ErrConflict, got %v", err)
	}
	if _, _, err := s.RenameUser(ctx, carol.ID, "api"); err == nil {
		t.Fatalf("rename to a reserved slug should fail")
	}
	must("cleanup carol", s.DeleteUser(ctx, carol.ID))

	// ── deletion guards: protected bootstrap admin, then cascade ────────────
	if err := s.DeleteUser(ctx, admin.ID); err != ErrProtected {
		t.Fatalf("deleting default-workspace admin: want ErrProtected, got %v", err)
	}
	// Give Bob some data, then delete him → workspace + data cascade away.
	_, err = s.CreateSwimlane(ctx, bob.WorkspaceID, "sw-bob", "Bob lane", "#FF375F")
	must("bob swimlane", err)
	must("delete bob", s.DeleteUser(ctx, bob.ID))
	if _, _, err := s.GetUserByUsername(ctx, "bob"); err != ErrNotFound {
		t.Fatalf("bob should be gone: %v", err)
	}
	var laneCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM swimlane WHERE workspace_id = $1`, bob.WorkspaceID).Scan(&laneCount); err != nil {
		t.Fatalf("count bob lanes: %v", err)
	}
	if laneCount != 0 {
		t.Fatalf("deleting user should cascade their workspace data, %d lanes remain", laneCount)
	}
	// The admin's default workspace is untouched.
	ap, err := s.GetPlan(ctx, DefaultWorkspaceID)
	must("admin plan after", err)
	if len(ap.Swimlanes) != 1 {
		t.Fatalf("admin workspace perturbed: %d swimlanes", len(ap.Swimlanes))
	}

	t.Log("user accounts: bootstrap, per-user workspace isolation, role + deletion guards, and cascade all hold")
}
