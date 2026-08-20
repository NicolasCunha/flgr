package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/repository"
)

var validNotificationChannelTypes = map[string]bool{
	model.NotificationChannelTypeKafka: true,
	model.NotificationChannelTypeHTTP:  true,
}

// NotificationChannelService implements the notification-channel CRUD half
// of docs/business/requirements/0006-feature-flag-killswitch.md, gated
// entirely by FeatureFlag: Edit ("A user with the Feature Flag: Edit
// permission can add, update, disable, or remove a Kafka or HTTP
// notification channel on a flag") — unlike FeatureFlagValueService, there
// is no separate permission surface for this sub-resource; List only needs
// FeatureFlag: View, per the CRUD-implies-View rule every other resource
// follows (see router.go for the actual gating).
//
// HTTP channels get a server-generated signing secret (ADR-0009), stored
// AES-256-GCM-encrypted at rest (ADR-0010) via encryptSecret/decryptSecret.
// Per ADR-0009's explicit call-out ("viewable afterward — unlike a Service
// Key secret, the receiving system needs to know it to verify signatures,
// so it isn't a one-time reveal"), every read method here decrypts and
// returns the plaintext secret, not just Create.
type NotificationChannelService struct {
	channels      repository.NotificationChannelRepository
	flags         repository.FeatureFlagRepository
	encryptionKey []byte
}

// NewNotificationChannelService returns a NotificationChannelService backed
// by the given repositories. encryptionKey must already be the decoded,
// validated 32-byte AES-256 key (see service.DecodeEncryptionKey, called
// once at startup by internal/api's router wiring).
func NewNotificationChannelService(channels repository.NotificationChannelRepository, flags repository.FeatureFlagRepository, encryptionKey []byte) *NotificationChannelService {
	return &NotificationChannelService{channels: channels, flags: flags, encryptionKey: encryptionKey}
}

func (s *NotificationChannelService) getFlag(ctx context.Context, featureFlagID string) (*model.FeatureFlag, error) {
	f, err := s.flags.GetByID(ctx, featureFlagID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrFeatureFlagNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("fetching feature flag: %w", err)
	}
	return f, nil
}

// getChannel fetches the channel identified by id, confirming it belongs to
// featureFlagID — a channel id that exists but belongs to a different flag
// 404s the same as one that doesn't exist at all, rather than leaking
// across flags.
func (s *NotificationChannelService) getChannel(ctx context.Context, featureFlagID, id string) (*model.NotificationChannel, error) {
	ch, err := s.channels.GetByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNotificationChannelNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("fetching notification channel: %w", err)
	}
	if ch.FeatureFlagID != featureFlagID {
		return nil, ErrNotificationChannelNotFound
	}
	return ch, nil
}

// secretFor decrypts ch's signing secret. Returns "" for Kafka channels (or
// any channel with no stored secret) rather than erroring — there's simply
// nothing to reveal.
func (s *NotificationChannelService) secretFor(ch *model.NotificationChannel) (string, error) {
	if len(ch.SigningSecretEncrypted) == 0 {
		return "", nil
	}
	secret, err := decryptSecret(s.encryptionKey, ch.SigningSecretEncrypted)
	if err != nil {
		return "", fmt.Errorf("decrypting notification channel signing secret: %w", err)
	}
	return secret, nil
}

// CreateNotificationChannelInput is the payload for
// NotificationChannelService.Create.
type CreateNotificationChannelInput struct {
	ChannelType string
	Destination string
}

