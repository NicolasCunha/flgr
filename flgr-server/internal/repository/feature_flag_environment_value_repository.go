package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
)

// FeatureFlagEnvironmentValueRepository is the data access seam for the
// feature_flag_environment_values table, per
// docs/business/requirements/0005-feature-flag-management.md. No Delete —
// a value row's lifetime is tied to its flag (ON DELETE CASCADE from
// feature_flags), and 0005 never describes clearing a single environment's
// value back to "unconfigured" as its own action.
type FeatureFlagEnvironmentValueRepository interface {
	Create(ctx context.Context, v *model.FeatureFlagEnvironmentValue) error
	GetByFlagAndEnvironment(ctx context.Context, featureFlagID, environmentID string) (*model.FeatureFlagEnvironmentValue, error)
	ListByFlag(ctx context.Context, featureFlagID string) ([]model.FeatureFlagEnvironmentValue, error)
	ListByEnvironment(ctx context.Context, environmentID string) ([]model.FeatureFlagEnvironmentValue, error)
	Update(ctx context.Context, v *model.FeatureFlagEnvironmentValue) error
}

type featureFlagEnvironmentValueRepository struct {
	db *sql.DB
}

// NewFeatureFlagEnvironmentValueRepository returns a
// FeatureFlagEnvironmentValueRepository backed by db.
func NewFeatureFlagEnvironmentValueRepository(db *sql.DB) FeatureFlagEnvironmentValueRepository {
	return &featureFlagEnvironmentValueRepository{db: db}
}

const featureFlagEnvironmentValueSelectColumns = `SELECT
	id, feature_flag_id, environment_id, enabled, value,
	created_by_user_id, created_by_service_key_id, created_on,
	modified_by_user_id, modified_by_service_key_id, modified_on`

func (r *featureFlagEnvironmentValueRepository) Create(ctx context.Context, v *model.FeatureFlagEnvironmentValue) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO feature_flag_environment_values (
			id, feature_flag_id, environment_id, enabled, value,
			created_by_user_id, created_by_service_key_id, created_on,
			modified_by_user_id, modified_by_service_key_id, modified_on
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		v.ID, v.FeatureFlagID, v.EnvironmentID, v.Enabled, nullString(v.Value),
		nullString(v.CreatedByUserID), nullString(v.CreatedByServiceKeyID), v.CreatedOn,
		nullString(v.ModifiedByUserID), nullString(v.ModifiedByServiceKeyID), v.ModifiedOn,
	)
	if isUniqueConstraintErr(err) {
		return fmt.Errorf("creating feature flag environment value: %w", ErrFeatureFlagEnvironmentValueExists)
	}
	if err != nil {
		return fmt.Errorf("creating feature flag environment value: %w", err)
	}
	return nil
}

func (r *featureFlagEnvironmentValueRepository) GetByFlagAndEnvironment(ctx context.Context, featureFlagID, environmentID string) (*model.FeatureFlagEnvironmentValue, error) {
	row := r.db.QueryRowContext(ctx,
		featureFlagEnvironmentValueSelectColumns+" FROM feature_flag_environment_values WHERE feature_flag_id = ? AND environment_id = ?",
		featureFlagID, environmentID,
	)
	v, err := scanFeatureFlagEnvironmentValueRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("fetching feature flag environment value: %w", err)
	}
	return v, nil
}

func (r *featureFlagEnvironmentValueRepository) ListByFlag(ctx context.Context, featureFlagID string) ([]model.FeatureFlagEnvironmentValue, error) {
	rows, err := r.db.QueryContext(ctx,
		featureFlagEnvironmentValueSelectColumns+" FROM feature_flag_environment_values WHERE feature_flag_id = ? ORDER BY environment_id",
		featureFlagID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing feature flag environment values: %w", err)
	}
	defer func() { _ = rows.Close() }()

	// A scan failure here, or rows.Err() below, isn't reachable in a test
	// without a fault-injecting driver double, per the exception clause in
	// docs/architecture/adr/0004-testing-and-coverage-standards.md.
	var values []model.FeatureFlagEnvironmentValue
	for rows.Next() {
		v, err := scanFeatureFlagEnvironmentValueRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning feature flag environment value: %w", err)
		}
		values = append(values, *v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("listing feature flag environment values: %w", err)
	}

	return values, nil
}

func (r *featureFlagEnvironmentValueRepository) ListByEnvironment(ctx context.Context, environmentID string) ([]model.FeatureFlagEnvironmentValue, error) {
	rows, err := r.db.QueryContext(ctx,
		featureFlagEnvironmentValueSelectColumns+" FROM feature_flag_environment_values WHERE environment_id = ? ORDER BY feature_flag_id",
		environmentID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing feature flag environment values: %w", err)
	}
	defer func() { _ = rows.Close() }()

	// A scan failure here, or rows.Err() below, isn't reachable in a test
	// without a fault-injecting driver double, per the exception clause in
	// docs/architecture/adr/0004-testing-and-coverage-standards.md.
	var values []model.FeatureFlagEnvironmentValue
	for rows.Next() {
		v, err := scanFeatureFlagEnvironmentValueRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning feature flag environment value: %w", err)
		}
		values = append(values, *v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("listing feature flag environment values: %w", err)
	}

	return values, nil
}

func (r *featureFlagEnvironmentValueRepository) Update(ctx context.Context, v *model.FeatureFlagEnvironmentValue) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE feature_flag_environment_values SET
			enabled = ?, value = ?,
			modified_by_user_id = ?, modified_by_service_key_id = ?, modified_on = ?
		WHERE id = ?`,
		v.Enabled, nullString(v.Value),
		nullString(v.ModifiedByUserID), nullString(v.ModifiedByServiceKeyID), v.ModifiedOn,
		v.ID,
	)
	if err != nil {
		return fmt.Errorf("updating feature flag environment value: %w", err)
	}

	// RowsAffected's own error path isn't reachable with a real SQLite
	// driver query result; not exercised in a test without a fault-
	// injecting driver double, per the exception clause in ADR-0004.
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("updating feature flag environment value: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("updating feature flag environment value: %w", ErrNotFound)
	}
	return nil
}

func scanFeatureFlagEnvironmentValueRow(row rowScanner) (*model.FeatureFlagEnvironmentValue, error) {
	var v model.FeatureFlagEnvironmentValue
	var value, createdByUser, createdByServiceKey, modifiedByUser, modifiedByServiceKey sql.NullString

	err := row.Scan(
		&v.ID, &v.FeatureFlagID, &v.EnvironmentID, &v.Enabled, &value,
		&createdByUser, &createdByServiceKey, &v.CreatedOn,
		&modifiedByUser, &modifiedByServiceKey, &v.ModifiedOn,
	)
	if err != nil {
		return nil, err
	}

	v.Value = stringPtr(value)
	v.CreatedByUserID = stringPtr(createdByUser)
	v.CreatedByServiceKeyID = stringPtr(createdByServiceKey)
	v.ModifiedByUserID = stringPtr(modifiedByUser)
	v.ModifiedByServiceKeyID = stringPtr(modifiedByServiceKey)

	return &v, nil
}
