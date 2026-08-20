package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/repository"
)

// featureFlagKeyPattern enforces 0005's key format rule: "letters, numbers,
// hyphens, and underscores."
var featureFlagKeyPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

var validFeatureFlagTypes = map[string]bool{
	model.FeatureFlagTypeBoolean: true,
	model.FeatureFlagTypeString:  true,
	model.FeatureFlagTypeNumber:  true,
	model.FeatureFlagTypeJSON:    true,
}

// FeatureFlagService implements the flag-definition half of
// docs/business/requirements/0005-feature-flag-management.md — see
// FeatureFlagValueService for per-environment values, gated by the
// separate FeatureFlagValue permission (0003's CRUD-implies-View rule
// covers Create/Edit/Remove/View only, not Read/Write, so the two halves
// need independent authorization and are split into separate services).
// Authorization here (FeatureFlag: Create/Edit/Remove/View) is enforced by
// the caller via route middleware
// (internal/api/middleware.RequirePermission) — flags have no "self"
// concept, same as Environments/Service Keys.
type FeatureFlagService struct {
	flags repository.FeatureFlagRepository
}

// NewFeatureFlagService returns a FeatureFlagService backed by flags.
func NewFeatureFlagService(flags repository.FeatureFlagRepository) *FeatureFlagService {
	return &FeatureFlagService{flags: flags}
}

// CreateFeatureFlagInput is the payload for FeatureFlagService.Create.
type CreateFeatureFlagInput struct {
	Key         string
	Name        string
	Description *string
	Type        string
}

// Create creates a new feature flag definition. Type is fixed for the
// life of the flag — see UpdateFeatureFlagInput, which has no Type field
// at all, per 0005 ("the type is fixed once the flag is created").
func (s *FeatureFlagService) Create(ctx context.Context, actor *model.User, input CreateFeatureFlagInput) (*model.FeatureFlag, error) {
	if actor == nil {
		return nil, ErrForbidden
	}

	key := strings.TrimSpace(input.Key)
	if key == "" || !featureFlagKeyPattern.MatchString(key) {
		return nil, validationErrorf("key is required and may only contain letters, numbers, hyphens, and underscores")
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, validationErrorf("name is required")
	}
	if !validFeatureFlagTypes[input.Type] {
		return nil, validationErrorf("type must be one of %q, %q, %q, %q",
			model.FeatureFlagTypeBoolean, model.FeatureFlagTypeString, model.FeatureFlagTypeNumber, model.FeatureFlagTypeJSON)
	}

	if _, err := s.flags.GetByKey(ctx, key); err == nil {
		return nil, ErrFeatureFlagKeyInUse
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("checking feature flag key uniqueness: %w", err)
	}

	ts := now()
	f := &model.FeatureFlag{
		ID:          newID(),
		Key:         key,
		Name:        name,
		Description: input.Description,
		Type:        input.Type,
		AuditFields: model.AuditFields{
			CreatedOn:        ts,
			ModifiedOn:       ts,
			CreatedByUserID:  &actor.ID,
			ModifiedByUserID: &actor.ID,
		},
	}
	if err := s.flags.Create(ctx, f); err != nil {
		if errors.Is(err, repository.ErrFeatureFlagKeyInUse) {
			return nil, ErrFeatureFlagKeyInUse
		}
		return nil, fmt.Errorf("creating feature flag: %w", err)
	}

	return f, nil
}

// Get returns the feature flag with the given id.
func (s *FeatureFlagService) Get(ctx context.Context, id string) (*model.FeatureFlag, error) {
	f, err := s.flags.GetByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrFeatureFlagNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("fetching feature flag: %w", err)
	}
	return f, nil
}

// List returns a page of feature flags, per the pagination envelope in
// docs/architecture/adr/0007-api-design-conventions.md.
func (s *FeatureFlagService) List(ctx context.Context, page, pageSize int) ([]model.FeatureFlag, int, error) {
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
	flags, total, err := s.flags.List(ctx, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("listing feature flags: %w", err)
	}
	return flags, total, nil
}

// UpdateFeatureFlagInput is the payload for FeatureFlagService.Update. Nil
// fields are left unchanged, per PATCH's partial-update semantics
// (docs/architecture/adr/0007-api-design-conventions.md). Deliberately has
// no Key or Type field — both are immutable after creation, per 0005.
type UpdateFeatureFlagInput struct {
	Name        *string
	Description *string
}

// Update applies input to the feature flag identified by id.
func (s *FeatureFlagService) Update(ctx context.Context, actor *model.User, id string, input UpdateFeatureFlagInput) (*model.FeatureFlag, error) {
	if actor == nil {
		return nil, ErrForbidden
	}

	f, err := s.flags.GetByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrFeatureFlagNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("fetching feature flag: %w", err)
	}

	if input.Name != nil {
		f.Name = strings.TrimSpace(*input.Name)
	}
	if input.Description != nil {
		f.Description = input.Description
	}
	if f.Name == "" {
		return nil, validationErrorf("name is required")
	}

	f.ModifiedByUserID = &actor.ID
	f.ModifiedOn = now()

	if err := s.flags.Update(ctx, f); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrFeatureFlagNotFound
		}
		return nil, fmt.Errorf("updating feature flag: %w", err)
	}

	return f, nil
}

// Delete removes the feature flag identified by id, along with its
// per-environment values (ON DELETE CASCADE from feature_flags), per
// 0005's Remove acceptance criterion.
func (s *FeatureFlagService) Delete(ctx context.Context, actor *model.User, id string) error {
	if actor == nil {
		return ErrForbidden
	}

	if err := s.flags.Delete(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrFeatureFlagNotFound
		}
		if errors.Is(err, repository.ErrInUse) {
			return ErrFeatureFlagInUse
		}
		return fmt.Errorf("deleting feature flag: %w", err)
	}
	return nil
}
