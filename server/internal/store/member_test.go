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

// TestMemberManagement covers invite / role change / remove and the last-owner
// guard. Needs ATLAS_TEST_DB.
func TestMemberManagement(t *testing.T) {
	dsn := os.Getenv("ATLAS_TEST_DB")
	if dsn == "" {
		t.Skip("set ATLAS_TEST_DB to run the member-management DB test")
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

	owner, err := s.CreateUser(ctx, "p3owner", "h", RoleEditor)
	if err != nil {
		t.Fatalf("owner: %v", err)
	}
	guest, err := s.CreateUser(ctx, "p3guest", "h", RoleEditor)
	if err != nil {
		t.Fatalf("guest: %v", err)
	}
	ws := owner.WorkspaceID

	if m, err := s.AddMember(ctx, ws, "p3guest", "editor"); err != nil || m.Role != "editor" {
		t.Fatalf("invite guest: %v %+v", err, m)
	}
	if r, _ := s.RoleInWorkspace(ctx, guest.ID, ws); r != WSRoleEditor {
		t.Fatalf("guest role = %q, want editor", r)
	}
	if _, err := s.AddMember(ctx, ws, "nobody-here", "viewer"); err != ErrNotFound {
		t.Fatalf("invite unknown: want ErrNotFound, got %v", err)
	}

	// Last-owner protection.
	if err := s.RemoveMember(ctx, ws, owner.ID); err != ErrLastOwner {
		t.Fatalf("remove last owner: want ErrLastOwner, got %v", err)
	}
	if err := s.SetMemberRole(ctx, ws, owner.ID, "editor"); err != ErrLastOwner {
		t.Fatalf("demote last owner: want ErrLastOwner, got %v", err)
	}

	// With a 2nd owner, the first can be demoted.
	if err := s.SetMemberRole(ctx, ws, guest.ID, "owner"); err != nil {
		t.Fatalf("promote guest: %v", err)
	}
	if err := s.SetMemberRole(ctx, ws, owner.ID, "viewer"); err != nil {
		t.Fatalf("demote with two owners: %v", err)
	}
	if err := s.RemoveMember(ctx, ws, guest.ID); err != ErrLastOwner {
		t.Fatalf("remove now-sole owner: want ErrLastOwner, got %v", err)
	}

	if mem, _ := s.ListMembers(ctx, ws); len(mem) != 2 {
		t.Fatalf("members = %d, want 2", len(mem))
	}
}
