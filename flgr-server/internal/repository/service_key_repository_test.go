package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
)

func newTestServiceKey(name string) *model.ServiceKey {
	now := time.Now().UTC().Truncate(time.Second)
	return &model.ServiceKey{
		ID:         uuid.NewString(),
		Name:       name,
		SecretHash: "sha256-hash",
		Status:     model.ServiceKeyStatusActive,
		CanRead:    true,
		CanWrite:   false,
		AuditFields: model.AuditFields{
			CreatedOn:  now,
			ModifiedOn: now,
		},
	}
}

func TestServiceKeyRepository_CreateAndGetByID(t *testing.T) {
	repo := NewServiceKeyRepository(newTestDB(t))
	ctx := context.Background()

	k := newTestServiceKey("checkout-service prod key")
	if err := repo.Create(ctx, k); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	got, err := repo.GetByID(ctx, k.ID)
	if err != nil {
		t.Fatalf("GetByID() returned unexpected error: %v", err)
	}
	if got.Name != "checkout-service prod key" || got.Status != model.ServiceKeyStatusActive || !got.CanRead || got.CanWrite {
		t.Errorf("GetByID() = %+v, want Name=checkout-service prod key Status=Active CanRead=true CanWrite=false", got)
	}
}

func TestServiceKeyRepository_Create_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewServiceKeyRepository(db)
	_ = db.Close()

	err := repo.Create(context.Background(), newTestServiceKey("closed"))
	if err == nil {
		t.Fatal("Create() on closed database expected an error, got nil")
	}
}

func TestServiceKeyRepository_GetByID_NotFound(t *testing.T) {
	repo := NewServiceKeyRepository(newTestDB(t))

	_, err := repo.GetByID(context.Background(), uuid.NewString())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByID() error = %v, want ErrNotFound", err)
	}
}

func TestServiceKeyRepository_GetByID_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewServiceKeyRepository(db)
	_ = db.Close()

	_, err := repo.GetByID(context.Background(), "irrelevant")
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByID() on closed database error = %v, want a non-nil, non-ErrNotFound error", err)
	}
}

func TestServiceKeyRepository_List(t *testing.T) {
	repo := NewServiceKeyRepository(newTestDB(t))
	ctx := context.Background()

	for _, name := range []string{"a-key", "b-key", "c-key"} {
		if err := repo.Create(ctx, newTestServiceKey(name)); err != nil {
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

func TestServiceKeyRepository_List_Empty(t *testing.T) {
	repo := NewServiceKeyRepository(newTestDB(t))

	page, total, err := repo.List(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("List() returned unexpected error: %v", err)
	}
	if total != 0 || len(page) != 0 {
		t.Errorf("List() = (%v, %d), want (empty, 0)", page, total)
	}
}

func TestServiceKeyRepository_List_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewServiceKeyRepository(db)
	_ = db.Close()

	if _, _, err := repo.List(context.Background(), 10, 0); err == nil {
		t.Fatal("List() on closed database expected an error, got nil")
	}
}

func TestServiceKeyRepository_Update(t *testing.T) {
	db := newTestDB(t)
	userRepo := NewUserRepository(db)
	repo := NewServiceKeyRepository(db)
	ctx := context.Background()

	actor := newTestUser("key-editor@example.com")
	if err := userRepo.Create(ctx, actor); err != nil {
		t.Fatalf("Create(actor) returned unexpected error: %v", err)
	}

	k := newTestServiceKey("update-me")
	if err := repo.Create(ctx, k); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	k.Name = "updated-name"
	k.Status = model.ServiceKeyStatusInactive
	k.CanRead = false
	k.CanWrite = true
	k.ModifiedByUserID = &actor.ID
	if err := repo.Update(ctx, k); err != nil {
		t.Fatalf("Update() returned unexpected error: %v", err)
	}

	got, err := repo.GetByID(ctx, k.ID)
	if err != nil {
		t.Fatalf("GetByID() returned unexpected error: %v", err)
	}
	if got.Name != "updated-name" || got.Status != model.ServiceKeyStatusInactive || got.CanRead || !got.CanWrite {
		t.Errorf("GetByID() after Update() = %+v, want Name=updated-name Status=Inactive CanRead=false CanWrite=true", got)
	}
}

func TestServiceKeyRepository_Update_NotFound(t *testing.T) {
	repo := NewServiceKeyRepository(newTestDB(t))

	err := repo.Update(context.Background(), newTestServiceKey("ghost"))
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Update() error = %v, want ErrNotFound", err)
	}
}

func TestServiceKeyRepository_Update_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewServiceKeyRepository(db)
	_ = db.Close()

	err := repo.Update(context.Background(), newTestServiceKey("closed-update"))
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("Update() on closed database error = %v, want a plain non-nil error", err)
	}
}