// Create adds a notification channel to featureFlagID. For HTTP channels, a
// signing secret is generated server-side and returned in plaintext
// alongside the (encrypted-at-rest) channel; Kafka channels never have one.
func (s *NotificationChannelService) Create(ctx context.Context, actor *model.User, featureFlagID string, input CreateNotificationChannelInput) (*model.NotificationChannel, string, error) {
	if actor == nil {
		return nil, "", ErrForbidden
	}

	if _, err := s.getFlag(ctx, featureFlagID); err != nil {
		return nil, "", err
	}

	if !validNotificationChannelTypes[input.ChannelType] {
		return nil, "", validationErrorf("channel_type must be %q or %q", model.NotificationChannelTypeKafka, model.NotificationChannelTypeHTTP)
	}
	destination := strings.TrimSpace(input.Destination)
	if destination == "" {
		return nil, "", validationErrorf("destination is required")
	}

	ts := now()
	ch := &model.NotificationChannel{
		ID:            newID(),
		FeatureFlagID: featureFlagID,
		ChannelType:   input.ChannelType,
		Destination:   destination,
		Enabled:       true,
		AuditFields: model.AuditFields{
			CreatedOn:        ts,
			ModifiedOn:       ts,
			CreatedByUserID:  &actor.ID,
			ModifiedByUserID: &actor.ID,
		},
	}

	var secret string
	if input.ChannelType == model.NotificationChannelTypeHTTP {
		var err error
		secret, err = generateToken()
		if err != nil {
			return nil, "", fmt.Errorf("generating notification channel signing secret: %w", err)
		}
		encrypted, err := encryptSecret(s.encryptionKey, secret)
		if err != nil {
			return nil, "", fmt.Errorf("encrypting notification channel signing secret: %w", err)
		}
		ch.SigningSecretEncrypted = encrypted
	}

	if err := s.channels.Create(ctx, ch); err != nil {
		return nil, "", fmt.Errorf("creating notification channel: %w", err)
	}

	return ch, secret, nil
}

// List returns every notification channel configured on featureFlagID,
// alongside a map of each channel's id to its decrypted signing secret
// ("" for Kafka channels).
func (s *NotificationChannelService) List(ctx context.Context, featureFlagID string) ([]model.NotificationChannel, map[string]string, error) {
	if _, err := s.getFlag(ctx, featureFlagID); err != nil {
		return nil, nil, err
	}

	channels, err := s.channels.ListByFeatureFlagID(ctx, featureFlagID)
	if err != nil {
		return nil, nil, fmt.Errorf("listing notification channels: %w", err)
	}

	secrets := make(map[string]string, len(channels))
	for i := range channels {
		secret, err := s.secretFor(&channels[i])
		if err != nil {
			return nil, nil, err
		}
		secrets[channels[i].ID] = secret
	}

	return channels, secrets, nil
}

// UpdateNotificationChannelInput is the payload for
// NotificationChannelService.Update. Nil fields are left unchanged, per
// PATCH's partial-update semantics (ADR-0007). Regenerating the signing
// secret is out of scope for this phase (not mentioned in 0006's AC), so
// there is deliberately no field for it here.
type UpdateNotificationChannelInput struct {
	Destination *string
	Enabled     *bool
}

// Update applies input to the channel identified by id on featureFlagID.
func (s *NotificationChannelService) Update(ctx context.Context, actor *model.User, featureFlagID, id string, input UpdateNotificationChannelInput) (*model.NotificationChannel, string, error) {
	if actor == nil {
		return nil, "", ErrForbidden
	}

	if _, err := s.getFlag(ctx, featureFlagID); err != nil {
		return nil, "", err
	}
	ch, err := s.getChannel(ctx, featureFlagID, id)
	if err != nil {
		return nil, "", err
	}

	if input.Destination != nil {
		ch.Destination = strings.TrimSpace(*input.Destination)
	}
	if ch.Destination == "" {
		return nil, "", validationErrorf("destination is required")
	}
	if input.Enabled != nil {
		ch.Enabled = *input.Enabled
	}

	ch.ModifiedByUserID = &actor.ID
	ch.ModifiedOn = now()

	if err := s.channels.Update(ctx, ch); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, "", ErrNotificationChannelNotFound
		}
		return nil, "", fmt.Errorf("updating notification channel: %w", err)
	}

	secret, err := s.secretFor(ch)
	if err != nil {
		return nil, "", err
	}
	return ch, secret, nil
}

// Delete removes the channel identified by id on featureFlagID — a real
// delete, not a soft one (channel rows aren't referenced as an audit
// created_by/modified_by target anywhere, same reasoning as
// Environments/Profiles per ADR-0005). A channel still referenced by
// notification_outbox rows is rejected with ErrNotificationChannelInUse.
func (s *NotificationChannelService) Delete(ctx context.Context, actor *model.User, featureFlagID, id string) error {
	if actor == nil {
		return ErrForbidden
	}

	if _, err := s.getFlag(ctx, featureFlagID); err != nil {
		return err
	}
	if _, err := s.getChannel(ctx, featureFlagID, id); err != nil {
		return err
	}

	if err := s.channels.Delete(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNotificationChannelNotFound
		}
		if errors.Is(err, repository.ErrInUse) {
			return ErrNotificationChannelInUse
		}
		return fmt.Errorf("deleting notification channel: %w", err)
	}
	return nil
}
