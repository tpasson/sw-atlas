package store

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tpasson/sw-atlas/server/internal/db"
)

// TestRoleInWorkspace verifies the P1 membership backfill: each user owns their
// home workspace and is a non-member of everyone else's. Needs ATLAS_TEST_DB.
func TestRoleInWorkspace(t *testing.T) {
	dsn := os.Getenv("ATLAS_TEST_DB")
	if dsn == "" {
		t.Skip("set ATLAS_TEST_DB to run the membership DB test")
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

	alice, err := s.CreateUser(ctx, "p1alice", "h", RoleEditor)
	if err != nil {
		t.Fatalf("create p1alice: %v", err)
	}
	bob, err := s.CreateUser(ctx, "p1bob", "h", RoleEditor)
	if err != nil {
		t.Fatalf("create p1bob: %v", err)
	}

	if r, _ := s.RoleInWorkspace(ctx, alice.ID, alice.WorkspaceID); r != WSRoleOwner {
		t.Fatalf("alice in own workspace = %q, want owner", r)
	}
	if r, _ := s.RoleInWorkspace(ctx, alice.ID, bob.WorkspaceID); r != "" {
		t.Fatalf("alice in bob's workspace = %q, want non-member", r)
	}
	if r, _ := s.RoleInWorkspace(ctx, "", alice.WorkspaceID); r != "" {
		t.Fatalf("empty user = %q, want non-member", r)
	}
}
