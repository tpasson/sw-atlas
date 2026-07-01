package store

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tpasson/sw-atlas/server/internal/db"
)

// TestGitColors covers the per-workspace synced-item colour scheme: defaults,
// round-trip, isolation, and the ""→inherit pointer helper. Needs a throwaway
// Postgres — set ATLAS_TEST_DB.
func TestGitColors(t *testing.T) {
	// Pure helper check (no DB needed).
	if gitColorPtr("") != nil {
		t.Fatal(`gitColorPtr("") should be nil (inherit lane)`)
	}
	if p := gitColorPtr("#abc"); p == nil || *p != "#abc" {
		t.Fatalf("gitColorPtr(#abc) wrong: %v", p)
	}

	dsn := os.Getenv("ATLAS_TEST_DB")
	if dsn == "" {
		t.Skip("set ATLAS_TEST_DB to run the git colours DB test")
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
	alice, err := s.CreateUser(ctx, "alice", "h", RoleUser)
	if err != nil {
		t.Fatalf("alice: %v", err)
	}

	// Unset → defaults.
	c, err := s.GetGitColors(ctx, DefaultWorkspaceID)
	if err != nil || c != DefaultGitColors() {
		t.Fatalf("unset should equal defaults: %+v err=%v", c, err)
	}

	// Round-trip a partial override; the helper field fills from defaults.
	custom := DefaultGitColors()
	custom.PROpen = "#00BB00"
	custom.IssueOpen = "" // deliberately inherit
	if err := s.SetGitColors(ctx, DefaultWorkspaceID, custom); err != nil {
		t.Fatalf("set: %v", err)
	}
	got, _ := s.GetGitColors(ctx, DefaultWorkspaceID)
	if got.PROpen != "#00BB00" || got.IssueOpen != "" || got.ReleasePre != "#FF9F0A" {
		t.Fatalf("round-trip wrong: %+v", got)
	}

	// Isolation: alice keeps the defaults.
	ac, _ := s.GetGitColors(ctx, alice.WorkspaceID)
	if ac != DefaultGitColors() {
		t.Fatalf("alice colours leaked: %+v", ac)
	}

	t.Log("git colours: defaults, round-trip, isolation, and ''→inherit helper all hold")
}
