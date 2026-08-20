package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/repository"
)

func newNotificationChannelServiceMocks() (*mockNotificationChannelRepository, *mockFeatureFlagRepository) {
	return new(mockNotificationChannelRepository), new(mockFeatureFlagRepository)
}

func newNotificationChannelService(channels *mockNotificationChannelRepository, flags *mockFeatureFlagRepository) *NotificationChannelService {
	return NewNotificationChannelService(channels, flags, validTestEncryptionKey())
}

// --- Create ---

func TestNotificationChannelService_Create_NilActor(t *testing.T) {
	channels, flags := newNotificationChannelServiceMocks()
	svc := newNotificationChannelService(channels, flags)

	_, _, err := svc.Create(context.Background(), nil, "f1", CreateNotificationChannelInput{ChannelType: model.NotificationChannelTypeKafka, Destination: "topic"})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Create() error = %v, want ErrForbidden", err)
	}
}

func TestNotificationChannelService_Create_FlagNotFound(t *testing.T) {
	channels, flags := newNotificationChannelServiceMocks()
	flags.On("GetByID", mock.Anything, "missing").Return(nil, repository.ErrNotFound)

	svc := newNotificationChannelService(channels, flags)
	actor := &model.User{ID: "actor-id"}
	_, _, err := svc.Create(context.Background(), actor, "missing", CreateNotificationChannelInput{ChannelType: model.NotificationChannelTypeKafka, Destination: "topic"})
	if !errors.Is(err, ErrFeatureFlagNotFound) {
		t.Fatalf("Create() error = %v, want ErrFeatureFlagNotFound", err)
	}
}

func TestNotificationChannelService_Create_FlagLookupError(t *testing.T) {
	channels, flags := newNotificationChannelServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(nil, errors.New("db down"))

	svc := newNotificationChannelService(channels, flags)
	actor := &model.User{ID: "actor-id"}
	_, _, err := svc.Create(context.Background(), actor, "f1", CreateNotificationChannelInput{ChannelType: model.NotificationChannelTypeKafka, Destination: "topic"})
	if err == nil || errors.Is(err, ErrFeatureFlagNotFound) {
		t.Fatalf("Create() error = %v, want a generic wrapped error", err)
	}
}

func TestNotificationChannelService_Create_InvalidChannelType(t *testing.T) {
	channels, flags := newNotificationChannelServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1"}, nil)

	svc := newNotificationChannelService(channels, flags)
	actor := &model.User{ID: "actor-id"}
	_, _, err := svc.Create(context.Background(), actor, "f1", CreateNotificationChannelInput{ChannelType: "Carrier Pigeon", Destination: "topic"})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Create() error = %v, want ErrValidation", err)
	}
	channels.AssertNotCalled(t, "Create")
}

func TestNotificationChannelService_Create_BlankDestination(t *testing.T) {
	channels, flags := newNotificationChannelServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1"}, nil)

	svc := newNotificationChannelService(channels, flags)
	actor := &model.User{ID: "actor-id"}
	_, _, err := svc.Create(context.Background(), actor, "f1", CreateNotificationChannelInput{ChannelType: model.NotificationChannelTypeKafka, Destination: "   "})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Create() error = %v, want ErrValidation", err)
	}
	channels.AssertNotCalled(t, "Create")
}

func TestNotificationChannelService_Create_Kafka_NoSecret(t *testing.T) {
	fixedNow(t)
	fixedID(t, "channel-1")
	channels, flags := newNotificationChannelServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1"}, nil)
	channels.On("Create", mock.Anything, mock.MatchedBy(func(c *model.NotificationChannel) bool {
		return c.ID == "channel-1" && c.ChannelType == model.NotificationChannelTypeKafka && c.SigningSecretEncrypted == nil && c.Enabled
	})).Return(nil)

	svc := newNotificationChannelService(channels, flags)
	actor := &model.User{ID: "actor-id"}
	ch, secret, err := svc.Create(context.Background(), actor, "f1", CreateNotificationChannelInput{ChannelType: model.NotificationChannelTypeKafka, Destination: "  topic  "})
	if err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}
	if secret != "" {
		t.Errorf("secret = %q, want empty for a Kafka channel", secret)
	}
	if ch.Destination != "topic" {
		t.Errorf("Destination = %q, want trimmed %q", ch.Destination, "topic")
	}
}

