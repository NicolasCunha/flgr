package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
)

// LoginAttemptRepository is the data access seam for the login_attempts
// table, backing the brute-force lockout in
// docs/architecture/adr/0006-authentication-and-session-strategy.md.
type LoginAttemptRepository interface {
	GetByEmail(ctx context.Context, email string) (*model.LoginAttempt, error)
	Upsert(ctx context.Context, a *model.LoginAttempt) error
	Reset(ctx context.Context, email string) error
}

type loginAttemptRepository struct {
	db *sql.DB
}

// NewLoginAttemptRepository returns a LoginAttemptRepository backed by db.
func NewLoginAttemptRepository(db *sql.DB) LoginAttemptRepository {
	return &loginAttemptRepository{db: db}
}

func (r *loginAttemptRepository) GetByEmail(ctx context.Context, email string) (*model.LoginAttempt, error) {
	var a model.LoginAttempt
	var firstFailedAt, lockedUntil sql.NullTime

	err := r.db.QueryRowContext(ctx, `
		SELECT id, email, failed_count, first_failed_at, locked_until
		FROM login_attempts WHERE email = ?`, email,
	).Scan(&a.ID, &a.Email, &a.FailedCount, &firstFailedAt, &lockedUntil)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("fetching login attempt: %w", err)
	}

	if firstFailedAt.Valid {
		a.FirstFailedAt = &firstFailedAt.Time
	}
	if lockedUntil.Valid {
		a.LockedUntil = &lockedUntil.Time
	}
	return &a, nil
}

func (r *loginAttemptRepository) Upsert(ctx context.Context, a *model.LoginAttempt) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO login_attempts (id, email, failed_count, first_failed_at, locked_until)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(email) DO UPDATE SET
			failed_count = excluded.failed_count,
			first_failed_at = excluded.first_failed_at,
			locked_until = excluded.locked_until`,
		a.ID, a.Email, a.FailedCount, a.FirstFailedAt, a.LockedUntil,
	)
	if err != nil {
		return fmt.Errorf("upserting login attempt: %w", err)
	}
	return nil
}

func (r *loginAttemptRepository) Reset(ctx context.Context, email string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM login_attempts WHERE email = ?", email)
	if err != nil {
		return fmt.Errorf("resetting login attempt: %w", err)
	}
	return nil
}
