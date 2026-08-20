package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/repository"
)

// UserAccessService manages a User's profile memberships and direct
// permission grants, per docs/business/requirements/0003-profile-and-permission-management.md.
// Split from UserService (which owns the User record itself) so growing
// RBAC-adjacent concerns doesn't keep expanding UserService's dependency
// list. Every method here requires the "User: Edit" permission, enforced
// by route middleware — there's no self-service exception for managing
// one's own access, unlike UserService.Get/.Update.
type UserAccessService struct {
	users           repository.UserRepository
	userProfiles    repository.UserProfileRepository
	userPermissions repository.UserPermissionRepository
	profiles        repository.ProfileRepository
	permissions     repository.PermissionRepository
}

// NewUserAccessService returns a UserAccessService backed by the given
// repositories.
func NewUserAccessService(
	users repository.UserRepository,
	userProfiles repository.UserProfileRepository,
	userPermissions repository.UserPermissionRepository,
	profiles repository.ProfileRepository,
	permissions repository.PermissionRepository,
) *UserAccessService {
	return &UserAccessService{
		users:           users,
		userProfiles:    userProfiles,
		userPermissions: userPermissions,
		profiles:        profiles,
		permissions:     permissions,
	}
}

func (s *UserAccessService) userExists(ctx context.Context, userID string) error {
	if _, err := s.users.GetByID(ctx, userID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrUserNotFound
		}
		return fmt.Errorf("fetching user: %w", err)
	}
	return nil
}

// ProfileIDs returns the profile ids userID is currently assigned to.
func (s *UserAccessService) ProfileIDs(ctx context.Context, userID string) ([]string, error) {
	if err := s.userExists(ctx, userID); err != nil {
		return nil, err
	}
	ids, err := s.userProfiles.ListProfileIDs(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("listing user profiles: %w", err)
	}
	return ids, nil
}

// ReplaceProfiles sets userID's profile memberships to exactly profileIDs.
func (s *UserAccessService) ReplaceProfiles(ctx context.Context, actor *model.User, userID string, profileIDs []string) error {
	if actor == nil {
		return ErrForbidden
	}
	if err := s.userExists(ctx, userID); err != nil {
		return err
	}
	for _, id := range profileIDs {
		if _, err := s.profiles.GetByID(ctx, id); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return validationErrorf("unknown profile id %q", id)
			}
			return fmt.Errorf("checking profile %q: %w", id, err)
		}
	}

	if err := s.userProfiles.ReplaceProfiles(ctx, userID, profileIDs, actor.ID); err != nil {
		return fmt.Errorf("updating user profiles: %w", err)
	}
	return nil
}

// DirectPermissionIDs returns the permission ids granted directly to
// userID, bypassing profiles.
func (s *UserAccessService) DirectPermissionIDs(ctx context.Context, userID string) ([]string, error) {
	if err := s.userExists(ctx, userID); err != nil {
		return nil, err
	}
	ids, err := s.userPermissions.ListPermissionIDs(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("listing user permissions: %w", err)
	}
	return ids, nil
}

// ReplaceDirectPermissions sets userID's direct permission grants to
// exactly permissionIDs.
func (s *UserAccessService) ReplaceDirectPermissions(ctx context.Context, actor *model.User, userID string, permissionIDs []string) error {
	if actor == nil {
		return ErrForbidden
	}
	if err := s.userExists(ctx, userID); err != nil {
		return err
	}

	catalog, err := s.permissions.List(ctx)
	if err != nil {
		return fmt.Errorf("listing permission catalog: %w", err)
	}
	valid := make(map[string]struct{}, len(catalog))
	for _, p := range catalog {
		valid[p.ID] = struct{}{}
	}
	for _, id := range permissionIDs {
		if _, ok := valid[id]; !ok {
			return validationErrorf("unknown permission id %q", id)
		}
	}

	if err := s.userPermissions.ReplacePermissions(ctx, userID, permissionIDs, actor.ID); err != nil {
		return fmt.Errorf("updating user permissions: %w", err)
	}
	return nil
}
