package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/repository"
)

// killswitchServiceMocks bundles every mocked repository KillswitchService
// (and the FeatureFlagValueService it wraps) needs.
type killswitchServiceMocks struct {
	flags     *mockFeatureFlagRepository
	envs      *mockEnvironmentRepository
	envValues *mockFeatureFlagEnvironmentValueRepository
	channels  *mockNotificationChannelRepository
	auditLogs *mockAuditLogRepository
	outbox    *mockNotificationOutboxRepository
}

func newKillswitchService() (*KillswitchService, *killswitchServiceMocks) {
	m := &killswitchServiceMocks{
		flags:     new(mockFeatureFlagRepository),
		envs:      new(mockEnvironmentRepository),
		envValues: new(mockFeatureFlagEnvironmentValueRepository),
		channels:  new(mockNotificationChannelRepository),
		auditLogs: new(mockAuditLogRepository),
		outbox:    new(mockNotificationOutboxRepository),
	}
	values := NewFeatureFlagValueService(m.flags, m.envs, m.envValues)
	svc := NewKillswitchService(values, m.flags, m.envs, m.envValues, m.channels, m.auditLogs, m.outbox)
	return svc, m
}

func TestKillswitchService_Trigger_NilActor(t *testing.T) {
	svc, _ := newKillswitchService()
	_, err := svc.Trigger(context.Background(), Actor{}, "f1", "env-1")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Trigger() error = %v, want ErrForbidden", err)
	}
}

func TestKillswitchService_Trigger_FlagNotFound(t *testing.T) {
	svc, m := newKillswitchService()
	m.flags.On("GetByID", mock.Anything, "missing").Return(nil, repository.ErrNotFound)

	actor := &model.User{ID: "actor-id"}
	_, err := svc.Trigger(context.Background(), ActorFromUser(actor), "missing", "env-1")
	if !errors.Is(err, ErrFeatureFlagNotFound) {
		t.Fatalf("Trigger() error = %v, want ErrFeatureFlagNotFound", err)
	}
}

func TestKillswitchService_Trigger_FlagLookupError(t *testing.T) {
	svc, m := newKillswitchService()
	m.flags.On("GetByID", mock.Anything, "f1").Return(nil, errors.New("db down"))

	actor := &model.User{ID: "actor-id"}
	_, err := svc.Trigger(context.Background(), ActorFromUser(actor), "f1", "env-1")
	if err == nil || errors.Is(err, ErrFeatureFlagNotFound) {
		t.Fatalf("Trigger() error = %v, want a generic wrapped error", err)
	}
}

func TestKillswitchService_Trigger_UnknownEnvironment(t *testing.T) {
	svc, m := newKillswitchService()
	m.flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1", Key: "checkout"}, nil)
	m.envs.On("GetByID", mock.Anything, "missing").Return(nil, repository.ErrNotFound)

	actor := &model.User{ID: "actor-id"}
	_, err := svc.Trigger(context.Background(), ActorFromUser(actor), "f1", "missing")
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Trigger() error = %v, want ErrValidation", err)
	}
}

func TestKillswitchService_Trigger_EnvironmentLookupError(t *testing.T) {
	svc, m := newKillswitchService()
	m.flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1", Key: "checkout"}, nil)
	m.envs.On("GetByID", mock.Anything, "env-1").Return(nil, errors.New("db down"))

	actor := &model.User{ID: "actor-id"}
	_, err := svc.Trigger(context.Background(), ActorFromUser(actor), "f1", "env-1")
	if err == nil || errors.Is(err, ErrValidation) {
		t.Fatalf("Trigger() error = %v, want a generic wrapped error", err)
	}
}

func TestKillswitchService_Trigger_ExistingValueLookupError(t *testing.T) {
	svc, m := newKillswitchService()
	m.flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1", Key: "checkout", Type: model.FeatureFlagTypeBoolean}, nil)
	m.envs.On("GetByID", mock.Anything, "env-1").Return(&model.Environment{ID: "env-1", Name: "prod"}, nil)
	m.envValues.On("GetByFlagAndEnvironment", mock.Anything, "f1", "env-1").Return(nil, errors.New("db down"))

	actor := &model.User{ID: "actor-id"}
	_, err := svc.Trigger(context.Background(), ActorFromUser(actor), "f1", "env-1")
	if err == nil {
		t.Fatal("Trigger() expected an error, got nil")
	}
}

