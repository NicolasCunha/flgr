package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
)

func newTestNotificationChannel(featureFlagID, channelType string) *model.NotificationChannel {
	now := time.Now().UTC().Truncate(time.Second)
	return &model.NotificationChannel{
		ID:            uuid.NewString(),
		FeatureFlagID: featureFlagID,
		ChannelType:   channelType,
		Destination:   "https://example.com/webhook",
		Enabled:       true,
		AuditFields:   model.AuditFields{CreatedOn: now, ModifiedOn: now},
	}
}

func TestNotificationChannelRepository_CreateAndGetByID(t *testing.T) {
	db := newTestDB(t)
	repo := NewNotificationChannelRepository(db)
	ctx := context.Background()

	flag := newTestFeatureFlag("channel-repo-flag")
	if err := NewFeatureFlagRepository(db).Create(ctx, flag); err != nil {
		t.Fatalf("Create(flag) returned unexpected error: %v", err)
	}

	ch := newTestNotificationChannel(flag.ID, model.NotificationChannelTypeHTTP)
	ch.SigningSecretEncrypted = []byte("ciphertext")
	if err := repo.Create(ctx, ch); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	got, err := repo.GetByID(ctx, ch.ID)
	if err != nil {
		t.Fatalf("GetByID() returned unexpected error: %v", err)
	}
	if got.FeatureFlagID != flag.ID || got.ChannelType != model.NotificationChannelTypeHTTP || got.Destination != ch.Destination || !got.Enabled {
		t.Errorf("GetByID() = %+v, want matching FeatureFlagID/ChannelType/Destination, Enabled=true", got)
	}
	if string(got.SigningSecretEncrypted) != "ciphertext" {
		t.Errorf("SigningSecretEncrypted = %q, want %q", got.SigningSecretEncrypted, "ciphertext")
	}
}

func TestNotificationChannelRepository_Create_KafkaHasNoSecret(t *testing.T) {
	db := newTestDB(t)
	repo := NewNotificationChannelRepository(db)
	ctx := context.Background()

	flag := newTestFeatureFlag("channel-repo-kafka")
	if err := NewFeatureFlagRepository(db).Create(ctx, flag); err != nil {
		t.Fatalf("Create(flag) returned unexpected error: %v", err)
	}

	ch := newTestNotificationChannel(flag.ID, model.NotificationChannelTypeKafka)
	if err := repo.Create(ctx, ch); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	got, err := repo.GetByID(ctx, ch.ID)
	if err != nil {
		t.Fatalf("GetByID() returned unexpected error: %v", err)
	}
	if got.SigningSecretEncrypted != nil {
		t.Errorf("SigningSecretEncrypted = %v, want nil for a Kafka channel", got.SigningSecretEncrypted)
	}
}

func TestNotificationChannelRepository_Create_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	flag := newTestFeatureFlag("channel-repo-closed")
	if err := NewFeatureFlagRepository(db).Create(context.Background(), flag); err != nil {
		t.Fatalf("Create(flag) returned unexpected error: %v", err)
	}
	repo := NewNotificationChannelRepository(db)
	_ = db.Close()

	err := repo.Create(context.Background(), newTestNotificationChannel(flag.ID, model.NotificationChannelTypeHTTP))
	if err == nil {
		t.Fatal("Create() on closed database expected an error, got nil")
	}
}

func TestNotificationChannelRepository_GetByID_NotFound(t *testing.T) {
	db := newTestDB(t)
	repo := NewNotificationChannelRepository(db)

	_, err := repo.GetByID(context.Background(), "does-not-exist")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByID() error = %v, want ErrNotFound", err)
	}
}

func TestNotificationChannelRepository_GetByID_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewNotificationChannelRepository(db)
	_ = db.Close()

	_, err := repo.GetByID(context.Background(), "irrelevant")
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByID() on closed database error = %v, want a plain non-nil error", err)
	}
}

