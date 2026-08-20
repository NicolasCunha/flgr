package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
)

func newTestFeatureFlag(key string) *model.FeatureFlag {
	now := time.Now().UTC().Truncate(time.Second)
	desc := "a test flag"
	return &model.FeatureFlag{
		ID:          uuid.NewString(),
		Key:         key,
		Name:        "Test Flag",
		Description: &desc,
		Type:        model.FeatureFlagTypeBoolean,
		AuditFields: model.AuditFields{CreatedOn: now, ModifiedOn: now},
	}
}

func TestFeatureFlagRepository_CreateAndGetByID(t *testing.T) {
	repo := NewFeatureFlagRepository(newTestDB(t))
	ctx := context.Background()

	f := newTestFeatureFlag("new-checkout-flow")
	if err := repo.Create(ctx, f); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	got, err := repo.GetByID(ctx, f.ID)
	if err != nil {
		t.Fatalf("GetByID() returned unexpected error: %v", err)
	}
	if got.Key != "new-checkout-flow" || got.Type != model.FeatureFlagTypeBoolean {
		t.Errorf("GetByID() = %+v, want Key=new-checkout-flow Type=Boolean", got)
	}
}

func TestFeatureFlagRepository_Create_DuplicateKey(t *testing.T) {
	repo := NewFeatureFlagRepository(newTestDB(t))
	ctx := context.Background()

	if err := repo.Create(ctx, newTestFeatureFlag("dup")); err != nil {
		t.Fatalf("Create() (first) returned unexpected error: %v", err)
	}
	err := repo.Create(ctx, newTestFeatureFlag("dup"))
	if !errors.Is(err, ErrFeatureFlagKeyInUse) {
		t.Fatalf("Create() (duplicate) error = %v, want ErrFeatureFlagKeyInUse", err)
	}
}

func TestFeatureFlagRepository_Create_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewFeatureFlagRepository(db)
	_ = db.Close()

	err := repo.Create(context.Background(), newTestFeatureFlag("closed"))
	if err == nil || errors.Is(err, ErrFeatureFlagKeyInUse) {
		t.Fatalf("Create() on closed database error = %v, want a plain non-nil error", err)
	}
}

func TestFeatureFlagRepository_GetByID_NotFound(t *testing.T) {
	repo := NewFeatureFlagRepository(newTestDB(t))

	_, err := repo.GetByID(context.Background(), uuid.NewString())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByID() error = %v, want ErrNotFound", err)
	}
}

func TestFeatureFlagRepository_GetByID_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewFeatureFlagRepository(db)
	_ = db.Close()

	_, err := repo.GetByID(context.Background(), "irrelevant")
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByID() on closed database error = %v, want a non-nil, non-ErrNotFound error", err)
	}
}

func TestFeatureFlagRepository_GetByKey(t *testing.T) {
	repo := NewFeatureFlagRepository(newTestDB(t))
	ctx := context.Background()

	f := newTestFeatureFlag("lookup-by-key")
	if err := repo.Create(ctx, f); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	got, err := repo.GetByKey(ctx, "lookup-by-key")
	if err != nil {
		t.Fatalf("GetByKey() returned unexpected error: %v", err)
	}
	if got.ID != f.ID {
		t.Errorf("GetByKey() ID = %q, want %q", got.ID, f.ID)
	}
}

func TestFeatureFlagRepository_GetByKey_NotFound(t *testing.T) {
	repo := NewFeatureFlagRepository(newTestDB(t))

	_, err := repo.GetByKey(context.Background(), "does-not-exist")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByKey() error = %v, want ErrNotFound", err)
	}
}

func TestFeatureFlagRepository_List(t *testing.T) {
	repo := NewFeatureFlagRepository(newTestDB(t))
	ctx := context.Background()

	for _, key := range []string{"a-flag", "b-flag", "c-flag"} {
		if err := repo.Create(ctx, newTestFeatureFlag(key)); err != nil {
			t.Fatalf("Create(%q) returned unexpected error: %v", key, err)
		}
	}

	page, total, err := repo.List(ctx, 2, 0)
	if err != nil {
		t.Fatalf("List() returned unexpected error: %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if len(page) != 2 {
		t.Errorf("len(page) = %d, want 2", len(page))
	}
}

func TestFeatureFlagRepository_List_Empty(t *testing.T) {
	repo := NewFeatureFlagRepository(newTestDB(t))

	page, total, err := repo.List(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("List() returned unexpected error: %v", err)
	}
	if total != 0 || len(page) != 0 {
		t.Errorf("List() = (%v, %d), want (empty, 0)", page, total)
	}
}

func TestFeatureFlagRepository_List_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewFeatureFlagRepository(db)
	_ = db.Close()

	if _, _, err := repo.List(context.Background(), 10, 0); err == nil {
		t.Fatal("List() on closed database expected an error, got nil")
	}
}

