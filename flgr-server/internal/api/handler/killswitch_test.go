package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/NicolasCunha/flgr/flgr-server/internal/api/middleware"
	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/repository"
	"github.com/NicolasCunha/flgr/flgr-server/internal/service"
)

// setupKillswitchAuthed mirrors setupFeatureFlagValueAuthed but for
// KillswitchService-backed routes. It also returns the underlying *sql.DB
// and a NotificationChannelService, so tests can seed channels beforehand
// and assert the audit_log/notification_outbox side effects directly
// afterward (KillswitchService itself exposes no read path for either).
func setupKillswitchAuthed(t *testing.T, method, path string, makeHandler func(*service.KillswitchService) gin.HandlerFunc) (router *gin.Engine, cookie *http.Cookie, db *sql.DB, channels *service.NotificationChannelService, actor *model.User, flagID, environmentID string) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	userService, authService, db := newTestServicesWithDB(t)
	actor, err := userService.Create(t.Context(), nil, service.CreateUserInput{
		FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com", Password: "supersecret",
	})
	if err != nil {
		t.Fatalf("Create(actor) returned unexpected error: %v", err)
	}
	_, token, err := authService.Login(t.Context(), "ada@example.com", "supersecret")
	if err != nil {
		t.Fatalf("Login() returned unexpected error: %v", err)
	}

	environmentService := service.NewEnvironmentService(
		repository.NewEnvironmentRepository(db),
		repository.NewEnvironmentCategoryRepository(db),
	)
	env, err := environmentService.Create(t.Context(), actor, service.CreateEnvironmentInput{Name: "prod", CategoryName: "Production"})
	if err != nil {
		t.Fatalf("seeding environment returned unexpected error: %v", err)
	}

	flagService := service.NewFeatureFlagService(repository.NewFeatureFlagRepository(db))
	flag, err := flagService.Create(t.Context(), actor, service.CreateFeatureFlagInput{Key: "checkout-flow", Name: "Checkout Flow", Type: model.FeatureFlagTypeBoolean})
	if err != nil {
		t.Fatalf("seeding feature flag returned unexpected error: %v", err)
	}

	valueService := service.NewFeatureFlagValueService(
		repository.NewFeatureFlagRepository(db),
		repository.NewEnvironmentRepository(db),
		repository.NewFeatureFlagEnvironmentValueRepository(db),
	)
	key, err := service.DecodeEncryptionKey(testEncryptionKey)
	if err != nil {
		t.Fatalf("DecodeEncryptionKey() returned unexpected error: %v", err)
	}
	channels = service.NewNotificationChannelService(repository.NewNotificationChannelRepository(db), repository.NewFeatureFlagRepository(db), key)
	killswitch := service.NewKillswitchService(
		valueService,
		repository.NewFeatureFlagRepository(db),
		repository.NewEnvironmentRepository(db),
		repository.NewFeatureFlagEnvironmentValueRepository(db),
		repository.NewNotificationChannelRepository(db),
		repository.NewAuditLogRepository(db),
		repository.NewNotificationOutboxRepository(db),
	)

	router = gin.New()
	router.Handle(method, path, middleware.RequireAuth(authService), makeHandler(killswitch))

	return router, &http.Cookie{Name: middleware.SessionCookieName, Value: token}, db, channels, actor, flag.ID, env.ID
}

