package repository

import (
	"context"
	"database/sql"
)

const userProfilesTable = "user_profiles"

// UserProfileRepository manages the user_profiles join table — which
// profiles a user belongs to — per
// docs/business/requirements/0003-profile-and-permission-management.md.
type UserProfileRepository interface {
	ListProfileIDs(ctx context.Context, userID string) ([]string, error)
	// ReplaceProfiles atomically sets userID's profile memberships to
	// exactly profileIDs, stamping actorUserID as the actor.
	ReplaceProfiles(ctx context.Context, userID string, profileIDs []string, actorUserID string) error
}

type userProfileRepository struct {
	db *sql.DB
}

// NewUserProfileRepository returns a UserProfileRepository backed by db.
func NewUserProfileRepository(db *sql.DB) UserProfileRepository {
	return &userProfileRepository{db: db}
}

func (r *userProfileRepository) ListProfileIDs(ctx context.Context, userID string) ([]string, error) {
	return listAssociatedIDs(ctx, r.db, userProfilesTable, "user_id", "profile_id", userID)
}

func (r *userProfileRepository) ReplaceProfiles(ctx context.Context, userID string, profileIDs []string, actorUserID string) error {
	return replaceAssociations(ctx, r.db, userProfilesTable, "user_id", "profile_id", userID, profileIDs, actorUserID)
}
