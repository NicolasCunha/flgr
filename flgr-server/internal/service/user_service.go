package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/repository"
)

const (
	minPasswordLength = 8

	defaultPageSize = 50
	maxPageSize     = 200
)

// now is a seam over time.Now so tests can control CreatedOn/ModifiedOn,
// mirroring the sqlOpen seam in internal/database/database.go.
var now = time.Now

// newID is a seam over uuid.NewString so tests can assert on generated IDs.
var newID = uuid.NewString

// PermissionChecker is the authorization seam UserService uses for the
// non-self branches of Get/Update — satisfied by *AuthzService. A narrow
// interface (rather than depending on *AuthzService directly) so it can
// be faked in tests without a real database.
type PermissionChecker interface {
	HasPermission(ctx context.Context, userID, resource, action string) (bool, error)
}

// UserService implements docs/business/requirements/0002-user-management.md.
type UserService struct {
	users repository.UserRepository
	authz PermissionChecker
}

// NewUserService returns a UserService backed by users, using authz for
// permission checks on actions targeting a User other than the actor.
func NewUserService(users repository.UserRepository, authz PermissionChecker) *UserService {
	return &UserService{users: users, authz: authz}
}

// CreateUserInput is the payload for UserService.Create.
type CreateUserInput struct {
	FirstName string
	LastName  string
	Email     string
	Password  string
}

// Create creates a new user, hashing its password. actor is the User
// performing the creation, or nil for the one system bootstrap case (the
// first user, created before anyone can authenticate — see
// CompleteSetup). The "User: Create" permission this requires per 0002 is
// enforced by the caller via route middleware
// (internal/api/middleware.RequirePermission), not here — creating a user
// has no "self" target the way Get/Update do, so a static per-route check
// is sufficient.
func (s *UserService) Create(ctx context.Context, actor *model.User, input CreateUserInput) (*model.User, error) {
	input.FirstName = strings.TrimSpace(input.FirstName)
	input.LastName = strings.TrimSpace(input.LastName)
	input.Email = normalizeEmail(input.Email)

	if err := validateName(input.FirstName, input.LastName); err != nil {
		return nil, err
	}
	if err := validateEmailFormat(input.Email); err != nil {
		return nil, err
	}
	if err := validatePasswordPresence(input.Password); err != nil {
		return nil, err
	}

	hash, err := hashPassword(input.Password)
	if err != nil {
		return nil, validationErrorf("%v", err)
	}

	if _, err := s.users.GetByEmail(ctx, input.Email); err == nil {
		return nil, ErrEmailInUse
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("checking email uniqueness: %w", err)
	}

	ts := now()
	u := &model.User{
		ID:           newID(),
		FirstName:    input.FirstName,
		LastName:     input.LastName,
		Email:        input.Email,
		PasswordHash: hash,
		Status:       model.UserStatusActive,
		AuditFields: model.AuditFields{
			CreatedOn:  ts,
			ModifiedOn: ts,
		},
	}
	if actor != nil {
		u.CreatedByUserID = &actor.ID
		u.ModifiedByUserID = &actor.ID
	}

	if err := s.users.Create(ctx, u); err != nil {
		if errors.Is(err, repository.ErrEmailInUse) {
			return nil, ErrEmailInUse
		}
		return nil, fmt.Errorf("creating user: %w", err)
	}

	return u, nil
}

// Get returns the user with the given id. A user may always view their
// own record; viewing another's requires the "User: View" permission
// (or Create/Edit/Remove, which imply View — see AuthzService), per 0002.
func (s *UserService) Get(ctx context.Context, actor *model.User, id string) (*model.User, error) {
	if actor == nil {
		return nil, ErrForbidden
	}
	if actor.ID != id {
		allowed, err := s.authz.HasPermission(ctx, actor.ID, model.ResourceUser, model.ActionView)
		if err != nil {
			return nil, fmt.Errorf("checking permission: %w", err)
		}
		if !allowed {
			return nil, ErrForbidden
		}
	}

	u, err := s.users.GetByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("fetching user: %w", err)
	}
	return u, nil
}

// List returns a page of users, per the pagination envelope in
// docs/architecture/adr/0007-api-design-conventions.md. The "User: View"
// permission this requires is enforced by route middleware, same as
// Create — listing has no "self" target to special-case.
func (s *UserService) List(ctx context.Context, page, pageSize int) ([]model.User, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	offset := (page - 1) * pageSize
	users, total, err := s.users.List(ctx, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("listing users: %w", err)
	}
	return users, total, nil
}

// UpdateUserInput is the payload for UserService.Update. Nil fields are
// left unchanged, per PATCH's partial-update semantics
// (docs/architecture/adr/0007-api-design-conventions.md).
type UpdateUserInput struct {
	FirstName *string
	LastName  *string
	Email     *string
	Password  *string
	Status    *string
}

