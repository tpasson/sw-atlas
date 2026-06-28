package store

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tpasson/sw-atlas/server/internal/db"
)

// The built-in catalog must cover the legacy kinds 1:1 (so migration 00019 maps
// every existing item to a real type) and bind each to a coded family.
func TestDefaultItemTypes(t *testing.T) {
	byKey := map[string]ItemType{}
	for _, it := range DefaultItemTypes() {
		if !it.Builtin {
			t.Errorf("%s should be builtin", it.Key)
		}
		byKey[it.Key] = it
	}
	want := map[string]string{
		"milestone": FamilyTimelinePoint,
		"event":     FamilyTimelineRange,
		"point":     FamilyTimelinePoint,
	}
	for key, fam := range want {
		it, ok := byKey[key]
		if !ok {
			t.Fatalf("missing built-in type %q", key)
		}
		if it.Family != fam {
			t.Errorf("%s: family=%q want %q", key, it.Family, fam)
		}
		if it.Label == "" || it.Icon == "" {
			t.Errorf("%s: label/icon must be set", key)
		}
	}
}

// TestItemTypesRoundTrip checks custom-type persistence, the merge with built-ins,
// and that built-in / empty keys are dropped. Needs a throwaway Postgres.
func TestItemTypesRoundTrip(t *testing.T) {
	dsn := os.Getenv("ATLAS_TEST_DB")
	if dsn == "" {
		t.Skip("set ATLAS_TEST_DB to run the item-types DB test")
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

	if list, err := s.ListItemTypes(ctx, DefaultWorkspaceID); err != nil || len(list) != 3 {
		t.Fatalf("want 3 built-ins, got %d err=%v", len(list), err)
	}

	in := []ItemType{
		{Key: "bug", Label: "Bug", Family: FamilyTimelinePoint, Icon: "l:Bug", Color: "#FF3B30",
			Fields: []ItemField{{Key: "severity", Label: "Severity", Type: "select", Options: []string{"low", "high"}}}},
		{Key: "milestone", Label: "Hijack"}, // built-in key → dropped
		{Key: "", Label: "no key"},           // empty key → dropped
	}
	if err := s.SetItemTypes(ctx, DefaultWorkspaceID, in); err != nil {
		t.Fatalf("set: %v", err)
	}
	list, _ := s.ListItemTypes(ctx, DefaultWorkspaceID)
	if len(list) != 4 {
		t.Fatalf("want 3 built-ins + 1 custom, got %d", len(list))
	}
	var bug *ItemType
	for i := range list {
		if list[i].Key == "bug" {
			bug = &list[i]
		}
	}
	if bug == nil || bug.Builtin {
		t.Fatalf("custom 'bug' missing or marked builtin: %+v", bug)
	}
	if len(bug.Fields) != 1 || bug.Fields[0].Key != "severity" || len(bug.Fields[0].Options) != 2 {
		t.Errorf("bug fields wrong: %+v", bug.Fields)
	}
}
