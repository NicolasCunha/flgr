package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
)

// ServiceKeyRepository is the data access seam for the service_keys table,
// per docs/business/requirements/0004-service-key-management.md. No Delete
// — service keys are soft-deleted (status = Inactive) via Update, same as
// Users, since created_by_service_key_id/modified_by_service_key_id
// columns across the schema depend on the row surviving (the
// history-dependency rule in
// docs/architecture/adr/0005-audit-columns-and-soft-delete-convention.md).
type ServiceKeyRepository interface {
	Create(ctx context.Context, k *model.ServiceKey) error
	GetByID(ctx context.Context, id string) (*model.ServiceKey, error)
	GetBySecretHash(ctx context.Context, hash string) (*model.ServiceKey, error)
	List(ctx context.Context, limit, offset int) ([]model.ServiceKey, int, error)
	Update(ctx context.Context, k *model.ServiceKey) error
}

type serviceKeyRepository struct {
	db *sql.DB
}

// NewServiceKeyRepository returns a ServiceKeyRepository backed by db.
func NewServiceKeyRepository(db *sql.DB) ServiceKeyRepository {
	return &serviceKeyRepository{db: db}
}

const serviceKeySelectColumns = `SELECT
	id, name, secret_hash, status, can_read, can_write,
	created_by_user_id, created_by_service_key_id, created_on,
	modified_by_user_id, modified_by_service_key_id, modified_on`

func (r *serviceKeyRepository) Create(ctx context.Context, k *model.ServiceKey) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO service_keys (
			id, name, secret_hash, status, can_read, can_write,
			created_by_user_id, created_by_service_key_id, created_on,
			modified_by_user_id, modified_by_service_key_id, modified_on
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		k.ID, k.Name, k.SecretHash, k.Status, k.CanRead, k.CanWrite,
		nullString(k.CreatedByUserID), nullString(k.CreatedByServiceKeyID), k.CreatedOn,
		nullString(k.ModifiedByUserID), nullString(k.ModifiedByServiceKeyID), k.ModifiedOn,
	)
	if err != nil {
		return fmt.Errorf("creating service key: %w", err)
	}
	return nil
}

func (r *serviceKeyRepository) GetByID(ctx context.Context, id string) (*model.ServiceKey, error) {
	return scanServiceKey(r.db.QueryRowContext(ctx, serviceKeySelectColumns+" FROM service_keys WHERE id = ?", id))
}

func (r *serviceKeyRepository) GetBySecretHash(ctx context.Context, hash string) (*model.ServiceKey, error) {
	return scanServiceKey(r.db.QueryRowContext(ctx, serviceKeySelectColumns+" FROM service_keys WHERE secret_hash = ?", hash))
}

func (r *serviceKeyRepository) List(ctx context.Context, limit, offset int) ([]model.ServiceKey, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(1) FROM service_keys").Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting service keys: %w", err)
	}

	// A failure here specifically (count succeeds, the page query fails)
	// isn't reachable in a test without a fault-injecting driver double,
	// per the exception clause in docs/architecture/adr/0004-testing-and-coverage-standards.md.
	rows, err := r.db.QueryContext(ctx, serviceKeySelectColumns+" FROM service_keys ORDER BY name LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("listing service keys: %w", err)
	}
	defer func() { _ = rows.Close() }()

	// A scan failure here, or rows.Err() below, isn't reachable in a test
	// without a fault-injecting driver double, per the same exception.
	var keys []model.ServiceKey
	for rows.Next() {
		k, err := scanServiceKeyRow(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning service key: %w", err)
		}
		keys = append(keys, *k)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("listing service keys: %w", err)
	}

	return keys, total, nil
}

func (r *serviceKeyRepository) Update(ctx context.Context, k *model.ServiceKey) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE service_keys SET
			name = ?, status = ?, can_read = ?, can_write = ?,
			modified_by_user_id = ?, modified_by_service_key_id = ?, modified_on = ?
		WHERE id = ?`,
		k.Name, k.Status, k.CanRead, k.CanWrite,
		nullString(k.ModifiedByUserID), nullString(k.ModifiedByServiceKeyID), k.ModifiedOn,
		k.ID,
	)
	if err != nil {
		return fmt.Errorf("updating service key: %w", err)
	}

	// RowsAffected's own error path isn't reachable with a real SQLite
	// driver query result; not exercised in a test without a fault-
	// injecting driver double, per the exception clause in ADR-0004.
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("updating service key: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("updating service key: %w", ErrNotFound)
	}
	return nil
}

func scanServiceKey(row *sql.Row) (*model.ServiceKey, error) {
	k, err := scanServiceKeyRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("fetching service key: %w", err)
	}
	return k, nil
}

func scanServiceKeyRow(row rowScanner) (*model.ServiceKey, error) {
	var k model.ServiceKey
	var createdByUser, createdByServiceKey, modifiedByUser, modifiedByServiceKey sql.NullString

	err := row.Scan(
		&k.ID, &k.Name, &k.SecretHash, &k.Status, &k.CanRead, &k.CanWrite,
		&createdByUser, &createdByServiceKey, &k.CreatedOn,
		&modifiedByUser, &modifiedByServiceKey, &k.ModifiedOn,
	)
	if err != nil {
		return nil, err
	}

	k.CreatedByUserID = stringPtr(createdByUser)
	k.CreatedByServiceKeyID = stringPtr(createdByServiceKey)
	k.ModifiedByUserID = stringPtr(modifiedByUser)
	k.ModifiedByServiceKeyID = stringPtr(modifiedByServiceKey)

	return &k, nil
}