// Update applies input to the user identified by targetID, on behalf of
// actor. A user may always update their own FirstName/LastName/Email/
// Password, per 0002; they can never change their own Status this way
// (see Deactivate for the "User: Remove"-gated path that does). Changing
// another user's record — including Status — requires the "User: Edit"
// permission.
func (s *UserService) Update(ctx context.Context, actor *model.User, targetID string, input UpdateUserInput) (*model.User, error) {
	if actor == nil {
		return nil, ErrForbidden
	}

	target, err := s.users.GetByID(ctx, targetID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("fetching user: %w", err)
	}

	isSelf := actor.ID == target.ID
	if isSelf && input.Status != nil {
		return nil, fmt.Errorf("%w: cannot change your own status", ErrForbidden)
	}
	if !isSelf {
		allowed, err := s.authz.HasPermission(ctx, actor.ID, model.ResourceUser, model.ActionEdit)
		if err != nil {
			return nil, fmt.Errorf("checking permission: %w", err)
		}
		if !allowed {
			return nil, ErrForbidden
		}
	}

	return s.applyUpdate(ctx, actor, target, input)
}

func (s *UserService) applyUpdate(ctx context.Context, actor *model.User, target *model.User, input UpdateUserInput) (*model.User, error) {
	if input.FirstName != nil {
		target.FirstName = strings.TrimSpace(*input.FirstName)
	}
	if input.LastName != nil {
		target.LastName = strings.TrimSpace(*input.LastName)
	}
	if input.Email != nil {
		target.Email = normalizeEmail(*input.Email)
	}
	if input.Status != nil {
		if *input.Status != model.UserStatusActive && *input.Status != model.UserStatusInactive {
			return nil, validationErrorf("status must be %q or %q", model.UserStatusActive, model.UserStatusInactive)
		}
		target.Status = *input.Status
	}

	if err := validateName(target.FirstName, target.LastName); err != nil {
		return nil, err
	}
	if err := validateEmailFormat(target.Email); err != nil {
		return nil, err
	}

	if input.Password != nil {
		if err := validatePasswordPresence(*input.Password); err != nil {
			return nil, err
		}
		hash, err := hashPassword(*input.Password)
		if err != nil {
			return nil, validationErrorf("%v", err)
		}
		target.PasswordHash = hash
	}

	target.ModifiedByUserID = &actor.ID
	target.ModifiedOn = now()

	if err := s.users.Update(ctx, target); err != nil {
		if errors.Is(err, repository.ErrEmailInUse) {
			return nil, ErrEmailInUse
		}
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("updating user: %w", err)
	}

	return target, nil
}

// SetupNeeded reports whether the first-run setup wizard should be shown
// — true only while the users table is empty. Users are only ever created
// by other Users (0002) and there's no public signup endpoint, so a fresh
// install needs exactly this one bootstrap path to create the first one.
func (s *UserService) SetupNeeded(ctx context.Context) (bool, error) {
	_, total, err := s.users.List(ctx, 1, 0)
	if err != nil {
		return false, fmt.Errorf("checking for existing users: %w", err)
	}
	return total == 0, nil
}

// CompleteSetup creates the first User from the setup wizard, with no
// authenticated actor (the system-seeded-row exception in
// docs/architecture/adr/0005-audit-columns-and-soft-delete-convention.md).
// It fails with ErrSetupAlreadyComplete once any user exists — this path
// creates the fresh-install admin, never a way to add further users.
//
// Two concurrent first requests could both observe SetupNeeded() = true
// and each create a user (a check-then-act race, same caveat as the
// email-uniqueness pre-check in Create) — acceptable here since this path
// only matters for the single-operator moment right after install.
func (s *UserService) CompleteSetup(ctx context.Context, input CreateUserInput) (*model.User, error) {
	needed, err := s.SetupNeeded(ctx)
	if err != nil {
		return nil, err
	}
	if !needed {
		return nil, ErrSetupAlreadyComplete
	}

	return s.Create(ctx, nil, input)
}

// Deactivate sets the target user's status to Inactive (a soft delete,
// per docs/architecture/adr/0005-audit-columns-and-soft-delete-convention.md),
// rather than removing the row. The "User: Remove" permission this
// requires is enforced by route middleware (deactivation always targets
// someone via a permission-gated action, unlike Update — there's no
// self-service path to deactivating your own account) — this method
// applies the change via applyUpdate directly rather than through Update,
// which would otherwise re-check "User: Edit" instead.
func (s *UserService) Deactivate(ctx context.Context, actor *model.User, targetID string) (*model.User, error) {
	if actor == nil {
		return nil, ErrForbidden
	}

	target, err := s.users.GetByID(ctx, targetID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("fetching user: %w", err)
	}

	inactive := model.UserStatusInactive
	return s.applyUpdate(ctx, actor, target, UpdateUserInput{Status: &inactive})
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validateName(firstName, lastName string) error {
	if firstName == "" || lastName == "" {
		return validationErrorf("first name and last name are required")
	}
	return nil
}

func validateEmailFormat(email string) error {
	if email == "" {
		return validationErrorf("email is required")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return validationErrorf("email is not well-formed")
	}
	return nil
}

func validatePasswordPresence(password string) error {
	if len(password) < minPasswordLength {
		return validationErrorf("password must be at least %d characters", minPasswordLength)
	}
	return nil
}
