package service

import (
	"context"
	"fmt"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/repository"
)

// PermissionService exposes the fixed permission catalog from
// docs/business/requirements/0003-profile-and-permission-management.md.
// Permissions are system-seeded — this is read-only.
type PermissionService struct {
	permissions repository.PermissionRepository
}

// NewPermissionService returns a PermissionService backed by permissions.
func NewPermissionService(permissions repository.PermissionRepository) *PermissionService {
	return &PermissionService{permissions: permissions}
}

// List returns the full permission catalog.
func (s *PermissionService) List(ctx context.Context) ([]model.Permission, error) {
	perms, err := s.permissions.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing permission catalog: %w", err)
	}
	return perms, nil
}
