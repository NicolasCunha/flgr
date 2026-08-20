package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
)

func newTestProfile(name string) *model.Profile {
	now := time.Now().UTC().Truncate(time.Second)
	desc := "a test profile"
	return &model.Profile{
		ID:          uuid.NewString(),
		Name:        name,
		Description: &desc,
		IsSystem:    false,
		AuditFields: model.AuditFields{CreatedOn: now, ModifiedOn: now},
	}
}

func TestProfileRepository_CreateAndGetByID(t *testing.T) {
	repo := NewProfileRepository(newTestDB(t))
	ctx := context.Background()

	p := newTestProfile("app-read")
	if err := repo.Create(ctx, p); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	got, err := repo.GetByID(ctx, p.ID)
	if err != nil {
		t.Fatalf("GetByID() returned unexpected error: %v", err)
	}
	if got.Name != "app-read" || got.Description == nil || *got.Description != "a test profile" {
		t.Errorf("GetByID() = %+v, want Name=app-read Description=\"a test profile\"", got)
	}
	if got.IsSystem {
		t.Error("IsSystem = true, want false for a created (non-system) profile")
	}
}

func TestProfileRepository_Create_DuplicateName(t *testing.T) {
	repo := NewProfileRepository(newTestDB(t))
	ctx := context.Background()

	if err := repo.Create(ctx, newTestProfile("dup")); err != nil {
		t.Fatalf("Create() (first) returned unexpected error: %v", err)
	}
	err := repo.Create(ctx, newTestProfile("dup"))
	if !errors.Is(err, ErrProfileNameInUse) {
		t.Fatalf("Create() (duplicate) error = %v, want ErrProfileNameInUse", err)
	}
}

func TestProfileRepository_Create_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewProfileRepository(db)
	_ = db.Close()

	err := repo.Create(context.Background(), newTestProfile("closed"))
	if err == nil || errors.Is(err, ErrProfileNameInUse) {
		t.Fatalf("Create() on closed database error = %v, want a plain non-nil error", err)
	}
}

func TestProfileRepository_GetByID_NotFound(t *testing.T) {
	repo := NewProfileRepository(newTestDB(t))

	_, err := repo.GetByID(context.Background(), uuid.NewString())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByID() error = %v, want ErrNotFound", err)
	}
}

func TestProfileRepository_GetByID_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewProfileRepository(db)
	_ = db.Close()

	_, err := repo.GetByID(context.Background(), "irrelevant")
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByID() on closed database error = %v, want a non-nil, non-ErrNotFound error", err)
	}
}

func TestProfileRepository_GetByName(t *testing.T) {
	repo := NewProfileRepository(newTestDB(t))
	ctx := context.Background()

	p := newTestProfile("lookup-by-name")
	if err := repo.Create(ctx, p); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	got, err := repo.GetByName(ctx, "lookup-by-name")
	if err != nil {
		t.Fatalf("GetByName() returned unexpected error: %v", err)
	}
	if got.ID != p.ID {
		t.Errorf("GetByName() ID = %q, want %q", got.ID, p.ID)
	}
}

func TestProfileRepository_GetByName_NotFound(t *testing.T) {
	repo := NewProfileRepository(newTestDB(t))

	_, err := repo.GetByName(context.Background(), "does-not-exist")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByName() error = %v, want ErrNotFound", err)
	}
}

func TestProfileRepository_List(t *testing.T) {
	repo := NewProfileRepository(newTestDB(t))
	ctx := context.Background()

	for _, name := range []string{"a-profile", "b-profile", "c-profile"} {
		if err := repo.Create(ctx, newTestProfile(name)); err != nil {
			t.Fatalf("Create(%q) returned unexpected error: %v", name, err)
		}
	}

	// +1 for the seeded Administrador profile.
	page, total, err := repo.List(ctx, 2, 0)
	if err != nil {
		t.Fatalf("List() returned unexpected error: %v", err)
	}
	if total != 4 {
		t.Errorf("total = %d, want 4", total)
	}
	if len(page) != 2 {
		t.Errorf("len(page) = %d, want 2", len(page))
	}
}

func TestProfileRepository_List_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewProfileRepository(db)
	_ = db.Close()

	if _, _, err := repo.List(context.Background(), 10, 0); err == nil {
		t.Fatal("List() on closed database expected an error, got nil")
	}
}