func TestKillswitchService_Trigger_UpsertError(t *testing.T) {
	svc, m := newKillswitchService()
	m.flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1", Key: "checkout", Type: model.FeatureFlagTypeBoolean}, nil)
	m.envs.On("GetByID", mock.Anything, "env-1").Return(&model.Environment{ID: "env-1", Name: "prod"}, nil)
	m.envValues.On("GetByFlagAndEnvironment", mock.Anything, "f1", "env-1").Return(nil, repository.ErrNotFound)
	m.envValues.On("Create", mock.Anything, mock.Anything).Return(errors.New("disk full"))

	actor := &model.User{ID: "actor-id"}
	_, err := svc.Trigger(context.Background(), ActorFromUser(actor), "f1", "env-1")
	if err == nil {
		t.Fatal("Trigger() expected an error, got nil")
	}
	m.auditLogs.AssertNotCalled(t, "Create")
	m.outbox.AssertNotCalled(t, "Create")
}

func TestKillswitchService_Trigger_AuditLogError(t *testing.T) {
	fixedNow(t)
	fixedID(t, "audit-1")
	svc, m := newKillswitchService()
	m.flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1", Key: "checkout", Type: model.FeatureFlagTypeBoolean}, nil)
	m.envs.On("GetByID", mock.Anything, "env-1").Return(&model.Environment{ID: "env-1", Name: "prod"}, nil)
	m.envValues.On("GetByFlagAndEnvironment", mock.Anything, "f1", "env-1").Return(&model.FeatureFlagEnvironmentValue{ID: "v1", Enabled: true}, nil)
	m.envValues.On("Update", mock.Anything, mock.Anything).Return(nil)
	m.auditLogs.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("disk full"))

	actor := &model.User{ID: "actor-id"}
	_, err := svc.Trigger(context.Background(), ActorFromUser(actor), "f1", "env-1")
	if err == nil {
		t.Fatal("Trigger() expected an error, got nil")
	}
	m.outbox.AssertNotCalled(t, "Create")
}

func TestKillswitchService_Trigger_ChannelListError(t *testing.T) {
	fixedNow(t)
	fixedID(t, "audit-1")
	svc, m := newKillswitchService()
	m.flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1", Key: "checkout", Type: model.FeatureFlagTypeBoolean}, nil)
	m.envs.On("GetByID", mock.Anything, "env-1").Return(&model.Environment{ID: "env-1", Name: "prod"}, nil)
	m.envValues.On("GetByFlagAndEnvironment", mock.Anything, "f1", "env-1").Return(&model.FeatureFlagEnvironmentValue{ID: "v1", Enabled: true}, nil)
	m.envValues.On("Update", mock.Anything, mock.Anything).Return(nil)
	m.auditLogs.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	m.channels.On("ListByFeatureFlagID", mock.Anything, "f1").Return(nil, errors.New("db down"))

	actor := &model.User{ID: "actor-id"}
	_, err := svc.Trigger(context.Background(), ActorFromUser(actor), "f1", "env-1")
	if err == nil {
		t.Fatal("Trigger() expected an error, got nil")
	}
}

