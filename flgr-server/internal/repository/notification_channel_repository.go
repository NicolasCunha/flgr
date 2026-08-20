package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
)

// NotificationChannelRepository is the data access seam for the
// feature_flag_notification_channels table, per
// docs/business/requirements/0006-feature-flag-killswitch.md.
type NotificationChannelRepository interface {
	Create(ctx context.Context, c *model.NotificationChannel) error
	GetByID(ctx context.Context, id string) (*model.NotificationChannel, error)
	ListByFeatureFlagID(ctx context.Context, featureFlagID string) ([]model.NotificationChannel, error)
	Update(ctx context.Context, c *model.NotificationChannel) error
	Delete(ctx context.Context, id string) error
}

type notificationChannelRepository struct {
	db *sql.DB
}

// NewNotificationChannelRepository returns a NotificationChannelRepository
// backed by db.
func NewNotificationChannelRepository(db *sql.DB) NotificationChannelRepository {
	return &notificationChannelRepository{db: db}
}

const notificationChannelSelectColumns = `SELECT
	id, feature_flag_id, channel_type, destination, signing_secret_encrypted, enabled,
	created_by_user_id, created_by_service_key_id, created_on,
	modified_by_user_id, modified_by_service_key_id, modified_on`

func (r *notificationChannelRepository) Create(ctx context.Context, c *model.NotificationChannel) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO feature_flag_notification_channels (
			id, feature_flag_id, channel_type, destination, signing_secret_encrypted, enabled,
			created_by_user_id, created_by_service_key_id, created_on,
			modified_by_user_id, modified_by_service_key_id, modified_on
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.FeatureFlagID, c.ChannelType, c.Destination, c.SigningSecretEncrypted, c.Enabled,
		nullString(c.CreatedByUserID), nullString(c.CreatedByServiceKeyID), c.CreatedOn,
		nullString(c.ModifiedByUserID), nullString(c.ModifiedByServiceKeyID), c.ModifiedOn,
	)
	if err != nil {
		return fmt.Errorf("creating notification channel: %w", err)
	}
	return nil
}

func (r *notificationChannelRepository) GetByID(ctx context.Context, id string) (*model.NotificationChannel, error) {
	return scanNotificationChannel(r.db.QueryRowContext(ctx, notificationChannelSelectColumns+" FROM feature_flag_notification_channels WHERE id = ?", id))
}

func (r *notificationChannelRepository) ListByFeatureFlagID(ctx context.Context, featureFlagID string) ([]model.NotificationChannel, error) {
	rows, err := r.db.QueryContext(ctx,
		notificationChannelSelectColumns+" FROM feature_flag_notification_channels WHERE feature_flag_id = ? ORDER BY created_on",
		featureFlagID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing notification channels: %w", err)
	}
	defer func() { _ = rows.Close() }()

	// A scan failure here, or rows.Err() below, isn't reachable in a test
	// without a fault-injecting driver double, per the exception clause in
	// docs/architecture/adr/0004-testing-and-coverage-standards.md.
	var channels []model.NotificationChannel
	for rows.Next() {
		c, err := scanNotificationChannelRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning notification channel: %w", err)
		}
		channels = append(channels, *c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("listing notification channels: %w", err)
	}

	return channels, nil
}

func (r *notificationChannelRepository) Update(ctx context.Context, c *model.NotificationChannel) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE feature_flag_notification_channels SET
			destination = ?, signing_secret_encrypted = ?, enabled = ?,
			modified_by_user_id = ?, modified_by_service_key_id = ?, modified_on = ?
		WHERE id = ?`,
		c.Destination, c.SigningSecretEncrypted, c.Enabled,
		nullString(c.ModifiedByUserID), nullString(c.ModifiedByServiceKeyID), c.ModifiedOn,
		c.ID,
	)
	if err != nil {
		return fmt.Errorf("updating notification channel: %w", err)
	}

	// RowsAffected's own error path isn't reachable with a real SQLite
	// driver query result; not exercised in a test without a fault-
	// injecting driver double, per the exception clause in ADR-0004.
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("updating notification channel: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("updating notification channel: %w", ErrNotFound)
	}
	return nil
}

func (r *notificationChannelRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM feature_flag_notification_channels WHERE id = ?", id)
	if isForeignKeyConstraintErr(err) {
		return fmt.Errorf("deleting notification channel: %w", ErrInUse)
	}
	if err != nil {
		return fmt.Errorf("deleting notification channel: %w", err)
	}

	// RowsAffected's own error path isn't reachable with a real SQLite
	// driver query result; not exercised in a test without a fault-
	// injecting driver double, per the exception clause in ADR-0004.
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("deleting notification channel: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("deleting notification channel: %w", ErrNotFound)
	}
	return nil
}

func scanNotificationChannel(row *sql.Row) (*model.NotificationChannel, error) {
	c, err := scanNotificationChannelRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("fetching notification channel: %w", err)
	}
	return c, nil
}

func scanNotificationChannelRow(row rowScanner) (*model.NotificationChannel, error) {
	var c model.NotificationChannel
	var createdByUser, createdByServiceKey, modifiedByUser, modifiedByServiceKey sql.NullString

	err := row.Scan(
		&c.ID, &c.FeatureFlagID, &c.ChannelType, &c.Destination, &c.SigningSecretEncrypted, &c.Enabled,
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
