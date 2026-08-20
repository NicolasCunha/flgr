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

// setupFeatureFlagValueAuthed mirrors setupServiceKeyAuthed but for
// FeatureFlagValueService-backed routes. It also returns a real feature
// flag id and environment id to write values against.
func setupFeatureFlagValueAuthed(t *testing.T, method, path string, makeHandler func(*service.FeatureFlagValueService) gin.HandlerFunc) (router *gin.Engine, cookie *http.Cookie, values *service.FeatureFlagValueService, actor *model.User, flagID, stringFlagID, environmentID string) {
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
	stringFlag, err := flagService.Create(t.Context(), actor, service.CreateFeatureFlagInput{Key: "welcome-message", Name: "Welcome Message", Type: model.FeatureFlagTypeString})
	if err != nil {
		t.Fatalf("seeding string feature flag returned unexpected error: %v", err)
	}

	valueService := service.NewFeatureFlagValueService(
		repository.NewFeatureFlagRepository(db),
		repository.NewEnvironmentRepository(db),
		repository.NewFeatureFlagEnvironmentValueRepository(db),
	)

	router = gin.New()
	router.Handle(method, path, middleware.RequireAuth(authService), makeHandler(valueService))

	return router, &http.Cookie{Name: middleware.SessionCookieName, Value: token}, valueService, actor, flag.ID, stringFlag.ID, env.ID
}

func TestFeatureFlagValueHandler_List_Empty(t *testing.T) {
	router, cookie, _, _, flagID, _, _ := setupFeatureFlagValueAuthed(t, http.MethodGet, "/feature-flags/:id/values", func(vs *service.FeatureFlagValueService) gin.HandlerFunc { return NewFeatureFlagValueHandler(vs).List })

	rec := doRequest(router, http.MethodGet, "/feature-flags/"+flagID+"/values", nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp struct {
		Data []featureFlagValueResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if len(resp.Data) != 0 {
		t.Errorf("len(Data) = %d, want 0 (no values configured yet)", len(resp.Data))
	}
}

func TestFeatureFlagValueHandler_List_WithValues(t *testing.T) {
	router, cookie, values, actor, flagID, _, envID := setupFeatureFlagValueAuthed(t, http.MethodGet, "/feature-flags/:id/values", func(vs *service.FeatureFlagValueService) gin.HandlerFunc {
		return NewFeatureFlagValueHandler(vs).List
	})
	if _, err := values.Upsert(context.Background(), service.ActorFromUser(actor), flagID, envID, service.UpsertValueInput{Enabled: true}); err != nil {
		t.Fatalf("Upsert() returned unexpected error: %v", err)
	}

	rec := doRequest(router, http.MethodGet, "/feature-flags/"+flagID+"/values", nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp struct {
		Data []featureFlagValueResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if len(resp.Data) != 1 || resp.Data[0].EnvironmentID != envID || !resp.Data[0].Enabled {
		t.Errorf("Data = %+v, want one enabled value for %q", resp.Data, envID)
	}
}

func TestFeatureFlagValueHandler_List_ServiceError(t *testing.T) {
	router, cookie, _, _, _, _, _ := setupFeatureFlagValueAuthed(t, http.MethodGet, "/feature-flags/:id/values", func(vs *service.FeatureFlagValueService) gin.HandlerFunc { return NewFeatureFlagValueHandler(vs).List })

	rec := doRequest(router, http.MethodGet, "/feature-flags/does-not-exist/values", nil, cookie)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestFeatureFlagValueHandler_Upsert_Success(t *testing.T) {
	router, cookie, _, _, flagID, _, envID := setupFeatureFlagValueAuthed(t, http.MethodPatch, "/feature-flags/:id/values/:environmentId", func(vs *service.FeatureFlagValueService) gin.HandlerFunc {
		return NewFeatureFlagValueHandler(vs).Upsert
	})

	body, _ := json.Marshal(map[string]any{"enabled": true})
	rec := doRequest(router, http.MethodPatch, "/feature-flags/"+flagID+"/values/"+envID, body, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp featureFlagValueResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if !resp.Enabled || resp.FeatureFlagID != flagID || resp.EnvironmentID != envID {
		t.Errorf("response = %+v, want Enabled=true FeatureFlagID=%q EnvironmentID=%q", resp, flagID, envID)
	}
}

func TestFeatureFlagValueHandler_Upsert_ListReflectsIt(t *testing.T) {
	router, cookie, values, actor, flagID, _, envID := setupFeatureFlagValueAuthed(t, http.MethodPatch, "/feature-flags/:id/values/:environmentId", func(vs *service.FeatureFlagValueService) gin.HandlerFunc {
		return NewFeatureFlagValueHandler(vs).Upsert
	})

	body, _ := json.Marshal(map[string]any{"enabled": true})
	rec := doRequest(router, http.MethodPatch, "/feature-flags/"+flagID+"/values/"+envID, body, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	got, err := values.List(context.Background(), flagID)
	if err != nil {
		t.Fatalf("List() returned unexpected error: %v", err)
	}
	if len(got) != 1 || !got[0].Enabled || got[0].ModifiedByUserID == nil || *got[0].ModifiedByUserID != actor.ID {
		t.Errorf("List() = %+v, want one enabled value modified by %q", got, actor.ID)
	}
}

func TestFeatureFlagValueHandler_Upsert_BindError(t *testing.T) {
	router, cookie, _, _, flagID, _, envID := setupFeatureFlagValueAuthed(t, http.MethodPatch, "/feature-flags/:id/values/:environmentId", func(vs *service.FeatureFlagValueService) gin.HandlerFunc {
		return NewFeatureFlagValueHandler(vs).Upsert
	})

	rec := doRequest(router, http.MethodPatch, "/feature-flags/"+flagID+"/values/"+envID, []byte("not json"), cookie)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestFeatureFlagValueHandler_Upsert_ServiceError(t *testing.T) {
	router, cookie, _, _, _, _, envID := setupFeatureFlagValueAuthed(t, http.MethodPatch, "/feature-flags/:id/values/:environmentId", func(vs *service.FeatureFlagValueService) gin.HandlerFunc {
		return NewFeatureFlagValueHandler(vs).Upsert
	})

	body, _ := json.Marshal(map[string]any{"enabled": true})
	rec := doRequest(router, http.MethodPatch, "/feature-flags/does-not-exist/values/"+envID, body, cookie)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestFeatureFlagValueHandler_Upsert_ValidationError(t *testing.T) {
	router, cookie, _, _, _, stringFlagID, envID := setupFeatureFlagValueAuthed(t, http.MethodPatch, "/feature-flags/:id/values/:environmentId", func(vs *service.FeatureFlagValueService) gin.HandlerFunc {
		return NewFeatureFlagValueHandler(vs).Upsert
	})

	// Enabling a non-Boolean flag with no value (and none previously
	// configured) is rejected — see resolveFeatureFlagValue.
	body, _ := json.Marshal(map[string]any{"enabled": true})
	rec := doRequest(router, http.MethodPatch, "/feature-flags/"+stringFlagID+"/values/"+envID, body, cookie)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}
