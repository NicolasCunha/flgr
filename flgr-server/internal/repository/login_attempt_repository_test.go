package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
)

func TestLoginAttemptRepository_GetByEmail_NotFound(t *testing.T) {
	repo := NewLoginAttemptRepository(newTestDB(t))

	_, err := repo.GetByEmail(context.Background(), "nobody@example.com")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByEmail() error = %v, want ErrNotFound", err)
	}
}

func TestLoginAttemptRepository_Upsert_InsertsThenUpdates(t *testing.T) {
	repo := NewLoginAttemptRepository(newTestDB(t))
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	a := &model.LoginAttempt{
		ID:            uuid.NewString(),
		Email:         "attempt@example.com",
		FailedCount:   1,
		FirstFailedAt: &now,
	}
	if err := repo.Upsert(ctx, a); err != nil {
		t.Fatalf("Upsert() (insert) returned unexpected error: %v", err)
	}

	got, err := repo.GetByEmail(ctx, "attempt@example.com")
	if err != nil {
		t.Fatalf("GetByEmail() returned unexpected error: %v", err)
	}
	if got.FailedCount != 1 {
		t.Errorf("FailedCount = %d, want 1", got.FailedCount)
	}
	if got.LockedUntil != nil {
		t.Errorf("LockedUntil = %v, want nil", got.LockedUntil)
	}

	locked := now.Add(15 * time.Minute)
	a.ID = got.ID
	a.FailedCount = 5
	a.LockedUntil = &locked
	if err := repo.Upsert(ctx, a); err != nil {
		t.Fatalf("Upsert() (update) returned unexpected error: %v", err)
	}

	got, err = repo.GetByEmail(ctx, "attempt@example.com")
	if err != nil {
		t.Fatalf("GetByEmail() (after update) returned unexpected error: %v", err)
	}
	if got.FailedCount != 5 {
		t.Errorf("FailedCount = %d, want 5", got.FailedCount)
	}
	if got.LockedUntil == nil || !got.LockedUntil.Equal(locked) {
		t.Errorf("LockedUntil = %v, want %v", got.LockedUntil, locked)
	}
}

func TestLoginAttemptRepository_Reset(t *testing.T) {
	repo := NewLoginAttemptRepository(newTestDB(t))
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	a := &model.LoginAttempt{
		ID:            uuid.NewString(),
		Email:         "reset@example.com",
		FailedCount:   3,
		FirstFailedAt: &now,
	}
	if err := repo.Upsert(ctx, a); err != nil {
		t.Fatalf("Upsert() returned unexpected error: %v", err)
	}

	if err := repo.Reset(ctx, "reset@example.com"); err != nil {
		t.Fatalf("Reset() returned unexpected error: %v", err)
	}

	_, err := repo.GetByEmail(ctx, "reset@example.com")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByEmail() after Reset() error = %v, want ErrNotFound", err)
	}
}

func TestLoginAttemptRepository_GetByEmail_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewLoginAttemptRepository(db)
	_ = db.Close()

	_, err := repo.GetByEmail(context.Background(), "irrelevant")
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByEmail() on closed database error = %v, want a non-nil, non-ErrNotFound error", err)
	}
}

func TestLoginAttemptRepository_Upsert_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewLoginAttemptRepository(db)
	_ = db.Close()

	err := repo.Upsert(context.Background(), &model.LoginAttempt{ID: uuid.NewString(), Email: "irrelevant@example.com"})
	if err == nil {
		t.Fatal("Upsert() on closed database expected an error, got nil")
	}
}

func TestLoginAttemptRepository_Reset_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewLoginAttemptRepository(db)
	_ = db.Close()

	err := repo.Reset(context.Background(), "irrelevant@example.com")
	if err == nil {
		t.Fatal("Reset() on closed database expected an error, got nil")
	}
}

func TestLoginAttemptRepository_Reset_NoOpWhenAbsent(t *testing.T) {
	repo := NewLoginAttemptRepository(newTestDB(t))

	if err := repo.Reset(context.Background(), "never-existed@example.com"); err != nil {
		t.Fatalf("Reset() returned unexpected error: %v", err)
	}
}
