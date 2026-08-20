package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
)

// developmentCategoryID looks up the seeded "Development" category, for
// tests that need a valid environment_category_id to satisfy the FK.
func developmentCategoryID(t *testing.T, db *sql.DB) string {
	t.Helper()
	c, err := NewEnvironmentCategoryRepository(db).GetByName(context.Background(), "Development")
	if err != nil {
		t.Fatalf("GetByName(Development) returned unexpected error: %v", err)
	}
	return c.ID
}

func newTestEnvironment(name, categoryID string) *model.Environment {
	now := time.Now().UTC().Truncate(time.Second)
	desc := "a test environment"
	return &model.Environment{
		ID:                    uuid.NewString(),
		Name:                  name,
		Description:           &desc,
		EnvironmentCategoryID: categoryID,
		AuditFields:           model.AuditFields{CreatedOn: now, ModifiedOn: now},
	}
}

func TestEnvironmentRepository_CreateAndGetByID(t *testing.T) {
	db := newTestDB(t)
	categoryID := developmentCategoryID(t, db)
	repo := NewEnvironmentRepository(db)
	ctx := context.Background()

	e := newTestEnvironment("app-dev", categoryID)
	if err := repo.Create(ctx, e); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	got, err := repo.GetByID(ctx, e.ID)
	if err != nil {
		t.Fatalf("GetByID() returned unexpected error: %v", err)
	}
	if got.Name != "app-dev" || got.EnvironmentCategoryID != categoryID {
		t.Errorf("GetByID() = %+v, want Name=app-dev EnvironmentCategoryID=%q", got, categoryID)
	}
}

func TestEnvironmentRepository_Create_DuplicateName(t *testing.T) {
	db := newTestDB(t)
	categoryID := developmentCategoryID(t, db)
	repo := NewEnvironmentRepository(db)
	ctx := context.Background()

	if err := repo.Create(ctx, newTestEnvironment("dup", categoryID)); err != nil {
		t.Fatalf("Create() (first) returned unexpected error: %v", err)
	}
	err := repo.Create(ctx, newTestEnvironment("dup", categoryID))
	if !errors.Is(err, ErrEnvironmentNameInUse) {
		t.Fatalf("Create() (duplicate) error = %v, want ErrEnvironmentNameInUse", err)
	}
}

func TestEnvironmentRepository_Create_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	categoryID := developmentCategoryID(t, db)
	repo := NewEnvironmentRepository(db)
	_ = db.Close()

	err := repo.Create(context.Background(), newTestEnvironment("closed", categoryID))
	if err == nil || errors.Is(err, ErrEnvironmentNameInUse) {
		t.Fatalf("Create() on closed database error = %v, want a plain non-nil error", err)
	}
}

func TestEnvironmentRepository_GetByID_NotFound(t *testing.T) {
	repo := NewEnvironmentRepository(newTestDB(t))

	_, err := repo.GetByID(context.Background(), uuid.NewString())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByID() error = %v, want ErrNotFound", err)
	}
}

func TestEnvironmentRepository_GetByID_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewEnvironmentRepository(db)
	_ = db.Close()

	_, err := repo.GetByID(context.Background(), "irrelevant")
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByID() on closed database error = %v, want a non-nil, non-ErrNotFound error", err)
	}
}

func TestEnvironmentRepository_GetByName(t *testing.T) {
	db := newTestDB(t)
	categoryID := developmentCategoryID(t, db)
	repo := NewEnvironmentRepository(db)
	ctx := context.Background()

	e := newTestEnvironment("lookup-by-name", categoryID)
	if err := repo.Create(ctx, e); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	got, err := repo.GetByName(ctx, "lookup-by-name")
	if err != nil {
		t.Fatalf("GetByName() returned unexpected error: %v", err)
	}
	if got.ID != e.ID {
		t.Errorf("GetByName() ID = %q, want %q", got.ID, e.ID)
	}
}

func TestEnvironmentRepository_GetByName_NotFound(t *testing.T) {
	repo := NewEnvironmentRepository(newTestDB(t))

	_, err := repo.GetByName(context.Background(), "does-not-exist")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByName() error = %v, want ErrNotFound", err)
	}
}

