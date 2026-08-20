package service

import (
	"context"
	"fmt"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/repository"
)

// EnvironmentCategoryService exposes the environment category list (the
// three seeded ones plus any custom ones created via
// EnvironmentService.resolveCategory), per
// docs/business/requirements/0001-environment-management.md. Read-only —
// categories are created only as a side effect of creating/editing an
// environment with a new category name, never directly.
type EnvironmentCategoryService struct {
	categories repository.EnvironmentCategoryRepository
}

// NewEnvironmentCategoryService returns an EnvironmentCategoryService
// backed by categories.
func NewEnvironmentCategoryService(categories repository.EnvironmentCategoryRepository) *EnvironmentCategoryService {
	return &EnvironmentCategoryService{categories: categories}
}

// List returns every environment category, for populating the category
// selector described in 0001 ("Development, Staging, Production, every
// previously created custom category, and Other").
func (s *EnvironmentCategoryService) List(ctx context.Context) ([]model.EnvironmentCategory, error) {
	categories, err := s.categories.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing environment categories: %w", err)
	}
	return categories, nil
}
