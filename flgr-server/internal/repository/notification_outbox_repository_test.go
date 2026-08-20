package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
)

// newTestNotificationOutboxEntry builds a Pending outbox entry referencing
// channelID, for tests that need a valid
// feature_flag_notification_channel_id to satisfy the FK.
func newTestNotificationOutboxEntry(channelID string) *model.NotificationOutboxEntry {
	now := time.Now().UTC().Truncate(time.Second)
	return &model.NotificationOutboxEntry{
		ID:                               uuid.NewString(),
		FeatureFlagNotificationChannelID: channelID,
		Payload:                          `{"event":"feature_flag.killed"}`,
		Status:                           model.NotificationOutboxStatusPending,
		AttemptCount:                     0,
		NextAttemptAt:                    &now,
		CreatedOn:                        now,
	}
}

func TestNotificationOutboxRepository_Create(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	flag := newTestFeatureFlag("outbox-repo-flag")
	if err := NewFeatureFlagRepository(db).Create(ctx, flag); err != nil {
		t.Fatalf("Create(flag) returned unexpected error: %v", err)
	}
	ch := newTestNotificationChannel(flag.ID, model.NotificationChannelTypeKafka)
	if err := NewNotificationChannelRepository(db).Create(ctx, ch); err != nil {
		t.Fatalf("Create(channel) returned unexpected error: %v", err)
	}

	repo := NewNotificationOutboxRepository(db)
	entry := newTestNotificationOutboxEntry(ch.ID)
	if err := repo.Create(ctx, entry); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	// Create has no Get/List yet (per this repository's doc comment — a
	// future delivery-worker phase adds those), so success is verified by
	// querying the row directly rather than through the repository.
	var gotStatus string
	var gotAttempt int
	row := db.QueryRow("SELECT status, attempt_count FROM notification_outbox WHERE id = ?", entry.ID)
	if err := row.Scan(&gotStatus, &gotAttempt); err != nil {
		t.Fatalf("querying created outbox row: %v", err)
	}
	if gotStatus != model.NotificationOutboxStatusPending || gotAttempt != 0 {
		t.Errorf("row = (status=%q, attempt_count=%d), want (Pending, 0)", gotStatus, gotAttempt)
	}
}

func TestNotificationOutboxRepository_Create_WithLastError(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	flag := newTestFeatureFlag("outbox-repo-flag-err")
	if err := NewFeatureFlagRepository(db).Create(ctx, flag); err != nil {
		t.Fatalf("Create(flag) returned unexpected error: %v", err)
	}
	ch := newTestNotificationChannel(flag.ID, model.NotificationChannelTypeHTTP)
	if err := NewNotificationChannelRepository(db).Create(ctx, ch); err != nil {
		t.Fatalf("Create(channel) returned unexpected error: %v", err)
	}

	repo := NewNotificationOutboxRepository(db)
	entry := newTestNotificationOutboxEntry(ch.ID)
	lastErr := "connection refused"
	entry.LastError = &lastErr
	entry.AttemptCount = 1
	entry.Status = model.NotificationOutboxStatusFailed
	if err := repo.Create(ctx, entry); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	var gotLastError string
	row := db.QueryRow("SELECT last_error FROM notification_outbox WHERE id = ?", entry.ID)
	if err := row.Scan(&gotLastError); err != nil {
		t.Fatalf("querying created outbox row: %v", err)
	}
	if gotLastError != lastErr {
		t.Errorf("last_error = %q, want %q", gotLastError, lastErr)
	}
}

func TestNotificationOutboxRepository_Create_UnknownChannel(t *testing.T) {
	db := newTestDB(t)
	repo := NewNotificationOutboxRepository(db)

	err := repo.Create(context.Background(), newTestNotificationOutboxEntry("does-not-exist"))
	if err == nil {
		t.Fatal("Create() with an unknown channel id expected an error (FK violation), got nil")
	}
}

func TestNotificationOutboxRepository_Create_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	flag := newTestFeatureFlag("outbox-repo-flag-closed")
	if err := NewFeatureFlagRepository(db).Create(ctx, flag); err != nil {
		t.Fatalf("Create(flag) returned unexpected error: %v", err)
	}
	ch := newTestNotificationChannel(flag.ID, model.NotificationChannelTypeKafka)
	if err := NewNotificationChannelRepository(db).Create(ctx, ch); err != nil {
		t.Fatalf("Create(channel) returned unexpected error: %v", err)
	}

	repo := NewNotificationOutboxRepository(db)
	_ = db.Close()

	err := repo.Create(context.Background(), newTestNotificationOutboxEntry(ch.ID))
	if err == nil {
		t.Fatal("Create() on closed database expected an error, got nil")
	}
}
