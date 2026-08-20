package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/repository"
)

// ProfileService implements docs/business/requirements/0003-profile-and-permission-management.md.
// Authorization (Profile: Create/Edit/Remove/View) is enforced by the
// caller via route middleware (internal/api/middleware.RequirePermission)
// — profiles have no "self" concept the way Users do, so unlike
// UserService there's no per-request actor-vs-target branching to do here.
type ProfileService struct {
	profiles           repository.ProfileRepository
	profilePermissions repository.ProfilePermissionRepository
	permissions        repository.PermissionRepository
}

// NewProfileService returns a ProfileService backed by the given
// repositories.
func NewProfileService(profiles repository.ProfileRepository, profilePermissions repository.ProfilePermissionRepository, permissions repository.PermissionRepository) *ProfileService {
	return &ProfileService{profiles: profiles, profilePermissions: profilePermissions, permissions: permissions}
}

// CreateProfileInput is the payload for ProfileService.Create.
type CreateProfileInput struct {
	Name          string
	Description   *string
	PermissionIDs []string
}

// Create creates a new, non-system profile with the given permissions.
func (s *ProfileService) Create(ctx context.Context, actor *model.User, input CreateProfileInput) (*model.Profile, error) {
	if actor == nil {
		return nil, ErrForbidden
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, validationErrorf("name is required")
	}
	if err := s.validatePermissionIDs(ctx, input.PermissionIDs); err != nil {
		return nil, err
	}

	if _, err := s.profiles.GetByName(ctx, name); err == nil {
		return nil, ErrProfileNameInUse
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("checking profile name uniqueness: %w", err)
	}

	ts := now()
	p := &model.Profile{
		ID:          newID(),
		Name:        name,
		Description: input.Description,
		IsSystem:    false,
		AuditFields: model.AuditFields{
			CreatedOn:        ts,
			ModifiedOn:       ts,
			CreatedByUserID:  &actor.ID,
			ModifiedByUserID: &actor.ID,
		},
	}
	if err := s.profiles.Create(ctx, p); err != nil {
		if errors.Is(err, repository.ErrProfileNameInUse) {
			return nil, ErrProfileNameInUse
		}
		return nil, fmt.Errorf("creating profile: %w", err)
	}

	if len(input.PermissionIDs) > 0 {
		if err := s.profilePermissions.ReplacePermissions(ctx, p.ID, input.PermissionIDs, actor.ID); err != nil {
			return nil, fmt.Errorf("assigning profile permissions: %w", err)
		}
	}

	return p, nil
}

// Get returns the profile with the given id.
func (s *ProfileService) Get(ctx context.Context, id string) (*model.Profile, error) {
	p, err := s.profiles.GetByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrProfileNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("fetching profile: %w", err)
	}
	return p, nil
}

// List returns a page of profiles, per the pagination envelope in
// docs/architecture/adr/0007-api-design-conventions.md.
func (s *ProfileService) List(ctx context.Context, page, pageSize int) ([]model.Profile, int, error) {
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
	profiles, total, err := s.profiles.List(ctx, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("listing profiles: %w", err)
	}
	return profiles, total, nil
}

// PermissionIDs returns the permission ids directly assigned to profile
// id (not its effective/resolved permissions — this is the raw
// assignment, used to populate an edit form).
func (s *ProfileService) PermissionIDs(ctx context.Context, id string) ([]string, error) {
	if _, err := s.Get(ctx, id); err != nil {
		return nil, err
	}
	ids, err := s.profilePermissions.ListPermissionIDs(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("listing profile permissions: %w", err)
	}
	return ids, nil
}

// UpdateProfileInput is the payload for ProfileService.Update. Nil fields
// are left unchanged, per PATCH's partial-update semantics
// (docs/architecture/adr/0007-api-design-conventions.md). A non-nil
// PermissionIDs replaces the profile's entire permission set.
type UpdateProfileInput struct {
	Name          *string
	Description   *string
	PermissionIDs *[]string
}

