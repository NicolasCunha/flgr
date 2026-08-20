package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/repository"
)

// ServiceKeyService implements docs/business/requirements/0004-service-key-management.md.
// Authorization (ServiceKey: Create/Edit/Remove/View) is enforced by the
// caller via route middleware (internal/api/middleware.RequirePermission)
// — service keys have no "self" concept, same as Environments.
//
// Every read/write method returns a key's environment ids alongside the
// key itself, fetched from ServiceKeyEnvironmentRepository within the same
// call — collapsing what would otherwise be two DB round trips the caller
// has to sequence (and separately error-check) into one. Handlers call
// each method exactly once, matching every other handler's single
// call-then-respondError shape.
type ServiceKeyService struct {
	serviceKeys  repository.ServiceKeyRepository
	environments repository.EnvironmentRepository
	links        repository.ServiceKeyEnvironmentRepository
}

// NewServiceKeyService returns a ServiceKeyService backed by the given
// repositories.
func NewServiceKeyService(serviceKeys repository.ServiceKeyRepository, environments repository.EnvironmentRepository, links repository.ServiceKeyEnvironmentRepository) *ServiceKeyService {
	return &ServiceKeyService{serviceKeys: serviceKeys, environments: environments, links: links}
}

// CreateServiceKeyInput is the payload for ServiceKeyService.Create.
type CreateServiceKeyInput struct {
	Name           string
	CanRead        bool
	CanWrite       bool
	EnvironmentIDs []string
}

// Create creates a new service key, returning the persisted record, its
// environment ids, and its plaintext secret — the only time the secret is
// ever available, per 0004 ("shown in full only once, at creation time").
func (s *ServiceKeyService) Create(ctx context.Context, actor *model.User, input CreateServiceKeyInput) (*model.ServiceKey, []string, string, error) {
	if actor == nil {
		return nil, nil, "", ErrForbidden
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, nil, "", validationErrorf("name is required")
	}
	if !input.CanRead && !input.CanWrite {
		return nil, nil, "", validationErrorf("at least one of can_read/can_write must be enabled")
	}
	if len(input.EnvironmentIDs) == 0 {
		return nil, nil, "", validationErrorf("at least one environment is required")
	}
	if err := s.validateEnvironmentIDs(ctx, input.EnvironmentIDs); err != nil {
		return nil, nil, "", err
	}

	secret, err := generateToken()
	if err != nil {
		return nil, nil, "", fmt.Errorf("generating service key secret: %w", err)
	}

	ts := now()
	k := &model.ServiceKey{
		ID:         newID(),
		Name:       name,
		SecretHash: hashToken(secret),
		Status:     model.ServiceKeyStatusActive,
		CanRead:    input.CanRead,
		CanWrite:   input.CanWrite,
		AuditFields: model.AuditFields{
			CreatedOn:        ts,
			ModifiedOn:       ts,
			CreatedByUserID:  &actor.ID,
			ModifiedByUserID: &actor.ID,
		},
	}
	if err := s.serviceKeys.Create(ctx, k); err != nil {
		return nil, nil, "", fmt.Errorf("creating service key: %w", err)
	}

	if err := s.links.ReplaceEnvironments(ctx, k.ID, input.EnvironmentIDs, actor.ID); err != nil {
		return nil, nil, "", fmt.Errorf("linking service key environments: %w", err)
	}

	return k, input.EnvironmentIDs, secret, nil
}

// validateEnvironmentIDs confirms every id in ids refers to a real
// environment, so a bad id fails as a clean 400 instead of surfacing a raw
// FK-violation error from replaceAssociations — the same pre-check pattern
// UserAccessService.ReplaceProfiles uses for profile ids.
func (s *ServiceKeyService) validateEnvironmentIDs(ctx context.Context, ids []string) error {
	for _, id := range ids {
		if _, err := s.environments.GetByID(ctx, id); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return validationErrorf("unknown environment id %q", id)
			}
			return fmt.Errorf("checking environment %q: %w", id, err)
		}
	}
	return nil
}

func (s *ServiceKeyService) getByID(ctx context.Context, id string) (*model.ServiceKey, error) {
	k, err := s.serviceKeys.GetByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrServiceKeyNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("fetching service key: %w", err)
	}
	return k, nil
}

func (s *ServiceKeyService) environmentIDs(ctx context.Context, serviceKeyID string) ([]string, error) {
	ids, err := s.links.ListEnvironmentIDs(ctx, serviceKeyID)
	if err != nil {
		return nil, fmt.Errorf("listing service key environments: %w", err)
	}
	return ids, nil
}

// Get returns the service key with the given id, plus the environment ids
// it may access (metadata only — the secret itself is never retrievable
// after creation, per 0004).
func (s *ServiceKeyService) Get(ctx context.Context, id string) (*model.ServiceKey, []string, error) {
	k, err := s.getByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	envIDs, err := s.environmentIDs(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	return k, envIDs, nil
}

// List returns a page of service keys, per the pagination envelope in
// docs/architecture/adr/0007-api-design-conventions.md, alongside a map of
// each key's id to its environment ids.
func (s *ServiceKeyService) List(ctx context.Context, page, pageSize int) ([]model.ServiceKey, map[string][]string, int, error) {
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
	keys, total, err := s.serviceKeys.List(ctx, pageSize, offset)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("listing service keys: %w", err)
	}

	envIDsByKey := make(map[string][]string, len(keys))
	for _, k := range keys {
		envIDs, err := s.environmentIDs(ctx, k.ID)
		if err != nil {
			return nil, nil, 0, err
		}
		envIDsByKey[k.ID] = envIDs
	}

	return keys, envIDsByKey, total, nil
}