func TestNotificationChannelRepository_ListByFeatureFlagID(t *testing.T) {
	db := newTestDB(t)
	repo := NewNotificationChannelRepository(db)
	ctx := context.Background()

	flag := newTestFeatureFlag("channel-repo-list")
	if err := NewFeatureFlagRepository(db).Create(ctx, flag); err != nil {
		t.Fatalf("Create(flag) returned unexpected error: %v", err)
	}
	other := newTestFeatureFlag("channel-repo-list-other")
	if err := NewFeatureFlagRepository(db).Create(ctx, other); err != nil {
		t.Fatalf("Create(other flag) returned unexpected error: %v", err)
	}

	ch1 := newTestNotificationChannel(flag.ID, model.NotificationChannelTypeKafka)
	if err := repo.Create(ctx, ch1); err != nil {
		t.Fatalf("Create(ch1) returned unexpected error: %v", err)
	}
	ch2 := newTestNotificationChannel(flag.ID, model.NotificationChannelTypeHTTP)
	if err := repo.Create(ctx, ch2); err != nil {
		t.Fatalf("Create(ch2) returned unexpected error: %v", err)
	}
	// A channel on a different flag must not leak into this list.
	if err := repo.Create(ctx, newTestNotificationChannel(other.ID, model.NotificationChannelTypeKafka)); err != nil {
		t.Fatalf("Create(other channel) returned unexpected error: %v", err)
	}

	got, err := repo.ListByFeatureFlagID(ctx, flag.ID)
	if err != nil {
		t.Fatalf("ListByFeatureFlagID() returned unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(ListByFeatureFlagID()) = %d, want 2", len(got))
	}
	if got[0].ID != ch1.ID || got[1].ID != ch2.ID {
		t.Errorf("ListByFeatureFlagID() = %+v, want ordered [ch1, ch2] by created_on", got)
	}
}

func TestNotificationChannelRepository_ListByFeatureFlagID_Empty(t *testing.T) {
	db := newTestDB(t)
	repo := NewNotificationChannelRepository(db)
	ctx := context.Background()

	flag := newTestFeatureFlag("channel-repo-list-empty")
	if err := NewFeatureFlagRepository(db).Create(ctx, flag); err != nil {
		t.Fatalf("Create(flag) returned unexpected error: %v", err)
	}

	got, err := repo.ListByFeatureFlagID(ctx, flag.ID)
	if err != nil {
		t.Fatalf("ListByFeatureFlagID() returned unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("ListByFeatureFlagID() = %v, want empty", got)
	}
}

func TestNotificationChannelRepository_ListByFeatureFlagID_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewNotificationChannelRepository(db)
	_ = db.Close()

	if _, err := repo.ListByFeatureFlagID(context.Background(), "irrelevant"); err == nil {
		t.Fatal("ListByFeatureFlagID() on closed database expected an error, got nil")
	}
}

func TestNotificationChannelRepository_Update(t *testing.T) {
	db := newTestDB(t)
	userRepo := NewUserRepository(db)
	repo := NewNotificationChannelRepository(db)
	ctx := context.Background()

	actor := newTestUser("channel-editor@example.com")
	if err := userRepo.Create(ctx, actor); err != nil {
		t.Fatalf("Create(actor) returned unexpected error: %v", err)
	}
	flag := newTestFeatureFlag("channel-repo-update")
	if err := NewFeatureFlagRepository(db).Create(ctx, flag); err != nil {
		t.Fatalf("Create(flag) returned unexpected error: %v", err)
	}

	ch := newTestNotificationChannel(flag.ID, model.NotificationChannelTypeHTTP)
	if err := repo.Create(ctx, ch); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	ch.Destination = "https://example.com/updated"
	ch.Enabled = false
	ch.SigningSecretEncrypted = []byte("new-ciphertext")
	ch.ModifiedByUserID = &actor.ID
	if err := repo.Update(ctx, ch); err != nil {
		t.Fatalf("Update() returned unexpected error: %v", err)
	}

	got, err := repo.GetByID(ctx, ch.ID)
	if err != nil {
		t.Fatalf("GetByID() returned unexpected error: %v", err)
	}
	if got.Destination != "https://example.com/updated" || got.Enabled {
		t.Errorf("GetByID() after Update() = %+v, want Destination updated, Enabled=false", got)
	}
	if string(got.SigningSecretEncrypted) != "new-ciphertext" {
		t.Errorf("SigningSecretEncrypted = %q, want %q", got.SigningSecretEncrypted, "new-ciphertext")
	}
}