func TestKillswitchService_Trigger_OutboxCreateError(t *testing.T) {
	fixedNow(t)
	fixedID(t, "audit-1")
	svc, m := newKillswitchService()
	m.flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1", Key: "checkout", Type: model.FeatureFlagTypeBoolean}, nil)
	m.envs.On("GetByID", mock.Anything, "env-1").Return(&model.Environment{ID: "env-1", Name: "prod"}, nil)
	m.envValues.On("GetByFlagAndEnvironment", mock.Anything, "f1", "env-1").Return(&model.FeatureFlagEnvironmentValue{ID: "v1", Enabled: true}, nil)
	m.envValues.On("Update", mock.Anything, mock.Anything).Return(nil)
	m.auditLogs.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	m.channels.On("ListByFeatureFlagID", mock.Anything, "f1").Return([]model.NotificationChannel{
		{ID: "ch1", Enabled: true},
	}, nil)
	m.outbox.On("Create", mock.Anything, mock.Anything).Return(errors.New("disk full"))

	actor := &model.User{ID: "actor-id"}
	_, err := svc.Trigger(context.Background(), ActorFromUser(actor), "f1", "env-1")
	if err == nil {
		t.Fatal("Trigger() expected an error, got nil")
	}
}

// TestKillswitchService_Trigger_Success_NoConfiguredValueYet covers the
// "flag has no configured row for this environment" branch — it evaluates
// as disabled beforehand (0005), so the audit entry's old_value must
// reflect that, and the trigger must still succeed (creating a new,
// disabled row) rather than erroring.
func TestKillswitchService_Trigger_Success_NoConfiguredValueYet(t *testing.T) {
	fixedNow(t)
	fixedID(t, "audit-1")
	svc, m := newKillswitchService()
	m.flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1", Key: "checkout", Type: model.FeatureFlagTypeBoolean}, nil)
	m.envs.On("GetByID", mock.Anything, "env-1").Return(&model.Environment{ID: "env-1", Name: "prod"}, nil)
	// Called twice: once by KillswitchService.Trigger's own wasEnabled
	// lookup, once by FeatureFlagValueService.Upsert's existing-row check.
	m.envValues.On("GetByFlagAndEnvironment", mock.Anything, "f1", "env-1").Return(nil, repository.ErrNotFound)
	m.envValues.On("Create", mock.Anything, mock.MatchedBy(func(v *model.FeatureFlagEnvironmentValue) bool {
		return !v.Enabled
	})).Return(nil)

	var loggedLog *model.AuditLog
	var loggedLink *model.AuditLogFeatureFlag
	m.auditLogs.On("Create", mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		loggedLog = args.Get(1).(*model.AuditLog)
		loggedLink = args.Get(2).(*model.AuditLogFeatureFlag)
	}).Return(nil)
	m.channels.On("ListByFeatureFlagID", mock.Anything, "f1").Return(nil, nil)

	actor := &model.User{ID: "actor-id"}
	value, err := svc.Trigger(context.Background(), ActorFromUser(actor), "f1", "env-1")
	if err != nil {
		t.Fatalf("Trigger() returned unexpected error: %v", err)
	}
	if value.Enabled {
		t.Error("value.Enabled = true, want false")
	}

	if loggedLog == nil {
		t.Fatal("AuditLogRepository.Create was not called")
	}
	if loggedLog.EntityType != model.AuditEntityTypeFeatureFlag || loggedLog.Action != model.AuditActionKillswitchTriggered {
		t.Errorf("audit log = (entity_type=%q, action=%q), want (%q, %q)", loggedLog.EntityType, loggedLog.Action, model.AuditEntityTypeFeatureFlag, model.AuditActionKillswitchTriggered)
	}
	if loggedLog.ActorUserID == nil || *loggedLog.ActorUserID != actor.ID {
		t.Errorf("audit log ActorUserID = %v, want %q", loggedLog.ActorUserID, actor.ID)
	}
	if loggedLog.Source != model.AuditSourceAPI {
		t.Errorf("audit log Source = %q, want %q", loggedLog.Source, model.AuditSourceAPI)
	}
	if loggedLog.OldValue == nil || *loggedLog.OldValue != `{"enabled":false}` {
		t.Errorf("audit log OldValue = %v, want %q (no prior row evaluates as disabled)", loggedLog.OldValue, `{"enabled":false}`)
	}
	if loggedLog.NewValue == nil || *loggedLog.NewValue != `{"enabled":false}` {
		t.Errorf("audit log NewValue = %v, want %q", loggedLog.NewValue, `{"enabled":false}`)
	}
	if loggedLink.FeatureFlagID != "f1" || loggedLink.EnvironmentID == nil || *loggedLink.EnvironmentID != "env-1" {
		t.Errorf("audit log link = %+v, want FeatureFlagID=f1 EnvironmentID=env-1", loggedLink)
	}

	// Zero enabled channels: no outbox writes attempted, and the trigger
	// still succeeds.
	m.outbox.AssertNotCalled(t, "Create")
}

