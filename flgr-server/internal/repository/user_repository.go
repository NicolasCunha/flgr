package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
)

// UserRepository is the data access seam for the users table, per
// docs/business/requirements/0002-user-management.md. Consumed as an
// interface by the service layer so it can be faked in tests.
type UserRepository interface {
	Create(ctx context.Context, u *model.User) error
	GetByID(ctx context.Context, id string) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	List(ctx context.Context, limit, offset int) ([]model.User, int, error)
	Update(ctx context.Context, u *model.User) error
}

type userRepository struct {
	db *sql.DB
}

// NewUserRepository returns a UserRepository backed by db.
func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, u *model.User) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO users (
			id, first_name, last_name, email, password_hash, status,
			created_by_user_id, created_by_service_key_id, created_on,
			modified_by_user_id, modified_by_service_key_id, modified_on
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		u.ID, u.FirstName, u.LastName, u.Email, u.PasswordHash, u.Status,
		nullString(u.CreatedByUserID), nullString(u.CreatedByServiceKeyID), u.CreatedOn,
		nullString(u.ModifiedByUserID), nullString(u.ModifiedByServiceKeyID), u.ModifiedOn,
	)
	if isUniqueConstraintErr(err) {
		return fmt.Errorf("creating user: %w", ErrEmailInUse)
	}
	if err != nil {
		return fmt.Errorf("creating user: %w", err)
	}
	return nil
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	return r.scanOne(r.db.QueryRowContext(ctx, userSelectColumns+" FROM users WHERE id = ?", id))
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	return r.scanOne(r.db.QueryRowContext(ctx, userSelectColumns+" FROM users WHERE email = ?", email))
}

func (r *userRepository) List(ctx context.Context, limit, offset int) ([]model.User, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(1) FROM users").Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting users: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, userSelectColumns+" FROM users ORDER BY last_name, first_name LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("listing users: %w", err)
	}
	defer func() { _ = rows.Close() }()

	// A scan failure here, or rows.Err() below, would require a row that
	// passed SQLite's own column types but still fails Go's Scan, or a
	// connection drop mid-iteration; not reachable in a test without a
	// fault-injecting driver double, per the exception clause in
	// docs/architecture/adr/0004-testing-and-coverage-standards.md.
	var users []model.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning user: %w", err)
		}
		users = append(users, *u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("listing users: %w", err)
	}

	return users, total, nil
}

func (r *userRepository) Update(ctx context.Context, u *model.User) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE users SET
			first_name = ?, last_name = ?, email = ?, password_hash = ?, status = ?,
			modified_by_user_id = ?, modified_by_service_key_id = ?, modified_on = ?
		WHERE id = ?`,
		u.FirstName, u.LastName, u.Email, u.PasswordHash, u.Status,
		nullString(u.ModifiedByUserID), nullString(u.ModifiedByServiceKeyID), u.ModifiedOn,
		u.ID,
	)
	if isUniqueConstraintErr(err) {
		return fmt.Errorf("updating user: %w", ErrEmailInUse)
	}
	if err != nil {
		return fmt.Errorf("updating user: %w", err)
	}

	// RowsAffected's own error path isn't reachable with a real SQLite
	// driver query result; not exercised in a test without a fault-
	// injecting driver double, per the exception clause in
	// docs/architecture/adr/0004-testing-and-coverage-standards.md.
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("updating user: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("updating user: %w", ErrNotFound)
	}
	return nil
}

const userSelectColumns = `SELECT
	id, first_name, last_name, email, password_hash, status,
	created_by_user_id, created_by_service_key_id, created_on,
	modified_by_user_id, modified_by_service_key_id, modified_on`

// rowScanner is satisfied by both *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

func (r *userRepository) scanOne(row *sql.Row) (*model.User, error) {
	u, err := scanUser(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("fetching user: %w", err)
	}
	return u, nil
}

func scanUser(row rowScanner) (*model.User, error) {
	var u model.User
	var createdByUser, createdByServiceKey, modifiedByUser, modifiedByServiceKey sql.NullString

	err := row.Scan(
		&u.ID, &u.FirstName, &u.LastName, &u.Email, &u.PasswordHash, &u.Status,
		&createdByUser, &createdByServiceKey, &u.CreatedOn,
		&modifiedByUser, &modifiedByServiceKey, &u.ModifiedOn,
	)
	if err != nil {
		return nil, err
	}

	u.CreatedByUserID = stringPtr(createdByUser)
	u.CreatedByServiceKeyID = stringPtr(createdByServiceKey)
	u.ModifiedByUserID = stringPtr(modifiedByUser)
	u.ModifiedByServiceKeyID = stringPtr(modifiedByServiceKey)

	return &u, nil
}
