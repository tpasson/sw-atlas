package store

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tpasson/sw-atlas/server/internal/db"
)

// TestUISettings covers the per-workspace display-settings blob: round-trip,
// isolation between workspaces, and JSON validation. Needs a throwaway Postgres
// — set ATLAS_TEST_DB.
func TestUISettings(t *testing.T) {
	dsn := os.Getenv("ATLAS_TEST_DB")
	if dsn == "" {
		t.Skip("set ATLAS_TEST_DB to run the UI settings test")
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
	alice, err := s.CreateUser(ctx, "alice", "h", RoleEditor)
	if err != nil {
		t.Fatalf("alice: %v", err)
	}

	// Unset → nil (client falls back to defaults).
	if raw, err := s.GetUISettings(ctx, DefaultWorkspaceID); err != nil || raw != nil {
		t.Fatalf("unset settings should be nil: raw=%s err=%v", raw, err)
	}

	fontSize := func(raw json.RawMessage) float64 {
		var m struct {
			Items struct {
				FontSize float64 `json:"fontSize"`
			} `json:"items"`
		}
		_ = json.Unmarshal(raw, &m)
		return m.Items.FontSize
	}

	// Round-trip on the default workspace.
	if err := s.SetUISettings(ctx, DefaultWorkspaceID, json.RawMessage(`{"items":{"fontSize":99}}`)); err != nil {
		t.Fatalf("set default: %v", err)
	}
	raw, err := s.GetUISettings(ctx, DefaultWorkspaceID)
	if err != nil || fontSize(raw) != 99 {
		t.Fatalf("default round-trip: raw=%s err=%v", raw, err)
	}

	// Isolation: alice's settings are independent.
	if err := s.SetUISettings(ctx, alice.WorkspaceID, json.RawMessage(`{"items":{"fontSize":42}}`)); err != nil {
		t.Fatalf("set alice: %v", err)
	}
	ar, _ := s.GetUISettings(ctx, alice.WorkspaceID)
	dr, _ := s.GetUISettings(ctx, DefaultWorkspaceID)
	if fontSize(ar) != 42 || fontSize(dr) != 99 {
		t.Fatalf("isolation broken: alice=%v default=%v", fontSize(ar), fontSize(dr))
	}

	// Invalid JSON is rejected.
	if err := s.SetUISettings(ctx, DefaultWorkspaceID, json.RawMessage(`not json`)); err == nil {
		t.Fatalf("invalid JSON should be rejected")
	}

	t.Log("UI settings: round-trip, per-workspace isolation, and JSON validation all hold")
}
