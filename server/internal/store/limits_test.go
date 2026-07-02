package store

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tpasson/sw-atlas/server/internal/db"
)

func TestLimits(t *testing.T) {
	dsn := os.Getenv("ATLAS_TEST_DB")
	if dsn == "" {
		t.Skip("set ATLAS_TEST_DB to run the limits test")
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

	must("bootstrap", s.EnsureBootstrapAdmin(ctx, "editor", "h"))
	u, err := s.CreateUser(ctx, "quota", "h", RoleUser)
	must("user", err)

	// An unset config yields the recommended defaults.
	if lim, _ := s.GetLimits(ctx); lim != DefaultLimits {
		t.Fatalf("unset limits should be DefaultLimits, got %+v", lim)
	}

	// ── item cap: 2 items ok, the 3rd is rejected ───────────────────────────
	must("set limits", s.SetLimits(ctx, Limits{MaxItemsPerPlan: 2, MaxProjectsPerUser: 1}))
	_, err = s.CreateSwimlane(ctx, u.WorkspaceID, "lane", "L", "#0A84FF")
	must("lane", err)
	_, err = s.CreateItem(ctx, u.WorkspaceID, Item{ID: "i1", SwimlaneID: "lane", Year: 2026, Month: 1, Title: "A", Kind: "milestone"})
	must("item 1", err)
	_, err = s.CreateItem(ctx, u.WorkspaceID, Item{ID: "i2", SwimlaneID: "lane", Year: 2026, Month: 2, Title: "B", Kind: "milestone"})
	must("item 2", err)
	if _, err := s.CreateItem(ctx, u.WorkspaceID, Item{ID: "i3", SwimlaneID: "lane", Year: 2026, Month: 3, Title: "C", Kind: "milestone"}); err != ErrLimitReached {
		t.Fatalf("3rd item should hit the cap: want ErrLimitReached, got %v", err)
	}

	// ── owned-project count excludes the personal home plan ─────────────────
	if n, _ := s.CountOwnedProjects(ctx, u.ID); n != 0 {
		t.Fatalf("new user should own 0 projects (home excluded), got %d", n)
	}
	_, err = s.CreateWorkspace(ctx, u.ID, "Project One")
	must("project", err)
	if n, _ := s.CountOwnedProjects(ctx, u.ID); n != 1 {
		t.Fatalf("after one project, owned projects = 1, got %d", n)
	}

	t.Log("limits: defaults, per-plan item cap, and owned-project count all hold")
}
