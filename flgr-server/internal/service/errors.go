// Package service holds flgr-server's business logic, orchestrating
// repository interfaces per backend.md's Layering rule.
package service

import (
	"errors"
	"fmt"
)

// ValidationError carries a caller-facing validation message. It matches
// errors.Is(err, ErrValidation), so callers that only care about the
// error class don't need to know about this type.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

func (e *ValidationError) Is(target error) bool { return target == ErrValidation }

func validationErrorf(format string, args ...any) error {
	return &ValidationError{Message: fmt.Sprintf(format, args...)}
}

var (
	// ErrValidation is returned when caller-supplied input fails
	// validation (missing required field, malformed email, etc.). Check
	// with errors.Is — the concrete error is a *ValidationError carrying
	// the caller-facing message.
	ErrValidation = errors.New("service: validation failed")

	// ErrEmailInUse is returned when a user's email collides with an
	// existing account, per docs/business/requirements/0002-user-management.md.
	ErrEmailInUse = errors.New("service: email already in use")

	// ErrUserNotFound is returned when a referenced user does not exist.
	ErrUserNotFound = errors.New("service: user not found")

	// ErrInvalidCredentials is returned when a login's email/password
	// combination doesn't match an active user.
	ErrInvalidCredentials = errors.New("service: invalid credentials")

	// ErrAccountLocked is returned when a login is rejected by the
	// brute-force lockout in docs/architecture/adr/0006-authentication-and-session-strategy.md.
	ErrAccountLocked = errors.New("service: account temporarily locked")

	// ErrUserInactive is returned when a login is attempted for a user
	// whose status is not Active.
	ErrUserInactive = errors.New("service: user is inactive")

	// ErrSessionInvalid is returned when a session token doesn't match a
	// live, unexpired session for an active user.
	ErrSessionInvalid = errors.New("service: session invalid or expired")

	// ErrForbidden is returned when an authenticated actor attempts an
	// action they're not allowed to perform on the target resource
	// (e.g. a user changing their own status through self-service).
	ErrForbidden = errors.New("service: forbidden")

	// ErrSetupAlreadyComplete is returned by UserService.CompleteSetup
	// once a user already exists — the setup wizard only ever runs once.
	ErrSetupAlreadyComplete = errors.New("service: setup already complete")

	// ErrProfileNotFound is returned when a referenced profile does not exist.
	ErrProfileNotFound = errors.New("service: profile not found")

	// ErrProfileNameInUse is returned when a profile's name collides with
	// an existing one, per docs/business/requirements/0003-profile-and-permission-management.md.
	ErrProfileNameInUse = errors.New("service: profile name already in use")

	// ErrLastFullCatalogProfile is returned by ProfileService.Delete when
	// removing the profile would leave the instance with no profile
	// holding the full permission catalog — see 0003's rule protecting
	// the seeded Administrador profile.
	ErrLastFullCatalogProfile = errors.New("service: cannot remove the only profile holding the full permission catalog")

	// ErrEnvironmentNotFound is returned when a referenced environment
	// does not exist.
	ErrEnvironmentNotFound = errors.New("service: environment not found")

	// ErrEnvironmentNameInUse is returned when an environment's name
	// collides with an existing one, per
	// docs/business/requirements/0001-environment-management.md.
	ErrEnvironmentNameInUse = errors.New("service: environment name already in use")

	// ErrEnvironmentInUse is returned by EnvironmentService.Delete when
	// another record (e.g. a feature flag value or service key
	// environment grant, once those exist) still references the
	// environment, per 0001.
	ErrEnvironmentInUse = errors.New("service: environment is still in use")

	// ErrServiceKeyNotFound is returned when a referenced service key does
	// not exist, per docs/business/requirements/0004-service-key-management.md.
	ErrServiceKeyNotFound = errors.New("service: service key not found")

	// ErrFeatureFlagNotFound is returned when a referenced feature flag
	// does not exist, per docs/business/requirements/0005-feature-flag-management.md.
	ErrFeatureFlagNotFound = errors.New("service: feature flag not found")

	// ErrFeatureFlagKeyInUse is returned when a feature flag's key
	// collides with an existing one, per 0005.
	ErrFeatureFlagKeyInUse = errors.New("service: feature flag key already in use")

	// ErrFeatureFlagInUse is returned by FeatureFlagService.Delete when
	// another record still references the flag (e.g. an audit log entry,
	// once 0008 is implemented).
	ErrFeatureFlagInUse = errors.New("service: feature flag is still in use")

	// ErrNotificationChannelNotFound is returned when a referenced
	// notification channel does not exist, or exists but belongs to a
	// different flag than the one requested, per
	// docs/business/requirements/0006-feature-flag-killswitch.md.
	ErrNotificationChannelNotFound = errors.New("service: notification channel not found")

	// ErrNotificationChannelInUse is returned by
	// NotificationChannelService.Delete when a notification_outbox row
	// still references the channel, per ADR-0009.
	ErrNotificationChannelInUse = errors.New("service: notification channel is still in use")
)