// UpdateServiceKeyInput is the payload for ServiceKeyService.Update. Nil
// fields are left unchanged, per PATCH's partial-update semantics
// (docs/architecture/adr/0007-api-design-conventions.md). EnvironmentIDs
// follows the same nil-vs-empty distinction as every other field here: a
// nil slice (the JSON key omitted) leaves the scope unchanged, while a
// non-nil empty slice ([]) is rejected below, since 0004 requires at least
// one linked environment at all times.
type UpdateServiceKeyInput struct {
	Name           *string
	Status         *string
	CanRead        *bool
	CanWrite       *bool
	EnvironmentIDs []string
}

// Update applies input to the service key identified by id, returning it
// with its (possibly just-replaced) environment ids.
func (s *ServiceKeyService) Update(ctx context.Context, actor *model.User, id string, input UpdateServiceKeyInput) (*model.ServiceKey, []string, error) {
	if actor == nil {
		return nil, nil, ErrForbidden
	}

	k, err := s.getByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	if input.Name != nil {
		k.Name = strings.TrimSpace(*input.Name)
	}
	if input.Status != nil {
		if *input.Status != model.ServiceKeyStatusActive && *input.Status != model.ServiceKeyStatusInactive {
			return nil, nil, validationErrorf("status must be %q or %q", model.ServiceKeyStatusActive, model.ServiceKeyStatusInactive)
		}
		k.Status = *input.Status
	}
	if input.CanRead != nil {
		k.CanRead = *input.CanRead
	}
	if input.CanWrite != nil {
		k.CanWrite = *input.CanWrite
	}
	if k.Name == "" {
		return nil, nil, validationErrorf("name is required")
	}
	if !k.CanRead && !k.CanWrite {
		return nil, nil, validationErrorf("at least one of can_read/can_write must be enabled")
	}

	if input.EnvironmentIDs != nil {
		if len(input.EnvironmentIDs) == 0 {
			return nil, nil, validationErrorf("at least one environment is required")
		}
		if err := s.validateEnvironmentIDs(ctx, input.EnvironmentIDs); err != nil {
			return nil, nil, err
		}
	}

	k.ModifiedByUserID = &actor.ID
	k.ModifiedOn = now()

	if err := s.serviceKeys.Update(ctx, k); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil, ErrServiceKeyNotFound
		}
		return nil, nil, fmt.Errorf("updating service key: %w", err)
	}

	if input.EnvironmentIDs != nil {
		if err := s.links.ReplaceEnvironments(ctx, k.ID, input.EnvironmentIDs, actor.ID); err != nil {
			return nil, nil, fmt.Errorf("updating service key environments: %w", err)
		}
		return k, input.EnvironmentIDs, nil
	}

	envIDs, err := s.environmentIDs(ctx, k.ID)
	if err != nil {
		return nil, nil, err
	}
	return k, envIDs, nil
}

// Authenticate looks up the service key whose secret hashes to secret and
// returns it alongside its environment ids, in the same
// fold-a-second-query-in convention documented on the struct above.
// ErrForbidden — the same sentinel Create uses for "no actor" — is
// returned when no key matches, or when the matching key is Inactive, per
// docs/business/requirements/0007-feature-flag-evaluation-api.md; an
// invalid secret and an inactive one are rejected identically. Whether the
// key's CanRead/CanWrite/environment scope covers the caller's specific
// request is checked by the caller (middleware.RequireServiceKeyAccess /
// RequireWriteAccess), not here — Authenticate only establishes identity.
func (s *ServiceKeyService) Authenticate(ctx context.Context, secret string) (*model.ServiceKey, []string, error) {
	k, err := s.serviceKeys.GetBySecretHash(ctx, hashToken(secret))
	if errors.Is(err, repository.ErrNotFound) {
		return nil, nil, ErrForbidden
	}
	if err != nil {
		return nil, nil, fmt.Errorf("fetching service key by secret: %w", err)
	}
	if !k.IsActive() {
		return nil, nil, ErrForbidden
	}

	envIDs, err := s.environmentIDs(ctx, k.ID)
	if err != nil {
		return nil, nil, err
	}

	return k, envIDs, nil
}

// Deactivate sets the service key's status to Inactive (a soft delete, per
// docs/architecture/adr/0005-audit-columns-and-soft-delete-convention.md),
// rather than removing the row — service_keys rows are also referenced by
// created_by_service_key_id/modified_by_service_key_id columns across the
// schema, so a hard delete isn't an option even before considering history.
// The "ServiceKey: Remove" permission this requires is enforced by route
// middleware; unlike UserService.Deactivate, there's no double-gating risk
// here since Update itself performs no in-service permission check (it has
// no self-vs-other branch to protect, same reasoning as EnvironmentService).
func (s *ServiceKeyService) Deactivate(ctx context.Context, actor *model.User, id string) (*model.ServiceKey, []string, error) {
	inactive := model.ServiceKeyStatusInactive
	return s.Update(ctx, actor, id, UpdateServiceKeyInput{Status: &inactive})
}
