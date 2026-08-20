package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
)

// NotificationOutboxRepository is the data access seam for the
// notification_outbox table, per
// docs/architecture/adr/0009-kafka-and-webhook-notification-delivery.md.
// Enqueue-only for this phase: the Killswitch trigger writes one row per
// enabled channel; actually polling and delivering pending rows is a
// future phase's background worker, which will add its own read/update
// methods (e.g. claiming pending rows, marking Delivered/Failed) when it
// lands — not guessed at here.
type NotificationOutboxRepository interface {
	Create(ctx context.Context, entry *model.NotificationOutboxEntry) error
}

type notificationOutboxRepository struct {
	db *sql.DB
}

// NewNotificationOutboxRepository returns a NotificationOutboxRepository
// backed by db.
func NewNotificationOutboxRepository(db *sql.DB) NotificationOutboxRepository {
	return &notificationOutboxRepository{db: db}
}

func (r *notificationOutboxRepository) Create(ctx context.Context, entry *model.NotificationOutboxEntry) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO notification_outbox (
			id, feature_flag_notification_channel_id, payload, status,
			attempt_count, next_attempt_at, last_error, created_on, delivered_on
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ID, entry.FeatureFlagNotificationChannelID, entry.Payload, entry.Status,
		entry.AttemptCount, entry.NextAttemptAt, nullString(entry.LastError),
		entry.CreatedOn, entry.DeliveredOn,
	)
	if err != nil {
		return fmt.Errorf("creating notification outbox entry: %w", err)
	}
	return nil
}
