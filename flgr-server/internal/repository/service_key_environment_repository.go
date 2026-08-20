package repository

import (
	"context"
	"database/sql"
)

const serviceKeyEnvironmentsTable = "service_key_environments"

// ServiceKeyEnvironmentRepository manages the service_key_environments join
// table — which environments a service key may access — per
// docs/business/requirements/0004-service-key-management.md.
type ServiceKeyEnvironmentRepository interface {
	ListEnvironmentIDs(ctx context.Context, serviceKeyID string) ([]string, error)
	// ReplaceEnvironments atomically sets serviceKeyID's environment scope
	// to exactly environmentIDs, stamping actorUserID as the actor.
	ReplaceEnvironments(ctx context.Context, serviceKeyID string, environmentIDs []string, actorUserID string) error
}

type serviceKeyEnvironmentRepository struct {
	db *sql.DB
}

// NewServiceKeyEnvironmentRepository returns a
// ServiceKeyEnvironmentRepository backed by db.
func NewServiceKeyEnvironmentRepository(db *sql.DB) ServiceKeyEnvironmentRepository {
	return &serviceKeyEnvironmentRepository{db: db}
}

func (r *serviceKeyEnvironmentRepository) ListEnvironmentIDs(ctx context.Context, serviceKeyID string) ([]string, error) {
	return listAssociatedIDs(ctx, r.db, serviceKeyEnvironmentsTable, "service_key_id", "environment_id", serviceKeyID)
}

func (r *serviceKeyEnvironmentRepository) ReplaceEnvironments(ctx context.Context, serviceKeyID string, environmentIDs []string, actorUserID string) error {
	return replaceAssociations(ctx, r.db, serviceKeyEnvironmentsTable, "service_key_id", "environment_id", serviceKeyID, environmentIDs, actorUserID)
}
