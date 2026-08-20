// Package repository is the data access layer: SQL against SQLite behind
// interfaces, so the service layer can be tested without a real database
// (see backend.md's Layering rule).
package repository

import (
	"errors"
	"strings"
)

// ErrNotFound is returned by a Get* method when no row matches. Callers
// translate it into a domain-specific error (e.g. service.ErrUserNotFound).
var ErrNotFound = errors.New("repository: not found")

// ErrEmailInUse is returned by UserRepository.Create/Update when the
// email column's unique index rejects the write. It backstops the
// service layer's own pre-check (see internal/service/user_service.go)
// against the race between the check and the write.
var ErrEmailInUse = errors.New("repository: email already in use")

// ErrProfileNameInUse is returned by ProfileRepository.Create/Update when
// the name column's unique index rejects the write — the same
// pre-check-plus-backstop pattern as ErrEmailInUse.
var ErrProfileNameInUse = errors.New("repository: profile name already in use")

// ErrEnvironmentNameInUse is returned by EnvironmentRepository.Create/Update
// when the name column's unique index rejects the write — the same
// pre-check-plus-backstop pattern as ErrEmailInUse.
var ErrEnvironmentNameInUse = errors.New("repository: environment name already in use")

// ErrEnvironmentCategoryNameInUse is returned by
// EnvironmentCategoryRepository.Create when the name column's unique
// index rejects the write.
var ErrEnvironmentCategoryNameInUse = errors.New("repository: environment category name already in use")

// ErrInUse is returned by a Delete method when a foreign key elsewhere
// still references the row, per
// docs/architecture/adr/0005-audit-columns-and-soft-delete-convention.md
// (an entity with no history dependency may be hard-deleted, but existing
// references still block it) — e.g. an environment category still used
// by an environment, per docs/business/requirements/0001-environment-management.md.
var ErrInUse = errors.New("repository: still referenced by other records")

// ErrFeatureFlagKeyInUse is returned by FeatureFlagRepository.Create when
// the key column's unique index rejects the write — the same
// pre-check-plus-backstop pattern as ErrEmailInUse.
var ErrFeatureFlagKeyInUse = errors.New("repository: feature flag key already in use")

// ErrFeatureFlagEnvironmentValueExists is returned by
// FeatureFlagEnvironmentValueRepository.Create when a row for the same
// (feature_flag_id, environment_id) pair already exists — the service
// layer treats this as a race with a concurrent first-write and self-heals
// by refetching and calling Update instead, the same shape as
// EnvironmentService.resolveCategory's create-race handling.
var ErrFeatureFlagEnvironmentValueExists = errors.New("repository: feature flag environment value already exists")

// isUniqueConstraintErr reports whether err comes from a SQLite UNIQUE
// index violation. modernc.org/sqlite surfaces this as a plain error whose
// message contains SQLite's own "UNIQUE constraint failed" text, so string
// matching is the only stable seam available without depending on that
// driver's internal error types.
func isUniqueConstraintErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// isForeignKeyConstraintErr reports whether err comes from a SQLite
// FOREIGN KEY (RESTRICT) violation — the same string-matching approach
// as isUniqueConstraintErr, for the same reason.
func isForeignKeyConstraintErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "FOREIGN KEY constraint failed")
}