func TestEnvironmentRepository_List(t *testing.T) {
	db := newTestDB(t)
	categoryID := developmentCategoryID(t, db)
	repo := NewEnvironmentRepository(db)
	ctx := context.Background()

	for _, name := range []string{"a-env", "b-env", "c-env"} {
		if err := repo.Create(ctx, newTestEnvironment(name, categoryID)); err != nil {
			t.Fatalf("Create(%q) returned unexpected error: %v", name, err)
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

func TestEnvironmentRepository_List_Empty(t *testing.T) {
	repo := NewEnvironmentRepository(newTestDB(t))

	page, total, err := repo.List(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("List() returned unexpected error: %v", err)
	}
	if total != 0 || len(page) != 0 {
		t.Errorf("List() = (%v, %d), want (empty, 0)", page, total)
	}
}

func TestEnvironmentRepository_List_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewEnvironmentRepository(db)
	_ = db.Close()

	if _, _, err := repo.List(context.Background(), 10, 0); err == nil {
		t.Fatal("List() on closed database expected an error, got nil")
	}
}

func TestEnvironmentRepository_Update(t *testing.T) {
	db := newTestDB(t)
	categoryID := developmentCategoryID(t, db)
	userRepo := NewUserRepository(db)
	repo := NewEnvironmentRepository(db)
	ctx := context.Background()

	actor := newTestUser("env-editor@example.com")
	if err := userRepo.Create(ctx, actor); err != nil {
		t.Fatalf("Create(actor) returned unexpected error: %v", err)
	}

	e := newTestEnvironment("update-me", categoryID)
	if err := repo.Create(ctx, e); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	newDesc := "updated description"
	e.Name = "updated-name"
	e.Description = &newDesc
	e.ModifiedByUserID = &actor.ID
	if err := repo.Update(ctx, e); err != nil {
		t.Fatalf("Update() returned unexpected error: %v", err)
	}

	got, err := repo.GetByID(ctx, e.ID)
	if err != nil {
		t.Fatalf("GetByID() returned unexpected error: %v", err)
	}
	if got.Name != "updated-name" || got.Description == nil || *got.Description != "updated description" {
		t.Errorf("GetByID() after Update() = %+v, want Name=updated-name Description=\"updated description\"", got)
	}
}

func TestEnvironmentRepository_Update_NotFound(t *testing.T) {
	db := newTestDB(t)
	categoryID := developmentCategoryID(t, db)
	repo := NewEnvironmentRepository(db)

	err := repo.Update(context.Background(), newTestEnvironment("ghost", categoryID))
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Update() error = %v, want ErrNotFound", err)
	}
}

func TestEnvironmentRepository_Update_DuplicateName(t *testing.T) {
	db := newTestDB(t)
	categoryID := developmentCategoryID(t, db)
	repo := NewEnvironmentRepository(db)
	ctx := context.Background()

	a := newTestEnvironment("first", categoryID)
	b := newTestEnvironment("second", categoryID)
	if err := repo.Create(ctx, a); err != nil {
		t.Fatalf("Create(a) returned unexpected error: %v", err)
	}
	if err := repo.Create(ctx, b); err != nil {
		t.Fatalf("Create(b) returned unexpected error: %v", err)
	}

	b.Name = a.Name
	err := repo.Update(ctx, b)
	if !errors.Is(err, ErrEnvironmentNameInUse) {
		t.Fatalf("Update() error = %v, want ErrEnvironmentNameInUse", err)
	}
}

func TestEnvironmentRepository_Update_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	categoryID := developmentCategoryID(t, db)
	repo := NewEnvironmentRepository(db)
	_ = db.Close()

	err := repo.Update(context.Background(), newTestEnvironment("closed-update", categoryID))
	if err == nil || errors.Is(err, ErrEnvironmentNameInUse) || errors.Is(err, ErrNotFound) {
		t.Fatalf("Update() on closed database error = %v, want a plain non-nil error", err)
	}
}

func TestEnvironmentRepository_Delete(t *testing.T) {
	db := newTestDB(t)
	categoryID := developmentCategoryID(t, db)
	repo := NewEnvironmentRepository(db)
	ctx := context.Background()

	e := newTestEnvironment("delete-me", categoryID)
	if err := repo.Create(ctx, e); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	if err := repo.Delete(ctx, e.ID); err != nil {
		t.Fatalf("Delete() returned unexpected error: %v", err)
	}

	_, err := repo.GetByID(ctx, e.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByID() after Delete() error = %v, want ErrNotFound", err)
	}
}

func TestEnvironmentRepository_Delete_NotFound(t *testing.T) {
	repo := NewEnvironmentRepository(newTestDB(t))

	err := repo.Delete(context.Background(), uuid.NewString())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Delete() error = %v, want ErrNotFound", err)
	}
}

func TestEnvironmentRepository_Delete_InUse(t *testing.T) {
	db := newTestDB(t)
	categoryID := developmentCategoryID(t, db)
	repo := NewEnvironmentRepository(db)
	ctx := context.Background()

	e := newTestEnvironment("in-use", categoryID)
	if err := repo.Create(ctx, e); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	// Reference the environment from service_key_environments (0004,
	// schema-only so far — no repository/service exists yet) via raw SQL,
	// to exercise the real FK-in-use branch rather than mocking it.
	if _, err := db.Exec("INSERT INTO service_keys (id, name, secret_hash) VALUES ('sk1', 'test key', 'hash')"); err != nil {
		t.Fatalf("seeding service key: %v", err)
	}
	if _, err := db.Exec("INSERT INTO service_key_environments (id, service_key_id, environment_id) VALUES ('ske1', 'sk1', ?)", e.ID); err != nil {
		t.Fatalf("seeding service_key_environments: %v", err)
	}

	err := repo.Delete(ctx, e.ID)
	if !errors.Is(err, ErrInUse) {
		t.Fatalf("Delete() error = %v, want ErrInUse", err)
	}
}

func TestEnvironmentRepository_Delete_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewEnvironmentRepository(db)
	_ = db.Close()

	err := repo.Delete(context.Background(), "irrelevant")
	if err == nil {
		t.Fatal("Delete() on closed database expected an error, got nil")
	}
}
