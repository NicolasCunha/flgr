package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
)

// PermissionRepository is the read-only data access seam for the fixed
// permission catalog, per docs/business/requirements/0003-profile-and-permission-management.md
// — permissions are system-seeded, never created/edited/removed via the app.
type PermissionRepository interface {
	List(ctx context.Context) ([]model.Permission, error)
}

type permissionRepository struct {
	db *sql.DB
}

// NewPermissionRepository returns a PermissionRepository backed by db.
func NewPermissionRepository(db *sql.DB) PermissionRepository {
	return &permissionRepository{db: db}
}

func (r *permissionRepository) List(ctx context.Context) ([]model.Permission, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, resource, action,
			created_by_user_id, created_by_service_key_id, created_on,
			modified_by_user_id, modified_by_service_key_id, modified_on
		FROM permissions ORDER BY resource, action`)
	if err != nil {
		return nil, fmt.Errorf("listing permissions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	// A scan failure here, or rows.Err() below, isn't reachable in a test
	// without a fault-injecting driver double, per the exception clause
	// in docs/architecture/adr/0004-testing-and-coverage-standards.md.
	var permissions []model.Permission
	for rows.Next() {
		var p model.Permission
		var createdByUser, createdByServiceKey, modifiedByUser, modifiedByServiceKey sql.NullString

		if err := rows.Scan(
			&p.ID, &p.Resource, &p.Action,
			&createdByUser, &createdByServiceKey, &p.CreatedOn,
			&modifiedByUser, &modifiedByServiceKey, &p.ModifiedOn,
		); err != nil {
			return nil, fmt.Errorf("scanning permission: %w", err)
		}
		p.CreatedByUserID = stringPtr(createdByUser)
		p.CreatedByServiceKeyID = stringPtr(createdByServiceKey)
		p.ModifiedByUserID = stringPtr(modifiedByUser)
		p.ModifiedByServiceKeyID = stringPtr(modifiedByServiceKey)

		permissions = append(permissions, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("listing permissions: %w", err)
	}

	return permissions, nil
}