func TestNotificationChannelRepository_Update_NotFound(t *testing.T) {
	db := newTestDB(t)
	repo := NewNotificationChannelRepository(db)
	ctx := context.Background()

	flag := newTestFeatureFlag("channel-repo-update-notfound")
	if err := NewFeatureFlagRepository(db).Create(ctx, flag); err != nil {
		t.Fatalf("Create(flag) returned unexpected error: %v", err)
	}

	err := repo.Update(ctx, newTestNotificationChannel(flag.ID, model.NotificationChannelTypeKafka))
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Update() error = %v, want ErrNotFound", err)
	}
}

func TestNotificationChannelRepository_Update_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	flag := newTestFeatureFlag("channel-repo-update-closed")
	if err := NewFeatureFlagRepository(db).Create(context.Background(), flag); err != nil {
		t.Fatalf("Create(flag) returned unexpected error: %v", err)
	}
	repo := NewNotificationChannelRepository(db)
	_ = db.Close()

	err := repo.Update(context.Background(), newTestNotificationChannel(flag.ID, model.NotificationChannelTypeKafka))
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("Update() on closed database error = %v, want a plain non-nil error", err)
	}
}

func TestNotificationChannelRepository_Delete(t *testing.T) {
	db := newTestDB(t)
	repo := NewNotificationChannelRepository(db)
	ctx := context.Background()

	flag := newTestFeatureFlag("channel-repo-delete")
	if err := NewFeatureFlagRepository(db).Create(ctx, flag); err != nil {
		t.Fatalf("Create(flag) returned unexpected error: %v", err)
	}
	ch := newTestNotificationChannel(flag.ID, model.NotificationChannelTypeKafka)
	if err := repo.Create(ctx, ch); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	if err := repo.Delete(ctx, ch.ID); err != nil {
		t.Fatalf("Delete() returned unexpected error: %v", err)
	}

	_, err := repo.GetByID(ctx, ch.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByID() after Delete() error = %v, want ErrNotFound", err)
	}
}

func TestNotificationChannelRepository_Delete_NotFound(t *testing.T) {
	db := newTestDB(t)
	repo := NewNotificationChannelRepository(db)

	err := repo.Delete(context.Background(), "does-not-exist")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Delete() error = %v, want ErrNotFound", err)
	}
}

func TestNotificationChannelRepository_Delete_InUse(t *testing.T) {
	db := newTestDB(t)
	channelRepo := NewNotificationChannelRepository(db)
	outboxRepo := NewNotificationOutboxRepository(db)
	ctx := context.Background()

	flag := newTestFeatureFlag("channel-repo-delete-inuse")
	if err := NewFeatureFlagRepository(db).Create(ctx, flag); err != nil {
		t.Fatalf("Create(flag) returned unexpected error: %v", err)
	}
	ch := newTestNotificationChannel(flag.ID, model.NotificationChannelTypeKafka)
	if err := channelRepo.Create(ctx, ch); err != nil {
		t.Fatalf("Create(channel) returned unexpected error: %v", err)
	}
	if err := outboxRepo.Create(ctx, newTestNotificationOutboxEntry(ch.ID)); err != nil {
		t.Fatalf("Create(outbox entry) returned unexpected error: %v", err)
	}

	err := channelRepo.Delete(ctx, ch.ID)
	if !errors.Is(err, ErrInUse) {
		t.Fatalf("Delete() error = %v, want ErrInUse", err)
	}
}

func TestNotificationChannelRepository_Delete_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	flag := newTestFeatureFlag("channel-repo-delete-closed")
	if err := NewFeatureFlagRepository(db).Create(context.Background(), flag); err != nil {
		t.Fatalf("Create(flag) returned unexpected error: %v", err)
	}
	repo := NewNotificationChannelRepository(db)
	ch := newTestNotificationChannel(flag.ID, model.NotificationChannelTypeKafka)
	if err := repo.Create(context.Background(), ch); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}
	_ = db.Close()

	err := repo.Delete(context.Background(), ch.ID)
	if err == nil || errors.Is(err, ErrNotFound) || errors.Is(err, ErrInUse) {
		t.Fatalf("Delete() on closed database error = %v, want a plain non-nil error", err)
	}
}
