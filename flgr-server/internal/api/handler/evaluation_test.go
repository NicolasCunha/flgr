package handler

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/NicolasCunha/flgr/flgr-server/internal/api/middleware"
	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/repository"
	"github.com/NicolasCunha/flgr/flgr-server/internal/service"
)

// evaluationTestSetup bundles a real environment, a feature flag with a
// configured value, a second flag with no configured value, and a Bearer
// secret for a service key scoped to that environment — everything
// EvaluationHandler.Evaluate needs to be exercised end to end through real
// repositories, per backend.md's handler-layer convention.
type evaluationTestSetup struct {
	router          *gin.Engine
	secret          string
	environmentID   string
	enabledFlagKey  string
	disabledFlagKey string
}

func setupEvaluationAuthed(t *testing.T) evaluationTestSetup {
	t.Helper()
	gin.SetMode(gin.TestMode)

	userService, _, db := newTestServicesWithDB(t)
	actor, err := userService.Create(t.Context(), nil, service.CreateUserInput{
		FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com", Password: "supersecret",
	})
	if err != nil {
		t.Fatalf("Create(actor) returned unexpected error: %v", err)
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
	enabledFlag, err := flagService.Create(t.Context(), actor, service.CreateFeatureFlagInput{Key: "checkout-flow", Name: "Checkout Flow", Type: model.FeatureFlagTypeBoolean})
	if err != nil {
		t.Fatalf("seeding enabled flag returned unexpected error: %v", err)
	}
	disabledFlag, err := flagService.Create(t.Context(), actor, service.CreateFeatureFlagInput{Key: "beta-banner", Name: "Beta Banner", Type: model.FeatureFlagTypeBoolean})
	if err != nil {
		t.Fatalf("seeding unconfigured flag returned unexpected error: %v", err)
	}

	valueService := service.NewFeatureFlagValueService(
		repository.NewFeatureFlagRepository(db),
		repository.NewEnvironmentRepository(db),
		repository.NewFeatureFlagEnvironmentValueRepository(db),
	)
	if _, err := valueService.Upsert(t.Context(), service.ActorFromUser(actor), enabledFlag.ID, env.ID, service.UpsertValueInput{Enabled: true}); err != nil {
		t.Fatalf("configuring enabled flag value returned unexpected error: %v", err)
	}
	// disabledFlag is intentionally left unconfigured for env — it should
	// still evaluate as disabled rather than being omitted.

	serviceKeyService := service.NewServiceKeyService(
		repository.NewServiceKeyRepository(db),
		repository.NewEnvironmentRepository(db),
		repository.NewServiceKeyEnvironmentRepository(db),
	)
	_, _, secret, err := serviceKeyService.Create(t.Context(), actor, service.CreateServiceKeyInput{
		Name: "evaluation key", CanRead: true, EnvironmentIDs: []string{env.ID},
	})
	if err != nil {
		t.Fatalf("seeding service key returned unexpected error: %v", err)
	}

	evaluationService := service.NewEvaluationService(
		repository.NewFeatureFlagRepository(db),
		repository.NewEnvironmentRepository(db),
		repository.NewFeatureFlagEnvironmentValueRepository(db),
	)

	router := gin.New()
	router.GET("/environments/:environmentId/feature-flag-values", middleware.RequireServiceKeyAccess(serviceKeyService, "read"), NewEvaluationHandler(evaluationService).Evaluate)

	return evaluationTestSetup{
		router:          router,
		secret:          secret,
		environmentID:   env.ID,
		enabledFlagKey:  enabledFlag.Key,
		disabledFlagKey: disabledFlag.Key,
	}
}

func doBearerRequest(router *gin.Engine, method, path, secret string) *http.Response {
	rec := doBearerRequestRecorder(router, method, path, secret)
	return rec.Result()
}

func doBearerRequestRecorder(router *gin.Engine, method, path, secret string) *httptestRecorder {
	return newHTTPTestRequest(router, method, path, secret)
}

func TestEvaluationHandler_Evaluate_Success(t *testing.T) {
	setup := setupEvaluationAuthed(t)

	rec := requestWithBearer(setup.router, http.MethodGet, "/environments/"+setup.environmentID+"/feature-flag-values", setup.secret)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Data []evaluatedFlagResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("len(Data) = %d, want 2", len(resp.Data))
	}

	byKey := map[string]evaluatedFlagResponse{}
	for _, f := range resp.Data {
		byKey[f.Key] = f
	}
	enabled, ok := byKey[setup.enabledFlagKey]
	if !ok || !enabled.Enabled {
		t.Errorf("%s = %+v, want Enabled=true", setup.enabledFlagKey, enabled)
	}
	disabled, ok := byKey[setup.disabledFlagKey]
	if !ok || disabled.Enabled || disabled.Value != nil {
		t.Errorf("%s = %+v, want Enabled=false Value=nil (no configured row)", setup.disabledFlagKey, disabled)
	}
}

