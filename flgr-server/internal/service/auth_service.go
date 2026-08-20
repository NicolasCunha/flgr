package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/repository"
)

const (
	// sessionInactivityWindow is the sliding session lifetime, per
	// docs/architecture/adr/0006-authentication-and-session-strategy.md.
	sessionInactivityWindow = 30 * time.Minute

	// maxFailedAttempts, attemptWindow, and lockDuration implement the
	// login attempt protection in ADR-0006: 5 consecutive failures for
	// the same email within 15 minutes locks that email out for 15 minutes.
	maxFailedAttempts = 5
	attemptWindow     = 15 * time.Minute
	lockDuration      = 15 * time.Minute
)

// AuthService implements User authentication, per
// docs/architecture/adr/0006-authentication-and-session-strategy.md.
type AuthService struct {
	users    repository.UserRepository
	sessions repository.SessionRepository
	attempts repository.LoginAttemptRepository
}

// NewAuthService returns an AuthService backed by the given repositories.
func NewAuthService(users repository.UserRepository, sessions repository.SessionRepository, attempts repository.LoginAttemptRepository) *AuthService {
	return &AuthService{users: users, sessions: sessions, attempts: attempts}
}

// Login verifies email/password against the stored user, enforces the
// brute-force lockout, and on success creates a session. It returns the
// opaque session token to set as a cookie — only its hash is persisted.
func (s *AuthService) Login(ctx context.Context, email, password string) (*model.User, string, error) {
	email = normalizeEmail(email)
	nowTime := now()

	attempt, err := s.attempts.GetByEmail(ctx, email)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, "", fmt.Errorf("checking login attempts: %w", err)
	}
	if err == nil && attempt.LockedUntil != nil && attempt.LockedUntil.After(nowTime) {
		return nil, "", ErrAccountLocked
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, "", fmt.Errorf("fetching user: %w", err)
	}

	if err != nil || !comparePassword(user.PasswordHash, password) {
		if recErr := s.recordFailedAttempt(ctx, email, attempt, nowTime); recErr != nil {
			return nil, "", recErr
		}
		return nil, "", ErrInvalidCredentials
	}

	if !user.IsActive() {
		return nil, "", ErrUserInactive
	}

	if err := s.attempts.Reset(ctx, email); err != nil {
		return nil, "", fmt.Errorf("resetting login attempts: %w", err)
	}

	token, err := generateToken()
	if err != nil {
		return nil, "", err
	}

	session := &model.Session{
		ID:         newID(),
		TokenHash:  hashToken(token),
		UserID:     user.ID,
		IssuedAt:   nowTime,
		ExpiresAt:  nowTime.Add(sessionInactivityWindow),
		LastSeenAt: nowTime,
	}
	if err := s.sessions.Create(ctx, session); err != nil {
		return nil, "", fmt.Errorf("creating session: %w", err)
	}

	return user, token, nil
}

// recordFailedAttempt increments (or starts) the failed-attempt counter
// for email, per ADR-0006's 15-minute window, locking the email out once
// maxFailedAttempts is reached within that window.
func (s *AuthService) recordFailedAttempt(ctx context.Context, email string, existing *model.LoginAttempt, nowTime time.Time) error {
	a := &model.LoginAttempt{Email: email}

	withinWindow := existing != nil && existing.FirstFailedAt != nil && nowTime.Sub(*existing.FirstFailedAt) < attemptWindow
	switch {
	case withinWindow:
		a.ID = existing.ID
		a.FailedCount = existing.FailedCount + 1
		a.FirstFailedAt = existing.FirstFailedAt
	case existing != nil:
		a.ID = existing.ID
		a.FailedCount = 1
		a.FirstFailedAt = &nowTime
	default:
		a.ID = newID()
		a.FailedCount = 1
		a.FirstFailedAt = &nowTime
	}

	if a.FailedCount >= maxFailedAttempts {
		locked := nowTime.Add(lockDuration)
		a.LockedUntil = &locked
	}

	if err := s.attempts.Upsert(ctx, a); err != nil {
		return fmt.Errorf("recording failed login attempt: %w", err)
	}
	return nil
}

// Logout deletes the session identified by token. It's idempotent: an
// already-invalid token is not an error.
func (s *AuthService) Logout(ctx context.Context, token string) error {
	session, err := s.sessions.GetByTokenHash(ctx, hashToken(token))
	if errors.Is(err, repository.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("fetching session: %w", err)
	}

	if err := s.sessions.Delete(ctx, session.ID); err != nil {
		return fmt.Errorf("deleting session: %w", err)
	}
	return nil
}

// ValidateSession resolves token to its User, per ADR-0006: the session
// must be unexpired and the user must still be Active, both read fresh
// from the database on every call so deactivation takes effect
// immediately. A valid call slides the session's expiration forward.
func (s *AuthService) ValidateSession(ctx context.Context, token string) (*model.User, error) {
	tokenHash := hashToken(token)
	session, err := s.sessions.GetByTokenHash(ctx, tokenHash)
	if errors.Is(err, repository.ErrNotFound) {
		// tokenHash itself isn't the secret (it's already a one-way hash of
		// it, same as what's stored), but log only a short prefix anyway —
		// this line is diagnostic-only, meant to be pasted into a bug
		// report, and shouldn't need the full value for that.
		log.Printf("auth: ValidateSession rejected — no session for token hash %s…", tokenHash[:8])
		return nil, ErrSessionInvalid
	}
	if err != nil {
		return nil, fmt.Errorf("fetching session: %w", err)
	}

	nowTime := now()
	if session.ExpiresAt.Before(nowTime) {
		// Logged with full RFC3339Nano precision (including zone offset) on
		// purpose — if sessions are expiring sooner than the 30-minute
		// sliding window, comparing these two timestamps is the fastest way
		// to tell a real idle timeout from a bug (e.g. a UTC/local mismatch
		// somewhere in the read/write round trip through SQLite).
		log.Printf("auth: ValidateSession rejected — session %s expired (expiresAt=%s, now=%s, userID=%s)",
			session.ID, session.ExpiresAt.Format(time.RFC3339Nano), nowTime.Format(time.RFC3339Nano), session.UserID)
		return nil, ErrSessionInvalid
	}

	user, err := s.users.GetByID(ctx, session.UserID)
	if errors.Is(err, repository.ErrNotFound) {
		log.Printf("auth: ValidateSession rejected — session %s references missing user %s", session.ID, session.UserID)
		return nil, ErrSessionInvalid
	}
	if err != nil {
		return nil, fmt.Errorf("fetching session user: %w", err)
	}
	if !user.IsActive() {
		log.Printf("auth: ValidateSession rejected — session %s user %s is not active (status=%s)", session.ID, user.ID, user.Status)
		return nil, ErrSessionInvalid
	}

	newExpiry := nowTime.Add(sessionInactivityWindow)
	if err := s.sessions.Touch(ctx, session.ID, nowTime, newExpiry); err != nil {
		return nil, fmt.Errorf("touching session: %w", err)
	}

	return user, nil
}