func TestNotificationChannelService_Create_HTTP_GeneratesEncryptedSecret(t *testing.T) {
	fixedNow(t)
	fixedID(t, "channel-2")
	channels, flags := newNotificationChannelServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1"}, nil)
	var captured *model.NotificationChannel
	channels.On("Create", mock.Anything, mock.MatchedBy(func(c *model.NotificationChannel) bool {
		captured = c
		return c.ChannelType == model.NotificationChannelTypeHTTP
	})).Return(nil)

	svc := newNotificationChannelService(channels, flags)
	actor := &model.User{ID: "actor-id"}
	ch, secret, err := svc.Create(context.Background(), actor, "f1", CreateNotificationChannelInput{ChannelType: model.NotificationChannelTypeHTTP, Destination: "https://example.com/hook"})
	if err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}
	if secret == "" {
		t.Fatal("secret is empty, want a generated plaintext signing secret for an HTTP channel")
	}
	if len(ch.SigningSecretEncrypted) == 0 {
		t.Fatal("SigningSecretEncrypted is empty, want it populated (encrypted) on the persisted channel")
	}
	if string(ch.SigningSecretEncrypted) == secret {
		t.Error("SigningSecretEncrypted must not equal the plaintext secret")
	}
	if captured == nil || len(captured.SigningSecretEncrypted) == 0 {
		t.Fatal("Create() was not called with an encrypted secret set")
	}

	// Round-trip: decrypting what was persisted must yield the plaintext
	// returned to the caller.
	decrypted, err := decryptSecret(validTestEncryptionKey(), captured.SigningSecretEncrypted)
	if err != nil {
		t.Fatalf("decrypting persisted secret returned unexpected error: %v", err)
	}
	if decrypted != secret {
		t.Errorf("decrypted persisted secret = %q, want returned secret %q", decrypted, secret)
	}
}

func TestNotificationChannelService_Create_HTTP_GenerateTokenError(t *testing.T) {
	channels, flags := newNotificationChannelServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1"}, nil)

	original := randRead
	randRead = func(b []byte) (int, error) { return 0, errors.New("entropy source unavailable") }
	t.Cleanup(func() { randRead = original })

	svc := newNotificationChannelService(channels, flags)
	actor := &model.User{ID: "actor-id"}
	_, _, err := svc.Create(context.Background(), actor, "f1", CreateNotificationChannelInput{ChannelType: model.NotificationChannelTypeHTTP, Destination: "https://example.com"})
	if err == nil {
		t.Fatal("Create() expected an error when generating the signing secret fails, got nil")
	}
	channels.AssertNotCalled(t, "Create")
}

func TestNotificationChannelService_Create_HTTP_EncryptError(t *testing.T) {
	channels, flags := newNotificationChannelServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1"}, nil)

	// generateToken()'s randRead call must succeed (so a plaintext secret
	// is produced); the second randRead call, inside encryptSecret's nonce
	// generation, is the one that fails.
	original := randRead
	calls := 0
	randRead = func(b []byte) (int, error) {
		calls++
		if calls == 1 {
			return original(b)
		}
		return 0, errors.New("entropy source unavailable")
	}
	t.Cleanup(func() { randRead = original })

	svc := newNotificationChannelService(channels, flags)
	actor := &model.User{ID: "actor-id"}
	_, _, err := svc.Create(context.Background(), actor, "f1", CreateNotificationChannelInput{ChannelType: model.NotificationChannelTypeHTTP, Destination: "https://example.com"})
	if err == nil {
		t.Fatal("Create() expected an error when encrypting the signing secret fails, got nil")
	}
	channels.AssertNotCalled(t, "Create")
}

func TestNotificationChannelService_Create_RepositoryError(t *testing.T) {
	channels, flags := newNotificationChannelServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1"}, nil)
	channels.On("Create", mock.Anything, mock.Anything).Return(errors.New("disk full"))

	svc := newNotificationChannelService(channels, flags)
	actor := &model.User{ID: "actor-id"}
	_, _, err := svc.Create(context.Background(), actor, "f1", CreateNotificationChannelInput{ChannelType: model.NotificationChannelTypeKafka, Destination: "topic"})
	if err == nil {
		t.Fatal("Create() expected an error, got nil")
	}
}

