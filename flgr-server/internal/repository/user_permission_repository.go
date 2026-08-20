package repository

import (
	"context"
	"database/sql"
)

const userPermissionsTable = "user_permissions"

// UserPermissionRepository manages the user_permissions join table —
// permissions granted directly to a user, bypassing profiles — per
// docs/business/requirements/0003-profile-and-permission-management.md.
type UserPermissionRepository interface {
	ListPermissionIDs(ctx context.Context, userID string) ([]string, error)
	// ReplacePermissions atomically sets userID's direct permission
	// grants to exactly permissionIDs, stamping actorUserID as the actor.
	ReplacePermissions(ctx context.Context, userID string, permissionIDs []string, actorUserID string) error
}

type userPermissionRepository struct {
	db *sql.DB
}

// NewUserPermissionRepository returns a UserPermissionRepository backed
// by db.
func NewUserPermissionRepository(db *sql.DB) UserPermissionRepository {
	return &userPermissionRepository{db: db}
}

func (r *userPermissionRepository) ListPermissionIDs(ctx context.Context, userID string) ([]string, error) {
	return listAssociatedIDs(ctx, r.db, userPermissionsTable, "user_id", "permission_id", userID)
}

func (r *userPermissionRepository) ReplacePermissions(ctx context.Context, userID string, permissionIDs []string, actorUserID string) error {
	return replaceAssociations(ctx, r.db, userPermissionsTable, "user_id", "permission_id", userID, permissionIDs, actorUserID)
}
