package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
)

// EnvironmentRepository is the data access seam for the environments
// table, per docs/business/requirements/0001-environment-management.md.
type EnvironmentRepository interface {
	Create(ctx context.Context, e *model.Environment) error
	GetByID(ctx context.Context, id string) (*model.Environment, error)
	GetByName(ctx context.Context, name string) (*model.Environment, error)
	List(ctx context.Context, limit, offset int) ([]model.Environment, int, error)
	Update(ctx context.Context, e *model.Environment) error
	Delete(ctx context.Context, id string) error
}

type environmentRepository struct {
	db *sql.DB
}

// NewEnvironmentRepository returns an EnvironmentRepository backed by db.
func NewEnvironmentRepository(db *sql.DB) EnvironmentRepository {
	return &environmentRepository{db: db}
}

const environmentSelectColumns = `SELECT
	id, name, description, environment_category_id,
	created_by_user_id, created_by_service_key_id, created_on,
	modified_by_user_id, modified_by_service_key_id, modified_on`

func (r *environmentRepository) Create(ctx context.Context, e *model.Environment) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO environments (
			id, name, description, environment_category_id,
			created_by_user_id, created_by_service_key_id, created_on,
			modified_by_user_id, modified_by_service_key_id, modified_on
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.Name, nullString(e.Description), e.EnvironmentCategoryID,
		nullString(e.CreatedByUserID), nullString(e.CreatedByServiceKeyID), e.CreatedOn,
		nullString(e.ModifiedByUserID), nullString(e.ModifiedByServiceKeyID), e.ModifiedOn,
	)
	if isUniqueConstraintErr(err) {
		return fmt.Errorf("creating environment: %w", ErrEnvironmentNameInUse)
	}
	if err != nil {
		return fmt.Errorf("creating environment: %w", err)
	}
	return nil
}

func (r *environmentRepository) GetByID(ctx context.Context, id string) (*model.Environment, error) {
	return scanEnvironment(r.db.QueryRowContext(ctx, environmentSelectColumns+" FROM environments WHERE id = ?", id))
}

func (r *environmentRepository) GetByName(ctx context.Context, name string) (*model.Environment, error) {
	return scanEnvironment(r.db.QueryRowContext(ctx, environmentSelectColumns+" FROM environments WHERE name = ?", name))
}

func (r *environmentRepository) List(ctx context.Context, limit, offset int) ([]model.Environment, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(1) FROM environments").Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting environments: %w", err)
	}

	// A failure here specifically (count succeeds, the page query fails)
	// isn't reachable in a test without a fault-injecting driver double,
	// per the exception clause in docs/architecture/adr/0004-testing-and-coverage-standards.md.
	rows, err := r.db.QueryContext(ctx, environmentSelectColumns+" FROM environments ORDER BY name LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("listing environments: %w", err)
	}
	defer func() { _ = rows.Close() }()

	// A scan failure here, or rows.Err() below, isn't reachable in a test
	// without a fault-injecting driver double, per the same exception.
	var environments []model.Environment
	for rows.Next() {
		e, err := scanEnvironmentRow(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning environment: %w", err)
		}
		environments = append(environments, *e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("listing environments: %w", err)
	}

	return environments, total, nil
}

func (r *environmentRepository) Update(ctx context.Context, e *model.Environment) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE environments SET
			name = ?, description = ?, environment_category_id = ?,
			modified_by_user_id = ?, modified_by_service_key_id = ?, modified_on = ?
		WHERE id = ?`,
		e.Name, nullString(e.Description), e.EnvironmentCategoryID,
		nullString(e.ModifiedByUserID), nullString(e.ModifiedByServiceKeyID), e.ModifiedOn,
		e.ID,
	)
	if isUniqueConstraintErr(err) {
		return fmt.Errorf("updating environment: %w", ErrEnvironmentNameInUse)
	}
	if err != nil {
		return fmt.Errorf("updating environment: %w", err)
	}

	// RowsAffected's own error path isn't reachable with a real SQLite
	// driver query result; not exercised in a test without a fault-
	// injecting driver double, per the exception clause in ADR-0004.
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("updating environment: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("updating environment: %w", ErrNotFound)
	}
	return nil
}

func (r *environmentRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM environments WHERE id = ?", id)
	if isForeignKeyConstraintErr(err) {
		return fmt.Errorf("deleting environment: %w", ErrInUse)
	}
	if err != nil {
		return fmt.Errorf("deleting environment: %w", err)
	}

	// RowsAffected's own error path isn't reachable with a real SQLite
	// driver query result; not exercised in a test without a fault-
	// injecting driver double, per the exception clause in ADR-0004.
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("deleting environment: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("deleting environment: %w", ErrNotFound)
	}
	return nil
}

func scanEnvironment(row *sql.Row) (*model.Environment, error) {
	e, err := scanEnvironmentRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("fetching environment: %w", err)
	}
	return e, nil
}

func scanEnvironmentRow(row rowScanner) (*model.Environment, error) {
	var e model.Environment
	var description, createdByUser, createdByServiceKey, modifiedByUser, modifiedByServiceKey sql.NullString

	err := row.Scan(
		&e.ID, &e.Name, &description, &e.EnvironmentCategoryID,
		&createdByUser, &createdByServiceKey, &e.CreatedOn,
		&modifiedByUser, &modifiedByServiceKey, &e.ModifiedOn,
	)
	if err != nil {
		return nil, err
	}

	e.Description = stringPtr(description)
	e.CreatedByUserID = stringPtr(createdByUser)
	e.CreatedByServiceKeyID = stringPtr(createdByServiceKey)
	e.ModifiedByUserID = stringPtr(modifiedByUser)
	e.ModifiedByServiceKeyID = stringPtr(modifiedByServiceKey)

	return &e, nil
}
