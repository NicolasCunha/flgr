package repository

import (
	"context"
	"testing"
)

func TestProfilePermissionRepository_ListPermissionIDs_Empty(t *testing.T) {
	db := newTestDB(t)
	profileRepo := NewProfileRepository(db)
	repo := NewProfilePermissionRepository(db)
	ctx := context.Background()

	p := newTestProfile("empty-perms")
	if err := profileRepo.Create(ctx, p); err != nil {
		t.Fatalf("Create(profile) returned unexpected error: %v", err)
	}

	ids, err := repo.ListPermissionIDs(ctx, p.ID)
	if err != nil {
		t.Fatalf("ListPermissionIDs() returned unexpected error: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("ListPermissionIDs() = %v, want empty", ids)
	}
}

func TestProfilePermissionRepository_ReplacePermissions_SetsThenReplaces(t *testing.T) {
	db := newTestDB(t)
	userRepo := NewUserRepository(db)
	profileRepo := NewProfileRepository(db)
	repo := NewProfilePermissionRepository(db)
	ctx := context.Background()

	actor := newTestUser("actor@example.com")
	if err := userRepo.Create(ctx, actor); err != nil {
		t.Fatalf("Create(user) returned unexpected error: %v", err)
	}
	p := newTestProfile("replace-perms")
	if err := profileRepo.Create(ctx, p); err != nil {
		t.Fatalf("Create(profile) returned unexpected error: %v", err)
	}

	if err := repo.ReplacePermissions(ctx, p.ID, []string{"user:view", "user:create"}, actor.ID); err != nil {
		t.Fatalf("ReplacePermissions() (first) returned unexpected error: %v", err)
	}
	ids, err := repo.ListPermissionIDs(ctx, p.ID)
	if err != nil {
		t.Fatalf("ListPermissionIDs() returned unexpected error: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("len(ids) = %d, want 2", len(ids))
	}

	if err := repo.ReplacePermissions(ctx, p.ID, []string{"user:edit"}, actor.ID); err != nil {
		t.Fatalf("ReplacePermissions() (second) returned unexpected error: %v", err)
	}
	ids, err = repo.ListPermissionIDs(ctx, p.ID)
	if err != nil {
		t.Fatalf("ListPermissionIDs() (after replace) returned unexpected error: %v", err)
	}
	if len(ids) != 1 || ids[0] != "user:edit" {
		t.Fatalf("ListPermissionIDs() (after replace) = %v, want [user:edit]", ids)
	}

	if err := repo.ReplacePermissions(ctx, p.ID, nil, actor.ID); err != nil {
		t.Fatalf("ReplacePermissions() (clear) returned unexpected error: %v", err)
	}
	ids, err = repo.ListPermissionIDs(ctx, p.ID)
	if err != nil {
		t.Fatalf("ListPermissionIDs() (after clear) returned unexpected error: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("ListPermissionIDs() (after clear) = %v, want empty", ids)
	}
}

func TestProfilePermissionRepository_ListPermissionIDs_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewProfilePermissionRepository(db)
	_ = db.Close()

	if _, err := repo.ListPermissionIDs(context.Background(), "irrelevant"); err == nil {
		t.Fatal("ListPermissionIDs() on closed database expected an error, got nil")
	}
}

func TestProfilePermissionRepository_ReplacePermissions_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewProfilePermissionRepository(db)
	_ = db.Close()

	err := repo.ReplacePermissions(context.Background(), "irrelevant", []string{"user:view"}, "actor")
	if err == nil {
		t.Fatal("ReplacePermissions() on closed database expected an error, got nil")
	}
}

func TestProfilePermissionRepository_ReplacePermissions_InvalidPermissionID(t *testing.T) {
	db := newTestDB(t)
	userRepo := NewUserRepository(db)
	profileRepo := NewProfileRepository(db)
	repo := NewProfilePermissionRepository(db)
	ctx := context.Background()

	actor := newTestUser("actor2@example.com")
	if err := userRepo.Create(ctx, actor); err != nil {
		t.Fatalf("Create(user) returned unexpected error: %v", err)
	}
	p := newTestProfile("bad-perm-id")
	if err := profileRepo.Create(ctx, p); err != nil {
		t.Fatalf("Create(profile) returned unexpected error: %v", err)
	}

	err := repo.ReplacePermissions(ctx, p.ID, []string{"does-not-exist"}, actor.ID)
	if err == nil {
		t.Fatal("ReplacePermissions() with an unknown permission id expected an error (FK violation), got nil")
	}

	// The transaction must have rolled back — no rows left behind.
	ids, listErr := repo.ListPermissionIDs(ctx, p.ID)
	if listErr != nil {
		t.Fatalf("ListPermissionIDs() returned unexpected error: %v", listErr)
	}
	if len(ids) != 0 {
		t.Errorf("ListPermissionIDs() after failed replace = %v, want empty (rolled back)", ids)
	}
}