// --- List ---

func TestNotificationChannelService_List_Success(t *testing.T) {
	channels, flags := newNotificationChannelServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1"}, nil)

	key := validTestEncryptionKey()
	encrypted, err := encryptSecret(key, "hmac-secret")
	if err != nil {
		t.Fatalf("encryptSecret() returned unexpected error: %v", err)
	}
	channels.On("ListByFeatureFlagID", mock.Anything, "f1").Return([]model.NotificationChannel{
		{ID: "kafka-1", ChannelType: model.NotificationChannelTypeKafka},
		{ID: "http-1", ChannelType: model.NotificationChannelTypeHTTP, SigningSecretEncrypted: encrypted},
	}, nil)

	svc := newNotificationChannelService(channels, flags)
	got, secrets, err := svc.List(context.Background(), "f1")
	if err != nil {
		t.Fatalf("List() returned unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(got))
	}
	if secrets["kafka-1"] != "" {
		t.Errorf("secrets[kafka-1] = %q, want empty for a Kafka channel", secrets["kafka-1"])
	}
	if secrets["http-1"] != "hmac-secret" {
		t.Errorf("secrets[http-1] = %q, want the decrypted plaintext %q (List must return it, not just Create)", secrets["http-1"], "hmac-secret")
	}
}

func TestNotificationChannelService_List_FlagNotFound(t *testing.T) {
	channels, flags := newNotificationChannelServiceMocks()
	flags.On("GetByID", mock.Anything, "missing").Return(nil, repository.ErrNotFound)

	svc := newNotificationChannelService(channels, flags)
	_, _, err := svc.List(context.Background(), "missing")
	if !errors.Is(err, ErrFeatureFlagNotFound) {
		t.Fatalf("List() error = %v, want ErrFeatureFlagNotFound", err)
	}
}

func TestNotificationChannelService_List_RepositoryError(t *testing.T) {
	channels, flags := newNotificationChannelServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1"}, nil)
	channels.On("ListByFeatureFlagID", mock.Anything, "f1").Return(nil, errors.New("db down"))

	svc := newNotificationChannelService(channels, flags)
	_, _, err := svc.List(context.Background(), "f1")
	if err == nil {
		t.Fatal("List() expected an error, got nil")
	}
}

func TestNotificationChannelService_List_DecryptError(t *testing.T) {
	channels, flags := newNotificationChannelServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1"}, nil)
	channels.On("ListByFeatureFlagID", mock.Anything, "f1").Return([]model.NotificationChannel{
		{ID: "http-1", ChannelType: model.NotificationChannelTypeHTTP, SigningSecretEncrypted: []byte("not-valid-ciphertext")},
	}, nil)

	svc := newNotificationChannelService(channels, flags)
	_, _, err := svc.List(context.Background(), "f1")
	if err == nil {
		t.Fatal("List() expected an error when a stored secret fails to decrypt, got nil")
	}
}

// --- Update ---

func TestNotificationChannelService_Update_NilActor(t *testing.T) {
	channels, flags := newNotificationChannelServiceMocks()
	svc := newNotificationChannelService(channels, flags)

	_, _, err := svc.Update(context.Background(), nil, "f1", "c1", UpdateNotificationChannelInput{})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Update() error = %v, want ErrForbidden", err)
	}
}

func TestNotificationChannelService_Update_FlagNotFound(t *testing.T) {
	channels, flags := newNotificationChannelServiceMocks()
	flags.On("GetByID", mock.Anything, "missing").Return(nil, repository.ErrNotFound)

	svc := newNotificationChannelService(channels, flags)
	actor := &model.User{ID: "actor-id"}
	_, _, err := svc.Update(context.Background(), actor, "missing", "c1", UpdateNotificationChannelInput{})
	if !errors.Is(err, ErrFeatureFlagNotFound) {
		t.Fatalf("Update() error = %v, want ErrFeatureFlagNotFound", err)
	}
}

