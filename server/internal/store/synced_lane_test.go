package store

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tpasson/sw-atlas/server/internal/db"
)

// TestSyncedLaneRules covers the read-only-source-lane behaviour: items can't be
// added, the name is owned by the source, but the consumer may recolour the lane
// and that colour survives a re-sync. Needs a throwaway Postgres — set ATLAS_TEST_DB.
func TestSyncedLaneRules(t *testing.T) {
	dsn := os.Getenv("ATLAS_TEST_DB")
	if dsn == "" {
		t.Skip("set ATLAS_TEST_DB to run the synced lane rules test")
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
	ws := DefaultWorkspaceID

	// Mirror one lane in as if from a GitHub source.
	wire := subWire{
		Swimlanes:  []wireLane{{ID: "r1", Name: "sw-atlas", Color: "#6E5494"}},
		Milestones: []Item{{ID: "rel1", SwimlaneID: "r1", Year: 2026, Month: 6, Title: "v1.0.0", Kind: "milestone"}},
	}
	if err := s.applyMirror(ctx, ws, "src-1", "https://example/repo", "github", wire); err != nil {
		t.Fatalf("applyMirror: %v", err)
	}

	plan, err := s.GetPlan(ctx, ws)
	if err != nil {
		t.Fatalf("GetPlan: %v", err)
	}
	var lane *Swimlane
	for i := range plan.Swimlanes {
		if plan.Swimlanes[i].SourceSystem != nil {
			lane = &plan.Swimlanes[i]
		}
	}
	if lane == nil || lane.SourceKind == nil || *lane.SourceKind != "github" {
		t.Fatalf("synced lane should expose sourceKind=github, got %+v", lane)
	}

	// Items can't be added to a synced lane.
	if _, err := s.CreateItem(ctx, ws, Item{ID: "x", SwimlaneID: lane.ID, Year: 2026, Month: 6, Title: "manual", Kind: "milestone"}); err != ErrLocked {
		t.Fatalf("adding to synced lane: want ErrLocked, got %v", err)
	}

	// The name is owned by the source...
	name := "Renamed"
	if err := s.UpdateSwimlane(ctx, ws, lane.ID, &name, nil); err != ErrLocked {
		t.Fatalf("renaming synced lane: want ErrLocked, got %v", err)
	}
	// ...but the colour is the consumer's to change.
	red := "#FF0000"
	if err := s.UpdateSwimlane(ctx, ws, lane.ID, nil, &red); err != nil {
		t.Fatalf("recolour synced lane: %v", err)
	}

	// Re-syncing the source must not clobber the consumer's colour (or kind).
	if err := s.applyMirror(ctx, ws, "src-1", "https://example/repo", "github", wire); err != nil {
		t.Fatalf("re-sync: %v", err)
	}
	plan, _ = s.GetPlan(ctx, ws)
	for _, sw := range plan.Swimlanes {
		if sw.ID == lane.ID {
			if sw.Color != red {
				t.Fatalf("re-sync wiped the user colour: %q", sw.Color)
			}
			if sw.SourceKind == nil || *sw.SourceKind != "github" {
				t.Fatalf("re-sync lost sourceKind: %v", sw.SourceKind)
			}
		}
	}

	t.Log("synced lane: no add, name locked, colour editable + survives re-sync, sourceKind exposed")
}