// TestKillswitchService_Trigger_Success_WasEnabled covers the old_value
// capturing a prior enabled row correctly, and enqueues one outbox row per
// enabled channel while skipping disabled ones.
func TestKillswitchService_Trigger_Success_WasEnabled(t *testing.T) {
	fixedNow(t)
	fixedID(t, "audit-1")
	svc, m := newKillswitchService()
	m.flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1", Key: "checkout", Type: model.FeatureFlagTypeBoolean}, nil)
	m.envs.On("GetByID", mock.Anything, "env-1").Return(&model.Environment{ID: "env-1", Name: "prod"}, nil)
	m.envValues.On("GetByFlagAndEnvironment", mock.Anything, "f1", "env-1").Return(&model.FeatureFlagEnvironmentValue{ID: "v1", Enabled: true}, nil)
	m.envValues.On("Update", mock.Anything, mock.MatchedBy(func(v *model.FeatureFlagEnvironmentValue) bool {
		return v.ID == "v1" && !v.Enabled
	})).Return(nil)

	var loggedLog *model.AuditLog
	m.auditLogs.On("Create", mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		loggedLog = args.Get(1).(*model.AuditLog)
	}).Return(nil)

	m.channels.On("ListByFeatureFlagID", mock.Anything, "f1").Return([]model.NotificationChannel{
		{ID: "ch-enabled-1", Enabled: true},
		{ID: "ch-disabled", Enabled: false},
		{ID: "ch-enabled-2", Enabled: true},
	}, nil)

	var createdEntries []*model.NotificationOutboxEntry
	m.outbox.On("Create", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		createdEntries = append(createdEntries, args.Get(1).(*model.NotificationOutboxEntry))
	}).Return(nil)

	actor := &model.User{ID: "actor-id"}
	_, err := svc.Trigger(context.Background(), ActorFromUser(actor), "f1", "env-1")
	if err != nil {
		t.Fatalf("Trigger() returned unexpected error: %v", err)
	}

	if loggedLog.OldValue == nil || *loggedLog.OldValue != `{"enabled":true}` {
		t.Errorf("audit log OldValue = %v, want %q (prior row was enabled)", loggedLog.OldValue, `{"enabled":true}`)
	}

	if len(createdEntries) != 2 {
		t.Fatalf("len(outbox entries created) = %d, want 2 (one per enabled channel, skipping the disabled one)", len(createdEntries))
	}
	seen := map[string]bool{}
	for _, e := range createdEntries {
		seen[e.FeatureFlagNotificationChannelID] = true
		if e.Status != model.NotificationOutboxStatusPending {
			t.Errorf("entry.Status = %q, want %q", e.Status, model.NotificationOutboxStatusPending)
		}
		if e.AttemptCount != 0 {
			t.Errorf("entry.AttemptCount = %d, want 0", e.AttemptCount)
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(e.Payload), &payload); err != nil {
			t.Fatalf("unmarshaling payload: %v", err)
		}
		if payload["event"] != "feature_flag.killed" || payload["feature_flag_key"] != "checkout" || payload["environment"] != "prod" || payload["enabled"] != false {
			t.Errorf("payload = %+v, want event/feature_flag_key/environment/enabled set correctly", payload)
		}
	}
	if !seen["ch-enabled-1"] || !seen["ch-enabled-2"] {
		t.Errorf("created entries = %+v, want one for ch-enabled-1 and one for ch-enabled-2", createdEntries)
	}
	if seen["ch-disabled"] {
		t.Error("an outbox entry was created for a disabled channel, want it skipped")
	}
}
