package repository

import (
	"context"
	"testing"
)

func TestAuthzRepository_EffectivePermissions_Union(t *testing.T) {
	db := newTestDB(t)
	userRepo := NewUserRepository(db)
	profileRepo := NewProfileRepository(db)
	profilePermRepo := NewProfilePermissionRepository(db)
	userProfileRepo := NewUserProfileRepository(db)
	userPermRepo := NewUserPermissionRepository(db)
	authzRepo := NewAuthzRepository(db)
	ctx := context.Background()

	actor := newTestUser("actor@example.com")
	if err := userRepo.Create(ctx, actor); err != nil {
		t.Fatalf("Create(actor) returned unexpected error: %v", err)
	}
	target := newTestUser("target@example.com")
	if err := userRepo.Create(ctx, target); err != nil {
		t.Fatalf("Create(target) returned unexpected error: %v", err)
	}
	profile := newTestProfile("via-profile")
	if err := profileRepo.Create(ctx, profile); err != nil {
		t.Fatalf("Create(profile) returned unexpected error: %v", err)
	}

	if err := profilePermRepo.ReplacePermissions(ctx, profile.ID, []string{"environment:view", "environment:create"}, actor.ID); err != nil {
		t.Fatalf("ReplacePermissions(profile) returned unexpected error: %v", err)
	}
	if err := userProfileRepo.ReplaceProfiles(ctx, target.ID, []string{profile.ID}, actor.ID); err != nil {
		t.Fatalf("ReplaceProfiles() returned unexpected error: %v", err)
	}
	if err := userPermRepo.ReplacePermissions(ctx, target.ID, []string{"user:view"}, actor.ID); err != nil {
		t.Fatalf("ReplacePermissions(user) returned unexpected error: %v", err)
	}

	perms, err := authzRepo.EffectivePermissions(ctx, target.ID)
	if err != nil {
		t.Fatalf("EffectivePermissions() returned unexpected error: %v", err)
	}

	got := make(map[string]bool, len(perms))
	for _, p := range perms {
		got[p.ID] = true
	}
	want := []string{"environment:view", "environment:create", "user:view"}
	if len(got) != len(want) {
		t.Fatalf("EffectivePermissions() = %v, want exactly %v", perms, want)
	}
	for _, id := range want {
		if !got[id] {
			t.Errorf("EffectivePermissions() missing %q", id)
		}
	}
}

func TestAuthzRepository_EffectivePermissions_NoGrants(t *testing.T) {
	db := newTestDB(t)
	userRepo := NewUserRepository(db)
	authzRepo := NewAuthzRepository(db)
	ctx := context.Background()

	target := newTestUser("nogrants@example.com")
	if err := userRepo.Create(ctx, target); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	perms, err := authzRepo.EffectivePermissions(ctx, target.ID)
	if err != nil {
		t.Fatalf("EffectivePermissions() returned unexpected error: %v", err)
	}
	if len(perms) != 0 {
		t.Errorf("EffectivePermissions() = %v, want empty", perms)
	}
}

func TestAuthzRepository_EffectivePermissions_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewAuthzRepository(db)
	_ = db.Close()

	if _, err := repo.EffectivePermissions(context.Background(), "irrelevant"); err == nil {
		t.Fatal("EffectivePermissions() on closed database expected an error, got nil")
	}
}
