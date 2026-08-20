package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
)

// EnvironmentCategoryRepository is the data access seam for the
// environment_categories table, per
// docs/business/requirements/0001-environment-management.md.
type EnvironmentCategoryRepository interface {
	Create(ctx context.Context, c *model.EnvironmentCategory) error
	GetByID(ctx context.Context, id string) (*model.EnvironmentCategory, error)
	GetByName(ctx context.Context, name string) (*model.EnvironmentCategory, error)
	ListAll(ctx context.Context) ([]model.EnvironmentCategory, error)
}

type environmentCategoryRepository struct {
	db *sql.DB
}

// NewEnvironmentCategoryRepository returns an EnvironmentCategoryRepository
// backed by db.
func NewEnvironmentCategoryRepository(db *sql.DB) EnvironmentCategoryRepository {
	return &environmentCategoryRepository{db: db}
}

const environmentCategorySelectColumns = `SELECT
	id, name, is_system,
	created_by_user_id, created_by_service_key_id, created_on,
	modified_by_user_id, modified_by_service_key_id, modified_on`

func (r *environmentCategoryRepository) Create(ctx context.Context, c *model.EnvironmentCategory) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO environment_categories (
			id, name, is_system,
			created_by_user_id, created_by_service_key_id, created_on,
			modified_by_user_id, modified_by_service_key_id, modified_on
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.Name, c.IsSystem,
		nullString(c.CreatedByUserID), nullString(c.CreatedByServiceKeyID), c.CreatedOn,
		nullString(c.ModifiedByUserID), nullString(c.ModifiedByServiceKeyID), c.ModifiedOn,
	)
	if isUniqueConstraintErr(err) {
		return fmt.Errorf("creating environment category: %w", ErrEnvironmentCategoryNameInUse)
	}
	if err != nil {
		return fmt.Errorf("creating environment category: %w", err)
	}
	return nil
}

func (r *environmentCategoryRepository) GetByID(ctx context.Context, id string) (*model.EnvironmentCategory, error) {
	return scanEnvironmentCategory(r.db.QueryRowContext(ctx, environmentCategorySelectColumns+" FROM environment_categories WHERE id = ?", id))
}

func (r *environmentCategoryRepository) GetByName(ctx context.Context, name string) (*model.EnvironmentCategory, error) {
	return scanEnvironmentCategory(r.db.QueryRowContext(ctx, environmentCategorySelectColumns+" FROM environment_categories WHERE name = ?", name))
}

func (r *environmentCategoryRepository) ListAll(ctx context.Context) ([]model.EnvironmentCategory, error) {
	rows, err := r.db.QueryContext(ctx, environmentCategorySelectColumns+" FROM environment_categories ORDER BY is_system DESC, name")
	if err != nil {
		return nil, fmt.Errorf("listing environment categories: %w", err)
	}
	defer func() { _ = rows.Close() }()

	// A scan failure here, or rows.Err() below, isn't reachable in a test
	// without a fault-injecting driver double, per the exception clause
	// in docs/architecture/adr/0004-testing-and-coverage-standards.md.
	var categories []model.EnvironmentCategory
	for rows.Next() {
		c, err := scanEnvironmentCategoryRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning environment category: %w", err)
		}
		categories = append(categories, *c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("listing environment categories: %w", err)
	}

	return categories, nil
}

func scanEnvironmentCategory(row *sql.Row) (*model.EnvironmentCategory, error) {
	c, err := scanEnvironmentCategoryRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("fetching environment category: %w", err)
	}
	return c, nil
}

func scanEnvironmentCategoryRow(row rowScanner) (*model.EnvironmentCategory, error) {
	var c model.EnvironmentCategory
	var createdByUser, createdByServiceKey, modifiedByUser, modifiedByServiceKey sql.NullString

	err := row.Scan(
		&c.ID, &c.Name, &c.IsSystem,
		&createdByUser, &createdByServiceKey, &c.CreatedOn,
		&modifiedByUser, &modifiedByServiceKey, &c.ModifiedOn,
	)
	if err != nil {
		return nil, err
	}

	c.CreatedByUserID = stringPtr(createdByUser)
	c.CreatedByServiceKeyID = stringPtr(createdByServiceKey)
	c.ModifiedByUserID = stringPtr(modifiedByUser)
	c.ModifiedByServiceKeyID = stringPtr(modifiedByServiceKey)

	return &c, nil
}
