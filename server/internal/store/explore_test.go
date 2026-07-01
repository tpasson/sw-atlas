package store

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tpasson/sw-atlas/server/internal/db"
)

// TestExploreDirectory covers the discovery landing page (#15): only public
// workspaces are listed, the card summary is populated, and featuring pins a
// plan first. Needs a throwaway Postgres — set ATLAS_TEST_DB.
func TestExploreDirectory(t *testing.T) {
	dsn := os.Getenv("ATLAS_TEST_DB")
	if dsn == "" {
		t.Skip("set ATLAS_TEST_DB to run the explore directory test")
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

	must("bootstrap", s.EnsureBootstrapAdmin(ctx, "editor", "h")) // default workspace, public
	alice, err := s.CreateUser(ctx, "alice", "h", RoleUser)     // private by default
	must("alice", err)
	_, err = s.CreateUser(ctx, "bob", "h", RoleUser) // stays private
	must("bob", err)

	// Alice goes public and adds an upcoming milestone.
	_, err = s.CreateSwimlane(ctx, alice.WorkspaceID, "lane-a", "Alice Lane", "#0A84FF")
	must("alice lane", err)
	when := "2099-12-01"
	_, err = s.CreateItem(ctx, alice.WorkspaceID, Item{ID: "it-a", SwimlaneID: "lane-a", Year: 2099, Month: 12, Title: "Alpha Launch", Kind: "milestone", When: &when})
	must("alice item", err)
	must("alice public", s.SetPublicRead(ctx, alice.WorkspaceID, true))

	// ── only public workspaces are listed ───────────────────────────────────
	dir, err := s.ListPublicWorkspaces(ctx)
	must("list", err)
	slugs := map[string]PublicWorkspace{}
	for _, w := range dir {
		slugs[w.Slug] = w
	}
	if _, ok := slugs["default"]; !ok {
		t.Fatalf("public default workspace missing from explore")
	}
	a, ok := slugs["alice"]
	if !ok {
		t.Fatalf("public alice workspace missing from explore")
	}
	if _, ok := slugs["bob"]; ok {
		t.Fatalf("private bob workspace must NOT appear in explore")
	}

	// ── card summary populated ──────────────────────────────────────────────
	if a.ItemCount != 1 || a.OwnerName != "alice" {
		t.Fatalf("alice card wrong: %+v", a)
	}
	if a.NextTitle == nil || *a.NextTitle != "Alpha Launch" || a.NextDate == nil || *a.NextDate != when {
		t.Fatalf("alice next-milestone summary wrong: title=%v date=%v", a.NextTitle, a.NextDate)
	}

	// ── featuring pins the plan first; unknown slug is ErrNotFound ──────────
	must("feature", s.SetWorkspaceFeatured(ctx, "alice", true))
	dir, err = s.ListPublicWorkspaces(ctx)
	must("list2", err)
	if len(dir) == 0 || dir[0].Slug != "alice" || !dir[0].Featured {
		t.Fatalf("featured plan should sort first and be flagged, got %+v", dir[0])
	}
	if err := s.SetWorkspaceFeatured(ctx, "ghost", true); err != ErrNotFound {
		t.Fatalf("featuring unknown slug: want ErrNotFound, got %v", err)
	}

	t.Log("explore directory: public-only listing, card summary, and featured-first ordering all hold")
}
