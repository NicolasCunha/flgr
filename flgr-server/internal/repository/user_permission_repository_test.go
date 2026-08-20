package repository

import (
	"context"
	"testing"
)

func TestUserPermissionRepository_ReplaceThenList(t *testing.T) {
	db := newTestDB(t)
	userRepo := NewUserRepository(db)
	repo := NewUserPermissionRepository(db)
	ctx := context.Background()

	actor := newTestUser("actor@example.com")
	if err := userRepo.Create(ctx, actor); err != nil {
		t.Fatalf("Create(actor) returned unexpected error: %v", err)
	}
	target := newTestUser("target@example.com")
	if err := userRepo.Create(ctx, target); err != nil {
		t.Fatalf("Create(target) returned unexpected error: %v", err)
	}

	ids, err := repo.ListPermissionIDs(ctx, target.ID)
	if err != nil {
		t.Fatalf("ListPermissionIDs() returned unexpected error: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("ListPermissionIDs() = %v, want empty before any grant", ids)
	}

	if err := repo.ReplacePermissions(ctx, target.ID, []string{"environment:view"}, actor.ID); err != nil {
		t.Fatalf("ReplacePermissions() returned unexpected error: %v", err)
	}

	ids, err = repo.ListPermissionIDs(ctx, target.ID)
	if err != nil {
		t.Fatalf("ListPermissionIDs() (after grant) returned unexpected error: %v", err)
	}
	if len(ids) != 1 || ids[0] != "environment:view" {
		t.Fatalf("ListPermissionIDs() = %v, want [environment:view]", ids)
	}
}

func TestUserPermissionRepository_ListPermissionIDs_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewUserPermissionRepository(db)
	_ = db.Close()

	if _, err := repo.ListPermissionIDs(context.Background(), "irrelevant"); err == nil {
		t.Fatal("ListPermissionIDs() on closed database expected an error, got nil")
	}
}

func TestUserPermissionRepository_ReplacePermissions_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewUserPermissionRepository(db)
	_ = db.Close()

	err := repo.ReplacePermissions(context.Background(), "irrelevant", []string{"user:view"}, "actor")
	if err == nil {
		t.Fatal("ReplacePermissions() on closed database expected an error, got nil")
	}
}
