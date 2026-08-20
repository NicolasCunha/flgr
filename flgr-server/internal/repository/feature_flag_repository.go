package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
)

// FeatureFlagRepository is the data access seam for the feature_flags
// table, per docs/business/requirements/0005-feature-flag-management.md.
type FeatureFlagRepository interface {
	Create(ctx context.Context, f *model.FeatureFlag) error
	GetByID(ctx context.Context, id string) (*model.FeatureFlag, error)
	GetByKey(ctx context.Context, key string) (*model.FeatureFlag, error)
	List(ctx context.Context, limit, offset int) ([]model.FeatureFlag, int, error)
	ListAll(ctx context.Context) ([]model.FeatureFlag, error)
	Update(ctx context.Context, f *model.FeatureFlag) error
	Delete(ctx context.Context, id string) error
}

type featureFlagRepository struct {
	db *sql.DB
}

// NewFeatureFlagRepository returns a FeatureFlagRepository backed by db.
func NewFeatureFlagRepository(db *sql.DB) FeatureFlagRepository {
	return &featureFlagRepository{db: db}
}

const featureFlagSelectColumns = `SELECT
	id, key, name, description, type,
	created_by_user_id, created_by_service_key_id, created_on,
	modified_by_user_id, modified_by_service_key_id, modified_on`

func (r *featureFlagRepository) Create(ctx context.Context, f *model.FeatureFlag) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO feature_flags (
			id, key, name, description, type,
			created_by_user_id, created_by_service_key_id, created_on,
			modified_by_user_id, modified_by_service_key_id, modified_on
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		f.ID, f.Key, f.Name, nullString(f.Description), f.Type,
		nullString(f.CreatedByUserID), nullString(f.CreatedByServiceKeyID), f.CreatedOn,
		nullString(f.ModifiedByUserID), nullString(f.ModifiedByServiceKeyID), f.ModifiedOn,
	)
	if isUniqueConstraintErr(err) {
		return fmt.Errorf("creating feature flag: %w", ErrFeatureFlagKeyInUse)
	}
	if err != nil {
		return fmt.Errorf("creating feature flag: %w", err)
	}
	return nil
}

func (r *featureFlagRepository) GetByID(ctx context.Context, id string) (*model.FeatureFlag, error) {
	return scanFeatureFlag(r.db.QueryRowContext(ctx, featureFlagSelectColumns+" FROM feature_flags WHERE id = ?", id))
}

func (r *featureFlagRepository) GetByKey(ctx context.Context, key string) (*model.FeatureFlag, error) {
	return scanFeatureFlag(r.db.QueryRowContext(ctx, featureFlagSelectColumns+" FROM feature_flags WHERE key = ?", key))
}

func (r *featureFlagRepository) List(ctx context.Context, limit, offset int) ([]model.FeatureFlag, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(1) FROM feature_flags").Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting feature flags: %w", err)
	}

	// A failure here specifically (count succeeds, the page query fails)
	// isn't reachable in a test without a fault-injecting driver double,
	// per the exception clause in docs/architecture/adr/0004-testing-and-coverage-standards.md.
	rows, err := r.db.QueryContext(ctx, featureFlagSelectColumns+" FROM feature_flags ORDER BY key LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("listing feature flags: %w", err)
	}
	defer func() { _ = rows.Close() }()

	// A scan failure here, or rows.Err() below, isn't reachable in a test
	// without a fault-injecting driver double, per the same exception.
	var flags []model.FeatureFlag
	for rows.Next() {
		f, err := scanFeatureFlagRow(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning feature flag: %w", err)
		}
		flags = append(flags, *f)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("listing feature flags: %w", err)
	}

	return flags, total, nil
}

func (r *featureFlagRepository) ListAll(ctx context.Context) ([]model.FeatureFlag, error) {
	rows, err := r.db.QueryContext(ctx, featureFlagSelectColumns+" FROM feature_flags ORDER BY key")
	if err != nil {
		return nil, fmt.Errorf("listing feature flags: %w", err)
	}
	defer func() { _ = rows.Close() }()

	// A scan failure here, or rows.Err() below, isn't reachable in a test
	// without a fault-injecting driver double, per the exception clause in
	// docs/architecture/adr/0004-testing-and-coverage-standards.md.
	var flags []model.FeatureFlag
	for rows.Next() {
		f, err := scanFeatureFlagRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning feature flag: %w", err)
		}
		flags = append(flags, *f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("listing feature flags: %w", err)
	}

	return flags, nil
}

func (r *featureFlagRepository) Update(ctx context.Context, f *model.FeatureFlag) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE feature_flags SET
			name = ?, description = ?,
			modified_by_user_id = ?, modified_by_service_key_id = ?, modified_on = ?
		WHERE id = ?`,
		f.Name, nullString(f.Description),
		nullString(f.ModifiedByUserID), nullString(f.ModifiedByServiceKeyID), f.ModifiedOn,
		f.ID,
	)
	if err != nil {
		return fmt.Errorf("updating feature flag: %w", err)
	}

	// RowsAffected's own error path isn't reachable with a real SQLite
	// driver query result; not exercised in a test without a fault-
	// injecting driver double, per the exception clause in ADR-0004.
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("updating feature flag: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("updating feature flag: %w", ErrNotFound)
	}
	return nil
}

func (r *featureFlagRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM feature_flags WHERE id = ?", id)
	if isForeignKeyConstraintErr(err) {
		return fmt.Errorf("deleting feature flag: %w", ErrInUse)
	}
	if err != nil {
		return fmt.Errorf("deleting feature flag: %w", err)
	}

	// RowsAffected's own error path isn't reachable with a real SQLite
	// driver query result; not exercised in a test without a fault-
	// injecting driver double, per the exception clause in ADR-0004.
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("deleting feature flag: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("deleting feature flag: %w", ErrNotFound)
	}
	return nil
}

func scanFeatureFlag(row *sql.Row) (*model.FeatureFlag, error) {
	f, err := scanFeatureFlagRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("fetching feature flag: %w", err)
	}
	return f, nil
}

func scanFeatureFlagRow(row rowScanner) (*model.FeatureFlag, error) {
	var f model.FeatureFlag
	var description, createdByUser, createdByServiceKey, modifiedByUser, modifiedByServiceKey sql.NullString

	err := row.Scan(
		&f.ID, &f.Key, &f.Name, &description, &f.Type,
		&createdByUser, &createdByServiceKey, &f.CreatedOn,
		&modifiedByUser, &modifiedByServiceKey, &f.ModifiedOn,
	)
	if err != nil {
		return nil, err
	}

	f.Description = stringPtr(description)
	f.CreatedByUserID = stringPtr(createdByUser)
	f.CreatedByServiceKeyID = stringPtr(createdByServiceKey)
	f.ModifiedByUserID = stringPtr(modifiedByUser)
	f.ModifiedByServiceKeyID = stringPtr(modifiedByServiceKey)

	return &f, nil
}
