package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
)

// ProfileRepository is the data access seam for the profiles table, per
// docs/business/requirements/0003-profile-and-permission-management.md.
type ProfileRepository interface {
	Create(ctx context.Context, p *model.Profile) error
	GetByID(ctx context.Context, id string) (*model.Profile, error)
	GetByName(ctx context.Context, name string) (*model.Profile, error)
	List(ctx context.Context, limit, offset int) ([]model.Profile, int, error)
	// ListAll returns every profile, unpaginated — used internally to
	// check the "at least one profile holds the full permission catalog"
	// invariant (see service.ProfileService.Delete), not exposed via HTTP.
	ListAll(ctx context.Context) ([]model.Profile, error)
	Update(ctx context.Context, p *model.Profile) error
	// Delete hard-deletes the profile. Profiles carry no created_by/
	// modified_by references from other tables (see
	// docs/architecture/adr/0005-audit-columns-and-soft-delete-convention.md),
	// so unlike users they have no soft-delete requirement.
	Delete(ctx context.Context, id string) error
}

type profileRepository struct {
	db *sql.DB
}

// NewProfileRepository returns a ProfileRepository backed by db.
func NewProfileRepository(db *sql.DB) ProfileRepository {
	return &profileRepository{db: db}
}

const profileSelectColumns = `SELECT
	id, name, description, is_system,
	created_by_user_id, created_by_service_key_id, created_on,
	modified_by_user_id, modified_by_service_key_id, modified_on`

func (r *profileRepository) Create(ctx context.Context, p *model.Profile) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO profiles (
			id, name, description, is_system,
			created_by_user_id, created_by_service_key_id, created_on,
			modified_by_user_id, modified_by_service_key_id, modified_on
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.Name, nullString(p.Description), p.IsSystem,
		nullString(p.CreatedByUserID), nullString(p.CreatedByServiceKeyID), p.CreatedOn,
		nullString(p.ModifiedByUserID), nullString(p.ModifiedByServiceKeyID), p.ModifiedOn,
	)
	if isUniqueConstraintErr(err) {
		return fmt.Errorf("creating profile: %w", ErrProfileNameInUse)
	}
	if err != nil {
		return fmt.Errorf("creating profile: %w", err)
	}
	return nil
}

func (r *profileRepository) GetByID(ctx context.Context, id string) (*model.Profile, error) {
	return scanProfile(r.db.QueryRowContext(ctx, profileSelectColumns+" FROM profiles WHERE id = ?", id))
}

func (r *profileRepository) GetByName(ctx context.Context, name string) (*model.Profile, error) {
	return scanProfile(r.db.QueryRowContext(ctx, profileSelectColumns+" FROM profiles WHERE name = ?", name))
}

func (r *profileRepository) List(ctx context.Context, limit, offset int) ([]model.Profile, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(1) FROM profiles").Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting profiles: %w", err)
	}

	// A failure here specifically (count succeeds, the page query fails)
	// isn't reachable in a test without a fault-injecting driver double,
	// per the exception clause in docs/architecture/adr/0004-testing-and-coverage-standards.md.
	profiles, err := queryProfiles(ctx, r.db, profileSelectColumns+" FROM profiles ORDER BY name LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return profiles, total, nil
}

func (r *profileRepository) ListAll(ctx context.Context) ([]model.Profile, error) {
	return queryProfiles(ctx, r.db, profileSelectColumns+" FROM profiles ORDER BY name")
}

func (r *profileRepository) Update(ctx context.Context, p *model.Profile) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE profiles SET
			name = ?, description = ?,
			modified_by_user_id = ?, modified_by_service_key_id = ?, modified_on = ?
		WHERE id = ?`,
		p.Name, nullString(p.Description),
		nullString(p.ModifiedByUserID), nullString(p.ModifiedByServiceKeyID), p.ModifiedOn,
		p.ID,
	)
	if isUniqueConstraintErr(err) {
		return fmt.Errorf("updating profile: %w", ErrProfileNameInUse)
	}
	if err != nil {
		return fmt.Errorf("updating profile: %w", err)
	}

	// RowsAffected's own error path isn't reachable with a real SQLite
	// driver query result; not exercised in a test without a fault-
	// injecting driver double, per the exception clause in
	// docs/architecture/adr/0004-testing-and-coverage-standards.md.
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("updating profile: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("updating profile: %w", ErrNotFound)
	}
	return nil
}

func (r *profileRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM profiles WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting profile: %w", err)
	}

	// RowsAffected's own error path isn't reachable with a real SQLite
	// driver query result; not exercised in a test without a fault-
	// injecting driver double, per the exception clause in ADR-0004.
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("deleting profile: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("deleting profile: %w", ErrNotFound)
	}
	return nil
}

func queryProfiles(ctx context.Context, db *sql.DB, query string, args ...any) ([]model.Profile, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing profiles: %w", err)
	}
	defer func() { _ = rows.Close() }()

	// A scan failure here, or rows.Err() below, isn't reachable in a test
	// without a fault-injecting driver double, per the exception clause
	// in ADR-0004.
	var profiles []model.Profile
	for rows.Next() {
		p, err := scanProfileRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning profile: %w", err)
		}
		profiles = append(profiles, *p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("listing profiles: %w", err)
	}

	return profiles, nil
}

func scanProfile(row *sql.Row) (*model.Profile, error) {
	p, err := scanProfileRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("fetching profile: %w", err)
	}
	return p, nil
}

func scanProfileRow(row rowScanner) (*model.Profile, error) {
	var p model.Profile
	var description, createdByUser, createdByServiceKey, modifiedByUser, modifiedByServiceKey sql.NullString

	err := row.Scan(
		&p.ID, &p.Name, &description, &p.IsSystem,
		&createdByUser, &createdByServiceKey, &p.CreatedOn,
		&modifiedByUser, &modifiedByServiceKey, &p.ModifiedOn,
	)
	if err != nil {
		return nil, err
	}

	p.Description = stringPtr(description)
	p.CreatedByUserID = stringPtr(createdByUser)
	p.CreatedByServiceKeyID = stringPtr(createdByServiceKey)
	p.ModifiedByUserID = stringPtr(modifiedByUser)
	p.ModifiedByServiceKeyID = stringPtr(modifiedByServiceKey)

	return &p, nil
}
