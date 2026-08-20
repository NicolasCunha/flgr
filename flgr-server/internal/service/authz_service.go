package service

import (
	"context"
	"fmt"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/repository"
)

// AuthzService resolves a User's effective permissions and answers
// permission checks, per docs/business/requirements/0003-profile-and-permission-management.md.
// Nothing here is cached — permissions are re-evaluated from current
// state on every call, per docs/architecture/adr/0006-authentication-and-session-strategy.md,
// so a revoked grant or profile change takes effect on the very next check.
type AuthzService struct {
	authz repository.AuthzRepository
}

// NewAuthzService returns an AuthzService backed by authz.
func NewAuthzService(authz repository.AuthzRepository) *AuthzService {
	return &AuthzService{authz: authz}
}

// EffectivePermissions returns every permission userID has, from either a
// direct grant or any profile they belong to (the union of both sources).
func (s *AuthzService) EffectivePermissions(ctx context.Context, userID string) ([]model.Permission, error) {
	perms, err := s.authz.EffectivePermissions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("resolving effective permissions: %w", err)
	}
	return perms, nil
}

// HasPermission reports whether userID has the given (resource, action)
// permission, applying 0003's CRUD-implies-View rule: a Create, Edit, or
// Remove grant on a resource always makes View on that resource
// effective too, without a separate grant.
func (s *AuthzService) HasPermission(ctx context.Context, userID, resource, action string) (bool, error) {
	perms, err := s.EffectivePermissions(ctx, userID)
	if err != nil {
		return false, err
	}
	return hasPermission(perms, resource, action), nil
}

func hasPermission(perms []model.Permission, resource, action string) bool {
	for _, p := range perms {
		if p.Resource != resource {
			continue
		}
		if p.Action == action {
			return true
		}
		if action == model.ActionView &&
			(p.Action == model.ActionCreate || p.Action == model.ActionEdit || p.Action == model.ActionRemove) {
			return true
		}
	}
	return false
}
