package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/repository"
)

// killswitchEvent is the notification payload shape shared by Kafka and
// HTTP deliveries, per the "Event payload" decision in
// docs/architecture/adr/0009-kafka-and-webhook-notification-delivery.md.
type killswitchEvent struct {
	Event          string    `json:"event"`
	FeatureFlagKey string    `json:"feature_flag_key"`
	Environment    string    `json:"environment"`
	Enabled        bool      `json:"enabled"`
	OccurredAt     time.Time `json:"occurred_at"`
}

// KillswitchService implements the Killswitch trigger from
// docs/business/requirements/0006-feature-flag-killswitch.md: an emergency
// action for a single (flag, environment) pair. Triggering it (1) disables
// the flag in that environment by reusing FeatureFlagValueService.Upsert
// rather than duplicating its merge/validation logic, (2) writes an
// audit_log entry (0008), and (3) enqueues one notification_outbox row per
// enabled channel configured on the flag (ADR-0009's outbox pattern) —
// actually delivering those rows (Kafka/HTTP, retries) is a future phase's
// background worker, not built here. Gated by the same FeatureFlagValue:
// Write permission as any other value change, per 0006.
type KillswitchService struct {
	values       *FeatureFlagValueService
	flags        repository.FeatureFlagRepository
	environments repository.EnvironmentRepository
	envValues    repository.FeatureFlagEnvironmentValueRepository
	channels     repository.NotificationChannelRepository
	auditLogs    repository.AuditLogRepository
	outbox       repository.NotificationOutboxRepository
}

// NewKillswitchService returns a KillswitchService backed by the given
// service/repositories.
func NewKillswitchService(
	values *FeatureFlagValueService,
	flags repository.FeatureFlagRepository,
	environments repository.EnvironmentRepository,
	envValues repository.FeatureFlagEnvironmentValueRepository,
	channels repository.NotificationChannelRepository,
	auditLogs repository.AuditLogRepository,
	outbox repository.NotificationOutboxRepository,
) *KillswitchService {
	return &KillswitchService{
		values:       values,
		flags:        flags,
		environments: environments,
		envValues:    envValues,
		channels:     channels,
		auditLogs:    auditLogs,
		outbox:       outbox,
	}
}

// Trigger disables flag featureFlagID in environmentID, records an
// audit_log entry, and enqueues an outbox row for every enabled
// notification channel configured on the flag, returning the updated
// value (the same shape FeatureFlagValueService.Upsert returns, so the
// caller can render it identically either way).
//
// These three writes are not wrapped in one shared database transaction —
// AuditLogRepository.Create and NotificationOutboxRepository.Create each
// commit independently of FeatureFlagValueService.Upsert, since no
// cross-repository transaction seam exists yet in this codebase. A true
// atomic outbox write (as ADR-0009 describes) would need that seam added
// at the repository layer, which is outside this phase's scope — see this
// phase's backend report.
func (s *KillswitchService) Trigger(ctx context.Context, actor Actor, featureFlagID, environmentID string) (*model.FeatureFlagEnvironmentValue, error) {
	if actor.IsZero() {
		return nil, ErrForbidden
	}

	flag, err := s.flags.GetByID(ctx, featureFlagID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrFeatureFlagNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("fetching feature flag: %w", err)
	}

	env, err := s.environments.GetByID(ctx, environmentID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, validationErrorf("unknown environment id %q", environmentID)
	}
	if err != nil {
		return nil, fmt.Errorf("fetching environment: %w", err)
	}

	// Capture the enabled state before flipping, for the audit entry's
	// old_value — a flag with no configured row for this environment
	// evaluates as disabled, per 0005, so that's the "old" state too.
	wasEnabled := false
	existing, err := s.envValues.GetByFlagAndEnvironment(ctx, featureFlagID, environmentID)
	if err != nil {
		if !errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("fetching feature flag environment value: %w", err)
		}
	} else {
		wasEnabled = existing.Enabled
	}

	value, err := s.values.Upsert(ctx, actor, featureFlagID, environmentID, UpsertValueInput{Enabled: false})
	if err != nil {
		return nil, err
	}

	if err := s.writeAuditLog(ctx, actor, featureFlagID, environmentID, wasEnabled); err != nil {
		return nil, err
	}

	if err := s.enqueueNotifications(ctx, flag, env); err != nil {
		return nil, err
	}

	return value, nil
}

func (s *KillswitchService) writeAuditLog(ctx context.Context, actor Actor, featureFlagID, environmentID string, wasEnabled bool) error {
	oldValue := fmt.Sprintf(`{"enabled":%t}`, wasEnabled)
	newValue := `{"enabled":false}`

	log := &model.AuditLog{
		ID:                newID(),
		EntityType:        model.AuditEntityTypeFeatureFlag,
		Action:            model.AuditActionKillswitchTriggered,
		ActorUserID:       actor.UserID,
		ActorServiceKeyID: actor.ServiceKeyID,
		// Source is always API here: middleware.RequireAuth's
		// session-cookie authentication is the only actor mechanism
		// wired into this route today, and there's no signal at this
		// layer distinguishing a browser UI call from a direct API call
		// made with the same session — see this phase's backend report
		// for the full reasoning behind defaulting to AuditSourceAPI.
		Source:     model.AuditSourceAPI,
		OldValue:   &oldValue,
		NewValue:   &newValue,
		OccurredOn: now(),
	}
	link := &model.AuditLogFeatureFlag{
		FeatureFlagID: featureFlagID,
		EnvironmentID: &environmentID,
	}
	if err := s.auditLogs.Create(ctx, log, link); err != nil {
		return fmt.Errorf("writing killswitch audit log: %w", err)
	}
	return nil
}

func (s *KillswitchService) enqueueNotifications(ctx context.Context, flag *model.FeatureFlag, env *model.Environment) error {
	channels, err := s.channels.ListByFeatureFlagID(ctx, flag.ID)
	if err != nil {
		return fmt.Errorf("listing notification channels: %w", err)
	}

	ts := now()
	payload, err := json.Marshal(killswitchEvent{
		Event:          "feature_flag.killed",
		FeatureFlagKey: flag.Key,
		Environment:    env.Name,
		Enabled:        false,
		OccurredAt:     ts,
	})
	if err != nil {
		return fmt.Errorf("marshaling killswitch event payload: %w", err)
	}

	for _, ch := range channels {
		if !ch.Enabled {
			continue
		}
		entry := &model.NotificationOutboxEntry{
			ID:                               newID(),
			FeatureFlagNotificationChannelID: ch.ID,
			Payload:                          string(payload),
			Status:                           model.NotificationOutboxStatusPending,
			AttemptCount:                     0,
			NextAttemptAt:                    &ts,
			CreatedOn:                        ts,
		}
		if err := s.outbox.Create(ctx, entry); err != nil {
			return fmt.Errorf("enqueuing notification outbox entry: %w", err)
		}
	}
	return nil
}
