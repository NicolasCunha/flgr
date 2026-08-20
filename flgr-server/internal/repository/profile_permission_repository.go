package repository

import (
	"context"
	"database/sql"
)

const profilePermissionsTable = "profile_permissions"

// ProfilePermissionRepository manages the profile_permissions join table
// — which permissions a profile grants — per
// docs/business/requirements/0003-profile-and-permission-management.md.
type ProfilePermissionRepository interface {
	ListPermissionIDs(ctx context.Context, profileID string) ([]string, error)
	// ReplacePermissions atomically sets profileID's granted permissions
	// to exactly permissionIDs, stamping actorUserID as the actor.
	ReplacePermissions(ctx context.Context, profileID string, permissionIDs []string, actorUserID string) error
}

type profilePermissionRepository struct {
	db *sql.DB
}

// NewProfilePermissionRepository returns a ProfilePermissionRepository
// backed by db.
func NewProfilePermissionRepository(db *sql.DB) ProfilePermissionRepository {
	return &profilePermissionRepository{db: db}
}

func (r *profilePermissionRepository) ListPermissionIDs(ctx context.Context, profileID string) ([]string, error) {
	return listAssociatedIDs(ctx, r.db, profilePermissionsTable, "profile_id", "permission_id", profileID)
}

func (r *profilePermissionRepository) ReplacePermissions(ctx context.Context, profileID string, permissionIDs []string, actorUserID string) error {
	return replaceAssociations(ctx, r.db, profilePermissionsTable, "profile_id", "permission_id", profileID, permissionIDs, actorUserID)
}
