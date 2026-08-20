package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/NicolasCunha/flgr/flgr-server/internal/api/middleware"
	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/repository"
	"github.com/NicolasCunha/flgr/flgr-server/internal/service"
)

// testEncryptionKey mirrors internal/api's testEncryptionKey — a valid
// base64-encoded 32-byte AES-256 key, test-only.
const testEncryptionKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="

// setupNotificationChannelAuthed mirrors setupFeatureFlagValueAuthed but
// for NotificationChannelService-backed routes.
func setupNotificationChannelAuthed(t *testing.T, method, path string, makeHandler func(*service.NotificationChannelService) gin.HandlerFunc) (router *gin.Engine, cookie *http.Cookie, channels *service.NotificationChannelService, actor *model.User, flagID string) {
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

	flagService := service.NewFeatureFlagService(repository.NewFeatureFlagRepository(db))
	flag, err := flagService.Create(t.Context(), actor, service.CreateFeatureFlagInput{Key: "checkout-flow", Name: "Checkout Flow", Type: model.FeatureFlagTypeBoolean})
	if err != nil {
		t.Fatalf("seeding feature flag returned unexpected error: %v", err)
	}

	key, err := service.DecodeEncryptionKey(testEncryptionKey)
	if err != nil {
		t.Fatalf("DecodeEncryptionKey() returned unexpected error: %v", err)
	}
	channels = service.NewNotificationChannelService(repository.NewNotificationChannelRepository(db), repository.NewFeatureFlagRepository(db), key)

	router = gin.New()
	router.Handle(method, path, middleware.RequireAuth(authService), makeHandler(channels))

	return router, &http.Cookie{Name: middleware.SessionCookieName, Value: token}, channels, actor, flag.ID
}

// --- Create ---

func TestNotificationChannelHandler_Create_Kafka_Success(t *testing.T) {
	router, cookie, _, _, flagID := setupNotificationChannelAuthed(t, http.MethodPost, "/feature-flags/:id/notification-channels", func(cs *service.NotificationChannelService) gin.HandlerFunc {
		return NewNotificationChannelHandler(cs).Create
	})

	body, _ := json.Marshal(map[string]string{"channel_type": model.NotificationChannelTypeKafka, "destination": "flag-events-topic"})
	rec := doRequest(router, http.MethodPost, "/feature-flags/"+flagID+"/notification-channels", body, cookie)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var resp notificationChannelResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if resp.ChannelType != model.NotificationChannelTypeKafka || resp.Destination != "flag-events-topic" || !resp.Enabled {
		t.Errorf("response = %+v, want ChannelType=Kafka Destination=flag-events-topic Enabled=true", resp)
	}
	if resp.SigningSecret != "" {
		t.Errorf("SigningSecret = %q, want empty for a Kafka channel", resp.SigningSecret)
	}
}

func TestNotificationChannelHandler_Create_HTTP_ReturnsPlaintextSecret(t *testing.T) {
	router, cookie, _, _, flagID := setupNotificationChannelAuthed(t, http.MethodPost, "/feature-flags/:id/notification-channels", func(cs *service.NotificationChannelService) gin.HandlerFunc {
		return NewNotificationChannelHandler(cs).Create
	})

	body, _ := json.Marshal(map[string]string{"channel_type": model.NotificationChannelTypeHTTP, "destination": "https://example.com/webhook"})
	rec := doRequest(router, http.MethodPost, "/feature-flags/"+flagID+"/notification-channels", body, cookie)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var resp notificationChannelResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if resp.ChannelType != model.NotificationChannelTypeHTTP {
		t.Errorf("ChannelType = %q, want HTTP", resp.ChannelType)
	}
	if resp.SigningSecret == "" {
		t.Error("SigningSecret is empty, want a generated plaintext secret for an HTTP channel")
	}
}