func TestKillswitchHandler_Trigger_Success(t *testing.T) {
	router, cookie, _, _, _, flagID, envID := setupKillswitchAuthed(t, http.MethodPost, "/feature-flags/:id/values/:environmentId/killswitch", func(ks *service.KillswitchService) gin.HandlerFunc {
		return NewKillswitchHandler(ks).Trigger
	})

	rec := doRequest(router, http.MethodPost, "/feature-flags/"+flagID+"/values/"+envID+"/killswitch", nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp featureFlagValueResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if resp.Enabled {
		t.Error("Enabled = true, want false after the killswitch fires")
	}
	if resp.FeatureFlagID != flagID || resp.EnvironmentID != envID {
		t.Errorf("response = %+v, want FeatureFlagID=%q EnvironmentID=%q", resp, flagID, envID)
	}
}

// TestKillswitchHandler_Trigger_WritesAuditLog confirms the audit_log side
// effect lands, end to end through the real repositories.
func TestKillswitchHandler_Trigger_WritesAuditLog(t *testing.T) {
	router, cookie, db, _, actor, flagID, envID := setupKillswitchAuthed(t, http.MethodPost, "/feature-flags/:id/values/:environmentId/killswitch", func(ks *service.KillswitchService) gin.HandlerFunc {
		return NewKillswitchHandler(ks).Trigger
	})

	rec := doRequest(router, http.MethodPost, "/feature-flags/"+flagID+"/values/"+envID+"/killswitch", nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var count int
	var action, actorUserID string
	row := db.QueryRow(
		"SELECT COUNT(*), action, actor_user_id FROM audit_log WHERE entity_type = ? GROUP BY action, actor_user_id",
		model.AuditEntityTypeFeatureFlag,
	)
	if err := row.Scan(&count, &action, &actorUserID); err != nil {
		t.Fatalf("querying audit_log: %v", err)
	}
	if count != 1 || action != model.AuditActionKillswitchTriggered || actorUserID != actor.ID {
		t.Errorf("audit_log = (count=%d, action=%q, actor=%q), want (1, %q, %q)", count, action, actorUserID, model.AuditActionKillswitchTriggered, actor.ID)
	}

	var linkFlagID string
	var linkEnvID sql.NullString
	linkRow := db.QueryRow("SELECT feature_flag_id, environment_id FROM audit_log_feature_flag")
	if err := linkRow.Scan(&linkFlagID, &linkEnvID); err != nil {
		t.Fatalf("querying audit_log_feature_flag: %v", err)
	}
	if linkFlagID != flagID || !linkEnvID.Valid || linkEnvID.String != envID {
		t.Errorf("audit_log_feature_flag = (flag=%q, env=%v), want (%q, %q)", linkFlagID, linkEnvID, flagID, envID)
	}
}

func TestKillswitchHandler_Trigger_FlagNotFound(t *testing.T) {
	router, cookie, _, _, _, _, envID := setupKillswitchAuthed(t, http.MethodPost, "/feature-flags/:id/values/:environmentId/killswitch", func(ks *service.KillswitchService) gin.HandlerFunc {
		return NewKillswitchHandler(ks).Trigger
	})

	rec := doRequest(router, http.MethodPost, "/feature-flags/does-not-exist/values/"+envID+"/killswitch", nil, cookie)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestKillswitchHandler_Trigger_UnknownEnvironment(t *testing.T) {
	router, cookie, _, _, _, flagID, _ := setupKillswitchAuthed(t, http.MethodPost, "/feature-flags/:id/values/:environmentId/killswitch", func(ks *service.KillswitchService) gin.HandlerFunc {
		return NewKillswitchHandler(ks).Trigger
	})

	rec := doRequest(router, http.MethodPost, "/feature-flags/"+flagID+"/values/does-not-exist/killswitch", nil, cookie)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

// TestKillswitchHandler_Trigger_EnqueuesOutboxForEnabledChannelsOnly
// confirms the end-to-end outbox side effect through the real
// repositories: an enabled channel gets an outbox row, a disabled one does
// not.
func TestKillswitchHandler_Trigger_EnqueuesOutboxForEnabledChannelsOnly(t *testing.T) {
	router, cookie, db, channels, actor, flagID, envID := setupKillswitchAuthed(t, http.MethodPost, "/feature-flags/:id/values/:environmentId/killswitch", func(ks *service.KillswitchService) gin.HandlerFunc {
		return NewKillswitchHandler(ks).Trigger
	})

	enabledCh, _, err := channels.Create(context.Background(), actor, flagID, service.CreateNotificationChannelInput{ChannelType: model.NotificationChannelTypeKafka, Destination: "topic-a"})
	if err != nil {
		t.Fatalf("seeding enabled channel returned unexpected error: %v", err)
	}
	disabledCh, _, err := channels.Create(context.Background(), actor, flagID, service.CreateNotificationChannelInput{ChannelType: model.NotificationChannelTypeKafka, Destination: "topic-b"})
	if err != nil {
		t.Fatalf("seeding disabled channel returned unexpected error: %v", err)
	}
	falseVal := false
	if _, _, err := channels.Update(context.Background(), actor, flagID, disabledCh.ID, service.UpdateNotificationChannelInput{Enabled: &falseVal}); err != nil {
		t.Fatalf("disabling channel returned unexpected error: %v", err)
	}

	rec := doRequest(router, http.MethodPost, "/feature-flags/"+flagID+"/values/"+envID+"/killswitch", nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	rows, err := db.Query("SELECT feature_flag_notification_channel_id, status FROM notification_outbox")
	if err != nil {
		t.Fatalf("querying notification_outbox: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var got []string
	for rows.Next() {
		var channelID, status string
		if err := rows.Scan(&channelID, &status); err != nil {
			t.Fatalf("scanning notification_outbox row: %v", err)
		}
		if status != model.NotificationOutboxStatusPending {
			t.Errorf("status = %q, want %q", status, model.NotificationOutboxStatusPending)
		}
		got = append(got, channelID)
	}
	if len(got) != 1 || got[0] != enabledCh.ID {
		t.Errorf("notification_outbox channel ids = %v, want exactly [%q] (the enabled channel only)", got, enabledCh.ID)
	}
}

// TestKillswitchHandler_Trigger_ZeroChannelsSucceeds confirms a flag with
// no configured notification channels still succeeds, with no outbox rows
// written.
func TestKillswitchHandler_Trigger_ZeroChannelsSucceeds(t *testing.T) {
	router, cookie, db, _, _, flagID, envID := setupKillswitchAuthed(t, http.MethodPost, "/feature-flags/:id/values/:environmentId/killswitch", func(ks *service.KillswitchService) gin.HandlerFunc {
		return NewKillswitchHandler(ks).Trigger
	})

	rec := doRequest(router, http.MethodPost, "/feature-flags/"+flagID+"/values/"+envID+"/killswitch", nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM notification_outbox").Scan(&count); err != nil {
		t.Fatalf("querying notification_outbox count: %v", err)
	}
	if count != 0 {
		t.Errorf("notification_outbox row count = %d, want 0 (no channels configured)", count)
	}
}