func TestProfileRepository_ListAll(t *testing.T) {
	repo := NewProfileRepository(newTestDB(t))
	ctx := context.Background()

	if err := repo.Create(ctx, newTestProfile("only-extra")); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	all, err := repo.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll() returned unexpected error: %v", err)
	}
	// The seeded Administrador profile + the one just created.
	if len(all) != 2 {
		t.Fatalf("len(all) = %d, want 2", len(all))
	}
}

func TestProfileRepository_ListAll_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewProfileRepository(db)
	_ = db.Close()

	if _, err := repo.ListAll(context.Background()); err == nil {
		t.Fatal("ListAll() on closed database expected an error, got nil")
	}
}

func TestProfileRepository_Update(t *testing.T) {
	db := newTestDB(t)
	repo := NewProfileRepository(db)
	userRepo := NewUserRepository(db)
	ctx := context.Background()

	actor := newTestUser("profile-editor@example.com")
	if err := userRepo.Create(ctx, actor); err != nil {
		t.Fatalf("Create(actor) returned unexpected error: %v", err)
	}

	p := newTestProfile("update-me")
	if err := repo.Create(ctx, p); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	newDesc := "updated description"
	p.Name = "updated-name"
	p.Description = &newDesc
	p.ModifiedByUserID = &actor.ID
	if err := repo.Update(ctx, p); err != nil {
		t.Fatalf("Update() returned unexpected error: %v", err)
	}

	got, err := repo.GetByID(ctx, p.ID)
	if err != nil {
		t.Fatalf("GetByID() returned unexpected error: %v", err)
	}
	if got.Name != "updated-name" || got.Description == nil || *got.Description != "updated description" {
		t.Errorf("GetByID() after Update() = %+v, want Name=updated-name Description=\"updated description\"", got)
	}
}

func TestProfileRepository_Update_NotFound(t *testing.T) {
	repo := NewProfileRepository(newTestDB(t))

	err := repo.Update(context.Background(), newTestProfile("ghost"))
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Update() error = %v, want ErrNotFound", err)
	}
}

func TestProfileRepository_Update_DuplicateName(t *testing.T) {
	repo := NewProfileRepository(newTestDB(t))
	ctx := context.Background()

	a := newTestProfile("first")
	b := newTestProfile("second")
	if err := repo.Create(ctx, a); err != nil {
		t.Fatalf("Create(a) returned unexpected error: %v", err)
	}
	if err := repo.Create(ctx, b); err != nil {
		t.Fatalf("Create(b) returned unexpected error: %v", err)
	}

	b.Name = a.Name
	err := repo.Update(ctx, b)
	if !errors.Is(err, ErrProfileNameInUse) {
		t.Fatalf("Update() error = %v, want ErrProfileNameInUse", err)
	}
}

func TestProfileRepository_Update_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewProfileRepository(db)
	_ = db.Close()

	err := repo.Update(context.Background(), newTestProfile("closed-update"))
	if err == nil || errors.Is(err, ErrProfileNameInUse) || errors.Is(err, ErrNotFound) {
		t.Fatalf("Update() on closed database error = %v, want a plain non-nil error", err)
	}
}

func TestProfileRepository_Delete(t *testing.T) {
	repo := NewProfileRepository(newTestDB(t))
	ctx := context.Background()

	p := newTestProfile("delete-me")
	if err := repo.Create(ctx, p); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	if err := repo.Delete(ctx, p.ID); err != nil {
		t.Fatalf("Delete() returned unexpected error: %v", err)
	}

	_, err := repo.GetByID(ctx, p.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByID() after Delete() error = %v, want ErrNotFound", err)
	}
}

func TestProfileRepository_Delete_NotFound(t *testing.T) {
	repo := NewProfileRepository(newTestDB(t))

	err := repo.Delete(context.Background(), uuid.NewString())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Delete() error = %v, want ErrNotFound", err)
	}
}

func TestProfileRepository_Delete_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewProfileRepository(db)
	_ = db.Close()

	err := repo.Delete(context.Background(), "irrelevant")
	if err == nil {
		t.Fatal("Delete() on closed database expected an error, got nil")
	}
}
