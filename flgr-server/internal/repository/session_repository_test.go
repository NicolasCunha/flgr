package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
)

func newTestSession(userID, tokenHash string) *model.Session {
	now := time.Now().UTC().Truncate(time.Second)
	return &model.Session{
		ID:         uuid.NewString(),
		TokenHash:  tokenHash,
		UserID:     userID,
		IssuedAt:   now,
		ExpiresAt:  now.Add(30 * time.Minute),
		LastSeenAt: now,
	}
}

func TestSessionRepository_CreateAndGetByTokenHash(t *testing.T) {
	db := newTestDB(t)
	userRepo := NewUserRepository(db)
	sessionRepo := NewSessionRepository(db)
	ctx := context.Background()

	u := newTestUser("session-user@example.com")
	if err := userRepo.Create(ctx, u); err != nil {
		t.Fatalf("Create(user) returned unexpected error: %v", err)
	}

	s := newTestSession(u.ID, "hash-1")
	if err := sessionRepo.Create(ctx, s); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	got, err := sessionRepo.GetByTokenHash(ctx, "hash-1")
	if err != nil {
		t.Fatalf("GetByTokenHash() returned unexpected error: %v", err)
	}
	if got.UserID != u.ID {
		t.Errorf("UserID = %q, want %q", got.UserID, u.ID)
	}
}

func TestSessionRepository_GetByTokenHash_NotFound(t *testing.T) {
	sessionRepo := NewSessionRepository(newTestDB(t))

	_, err := sessionRepo.GetByTokenHash(context.Background(), "does-not-exist")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByTokenHash() error = %v, want ErrNotFound", err)
	}
}

func TestSessionRepository_Touch(t *testing.T) {
	db := newTestDB(t)
	userRepo := NewUserRepository(db)
	sessionRepo := NewSessionRepository(db)
	ctx := context.Background()

	u := newTestUser("touch-user@example.com")
	if err := userRepo.Create(ctx, u); err != nil {
		t.Fatalf("Create(user) returned unexpected error: %v", err)
	}

	s := newTestSession(u.ID, "hash-touch")
	if err := sessionRepo.Create(ctx, s); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	newExpiry := time.Now().UTC().Truncate(time.Second).Add(60 * time.Minute)
	if err := sessionRepo.Touch(ctx, s.ID, newExpiry, newExpiry); err != nil {
		t.Fatalf("Touch() returned unexpected error: %v", err)
	}

	got, err := sessionRepo.GetByTokenHash(ctx, "hash-touch")
	if err != nil {
		t.Fatalf("GetByTokenHash() returned unexpected error: %v", err)
	}
	if !got.ExpiresAt.Equal(newExpiry) {
		t.Errorf("ExpiresAt = %v, want %v", got.ExpiresAt, newExpiry)
	}
}

func TestSessionRepository_Create_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewSessionRepository(db)
	_ = db.Close()

	err := repo.Create(context.Background(), newTestSession(uuid.NewString(), "hash-closed"))
	if err == nil {
		t.Fatal("Create() on closed database expected an error, got nil")
	}
}

func TestSessionRepository_GetByTokenHash_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewSessionRepository(db)
	_ = db.Close()

	_, err := repo.GetByTokenHash(context.Background(), "irrelevant")
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByTokenHash() on closed database error = %v, want a non-nil, non-ErrNotFound error", err)
	}
}

func TestSessionRepository_Touch_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewSessionRepository(db)
	_ = db.Close()

	err := repo.Touch(context.Background(), "irrelevant", time.Now(), time.Now())
	if err == nil {
		t.Fatal("Touch() on closed database expected an error, got nil")
	}
}

func TestSessionRepository_Delete_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewSessionRepository(db)
	_ = db.Close()

	err := repo.Delete(context.Background(), "irrelevant")
	if err == nil {
		t.Fatal("Delete() on closed database expected an error, got nil")
	}
}

func TestSessionRepository_Delete(t *testing.T) {
	db := newTestDB(t)
	userRepo := NewUserRepository(db)
	sessionRepo := NewSessionRepository(db)
	ctx := context.Background()

	u := newTestUser("delete-user@example.com")
	if err := userRepo.Create(ctx, u); err != nil {
		t.Fatalf("Create(user) returned unexpected error: %v", err)
	}

	s := newTestSession(u.ID, "hash-delete")
	if err := sessionRepo.Create(ctx, s); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	if err := sessionRepo.Delete(ctx, s.ID); err != nil {
		t.Fatalf("Delete() returned unexpected error: %v", err)
	}

	_, err := sessionRepo.GetByTokenHash(ctx, "hash-delete")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByTokenHash() after Delete() error = %v, want ErrNotFound", err)
	}
}