func TestNotificationChannelHandler_Create_BindError(t *testing.T) {
	router, cookie, _, _, flagID := setupNotificationChannelAuthed(t, http.MethodPost, "/feature-flags/:id/notification-channels", func(cs *service.NotificationChannelService) gin.HandlerFunc {
		return NewNotificationChannelHandler(cs).Create
	})

	rec := doRequest(router, http.MethodPost, "/feature-flags/"+flagID+"/notification-channels", []byte("not json"), cookie)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestNotificationChannelHandler_Create_ValidationError(t *testing.T) {
	router, cookie, _, _, flagID := setupNotificationChannelAuthed(t, http.MethodPost, "/feature-flags/:id/notification-channels", func(cs *service.NotificationChannelService) gin.HandlerFunc {
		return NewNotificationChannelHandler(cs).Create
	})

	body, _ := json.Marshal(map[string]string{"channel_type": "Carrier Pigeon", "destination": "somewhere"})
	rec := doRequest(router, http.MethodPost, "/feature-flags/"+flagID+"/notification-channels", body, cookie)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestNotificationChannelHandler_Create_FlagNotFound(t *testing.T) {
	router, cookie, _, _, _ := setupNotificationChannelAuthed(t, http.MethodPost, "/feature-flags/:id/notification-channels", func(cs *service.NotificationChannelService) gin.HandlerFunc {
		return NewNotificationChannelHandler(cs).Create
	})

	body, _ := json.Marshal(map[string]string{"channel_type": model.NotificationChannelTypeKafka, "destination": "topic"})
	rec := doRequest(router, http.MethodPost, "/feature-flags/does-not-exist/notification-channels", body, cookie)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

// --- List ---

func TestNotificationChannelHandler_List_Empty(t *testing.T) {
	router, cookie, _, _, flagID := setupNotificationChannelAuthed(t, http.MethodGet, "/feature-flags/:id/notification-channels", func(cs *service.NotificationChannelService) gin.HandlerFunc {
		return NewNotificationChannelHandler(cs).List
	})

	rec := doRequest(router, http.MethodGet, "/feature-flags/"+flagID+"/notification-channels", nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp struct {
		Data []notificationChannelResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if len(resp.Data) != 0 {
		t.Errorf("len(Data) = %d, want 0", len(resp.Data))
	}
}

func TestNotificationChannelHandler_List_SecretPresentForHTTPOnly(t *testing.T) {
	router, cookie, channels, actor, flagID := setupNotificationChannelAuthed(t, http.MethodGet, "/feature-flags/:id/notification-channels", func(cs *service.NotificationChannelService) gin.HandlerFunc {
		return NewNotificationChannelHandler(cs).List
	})

	if _, _, err := channels.Create(context.Background(), actor, flagID, service.CreateNotificationChannelInput{ChannelType: model.NotificationChannelTypeKafka, Destination: "topic"}); err != nil {
		t.Fatalf("seeding Kafka channel returned unexpected error: %v", err)
	}
	if _, _, err := channels.Create(context.Background(), actor, flagID, service.CreateNotificationChannelInput{ChannelType: model.NotificationChannelTypeHTTP, Destination: "https://example.com"}); err != nil {
		t.Fatalf("seeding HTTP channel returned unexpected error: %v", err)
	}

	rec := doRequest(router, http.MethodGet, "/feature-flags/"+flagID+"/notification-channels", nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp struct {
		Data []notificationChannelResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("len(Data) = %d, want 2", len(resp.Data))
	}
	for _, c := range resp.Data {
		switch c.ChannelType {
		case model.NotificationChannelTypeKafka:
			if c.SigningSecret != "" {
				t.Errorf("Kafka channel SigningSecret = %q, want empty", c.SigningSecret)
			}
		case model.NotificationChannelTypeHTTP:
			// List must return the secret decrypted, not just Create — per
			// ADR-0009's "viewable afterward" requirement.
			if c.SigningSecret == "" {
				t.Error("HTTP channel SigningSecret is empty, want the decrypted plaintext returned on List too")
			}
		}
	}
}

func TestNotificationChannelHandler_List_ServiceError(t *testing.T) {
	router, cookie, _, _, _ := setupNotificationChannelAuthed(t, http.MethodGet, "/feature-flags/:id/notification-channels", func(cs *service.NotificationChannelService) gin.HandlerFunc {
		return NewNotificationChannelHandler(cs).List
	})

	rec := doRequest(router, http.MethodGet, "/feature-flags/does-not-exist/notification-channels", nil, cookie)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

// --- Update ---

func TestNotificationChannelHandler_Update_Success(t *testing.T) {
	router, cookie, channels, actor, flagID := setupNotificationChannelAuthed(t, http.MethodPatch, "/feature-flags/:id/notification-channels/:channelId", func(cs *service.NotificationChannelService) gin.HandlerFunc {
		return NewNotificationChannelHandler(cs).Update
	})
	ch, _, err := channels.Create(context.Background(), actor, flagID, service.CreateNotificationChannelInput{ChannelType: model.NotificationChannelTypeKafka, Destination: "topic"})
	if err != nil {
		t.Fatalf("seeding channel returned unexpected error: %v", err)
	}

	body, _ := json.Marshal(map[string]any{"enabled": false})
	rec := doRequest(router, http.MethodPatch, "/feature-flags/"+flagID+"/notification-channels/"+ch.ID, body, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp notificationChannelResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if resp.Enabled {
		t.Error("Enabled = true, want false after Update")
	}
}

func TestNotificationChannelHandler_Update_BindError(t *testing.T) {
	router, cookie, channels, actor, flagID := setupNotificationChannelAuthed(t, http.MethodPatch, "/feature-flags/:id/notification-channels/:channelId", func(cs *service.NotificationChannelService) gin.HandlerFunc {
		return NewNotificationChannelHandler(cs).Update
	})
	ch, _, err := channels.Create(context.Background(), actor, flagID, service.CreateNotificationChannelInput{ChannelType: model.NotificationChannelTypeKafka, Destination: "topic"})
	if err != nil {
		t.Fatalf("seeding channel returned unexpected error: %v", err)
	}

	rec := doRequest(router, http.MethodPatch, "/feature-flags/"+flagID+"/notification-channels/"+ch.ID, []byte("not json"), cookie)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestNotificationChannelHandler_Update_NotFound(t *testing.T) {
	router, cookie, _, _, flagID := setupNotificationChannelAuthed(t, http.MethodPatch, "/feature-flags/:id/notification-channels/:channelId", func(cs *service.NotificationChannelService) gin.HandlerFunc {
		return NewNotificationChannelHandler(cs).Update
	})

	body, _ := json.Marshal(map[string]any{"enabled": false})
	rec := doRequest(router, http.MethodPatch, "/feature-flags/"+flagID+"/notification-channels/does-not-exist", body, cookie)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

// TestNotificationChannelHandler_Update_CrossFlagID exercises the IDOR
// guard end to end through the handler: a channel created on flagA, patched
// through flagB's URL, must 404 rather than succeed.
func TestNotificationChannelHandler_Update_CrossFlagID(t *testing.T) {
	router, cookie, channels, actor, flagID := setupNotificationChannelAuthed(t, http.MethodPatch, "/feature-flags/:id/notification-channels/:channelId", func(cs *service.NotificationChannelService) gin.HandlerFunc {
		return NewNotificationChannelHandler(cs).Update
	})
	ch, _, err := channels.Create(context.Background(), actor, flagID, service.CreateNotificationChannelInput{ChannelType: model.NotificationChannelTypeKafka, Destination: "topic"})
	if err != nil {
		t.Fatalf("seeding channel returned unexpected error: %v", err)
	}

	body, _ := json.Marshal(map[string]any{"enabled": false})
	rec := doRequest(router, http.MethodPatch, "/feature-flags/some-other-flag-id/notification-channels/"+ch.ID, body, cookie)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

// --- Delete ---

func TestNotificationChannelHandler_Delete_Success(t *testing.T) {
	router, cookie, channels, actor, flagID := setupNotificationChannelAuthed(t, http.MethodDelete, "/feature-flags/:id/notification-channels/:channelId", func(cs *service.NotificationChannelService) gin.HandlerFunc {
		return NewNotificationChannelHandler(cs).Delete
	})
	ch, _, err := channels.Create(context.Background(), actor, flagID, service.CreateNotificationChannelInput{ChannelType: model.NotificationChannelTypeKafka, Destination: "topic"})
	if err != nil {
		t.Fatalf("seeding channel returned unexpected error: %v", err)
	}

	rec := doRequest(router, http.MethodDelete, "/feature-flags/"+flagID+"/notification-channels/"+ch.ID, nil, cookie)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}

	got, _, err := channels.List(context.Background(), flagID)
	if err != nil {
		t.Fatalf("List() returned unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("List() after Delete() = %+v, want empty", got)
	}
}

func TestNotificationChannelHandler_Delete_NotFound(t *testing.T) {
	router, cookie, _, _, flagID := setupNotificationChannelAuthed(t, http.MethodDelete, "/feature-flags/:id/notification-channels/:channelId", func(cs *service.NotificationChannelService) gin.HandlerFunc {
		return NewNotificationChannelHandler(cs).Delete
	})

	rec := doRequest(router, http.MethodDelete, "/feature-flags/"+flagID+"/notification-channels/does-not-exist", nil, cookie)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}