// IDOR guard: a channel that exists but belongs to a different flag than
// the one in the URL must 404, not leak across flags.
func TestNotificationChannelService_Update_ChannelBelongsToDifferentFlag(t *testing.T) {
	channels, flags := newNotificationChannelServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1"}, nil)
	channels.On("GetByID", mock.Anything, "c1").Return(&model.NotificationChannel{ID: "c1", FeatureFlagID: "other-flag"}, nil)

	svc := newNotificationChannelService(channels, flags)
	actor := &model.User{ID: "actor-id"}
	_, _, err := svc.Update(context.Background(), actor, "f1", "c1", UpdateNotificationChannelInput{})
	if !errors.Is(err, ErrNotificationChannelNotFound) {
		t.Fatalf("Update() error = %v, want ErrNotificationChannelNotFound (cross-flag id must 404)", err)
	}
	channels.AssertNotCalled(t, "Update")
}

func TestNotificationChannelService_Update_ChannelNotFound(t *testing.T) {
	channels, flags := newNotificationChannelServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1"}, nil)
	channels.On("GetByID", mock.Anything, "missing").Return(nil, repository.ErrNotFound)

	svc := newNotificationChannelService(channels, flags)
	actor := &model.User{ID: "actor-id"}
	_, _, err := svc.Update(context.Background(), actor, "f1", "missing", UpdateNotificationChannelInput{})
	if !errors.Is(err, ErrNotificationChannelNotFound) {
		t.Fatalf("Update() error = %v, want ErrNotificationChannelNotFound", err)
	}
}

func TestNotificationChannelService_Update_ChannelLookupError(t *testing.T) {
	channels, flags := newNotificationChannelServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1"}, nil)
	channels.On("GetByID", mock.Anything, "c1").Return(nil, errors.New("db down"))

	svc := newNotificationChannelService(channels, flags)
	actor := &model.User{ID: "actor-id"}
	_, _, err := svc.Update(context.Background(), actor, "f1", "c1", UpdateNotificationChannelInput{})
	if err == nil || errors.Is(err, ErrNotificationChannelNotFound) {
		t.Fatalf("Update() error = %v, want a generic wrapped error", err)
	}
}

func TestNotificationChannelService_Update_BlankDestination(t *testing.T) {
	channels, flags := newNotificationChannelServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1"}, nil)
	channels.On("GetByID", mock.Anything, "c1").Return(&model.NotificationChannel{ID: "c1", FeatureFlagID: "f1", Destination: "topic"}, nil)

	blank := "   "
	svc := newNotificationChannelService(channels, flags)
	actor := &model.User{ID: "actor-id"}
	_, _, err := svc.Update(context.Background(), actor, "f1", "c1", UpdateNotificationChannelInput{Destination: &blank})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Update() error = %v, want ErrValidation", err)
	}
	channels.AssertNotCalled(t, "Update")
}

func TestNotificationChannelService_Update_PartialUpdate_EnabledOnly(t *testing.T) {
	fixedNow(t)
	channels, flags := newNotificationChannelServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1"}, nil)
	channels.On("GetByID", mock.Anything, "c1").Return(&model.NotificationChannel{ID: "c1", FeatureFlagID: "f1", Destination: "topic", Enabled: true}, nil)
	channels.On("Update", mock.Anything, mock.MatchedBy(func(c *model.NotificationChannel) bool {
		return c.Destination == "topic" && !c.Enabled
	})).Return(nil)

	disabled := false
	svc := newNotificationChannelService(channels, flags)
	actor := &model.User{ID: "actor-id"}
	got, secret, err := svc.Update(context.Background(), actor, "f1", "c1", UpdateNotificationChannelInput{Enabled: &disabled})
	if err != nil {
		t.Fatalf("Update() returned unexpected error: %v", err)
	}
	if got.Enabled {
		t.Error("Enabled = true, want false")
	}
	if secret != "" {
		t.Errorf("secret = %q, want empty (no secret on this Kafka-typed channel)", secret)
	}
}

