package store

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tpasson/sw-atlas/server/internal/db"
)

// TestWorkspaceResolution covers the Slice-C slug→workspace lookup that the URL
// router relies on. Needs a throwaway Postgres — set ATLAS_TEST_DB.
func TestWorkspaceResolution(t *testing.T) {
	dsn := os.Getenv("ATLAS_TEST_DB")
	if dsn == "" {
		t.Skip("set ATLAS_TEST_DB to run the workspace resolution test")
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

	if err := s.EnsureBootstrapAdmin(ctx, "editor", "h"); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	alice, err := s.CreateUser(ctx, "Alice", "h", RoleUser)
	if err != nil {
		t.Fatalf("create alice: %v", err)
	}

	// The default workspace resolves by its slug (case-insensitively).
	if ws, err := s.GetWorkspaceBySlug(ctx, "DeFaUlT"); err != nil || ws.ID != DefaultWorkspaceID {
		t.Fatalf("default by slug: ws=%+v err=%v", ws, err)
	}

	// A user's slug is their username, owned by them, and private by default.
	ws, err := s.GetWorkspaceBySlug(ctx, "alice")
	if err != nil {
		t.Fatalf("alice by slug: %v", err)
	}
	if ws.ID != alice.WorkspaceID || ws.Slug != "alice" {
		t.Fatalf("alice workspace mismatch: %+v (user ws %s)", ws, alice.WorkspaceID)
	}
	if ws.OwnerUserID == nil || *ws.OwnerUserID != alice.ID {
		t.Fatalf("alice workspace owner wrong: %+v", ws.OwnerUserID)
	}
	if pub, _ := s.GetPublicRead(ctx, ws.ID); pub {
		t.Fatalf("new user workspace should be private by default")
	}

	// Lookup by id round-trips, and unknown slugs are ErrNotFound.
	if got, err := s.GetWorkspace(ctx, alice.WorkspaceID); err != nil || got.Slug != "alice" {
		t.Fatalf("GetWorkspace by id: got=%+v err=%v", got, err)
	}
	if _, err := s.GetWorkspaceBySlug(ctx, "ghost"); err != ErrNotFound {
		t.Fatalf("unknown slug: want ErrNotFound, got %v", err)
	}

	t.Log("workspace slug resolution, ownership and private-by-default all hold")
}