// Update applies input to the profile identified by id. Changing
// permissions on a system profile (Administrador) is rejected — per
// 0003, only non-system profiles can have their permissions edited.
func (s *ProfileService) Update(ctx context.Context, actor *model.User, id string, input UpdateProfileInput) (*model.Profile, error) {
	if actor == nil {
		return nil, ErrForbidden
	}

	p, err := s.profiles.GetByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrProfileNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("fetching profile: %w", err)
	}

	if input.PermissionIDs != nil && p.IsSystem {
		return nil, fmt.Errorf("%w: cannot edit permissions on a system profile", ErrForbidden)
	}
	if input.PermissionIDs != nil {
		if err := s.validatePermissionIDs(ctx, *input.PermissionIDs); err != nil {
			return nil, err
		}
	}

	if input.Name != nil {
		p.Name = strings.TrimSpace(*input.Name)
	}
	if input.Description != nil {
		p.Description = input.Description
	}
	if p.Name == "" {
		return nil, validationErrorf("name is required")
	}

	p.ModifiedByUserID = &actor.ID
	p.ModifiedOn = now()

	if err := s.profiles.Update(ctx, p); err != nil {
		if errors.Is(err, repository.ErrProfileNameInUse) {
			return nil, ErrProfileNameInUse
		}
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrProfileNotFound
		}
		return nil, fmt.Errorf("updating profile: %w", err)
	}

	if input.PermissionIDs != nil {
		if err := s.profilePermissions.ReplacePermissions(ctx, id, *input.PermissionIDs, actor.ID); err != nil {
			return nil, fmt.Errorf("updating profile permissions: %w", err)
		}
	}

	return p, nil
}

// Delete removes profile id. Removing a profile that currently holds the
// full permission catalog (in practice, only ever the seeded Administrador
// profile — see 0003) is rejected unless another profile also holds the
// full catalog, guaranteeing the instance is never left without one.
func (s *ProfileService) Delete(ctx context.Context, actor *model.User, id string) error {
	if actor == nil {
		return ErrForbidden
	}

	p, err := s.profiles.GetByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrProfileNotFound
	}
	if err != nil {
		return fmt.Errorf("fetching profile: %w", err)
	}

	if p.IsSystem {
		holds, err := s.anotherProfileHoldsFullCatalog(ctx, id)
		if err != nil {
			return err
		}
		if !holds {
			return ErrLastFullCatalogProfile
		}
	}

	if err := s.profiles.Delete(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrProfileNotFound
		}
		return fmt.Errorf("deleting profile: %w", err)
	}
	return nil
}

func (s *ProfileService) anotherProfileHoldsFullCatalog(ctx context.Context, excludeProfileID string) (bool, error) {
	catalog, err := s.permissions.List(ctx)
	if err != nil {
		return false, fmt.Errorf("listing permission catalog: %w", err)
	}

	profiles, err := s.profiles.ListAll(ctx)
	if err != nil {
		return false, fmt.Errorf("listing profiles: %w", err)
	}

	for _, p := range profiles {
		if p.ID == excludeProfileID {
			continue
		}
		permIDs, err := s.profilePermissions.ListPermissionIDs(ctx, p.ID)
		if err != nil {
			return false, fmt.Errorf("listing profile permissions: %w", err)
		}
		if isFullCatalog(permIDs, catalog) {
			return true, nil
		}
	}
	return false, nil
}

func (s *ProfileService) validatePermissionIDs(ctx context.Context, ids []string) error {
	catalog, err := s.permissions.List(ctx)
	if err != nil {
		return fmt.Errorf("listing permission catalog: %w", err)
	}
	valid := make(map[string]struct{}, len(catalog))
	for _, p := range catalog {
		valid[p.ID] = struct{}{}
	}
	for _, id := range ids {
		if _, ok := valid[id]; !ok {
			return validationErrorf("unknown permission id %q", id)
		}
	}
	return nil
}

func isFullCatalog(permissionIDs []string, catalog []model.Permission) bool {
	if len(permissionIDs) != len(catalog) {
		return false
	}
	set := make(map[string]struct{}, len(permissionIDs))
	for _, id := range permissionIDs {
		set[id] = struct{}{}
	}
	for _, p := range catalog {
		if _, ok := set[p.ID]; !ok {
			return false
		}
	}
	return true
}