func TestNotificationChannelService_Update_ReturnsDecryptedSecret(t *testing.T) {
	fixedNow(t)
	channels, flags := newNotificationChannelServiceMocks()
	key := validTestEncryptionKey()
	encrypted, err := encryptSecret(key, "hmac-secret")
	if err != nil {
		t.Fatalf("encryptSecret() returned unexpected error: %v", err)
	}
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1"}, nil)
	channels.On("GetByID", mock.Anything, "c1").Return(&model.NotificationChannel{
		ID: "c1", FeatureFlagID: "f1", Destination: "https://example.com", ChannelType: model.NotificationChannelTypeHTTP, SigningSecretEncrypted: encrypted,
	}, nil)
	channels.On("Update", mock.Anything, mock.Anything).Return(nil)

	newDest := "https://example.com/v2"
	svc := newNotificationChannelService(channels, flags)
	actor := &model.User{ID: "actor-id"}
	_, secret, err := svc.Update(context.Background(), actor, "f1", "c1", UpdateNotificationChannelInput{Destination: &newDest})
	if err != nil {
		t.Fatalf("Update() returned unexpected error: %v", err)
	}
	if secret != "hmac-secret" {
		t.Errorf("secret = %q, want the decrypted %q (Update must return it too, not just Create)", secret, "hmac-secret")
	}
}

// TestNotificationChannelService_Update_SecretDecryptError covers the
// post-Update secretFor() error branch: the channel's persisted secret
// bytes fail to decrypt (e.g. corrupted at rest), which must surface as an
// error rather than a silently empty secret.
func TestNotificationChannelService_Update_SecretDecryptError(t *testing.T) {
	fixedNow(t)
	channels, flags := newNotificationChannelServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1"}, nil)
	channels.On("GetByID", mock.Anything, "c1").Return(&model.NotificationChannel{
		ID: "c1", FeatureFlagID: "f1", Destination: "https://example.com",
		ChannelType: model.NotificationChannelTypeHTTP, SigningSecretEncrypted: []byte("not-valid-ciphertext"),
	}, nil)
	channels.On("Update", mock.Anything, mock.Anything).Return(nil)

	svc := newNotificationChannelService(channels, flags)
	actor := &model.User{ID: "actor-id"}
	_, _, err := svc.Update(context.Background(), actor, "f1", "c1", UpdateNotificationChannelInput{})
	if err == nil {
		t.Fatal("Update() expected an error when the persisted secret fails to decrypt, got nil")
	}
}

func TestNotificationChannelService_Update_RepositoryNotFound(t *testing.T) {
	channels, flags := newNotificationChannelServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1"}, nil)
	channels.On("GetByID", mock.Anything, "c1").Return(&model.NotificationChannel{ID: "c1", FeatureFlagID: "f1", Destination: "topic"}, nil)
	channels.On("Update", mock.Anything, mock.Anything).Return(repository.ErrNotFound)

	svc := newNotificationChannelService(channels, flags)
	actor := &model.User{ID: "actor-id"}
	_, _, err := svc.Update(context.Background(), actor, "f1", "c1", UpdateNotificationChannelInput{})
	if !errors.Is(err, ErrNotificationChannelNotFound) {
		t.Fatalf("Update() error = %v, want ErrNotificationChannelNotFound", err)
	}
}

func TestNotificationChannelService_Update_RepositoryError(t *testing.T) {
	channels, flags := newNotificationChannelServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1"}, nil)
	channels.On("GetByID", mock.Anything, "c1").Return(&model.NotificationChannel{ID: "c1", FeatureFlagID: "f1", Destination: "topic"}, nil)
	channels.On("Update", mock.Anything, mock.Anything).Return(errors.New("disk full"))

	svc := newNotificationChannelService(channels, flags)
	actor := &model.User{ID: "actor-id"}
	_, _, err := svc.Update(context.Background(), actor, "f1", "c1", UpdateNotificationChannelInput{})
	if err == nil {
		t.Fatal("Update() expected an error, got nil")
	}
}

// --- Delete ---

func TestNotificationChannelService_Delete_NilActor(t *testing.T) {
	channels, flags := newNotificationChannelServiceMocks()
	svc := newNotificationChannelService(channels, flags)

	err := svc.Delete(context.Background(), nil, "f1", "c1")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Delete() error = %v, want ErrForbidden", err)
	}
}

