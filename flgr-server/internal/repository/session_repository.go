package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
)

// SessionRepository is the data access seam for the sessions table, per
// docs/architecture/adr/0006-authentication-and-session-strategy.md.
type SessionRepository interface {
	Create(ctx context.Context, s *model.Session) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*model.Session, error)
	Touch(ctx context.Context, id string, lastSeenAt, expiresAt time.Time) error
	Delete(ctx context.Context, id string) error
}

type sessionRepository struct {
	db *sql.DB
}

// NewSessionRepository returns a SessionRepository backed by db.
func NewSessionRepository(db *sql.DB) SessionRepository {
	return &sessionRepository{db: db}
}

func (r *sessionRepository) Create(ctx context.Context, s *model.Session) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO sessions (id, token_hash, user_id, issued_at, expires_at, last_seen_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		s.ID, s.TokenHash, s.UserID, s.IssuedAt, s.ExpiresAt, s.LastSeenAt,
	)
	if err != nil {
		return fmt.Errorf("creating session: %w", err)
	}
	return nil
}

func (r *sessionRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*model.Session, error) {
	var s model.Session
	err := r.db.QueryRowContext(ctx, `
		SELECT id, token_hash, user_id, issued_at, expires_at, last_seen_at
		FROM sessions WHERE token_hash = ?`, tokenHash,
	).Scan(&s.ID, &s.TokenHash, &s.UserID, &s.IssuedAt, &s.ExpiresAt, &s.LastSeenAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("fetching session: %w", err)
	}
	return &s, nil
}

func (r *sessionRepository) Touch(ctx context.Context, id string, lastSeenAt, expiresAt time.Time) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE sessions SET last_seen_at = ?, expires_at = ? WHERE id = ?",
		lastSeenAt, expiresAt, id,
	)
	if err != nil {
		return fmt.Errorf("touching session: %w", err)
	}
	return nil
}

func (r *sessionRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM sessions WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting session: %w", err)
	}
	return nil
}
