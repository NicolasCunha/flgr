package repository

import (
	"context"
	"testing"
)

func TestServiceKeyEnvironmentRepository_ReplaceThenList(t *testing.T) {
	db := newTestDB(t)
	userRepo := NewUserRepository(db)
	repo := NewServiceKeyEnvironmentRepository(db)
	ctx := context.Background()

	actor := newTestUser("sk-env-actor@example.com")
	if err := userRepo.Create(ctx, actor); err != nil {
		t.Fatalf("Create(actor) returned unexpected error: %v", err)
	}
	key := newTestServiceKey("scoped-key")
	if err := NewServiceKeyRepository(db).Create(ctx, key); err != nil {
		t.Fatalf("Create(key) returned unexpected error: %v", err)
	}
	categoryID := developmentCategoryID(t, db)
	env := newTestEnvironment("scoped-env", categoryID)
	if err := NewEnvironmentRepository(db).Create(ctx, env); err != nil {
		t.Fatalf("Create(env) returned unexpected error: %v", err)
	}

	ids, err := repo.ListEnvironmentIDs(ctx, key.ID)
	if err != nil {
		t.Fatalf("ListEnvironmentIDs() returned unexpected error: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("ListEnvironmentIDs() = %v, want empty before any assignment", ids)
	}

	if err := repo.ReplaceEnvironments(ctx, key.ID, []string{env.ID}, actor.ID); err != nil {
		t.Fatalf("ReplaceEnvironments() returned unexpected error: %v", err)
	}

	ids, err = repo.ListEnvironmentIDs(ctx, key.ID)
	if err != nil {
		t.Fatalf("ListEnvironmentIDs() (after assign) returned unexpected error: %v", err)
	}
	if len(ids) != 1 || ids[0] != env.ID {
		t.Fatalf("ListEnvironmentIDs() = %v, want [%s]", ids, env.ID)
	}
}

func TestServiceKeyEnvironmentRepository_ListEnvironmentIDs_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewServiceKeyEnvironmentRepository(db)
	_ = db.Close()

	if _, err := repo.ListEnvironmentIDs(context.Background(), "irrelevant"); err == nil {
		t.Fatal("ListEnvironmentIDs() on closed database expected an error, got nil")
	}
}

func TestServiceKeyEnvironmentRepository_ReplaceEnvironments_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewServiceKeyEnvironmentRepository(db)
	_ = db.Close()

	err := repo.ReplaceEnvironments(context.Background(), "irrelevant", []string{"some-id"}, "actor")
	if err == nil {
		t.Fatal("ReplaceEnvironments() on closed database expected an error, got nil")
	}
}
