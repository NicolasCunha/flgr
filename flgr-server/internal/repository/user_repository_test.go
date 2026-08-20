package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
)

func newTestUser(email string) *model.User {
	now := time.Now().UTC().Truncate(time.Second)
	return &model.User{
		ID:           uuid.NewString(),
		FirstName:    "Ada",
		LastName:     "Lovelace",
		Email:        email,
		PasswordHash: "bcrypt-hash",
		Status:       model.UserStatusActive,
		AuditFields: model.AuditFields{
			CreatedOn:  now,
			ModifiedOn: now,
		},
	}
}

func TestUserRepository_CreateAndGetByID(t *testing.T) {
	repo := NewUserRepository(newTestDB(t))
	ctx := context.Background()

	u := newTestUser("ada@example.com")
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	got, err := repo.GetByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("GetByID() returned unexpected error: %v", err)
	}
	if got.Email != u.Email || got.FirstName != u.FirstName {
		t.Errorf("GetByID() = %+v, want email/first name matching %+v", got, u)
	}
}

func TestUserRepository_Create_DuplicateEmail(t *testing.T) {
	repo := NewUserRepository(newTestDB(t))
	ctx := context.Background()

	if err := repo.Create(ctx, newTestUser("dup@example.com")); err != nil {
		t.Fatalf("Create() (first) returned unexpected error: %v", err)
	}

	err := repo.Create(ctx, newTestUser("dup@example.com"))
	if !errors.Is(err, ErrEmailInUse) {
		t.Fatalf("Create() (duplicate) error = %v, want ErrEmailInUse", err)
	}
}

func TestUserRepository_Create_WithActor(t *testing.T) {
	repo := NewUserRepository(newTestDB(t))
	ctx := context.Background()

	creator := newTestUser("creator@example.com")
	if err := repo.Create(ctx, creator); err != nil {
		t.Fatalf("Create() (creator) returned unexpected error: %v", err)
	}

	created := newTestUser("created@example.com")
	created.CreatedByUserID = &creator.ID
	created.ModifiedByUserID = &creator.ID
	if err := repo.Create(ctx, created); err != nil {
		t.Fatalf("Create() (with actor) returned unexpected error: %v", err)
	}

	got, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID() returned unexpected error: %v", err)
	}
	if got.CreatedByUserID == nil || *got.CreatedByUserID != creator.ID {
		t.Errorf("CreatedByUserID = %v, want %q", got.CreatedByUserID, creator.ID)
	}
	if got.CreatedByServiceKeyID != nil {
		t.Errorf("CreatedByServiceKeyID = %v, want nil", got.CreatedByServiceKeyID)
	}
}

func TestUserRepository_GetByID_NotFound(t *testing.T) {
	repo := NewUserRepository(newTestDB(t))

	_, err := repo.GetByID(context.Background(), uuid.NewString())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByID() error = %v, want ErrNotFound", err)
	}
}

func TestUserRepository_GetByEmail(t *testing.T) {
	repo := NewUserRepository(newTestDB(t))
	ctx := context.Background()

	u := newTestUser("lookup@example.com")
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	got, err := repo.GetByEmail(ctx, "lookup@example.com")
	if err != nil {
		t.Fatalf("GetByEmail() returned unexpected error: %v", err)
	}
	if got.ID != u.ID {
		t.Errorf("GetByEmail() ID = %q, want %q", got.ID, u.ID)
	}
}

func TestUserRepository_GetByEmail_NotFound(t *testing.T) {
	repo := NewUserRepository(newTestDB(t))

	_, err := repo.GetByEmail(context.Background(), "nobody@example.com")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByEmail() error = %v, want ErrNotFound", err)
	}
}

