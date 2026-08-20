package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/repository"
)

// EnvironmentService implements docs/business/requirements/0001-environment-management.md.
// Authorization (Environment: Create/Edit/Remove/View) is enforced by the
// caller via route middleware (internal/api/middleware.RequirePermission)
// — environments have no "self" concept, so there's no per-request
// actor-vs-target branching to do here (unlike UserService).
type EnvironmentService struct {
	environments repository.EnvironmentRepository
	categories   repository.EnvironmentCategoryRepository
}

// NewEnvironmentService returns an EnvironmentService backed by the given
// repositories.
func NewEnvironmentService(environments repository.EnvironmentRepository, categories repository.EnvironmentCategoryRepository) *EnvironmentService {
	return &EnvironmentService{environments: environments, categories: categories}
}

// CreateEnvironmentInput is the payload for EnvironmentService.Create.
// CategoryName is always a name — an existing category (e.g.
// "Development") or a brand-new one typed after picking "Other" in the
// UI; resolveCategory looks it up or creates it, per 0001's Content
// section ("that name is added to the list of selectable categories").
type CreateEnvironmentInput struct {
	Name         string
	Description  *string
	CategoryName string
}

// Create creates a new environment, resolving (or creating) its category
// by name.
func (s *EnvironmentService) Create(ctx context.Context, actor *model.User, input CreateEnvironmentInput) (*model.Environment, error) {
	if actor == nil {
		return nil, ErrForbidden
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, validationErrorf("name is required")
	}

	categoryID, err := s.resolveCategory(ctx, actor, input.CategoryName)
	if err != nil {
		return nil, err
	}

	if _, err := s.environments.GetByName(ctx, name); err == nil {
		return nil, ErrEnvironmentNameInUse
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("checking environment name uniqueness: %w", err)
	}

	ts := now()
	e := &model.Environment{
		ID:                    newID(),
		Name:                  name,
		Description:           input.Description,
		EnvironmentCategoryID: categoryID,
		AuditFields: model.AuditFields{
			CreatedOn:        ts,
			ModifiedOn:       ts,
			CreatedByUserID:  &actor.ID,
			ModifiedByUserID: &actor.ID,
		},
	}
	if err := s.environments.Create(ctx, e); err != nil {
		if errors.Is(err, repository.ErrEnvironmentNameInUse) {
			return nil, ErrEnvironmentNameInUse
		}
		return nil, fmt.Errorf("creating environment: %w", err)
	}

	return e, nil
}

// resolveCategory returns the id of the category named name, creating a
// new non-system one if it doesn't exist yet.
func (s *EnvironmentService) resolveCategory(ctx context.Context, actor *model.User, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", validationErrorf("category is required")
	}

	if c, err := s.categories.GetByName(ctx, name); err == nil {
		return c.ID, nil
	} else if !errors.Is(err, repository.ErrNotFound) {
		return "", fmt.Errorf("checking category: %w", err)
	}

	ts := now()
	c := &model.EnvironmentCategory{
		ID:       newID(),
		Name:     name,
		IsSystem: false,
		AuditFields: model.AuditFields{
			CreatedOn:        ts,
			ModifiedOn:       ts,
			CreatedByUserID:  &actor.ID,
			ModifiedByUserID: &actor.ID,
		},
	}
	if err := s.categories.Create(ctx, c); err != nil {
		if errors.Is(err, repository.ErrEnvironmentCategoryNameInUse) {
			// Lost a race with a concurrent create of the same new
			// category name — the winner's row is just as good.
			existing, getErr := s.categories.GetByName(ctx, name)
			if getErr != nil {
				return "", fmt.Errorf("re-fetching category after a create race: %w", getErr)
			}
			return existing.ID, nil
		}
		return "", fmt.Errorf("creating category: %w", err)
	}

	return c.ID, nil
}

// Get returns the environment with the given id.
func (s *EnvironmentService) Get(ctx context.Context, id string) (*model.Environment, error) {
	e, err := s.environments.GetByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrEnvironmentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("fetching environment: %w", err)
	}
	return e, nil
}

// List returns a page of environments, per the pagination envelope in
// docs/architecture/adr/0007-api-design-conventions.md.
func (s *EnvironmentService) List(ctx context.Context, page, pageSize int) ([]model.Environment, int, error) {
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
	environments, total, err := s.environments.List(ctx, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("listing environments: %w", err)
	}
	return environments, total, nil
}

// UpdateEnvironmentInput is the payload for EnvironmentService.Update.
// Nil fields are left unchanged, per PATCH's partial-update semantics
// (docs/architecture/adr/0007-api-design-conventions.md).
type UpdateEnvironmentInput struct {
	Name         *string
	Description  *string
	CategoryName *string
}

// Update applies input to the environment identified by id.
func (s *EnvironmentService) Update(ctx context.Context, actor *model.User, id string, input UpdateEnvironmentInput) (*model.Environment, error) {
	if actor == nil {
		return nil, ErrForbidden
	}

	e, err := s.environments.GetByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrEnvironmentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("fetching environment: %w", err)
	}

	if input.Name != nil {
		e.Name = strings.TrimSpace(*input.Name)
	}
	if input.Description != nil {
		e.Description = input.Description
	}
	if input.CategoryName != nil {
		categoryID, err := s.resolveCategory(ctx, actor, *input.CategoryName)
		if err != nil {
			return nil, err
		}
		e.EnvironmentCategoryID = categoryID
	}
	if e.Name == "" {
		return nil, validationErrorf("name is required")
	}

	e.ModifiedByUserID = &actor.ID
	e.ModifiedOn = now()

	if err := s.environments.Update(ctx, e); err != nil {
		if errors.Is(err, repository.ErrEnvironmentNameInUse) {
			return nil, ErrEnvironmentNameInUse
		}
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrEnvironmentNotFound
		}
		return nil, fmt.Errorf("updating environment: %w", err)
	}

	return e, nil
}

// Delete removes the environment identified by id. Rejected if another
// record still references it (e.g. a feature flag value or service key
// environment grant, once those exist), per 0001.
func (s *EnvironmentService) Delete(ctx context.Context, actor *model.User, id string) error {
	if actor == nil {
		return ErrForbidden
	}

	if err := s.environments.Delete(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrEnvironmentNotFound
		}
		if errors.Is(err, repository.ErrInUse) {
			return ErrEnvironmentInUse
		}
		return fmt.Errorf("deleting environment: %w", err)
	}
	return nil
}
