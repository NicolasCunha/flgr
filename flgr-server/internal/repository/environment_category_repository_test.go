package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
)

func newTestEnvironmentCategory(name string) *model.EnvironmentCategory {
	now := time.Now().UTC().Truncate(time.Second)
	return &model.EnvironmentCategory{
		ID:          uuid.NewString(),
		Name:        name,
		IsSystem:    false,
		AuditFields: model.AuditFields{CreatedOn: now, ModifiedOn: now},
	}
}

func TestEnvironmentCategoryRepository_SeededSystemCategories(t *testing.T) {
	repo := NewEnvironmentCategoryRepository(newTestDB(t))

	categories, err := repo.ListAll(context.Background())
	if err != nil {
		t.Fatalf("ListAll() returned unexpected error: %v", err)
	}
	if len(categories) != 3 {
		t.Fatalf("len(categories) = %d, want 3 (seeded)", len(categories))
	}
	for _, c := range categories {
		if !c.IsSystem {
			t.Errorf("category %q IsSystem = false, want true for a seeded category", c.Name)
		}
	}
}

func TestEnvironmentCategoryRepository_CreateAndGetByID(t *testing.T) {
	repo := NewEnvironmentCategoryRepository(newTestDB(t))
	ctx := context.Background()

	c := newTestEnvironmentCategory("Sandbox")
	if err := repo.Create(ctx, c); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	got, err := repo.GetByID(ctx, c.ID)
	if err != nil {
		t.Fatalf("GetByID() returned unexpected error: %v", err)
	}
	if got.Name != "Sandbox" || got.IsSystem {
		t.Errorf("GetByID() = %+v, want Name=Sandbox IsSystem=false", got)
	}
}

func TestEnvironmentCategoryRepository_Create_DuplicateName(t *testing.T) {
	repo := NewEnvironmentCategoryRepository(newTestDB(t))
	ctx := context.Background()

	if err := repo.Create(ctx, newTestEnvironmentCategory("Sandbox")); err != nil {
		t.Fatalf("Create() (first) returned unexpected error: %v", err)
	}
	err := repo.Create(ctx, newTestEnvironmentCategory("Sandbox"))
	if !errors.Is(err, ErrEnvironmentCategoryNameInUse) {
		t.Fatalf("Create() (duplicate) error = %v, want ErrEnvironmentCategoryNameInUse", err)
	}
}

func TestEnvironmentCategoryRepository_Create_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewEnvironmentCategoryRepository(db)
	_ = db.Close()

	err := repo.Create(context.Background(), newTestEnvironmentCategory("Sandbox"))
	if err == nil || errors.Is(err, ErrEnvironmentCategoryNameInUse) {
		t.Fatalf("Create() on closed database error = %v, want a plain non-nil error", err)
	}
}

func TestEnvironmentCategoryRepository_GetByID_NotFound(t *testing.T) {
	repo := NewEnvironmentCategoryRepository(newTestDB(t))

	_, err := repo.GetByID(context.Background(), uuid.NewString())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByID() error = %v, want ErrNotFound", err)
	}
}

func TestEnvironmentCategoryRepository_GetByID_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewEnvironmentCategoryRepository(db)
	_ = db.Close()

	_, err := repo.GetByID(context.Background(), "irrelevant")
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByID() on closed database error = %v, want a non-nil, non-ErrNotFound error", err)
	}
}

func TestEnvironmentCategoryRepository_GetByName(t *testing.T) {
	repo := NewEnvironmentCategoryRepository(newTestDB(t))

	got, err := repo.GetByName(context.Background(), "Development")
	if err != nil {
		t.Fatalf("GetByName() returned unexpected error: %v", err)
	}
	if !got.IsSystem {
		t.Errorf("IsSystem = false, want true for the seeded Development category")
	}
}

func TestEnvironmentCategoryRepository_GetByName_NotFound(t *testing.T) {
	repo := NewEnvironmentCategoryRepository(newTestDB(t))

	_, err := repo.GetByName(context.Background(), "does-not-exist")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByName() error = %v, want ErrNotFound", err)
	}
}

func TestEnvironmentCategoryRepository_ListAll_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewEnvironmentCategoryRepository(db)
	_ = db.Close()

	if _, err := repo.ListAll(context.Background()); err == nil {
		t.Fatal("ListAll() on closed database expected an error, got nil")
	}
}