func TestUserRepository_List(t *testing.T) {
	repo := NewUserRepository(newTestDB(t))
	ctx := context.Background()

	for _, email := range []string{"a@example.com", "b@example.com", "c@example.com"} {
		if err := repo.Create(ctx, newTestUser(email)); err != nil {
			t.Fatalf("Create(%q) returned unexpected error: %v", email, err)
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

	rest, total, err := repo.List(ctx, 2, 2)
	if err != nil {
		t.Fatalf("List() (second page) returned unexpected error: %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if len(rest) != 1 {
		t.Errorf("len(rest) = %d, want 1", len(rest))
	}
}

func TestUserRepository_List_Empty(t *testing.T) {
	repo := NewUserRepository(newTestDB(t))

	page, total, err := repo.List(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("List() returned unexpected error: %v", err)
	}
	if total != 0 || len(page) != 0 {
		t.Errorf("List() = (%v, %d), want (empty, 0)", page, total)
	}
}

func TestUserRepository_Update(t *testing.T) {
	repo := NewUserRepository(newTestDB(t))
	ctx := context.Background()

	u := newTestUser("update@example.com")
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	u.FirstName = "Grace"
	u.Status = model.UserStatusInactive
	u.ModifiedByUserID = &u.ID
	if err := repo.Update(ctx, u); err != nil {
		t.Fatalf("Update() returned unexpected error: %v", err)
	}

	got, err := repo.GetByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("GetByID() returned unexpected error: %v", err)
	}
	if got.FirstName != "Grace" || got.Status != model.UserStatusInactive {
		t.Errorf("GetByID() after Update() = %+v, want FirstName=Grace Status=Inactive", got)
	}
	if got.ModifiedByUserID == nil || *got.ModifiedByUserID != u.ID {
		t.Errorf("ModifiedByUserID = %v, want %q", got.ModifiedByUserID, u.ID)
	}
}

func TestUserRepository_Update_NotFound(t *testing.T) {
	repo := NewUserRepository(newTestDB(t))

	u := newTestUser("ghost@example.com")
	err := repo.Update(context.Background(), u)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Update() error = %v, want ErrNotFound", err)
	}
}

func TestUserRepository_Create_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewUserRepository(db)
	_ = db.Close()

	err := repo.Create(context.Background(), newTestUser("closed@example.com"))
	if err == nil || errors.Is(err, ErrEmailInUse) {
		t.Fatalf("Create() on closed database error = %v, want a non-nil, non-ErrEmailInUse error", err)
	}
}

func TestUserRepository_GetByID_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewUserRepository(db)
	_ = db.Close()

	_, err := repo.GetByID(context.Background(), "irrelevant")
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByID() on closed database error = %v, want a non-nil, non-ErrNotFound error", err)
	}
}

func TestUserRepository_List_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewUserRepository(db)
	_ = db.Close()

	_, _, err := repo.List(context.Background(), 10, 0)
	if err == nil {
		t.Fatal("List() on closed database expected an error, got nil")
	}
}

func TestUserRepository_Update_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewUserRepository(db)
	_ = db.Close()

	err := repo.Update(context.Background(), newTestUser("closed-update@example.com"))
	if err == nil || errors.Is(err, ErrEmailInUse) || errors.Is(err, ErrNotFound) {
		t.Fatalf("Update() on closed database error = %v, want a plain non-nil error", err)
	}
}

func TestUserRepository_Update_DuplicateEmail(t *testing.T) {
	repo := NewUserRepository(newTestDB(t))
	ctx := context.Background()

	a := newTestUser("first@example.com")
	b := newTestUser("second@example.com")
	if err := repo.Create(ctx, a); err != nil {
		t.Fatalf("Create(a) returned unexpected error: %v", err)
	}
	if err := repo.Create(ctx, b); err != nil {
		t.Fatalf("Create(b) returned unexpected error: %v", err)
	}

	b.Email = a.Email
	err := repo.Update(ctx, b)
	if !errors.Is(err, ErrEmailInUse) {
		t.Fatalf("Update() error = %v, want ErrEmailInUse", err)
	}
}
