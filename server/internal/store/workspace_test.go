package store

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tpasson/sw-atlas/server/internal/db"
)

// TestWorkspaceIsolation verifies the Slice-A tenant scoping: no store call ever
// leaks one workspace's data into another. Needs a throwaway Postgres — set
// ATLAS_TEST_DB (the it14.sh wrapper does this); skipped otherwise.
func TestWorkspaceIsolation(t *testing.T) {
	dsn := os.Getenv("ATLAS_TEST_DB")
	if dsn == "" {
		t.Skip("set ATLAS_TEST_DB to run the workspace isolation test")
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

	const A, B = DefaultWorkspaceID, "wsB"
	if _, err := pool.Exec(ctx, `INSERT INTO workspace (id,slug,name) VALUES ($1,$1,'B') ON CONFLICT DO NOTHING`, B); err != nil {
		t.Fatalf("seed wsB: %v", err)
	}
	must := func(label string, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("%s: %v", label, err)
		}
	}

	// ── populate both workspaces ────────────────────────────────────────────
	for _, w := range []string{A, B} {
		_, err := s.CreateSwimlane(ctx, w, "sw-"+w, "Lane "+w, "#0A84FF")
		must("CreateSwimlane "+w, err)
		for _, n := range []string{"1", "2"} {
			_, err := s.CreateItem(ctx, w, Item{ID: "it-" + w + "-" + n, SwimlaneID: "sw-" + w, Year: 2026, Month: 6, Title: "T" + n, Kind: "milestone"})
			must("CreateItem "+w, err)
		}
		must("AddLink "+w, s.AddLink(ctx, w, "it-"+w+"-1", "it-"+w+"-2", "depends-on"))
		must("SetPublicRead "+w, s.SetPublicRead(ctx, w, w == A))            // A=true, B=false
		must("SetPalette "+w, s.SetPalette(ctx, w, []string{"#" + w[:1] + "00000"}))
		must("SetGroups "+w, s.SetGroups(ctx, w, []Group{{ID: "g-" + w, Name: "G" + w}}))
		_, err = s.CreateSubscription(ctx, w, "sub-"+w, "Sub "+w, "https://x.example/"+w, "tok", 300)
		must("CreateSubscription "+w, err)
		_, err = s.CreateGitHubSource(ctx, w, "gh-"+w, GitHubSourceInput{Owner: "o", Repo: w, HTMLURL: "https://github.com/o/" + w, Provider: "github", Releases: true})
		must("CreateGitHubSource "+w, err)
	}
	_, err = s.CreateBaseline(ctx, A, "bl-A", "Baseline A", "")
	must("CreateBaseline A", err)

	// ── GetPlan is disjoint ─────────────────────────────────────────────────
	pa, err := s.GetPlan(ctx, A)
	must("GetPlan A", err)
	pb, err := s.GetPlan(ctx, B)
	must("GetPlan B", err)
	if len(pa.Swimlanes) != 1 || len(pb.Swimlanes) != 1 || pa.Swimlanes[0].ID == pb.Swimlanes[0].ID {
		t.Fatalf("swimlanes leaked: A=%d B=%d", len(pa.Swimlanes), len(pb.Swimlanes))
	}
	for _, m := range pa.Milestones {
		if m.SwimlaneID != "sw-"+A {
			t.Fatalf("GetPlan A leaked item from another workspace: %+v", m)
		}
	}
	if len(pa.Milestones) != 2 || len(pb.Milestones) != 2 || len(pa.Links) != 1 || len(pb.Links) != 1 {
		t.Fatalf("items/links leaked: A items=%d links=%d, B items=%d links=%d", len(pa.Milestones), len(pa.Links), len(pb.Milestones), len(pb.Links))
	}

	// ── cross-tenant access is ErrNotFound; the foreign row is untouched ────
	if err := s.UpdateItem(ctx, A, "it-"+B+"-1", Item{Title: "hacked", Kind: "milestone", SwimlaneID: "sw-" + B}); err != ErrNotFound {
		t.Fatalf("UpdateItem across tenants: want ErrNotFound, got %v", err)
	}
	if err := s.DeleteItem(ctx, A, "it-"+B+"-1"); err != ErrNotFound {
		t.Fatalf("DeleteItem across tenants: want ErrNotFound, got %v", err)
	}
	if err := s.DeleteSwimlane(ctx, A, "sw-"+B); err != ErrNotFound {
		t.Fatalf("DeleteSwimlane across tenants: want ErrNotFound, got %v", err)
	}
	if _, err := s.GetGitHubSource(ctx, A, "gh-"+B); err != ErrNotFound {
		t.Fatalf("GetGitHubSource across tenants: want ErrNotFound, got %v", err)
	}
	if _, err := s.GetSubscription(ctx, A, "sub-"+B); err != ErrNotFound {
		t.Fatalf("GetSubscription across tenants: want ErrNotFound, got %v", err)
	}
	if _, err := s.GetBaseline(ctx, B, "bl-A"); err != ErrNotFound {
		t.Fatalf("GetBaseline across tenants: want ErrNotFound, got %v", err)
	}
	// wsB still intact after the failed cross-tenant writes
	if pb2, _ := s.GetPlan(ctx, B); len(pb2.Milestones) != 2 {
		t.Fatalf("wsB items perturbed by cross-tenant writes: %d", len(pb2.Milestones))
	}

	// ── lists are per-workspace ─────────────────────────────────────────────
	if gh, _ := s.ListGitHubSources(ctx, A); len(gh) != 1 || gh[0].Repo != A {
		t.Fatalf("ListGitHubSources A leaked: %+v", gh)
	}
	if subs, _ := s.ListSubscriptions(ctx, A); len(subs) != 1 {
		t.Fatalf("ListSubscriptions A leaked: %d", len(subs))
	}
	if bl, _ := s.ListBaselines(ctx, B); len(bl) != 0 {
		t.Fatalf("ListBaselines B should be empty, got %d", len(bl))
	}

	// ── per-workspace settings differ ───────────────────────────────────────
	prA, _ := s.GetPublicRead(ctx, A)
	prB, _ := s.GetPublicRead(ctx, B)
	if !prA || prB {
		t.Fatalf("public_read leaked: A=%v B=%v (want true/false)", prA, prB)
	}
	if palA, _ := s.GetPalette(ctx, A); len(palA) != 1 || palA[0] != "#d00000" {
		t.Fatalf("palette A leaked: %+v", palA)
	}

	// ── baseline snapshot is scoped to its workspace ────────────────────────
	bA, err := s.GetBaseline(ctx, A, "bl-A")
	must("GetBaseline A", err)
	if len(bA.Items) != 2 {
		t.Fatalf("baseline A snapshot should hold only A's 2 items, got %d", len(bA.Items))
	}

	// ── FEDERATION FEED isolation (the crown jewel) ─────────────────────────
	if _, err := s.CreateShareScope(ctx, B, ShareScope{ID: "sc-B", Name: "B scope", DetailLevel: "full", Lanes: []string{"sw-" + B}}); err != nil {
		must("CreateShareScope B", err)
	}
	if _, err := s.CreateShareToken(ctx, B, "tok-B", "sc-B", "hash-B", "label", nil); err != nil {
		must("CreateShareToken B", err)
	}
	gotWs, scopeID, detail, err := s.ResolveToken(ctx, "hash-B")
	must("ResolveToken", err)
	if gotWs != B || scopeID != "sc-B" {
		t.Fatalf("ResolveToken returned wrong workspace/scope: ws=%q scope=%q", gotWs, scopeID)
	}
	feed, err := s.ResolveScopePlan(ctx, gotWs, scopeID, detail)
	must("ResolveScopePlan", err)
	for _, m := range feed.Milestones {
		if m.SwimlaneID != "sw-"+B {
			t.Fatalf("federation feed LEAKED a non-wsB item: %+v", m)
		}
	}
	if len(feed.Milestones) != 2 {
		t.Fatalf("federation feed should serve wsB's 2 items, got %d", len(feed.Milestones))
	}

	t.Log("workspace isolation holds across plan, settings, baselines, sources, subscriptions and the federation feed")
}