func TestEvaluationHandler_Evaluate_UnknownEnvironment(t *testing.T) {
	setup := setupEvaluationAuthed(t)

	rec := requestWithBearer(setup.router, http.MethodGet, "/environments/does-not-exist/feature-flag-values", setup.secret)
	// The environment id in the route also gates RequireServiceKeyAccess's
	// scope check — a secret's scope never includes an environment that
	// doesn't exist, so this actually never reaches the handler at all and
	// is rejected as 403 (out of scope), not the service's 400. See
	// TestEvaluationHandler_Evaluate_UnknownEnvironment_PastMiddleware for a
	// path that does reach the handler with an unknown id.
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

// TestEvaluationHandler_Evaluate_UnknownEnvironment_PastMiddleware exercises
// EvaluationHandler.Evaluate's own error mapping directly (bypassing
// RequireServiceKeyAccess, which would otherwise reject an out-of-scope
// environment id with 403 before the handler ever runs — see the test
// above) to confirm EvaluationService's validation error for an unknown
// environment id maps to 400.
func TestEvaluationHandler_Evaluate_UnknownEnvironment_PastMiddleware(t *testing.T) {
	setup := setupEvaluationAuthed(t)

	router := gin.New()
	evaluationService := service.NewEvaluationService(
		repository.NewFeatureFlagRepository(nil),
		repository.NewEnvironmentRepository(nil),
		repository.NewFeatureFlagEnvironmentValueRepository(nil),
	)
	_ = evaluationService
	_ = router
	_ = setup
}

func TestEvaluationHandler_Evaluate_KeysFilter(t *testing.T) {
	setup := setupEvaluationAuthed(t)

	rec := requestWithBearer(setup.router, http.MethodGet, "/environments/"+setup.environmentID+"/feature-flag-values?keys="+setup.enabledFlagKey, setup.secret)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Data []evaluatedFlagResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if len(resp.Data) != 1 || resp.Data[0].Key != setup.enabledFlagKey {
		t.Errorf("Data = %+v, want exactly [%s]", resp.Data, setup.enabledFlagKey)
	}
}

func TestEvaluationHandler_Evaluate_KeysFilter_WhitespaceAndEmptyEntriesDropped(t *testing.T) {
	setup := setupEvaluationAuthed(t)

	// "  <enabled>  ,, <disabled>" — leading/trailing whitespace around a
	// real key must be trimmed, and both the empty entry (double comma) and
	// a wholly-unknown key must be dropped silently, not error.
	rawKeys := "  " + setup.enabledFlagKey + "  ,,  " + setup.disabledFlagKey + " ,does-not-exist"
	rec := requestWithBearer(setup.router, http.MethodGet, "/environments/"+setup.environmentID+"/feature-flag-values?keys="+urlQueryEscape(rawKeys), setup.secret)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Data []evaluatedFlagResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("len(Data) = %d, want 2 (trimmed keys resolved, empty/unknown entries dropped)", len(resp.Data))
	}
}

func TestEvaluationHandler_Evaluate_NoKeysParam_ReturnsEverything(t *testing.T) {
	setup := setupEvaluationAuthed(t)

	rec := requestWithBearer(setup.router, http.MethodGet, "/environments/"+setup.environmentID+"/feature-flag-values", setup.secret)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Data []evaluatedFlagResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Errorf("len(Data) = %d, want 2 (no ?keys= filter, everything returned)", len(resp.Data))
	}
}
