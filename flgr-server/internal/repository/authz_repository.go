package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
)

// AuthzRepository resolves a User's effective permissions, per
// docs/business/requirements/0003-profile-and-permission-management.md:
// the union of direct grants and every profile the user belongs to.
type AuthzRepository interface {
	EffectivePermissions(ctx context.Context, userID string) ([]model.Permission, error)
}

type authzRepository struct {
	db *sql.DB
}

// NewAuthzRepository returns an AuthzRepository backed by db.
func NewAuthzRepository(db *sql.DB) AuthzRepository {
	return &authzRepository{db: db}
}

func (r *authzRepository) EffectivePermissions(ctx context.Context, userID string) ([]model.Permission, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT p.id, p.resource, p.action
		FROM permissions p
		WHERE p.id IN (
			SELECT permission_id FROM user_permissions WHERE user_id = ?
			UNION
			SELECT pp.permission_id
			FROM profile_permissions pp
			JOIN user_profiles up ON up.profile_id = pp.profile_id
			WHERE up.user_id = ?
		)
		ORDER BY p.resource, p.action`, userID, userID)
	if err != nil {
		return nil, fmt.Errorf("resolving effective permissions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	// A scan failure here, or rows.Err() below, isn't reachable in a test
	// without a fault-injecting driver double, per the exception clause
	// in docs/architecture/adr/0004-testing-and-coverage-standards.md.
	var permissions []model.Permission
	for rows.Next() {
		var p model.Permission
		if err := rows.Scan(&p.ID, &p.Resource, &p.Action); err != nil {
			return nil, fmt.Errorf("scanning permission: %w", err)
		}
		permissions = append(permissions, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("resolving effective permissions: %w", err)
	}

	return permissions, nil
}
