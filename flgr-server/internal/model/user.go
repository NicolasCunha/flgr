// Package model holds domain types shared across the service, repository,
// and handler layers, per flgr-server's layering rule (see backend.md).
package model

import "time"

const (
	UserStatusActive   = "Active"
	UserStatusInactive = "Inactive"
)

// User is a person who administers flgr through the web UI, per
// docs/business/requirements/0002-user-management.md.
type User struct {
	ID           string
	FirstName    string
	LastName     string
	Email        string
	PasswordHash string
	Status       string
	AuditFields
}

// IsActive reports whether the user is allowed to authenticate, per
// 0002's acceptance criteria.
func (u User) IsActive() bool {
	return u.Status == UserStatusActive
}

// Session is a server-side login session for a User, per
// docs/architecture/adr/0006-authentication-and-session-strategy.md.
// It intentionally does not embed AuditFields — its own IssuedAt already
// covers "created by/when" (always the session's own UserID), and a
// session is deleted on logout/expiry rather than modified.
type Session struct {
	ID         string
	TokenHash  string
	UserID     string
	IssuedAt   time.Time
	ExpiresAt  time.Time
	LastSeenAt time.Time
}

// LoginAttempt tracks failed User login attempts per email, for the
// brute-force lockout in ADR-0006. Tracked by email rather than by an
// authenticated actor, so it does not embed AuditFields either.
type LoginAttempt struct {
	ID            string
	Email         string
	FailedCount   int
	FirstFailedAt *time.Time
	LockedUntil   *time.Time
}