func TestFeatureFlagRepository_Update(t *testing.T) {
	db := newTestDB(t)
	userRepo := NewUserRepository(db)
	repo := NewFeatureFlagRepository(db)
	ctx := context.Background()

	actor := newTestUser("flag-editor@example.com")
	if err := userRepo.Create(ctx, actor); err != nil {
		t.Fatalf("Create(actor) returned unexpected error: %v", err)
	}

	f := newTestFeatureFlag("update-me")
	if err := repo.Create(ctx, f); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	newDesc := "updated description"
	f.Name = "Updated Name"
	f.Description = &newDesc
	f.ModifiedByUserID = &actor.ID
	if err := repo.Update(ctx, f); err != nil {
		t.Fatalf("Update() returned unexpected error: %v", err)
	}

	got, err := repo.GetByID(ctx, f.ID)
	if err != nil {
		t.Fatalf("GetByID() returned unexpected error: %v", err)
	}
	if got.Name != "Updated Name" || got.Description == nil || *got.Description != "updated description" {
		t.Errorf("GetByID() after Update() = %+v, want Name=Updated Name Description=\"updated description\"", got)
	}
	if got.Key != "update-me" {
		t.Errorf("GetByID() after Update() Key = %q, want unchanged %q", got.Key, "update-me")
	}
}

func TestFeatureFlagRepository_Update_NotFound(t *testing.T) {
	repo := NewFeatureFlagRepository(newTestDB(t))

	err := repo.Update(context.Background(), newTestFeatureFlag("ghost"))
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Update() error = %v, want ErrNotFound", err)
	}
}

func TestFeatureFlagRepository_Update_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewFeatureFlagRepository(db)
	_ = db.Close()

	err := repo.Update(context.Background(), newTestFeatureFlag("closed-update"))
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("Update() on closed database error = %v, want a plain non-nil error", err)
	}
}

func TestFeatureFlagRepository_Delete(t *testing.T) {
	repo := NewFeatureFlagRepository(newTestDB(t))
	ctx := context.Background()

	f := newTestFeatureFlag("delete-me")
	if err := repo.Create(ctx, f); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	if err := repo.Delete(ctx, f.ID); err != nil {
		t.Fatalf("Delete() returned unexpected error: %v", err)
	}

	_, err := repo.GetByID(ctx, f.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByID() after Delete() error = %v, want ErrNotFound", err)
	}
}

func TestFeatureFlagRepository_Delete_CascadesEnvironmentValues(t *testing.T) {
	db := newTestDB(t)
	repo := NewFeatureFlagRepository(db)
	valueRepo := NewFeatureFlagEnvironmentValueRepository(db)
	ctx := context.Background()

	categoryID := developmentCategoryID(t, db)
	env := newTestEnvironment("flag-delete-cascade-env", categoryID)
	if err := NewEnvironmentRepository(db).Create(ctx, env); err != nil {
		t.Fatalf("Create(env) returned unexpected error: %v", err)
	}

	f := newTestFeatureFlag("cascade-me")
	if err := repo.Create(ctx, f); err != nil {
		t.Fatalf("Create(flag) returned unexpected error: %v", err)
	}
	if err := valueRepo.Create(ctx, newTestFeatureFlagEnvironmentValue(f.ID, env.ID)); err != nil {
		t.Fatalf("Create(value) returned unexpected error: %v", err)
	}

	if err := repo.Delete(ctx, f.ID); err != nil {
		t.Fatalf("Delete() returned unexpected error: %v", err)
	}

	_, err := valueRepo.GetByFlagAndEnvironment(ctx, f.ID, env.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByFlagAndEnvironment() after flag Delete() error = %v, want ErrNotFound (cascaded)", err)
	}
}

func TestFeatureFlagRepository_Delete_NotFound(t *testing.T) {
	repo := NewFeatureFlagRepository(newTestDB(t))

	err := repo.Delete(context.Background(), uuid.NewString())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Delete() error = %v, want ErrNotFound", err)
	}
}

func TestFeatureFlagRepository_Delete_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewFeatureFlagRepository(db)
	_ = db.Close()

	err := repo.Delete(context.Background(), "irrelevant")
	if err == nil {
		t.Fatal("Delete() on closed database expected an error, got nil")
	}
}
