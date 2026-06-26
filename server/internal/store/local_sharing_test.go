package store

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tpasson/sw-atlas/server/internal/db"
)

// TestLocalSharing covers Slice-D intra-server publish & subscribe: a published
// scope is discoverable, a local subscription mirrors it read-only, and
// unpublishing stops the sync. Needs a throwaway Postgres — set ATLAS_TEST_DB.
func TestLocalSharing(t *testing.T) {
	dsn := os.Getenv("ATLAS_TEST_DB")
	if dsn == "" {
		t.Skip("set ATLAS_TEST_DB to run the local sharing test")
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
	alice, err := s.CreateUser(ctx, "alice", "h", RoleEditor)
	must("alice", err)
	bob, err := s.CreateUser(ctx, "bob", "h", RoleEditor)
	must("bob", err)

	// Alice builds a plan and a scope covering her lane.
	_, err = s.CreateSwimlane(ctx, alice.WorkspaceID, "lane-a", "Alice Roadmap", "#0A84FF")
	must("alice lane", err)
	_, err = s.CreateItem(ctx, alice.WorkspaceID, Item{ID: "it-a", SwimlaneID: "lane-a", Year: 2026, Month: 6, Title: "Alpha", Kind: "milestone"})
	must("alice item", err)
	_, err = s.CreateShareScope(ctx, alice.WorkspaceID, ShareScope{ID: "scope-a", Name: "Public Roadmap", DetailLevel: "full", Lanes: []string{"lane-a"}})
	must("alice scope", err)

	// ── directory: hidden until published, then discoverable, excludes own ──
	if avail, _ := s.ListPublishedScopes(ctx, bob.WorkspaceID); len(avail) != 0 {
		t.Fatalf("scope should be hidden until published, got %d", len(avail))
	}
	must("publish", s.SetShareScopePublished(ctx, alice.WorkspaceID, "scope-a", true))
	avail, err := s.ListPublishedScopes(ctx, bob.WorkspaceID)
	must("directory", err)
	if len(avail) != 1 || avail[0].ScopeID != "scope-a" || avail[0].WorkspaceSlug != "alice" || avail[0].OwnerName != "alice" {
		t.Fatalf("published scope not discoverable correctly: %+v", avail)
	}
	if own, _ := s.ListPublishedScopes(ctx, alice.WorkspaceID); len(own) != 0 {
		t.Fatalf("directory must exclude the caller's own scopes, got %d", len(own))
	}

	// ── bob subscribes locally → alice's lane mirrors in read-only ──────────
	_, err = s.CreateLocalSubscription(ctx, bob.WorkspaceID, "sub-1", "alice · Public Roadmap", alice.WorkspaceID, "scope-a", 300)
	must("local subscription", err)
	must("first sync", s.SyncSubscription(ctx, bob.WorkspaceID, "sub-1"))

	bp, err := s.GetPlan(ctx, bob.WorkspaceID)
	must("bob plan", err)
	var mirrored *Item
	for i := range bp.Milestones {
		if bp.Milestones[i].Title == "Alpha" {
			mirrored = &bp.Milestones[i]
		}
	}
	if len(bp.Swimlanes) != 1 || bp.Swimlanes[0].Name != "Alice Roadmap" || mirrored == nil {
		t.Fatalf("alice's plan did not mirror into bob: lanes=%d item=%v", len(bp.Swimlanes), mirrored)
	}
	// The mirror is isolated: it lives in bob's workspace, not alice's source id.
	if mirrored.ID == "it-a" {
		t.Fatalf("mirror should be a fresh local copy, not the source id")
	}
	// And it is read-only (managed by the subscription).
	if err := s.UpdateItem(ctx, bob.WorkspaceID, mirrored.ID, Item{Title: "Hacked", Kind: "milestone", SwimlaneID: bp.Swimlanes[0].ID}); err != ErrLocked {
		t.Fatalf("mirrored item should be read-only: want ErrLocked, got %v", err)
	}

	// ── unpublishing withdraws consent: the next sync errors, mirror remains ─
	must("unpublish", s.SetShareScopePublished(ctx, alice.WorkspaceID, "scope-a", false))
	if _, err := s.publishedScopeDetail(ctx, alice.WorkspaceID, "scope-a"); err != ErrNotFound {
		t.Fatalf("unpublished scope should be ErrNotFound, got %v", err)
	}
	must("sync after unpublish", s.SyncSubscription(ctx, bob.WorkspaceID, "sub-1"))
	sub, err := s.GetSubscription(ctx, bob.WorkspaceID, "sub-1")
	must("get sub", err)
	if sub.LastStatus == "" || sub.LastStatus[:5] != "error" {
		t.Fatalf("sync after unpublish should record an error, got %q", sub.LastStatus)
	}

	t.Log("local sharing: publish/discovery, read-only mirror, and unpublish-withdraws-consent all hold")
}
