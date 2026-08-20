package repository

import (
	"context"
	"testing"
)

func TestUserProfileRepository_ReplaceThenList(t *testing.T) {
	db := newTestDB(t)
	userRepo := NewUserRepository(db)
	profileRepo := NewProfileRepository(db)
	repo := NewUserProfileRepository(db)
	ctx := context.Background()

	actor := newTestUser("actor@example.com")
	if err := userRepo.Create(ctx, actor); err != nil {
		t.Fatalf("Create(actor) returned unexpected error: %v", err)
	}
	target := newTestUser("target@example.com")
	if err := userRepo.Create(ctx, target); err != nil {
		t.Fatalf("Create(target) returned unexpected error: %v", err)
	}
	profile := newTestProfile("some-profile")
	if err := profileRepo.Create(ctx, profile); err != nil {
		t.Fatalf("Create(profile) returned unexpected error: %v", err)
	}

	ids, err := repo.ListProfileIDs(ctx, target.ID)
	if err != nil {
		t.Fatalf("ListProfileIDs() returned unexpected error: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("ListProfileIDs() = %v, want empty before any assignment", ids)
	}

	if err := repo.ReplaceProfiles(ctx, target.ID, []string{profile.ID}, actor.ID); err != nil {
		t.Fatalf("ReplaceProfiles() returned unexpected error: %v", err)
	}

	ids, err = repo.ListProfileIDs(ctx, target.ID)
	if err != nil {
		t.Fatalf("ListProfileIDs() (after assign) returned unexpected error: %v", err)
	}
	if len(ids) != 1 || ids[0] != profile.ID {
		t.Fatalf("ListProfileIDs() = %v, want [%s]", ids, profile.ID)
	}
}

func TestUserProfileRepository_ListProfileIDs_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewUserProfileRepository(db)
	_ = db.Close()

	if _, err := repo.ListProfileIDs(context.Background(), "irrelevant"); err == nil {
		t.Fatal("ListProfileIDs() on closed database expected an error, got nil")
	}
}

func TestUserProfileRepository_ReplaceProfiles_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewUserProfileRepository(db)
	_ = db.Close()

	err := repo.ReplaceProfiles(context.Background(), "irrelevant", []string{"some-id"}, "actor")
	if err == nil {
		t.Fatal("ReplaceProfiles() on closed database expected an error, got nil")
	}
}