func TestNotificationChannelService_Delete_FlagNotFound(t *testing.T) {
	channels, flags := newNotificationChannelServiceMocks()
	flags.On("GetByID", mock.Anything, "missing").Return(nil, repository.ErrNotFound)

	svc := newNotificationChannelService(channels, flags)
	actor := &model.User{ID: "actor-id"}
	err := svc.Delete(context.Background(), actor, "missing", "c1")
	if !errors.Is(err, ErrFeatureFlagNotFound) {
		t.Fatalf("Delete() error = %v, want ErrFeatureFlagNotFound", err)
	}
}

func TestNotificationChannelService_Delete_ChannelBelongsToDifferentFlag(t *testing.T) {
	channels, flags := newNotificationChannelServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1"}, nil)
	channels.On("GetByID", mock.Anything, "c1").Return(&model.NotificationChannel{ID: "c1", FeatureFlagID: "other-flag"}, nil)

	svc := newNotificationChannelService(channels, flags)
	actor := &model.User{ID: "actor-id"}
	err := svc.Delete(context.Background(), actor, "f1", "c1")
	if !errors.Is(err, ErrNotificationChannelNotFound) {
		t.Fatalf("Delete() error = %v, want ErrNotificationChannelNotFound (cross-flag id must 404)", err)
	}
	channels.AssertNotCalled(t, "Delete")
}

func TestNotificationChannelService_Delete_Success(t *testing.T) {
	channels, flags := newNotificationChannelServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1"}, nil)
	channels.On("GetByID", mock.Anything, "c1").Return(&model.NotificationChannel{ID: "c1", FeatureFlagID: "f1"}, nil)
	channels.On("Delete", mock.Anything, "c1").Return(nil)

	svc := newNotificationChannelService(channels, flags)
	actor := &model.User{ID: "actor-id"}
	if err := svc.Delete(context.Background(), actor, "f1", "c1"); err != nil {
		t.Fatalf("Delete() returned unexpected error: %v", err)
	}
}

func TestNotificationChannelService_Delete_RepositoryNotFound(t *testing.T) {
	channels, flags := newNotificationChannelServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1"}, nil)
	channels.On("GetByID", mock.Anything, "c1").Return(&model.NotificationChannel{ID: "c1", FeatureFlagID: "f1"}, nil)
	channels.On("Delete", mock.Anything, "c1").Return(repository.ErrNotFound)

	svc := newNotificationChannelService(channels, flags)
	actor := &model.User{ID: "actor-id"}
	err := svc.Delete(context.Background(), actor, "f1", "c1")
	if !errors.Is(err, ErrNotificationChannelNotFound) {
		t.Fatalf("Delete() error = %v, want ErrNotificationChannelNotFound", err)
	}
}

func TestNotificationChannelService_Delete_InUse(t *testing.T) {
	channels, flags := newNotificationChannelServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1"}, nil)
	channels.On("GetByID", mock.Anything, "c1").Return(&model.NotificationChannel{ID: "c1", FeatureFlagID: "f1"}, nil)
	channels.On("Delete", mock.Anything, "c1").Return(repository.ErrInUse)

	svc := newNotificationChannelService(channels, flags)
	actor := &model.User{ID: "actor-id"}
	err := svc.Delete(context.Background(), actor, "f1", "c1")
	if !errors.Is(err, ErrNotificationChannelInUse) {
		t.Fatalf("Delete() error = %v, want ErrNotificationChannelInUse", err)
	}
}

func TestNotificationChannelService_Delete_RepositoryError(t *testing.T) {
	channels, flags := newNotificationChannelServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1"}, nil)
	channels.On("GetByID", mock.Anything, "c1").Return(&model.NotificationChannel{ID: "c1", FeatureFlagID: "f1"}, nil)
	channels.On("Delete", mock.Anything, "c1").Return(errors.New("disk full"))

	svc := newNotificationChannelService(channels, flags)
	actor := &model.User{ID: "actor-id"}
	err := svc.Delete(context.Background(), actor, "f1", "c1")
	if err == nil {
		t.Fatal("Delete() expected an error, got nil")
	}
}
