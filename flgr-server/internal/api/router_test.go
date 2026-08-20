package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/repository"
	"github.com/NicolasCunha/flgr/flgr-server/internal/service"
)

func TestNewRouter_HealthRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter(newTestDB(t), newTestConfig())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestNewRouter_SetupRoute_IsPublic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter(newTestDB(t), newTestConfig())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/setup", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestNewRouter_EnvironmentRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newTestDB(t)

	userRepo := repository.NewUserRepository(db)
	users := service.NewUserService(userRepo, service.NewAuthzService(repository.NewAuthzRepository(db)))
	admin, err := users.Create(t.Context(), nil, service.CreateUserInput{
		FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com", Password: "supersecret",
	})
	if err != nil {
		t.Fatalf("seeding admin returned unexpected error: %v", err)
	}
	if err := repository.NewUserProfileRepository(db).ReplaceProfiles(t.Context(), admin.ID, []string{model.AdministradorProfileID}, admin.ID); err != nil {
		t.Fatalf("granting Administrador profile returned unexpected error: %v", err)
	}

	router := NewRouter(db, newTestConfig())

	loginBody, _ := json.Marshal(map[string]string{"email": "ada@example.com", "password": "supersecret"})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d, body: %s", loginRec.Code, http.StatusOK, loginRec.Body.String())
	}
	cookie := loginRec.Result().Cookies()[0]

	categoriesReq := httptest.NewRequest(http.MethodGet, "/api/v1/environment-categories", nil)
	categoriesReq.AddCookie(cookie)
	categoriesRec := httptest.NewRecorder()
	router.ServeHTTP(categoriesRec, categoriesReq)
	if categoriesRec.Code != http.StatusOK {
		t.Fatalf("environment-categories status = %d, want %d, body: %s", categoriesRec.Code, http.StatusOK, categoriesRec.Body.String())
	}

	createBody, _ := json.Marshal(map[string]string{"name": "app-dev", "category_name": "Development"})
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/environments", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.AddCookie(cookie)
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create environment status = %d, want %d, body: %s", createRec.Code, http.StatusCreated, createRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/environments", nil)
	listReq.AddCookie(cookie)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list environments status = %d, want %d, body: %s", listRec.Code, http.StatusOK, listRec.Body.String())
	}
}

func TestNewRouter_EnvironmentRoutes_RequirePermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newTestDB(t)

	userRepo := repository.NewUserRepository(db)
	users := service.NewUserService(userRepo, service.NewAuthzService(repository.NewAuthzRepository(db)))
	// No Administrador profile granted — an ordinary, unprivileged user.
	if _, err := users.Create(t.Context(), nil, service.CreateUserInput{
		FirstName: "Grace", LastName: "Hopper", Email: "grace@example.com", Password: "supersecret",
	}); err != nil {
		t.Fatalf("seeding user returned unexpected error: %v", err)
	}

	router := NewRouter(db, newTestConfig())

	loginBody, _ := json.Marshal(map[string]string{"email": "grace@example.com", "password": "supersecret"})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)
	cookie := loginRec.Result().Cookies()[0]

	createBody, _ := json.Marshal(map[string]string{"name": "app-dev", "category_name": "Development"})
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/environments", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.AddCookie(cookie)
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusForbidden {
		t.Fatalf("create environment status = %d, want %d, body: %s", createRec.Code, http.StatusForbidden, createRec.Body.String())
	}
}

func TestNewRouter_ServiceKeyRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newTestDB(t)

	userRepo := repository.NewUserRepository(db)
	users := service.NewUserService(userRepo, service.NewAuthzService(repository.NewAuthzRepository(db)))
	admin, err := users.Create(t.Context(), nil, service.CreateUserInput{
		FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com", Password: "supersecret",
	})
	if err != nil {
		t.Fatalf("seeding admin returned unexpected error: %v", err)
	}
	if err := repository.NewUserProfileRepository(db).ReplaceProfiles(t.Context(), admin.ID, []string{model.AdministradorProfileID}, admin.ID); err != nil {
		t.Fatalf("granting Administrador profile returned unexpected error: %v", err)
	}

	environmentCategoryID, err := repository.NewEnvironmentCategoryRepository(db).GetByName(t.Context(), "Development")
	if err != nil {
		t.Fatalf("fetching seeded Development category returned unexpected error: %v", err)
	}
	ts := time.Now().UTC()
	env := &model.Environment{
		ID: "env-1", Name: "app-dev", EnvironmentCategoryID: environmentCategoryID.ID,
		AuditFields: model.AuditFields{CreatedOn: ts, ModifiedOn: ts, CreatedByUserID: &admin.ID, ModifiedByUserID: &admin.ID},
	}
	if err := repository.NewEnvironmentRepository(db).Create(t.Context(), env); err != nil {
		t.Fatalf("seeding environment returned unexpected error: %v", err)
	}

	router := NewRouter(db, newTestConfig())

	loginBody, _ := json.Marshal(map[string]string{"email": "ada@example.com", "password": "supersecret"})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d, body: %s", loginRec.Code, http.StatusOK, loginRec.Body.String())
	}
	cookie := loginRec.Result().Cookies()[0]

	createBody, _ := json.Marshal(map[string]any{"name": "checkout-service prod key", "can_read": true, "environment_ids": []string{"env-1"}})
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/service-keys", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.AddCookie(cookie)
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create service key status = %d, want %d, body: %s", createRec.Code, http.StatusCreated, createRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/service-keys", nil)
	listReq.AddCookie(cookie)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list service keys status = %d, want %d, body: %s", listRec.Code, http.StatusOK, listRec.Body.String())
	}
}

func TestNewRouter_ServiceKeyRoutes_RequirePermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newTestDB(t)

	userRepo := repository.NewUserRepository(db)
	users := service.NewUserService(userRepo, service.NewAuthzService(repository.NewAuthzRepository(db)))
	// No Administrador profile granted — an ordinary, unprivileged user.
	if _, err := users.Create(t.Context(), nil, service.CreateUserInput{
		FirstName: "Grace", LastName: "Hopper", Email: "grace@example.com", Password: "supersecret",
	}); err != nil {
		t.Fatalf("seeding user returned unexpected error: %v", err)
	}

	router := NewRouter(db, newTestConfig())

	loginBody, _ := json.Marshal(map[string]string{"email": "grace@example.com", "password": "supersecret"})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)
	cookie := loginRec.Result().Cookies()[0]

	createBody, _ := json.Marshal(map[string]any{"name": "x", "can_read": true, "environment_ids": []string{"env-1"}})
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/service-keys", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.AddCookie(cookie)
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusForbidden {
		t.Fatalf("create service key status = %d, want %d, body: %s", createRec.Code, http.StatusForbidden, createRec.Body.String())
	}
}

func TestNewRouter_FeatureFlagRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newTestDB(t)

	userRepo := repository.NewUserRepository(db)
	users := service.NewUserService(userRepo, service.NewAuthzService(repository.NewAuthzRepository(db)))
	admin, err := users.Create(t.Context(), nil, service.CreateUserInput{
		FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com", Password: "supersecret",
	})
	if err != nil {
		t.Fatalf("seeding admin returned unexpected error: %v", err)
	}
	if err := repository.NewUserProfileRepository(db).ReplaceProfiles(t.Context(), admin.ID, []string{model.AdministradorProfileID}, admin.ID); err != nil {
		t.Fatalf("granting Administrador profile returned unexpected error: %v", err)
	}

	environmentCategoryID, err := repository.NewEnvironmentCategoryRepository(db).GetByName(t.Context(), "Development")
	if err != nil {
		t.Fatalf("fetching seeded Development category returned unexpected error: %v", err)
	}
	ts := time.Now().UTC()
	env := &model.Environment{
		ID: "env-1", Name: "app-dev", EnvironmentCategoryID: environmentCategoryID.ID,
		AuditFields: model.AuditFields{CreatedOn: ts, ModifiedOn: ts, CreatedByUserID: &admin.ID, ModifiedByUserID: &admin.ID},
	}
	if err := repository.NewEnvironmentRepository(db).Create(t.Context(), env); err != nil {
		t.Fatalf("seeding environment returned unexpected error: %v", err)
	}

	router := NewRouter(db, newTestConfig())

	loginBody, _ := json.Marshal(map[string]string{"email": "ada@example.com", "password": "supersecret"})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d, body: %s", loginRec.Code, http.StatusOK, loginRec.Body.String())
	}
	cookie := loginRec.Result().Cookies()[0]

	createBody, _ := json.Marshal(map[string]string{"key": "new-checkout-flow", "name": "New Checkout Flow", "type": model.FeatureFlagTypeBoolean})
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/feature-flags", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.AddCookie(cookie)
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create feature flag status = %d, want %d, body: %s", createRec.Code, http.StatusCreated, createRec.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshaling create response: %v", err)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/feature-flags", nil)
	listReq.AddCookie(cookie)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list feature flags status = %d, want %d, body: %s", listRec.Code, http.StatusOK, listRec.Body.String())
	}

	upsertBody, _ := json.Marshal(map[string]any{"enabled": true})
	upsertReq := httptest.NewRequest(http.MethodPatch, "/api/v1/feature-flags/"+created.ID+"/values/env-1", bytes.NewReader(upsertBody))
	upsertReq.Header.Set("Content-Type", "application/json")
	upsertReq.AddCookie(cookie)
	upsertRec := httptest.NewRecorder()
	router.ServeHTTP(upsertRec, upsertReq)
	if upsertRec.Code != http.StatusOK {
		t.Fatalf("upsert feature flag value status = %d, want %d, body: %s", upsertRec.Code, http.StatusOK, upsertRec.Body.String())
	}

	valuesReq := httptest.NewRequest(http.MethodGet, "/api/v1/feature-flags/"+created.ID+"/values", nil)
	valuesReq.AddCookie(cookie)
	valuesRec := httptest.NewRecorder()
	router.ServeHTTP(valuesRec, valuesReq)
	if valuesRec.Code != http.StatusOK {
		t.Fatalf("list feature flag values status = %d, want %d, body: %s", valuesRec.Code, http.StatusOK, valuesRec.Body.String())
	}
}

func TestNewRouter_FeatureFlagRoutes_RequirePermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newTestDB(t)

	userRepo := repository.NewUserRepository(db)
	users := service.NewUserService(userRepo, service.NewAuthzService(repository.NewAuthzRepository(db)))
	// No Administrador profile granted — an ordinary, unprivileged user.
	if _, err := users.Create(t.Context(), nil, service.CreateUserInput{
		FirstName: "Grace", LastName: "Hopper", Email: "grace@example.com", Password: "supersecret",
	}); err != nil {
		t.Fatalf("seeding user returned unexpected error: %v", err)
	}

	router := NewRouter(db, newTestConfig())

	loginBody, _ := json.Marshal(map[string]string{"email": "grace@example.com", "password": "supersecret"})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)
	cookie := loginRec.Result().Cookies()[0]

	createBody, _ := json.Marshal(map[string]string{"key": "x", "name": "x", "type": model.FeatureFlagTypeBoolean})
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/feature-flags", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.AddCookie(cookie)
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusForbidden {
		t.Fatalf("create feature flag status = %d, want %d, body: %s", createRec.Code, http.StatusForbidden, createRec.Body.String())
	}
}

// seedAdminAndEnv seeds a user granted the Administrador profile plus a
// real environment (id "env-1"), logs the admin in, and returns the router
// and login cookie — the common setup every Phase 6 route wiring test below
// needs (a permission-gated route requires a flag and value/channel target
// to exercise, not just a bare permission check).
func seedAdminAndEnv(t *testing.T) (router *gin.Engine, cookie *http.Cookie, db *sql.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db = newTestDB(t)

	userRepo := repository.NewUserRepository(db)
	users := service.NewUserService(userRepo, service.NewAuthzService(repository.NewAuthzRepository(db)))
	admin, err := users.Create(t.Context(), nil, service.CreateUserInput{
		FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com", Password: "supersecret",
	})
	if err != nil {
		t.Fatalf("seeding admin returned unexpected error: %v", err)
	}
	if err := repository.NewUserProfileRepository(db).ReplaceProfiles(t.Context(), admin.ID, []string{model.AdministradorProfileID}, admin.ID); err != nil {
		t.Fatalf("granting Administrador profile returned unexpected error: %v", err)
	}

	environmentCategoryID, err := repository.NewEnvironmentCategoryRepository(db).GetByName(t.Context(), "Development")
	if err != nil {
		t.Fatalf("fetching seeded Development category returned unexpected error: %v", err)
	}
	ts := time.Now().UTC()
	env := &model.Environment{
		ID: "env-1", Name: "app-dev", EnvironmentCategoryID: environmentCategoryID.ID,
		AuditFields: model.AuditFields{CreatedOn: ts, ModifiedOn: ts, CreatedByUserID: &admin.ID, ModifiedByUserID: &admin.ID},
	}
	if err := repository.NewEnvironmentRepository(db).Create(t.Context(), env); err != nil {
		t.Fatalf("seeding environment returned unexpected error: %v", err)
	}

	router = NewRouter(db, newTestConfig())

	loginBody, _ := json.Marshal(map[string]string{"email": "ada@example.com", "password": "supersecret"})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d, body: %s", loginRec.Code, http.StatusOK, loginRec.Body.String())
	}

	return router, loginRec.Result().Cookies()[0], db
}

// seedUnprivilegedUser logs in an ordinary user (no profile granted) against
// an already-running router, for the RequirePermission negative-path tests.
func seedUnprivilegedUser(t *testing.T, db *sql.DB, router *gin.Engine) *http.Cookie {
	t.Helper()
	userRepo := repository.NewUserRepository(db)
	users := service.NewUserService(userRepo, service.NewAuthzService(repository.NewAuthzRepository(db)))
	if _, err := users.Create(t.Context(), nil, service.CreateUserInput{
		FirstName: "Grace", LastName: "Hopper", Email: "grace@example.com", Password: "supersecret",
	}); err != nil {
		t.Fatalf("seeding user returned unexpected error: %v", err)
	}

	loginBody, _ := json.Marshal(map[string]string{"email": "grace@example.com", "password": "supersecret"})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login (unprivileged) status = %d, want %d, body: %s", loginRec.Code, http.StatusOK, loginRec.Body.String())
	}
	return loginRec.Result().Cookies()[0]
}

func TestNewRouter_NotificationChannelRoutes(t *testing.T) {
	router, cookie, _ := seedAdminAndEnv(t)

	createBody, _ := json.Marshal(map[string]string{"key": "checkout-flow", "name": "Checkout Flow", "type": model.FeatureFlagTypeBoolean})
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/feature-flags", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.AddCookie(cookie)
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create feature flag status = %d, want %d, body: %s", createRec.Code, http.StatusCreated, createRec.Body.String())
	}
	var flag struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(createRec.Body.Bytes(), &flag); err != nil {
		t.Fatalf("unmarshaling create response: %v", err)
	}

	channelBody, _ := json.Marshal(map[string]string{"channel_type": model.NotificationChannelTypeKafka, "destination": "flag-events"})
	channelReq := httptest.NewRequest(http.MethodPost, "/api/v1/feature-flags/"+flag.ID+"/notification-channels", bytes.NewReader(channelBody))
	channelReq.Header.Set("Content-Type", "application/json")
	channelReq.AddCookie(cookie)
	channelRec := httptest.NewRecorder()
	router.ServeHTTP(channelRec, channelReq)
	if channelRec.Code != http.StatusCreated {
		t.Fatalf("create notification channel status = %d, want %d, body: %s", channelRec.Code, http.StatusCreated, channelRec.Body.String())
	}
	var channel struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(channelRec.Body.Bytes(), &channel); err != nil {
		t.Fatalf("unmarshaling create channel response: %v", err)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/feature-flags/"+flag.ID+"/notification-channels", nil)
	listReq.AddCookie(cookie)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list notification channels status = %d, want %d, body: %s", listRec.Code, http.StatusOK, listRec.Body.String())
	}

	updateBody, _ := json.Marshal(map[string]any{"enabled": false})
	updateReq := httptest.NewRequest(http.MethodPatch, "/api/v1/feature-flags/"+flag.ID+"/notification-channels/"+channel.ID, bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.AddCookie(cookie)
	updateRec := httptest.NewRecorder()
	router.ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("update notification channel status = %d, want %d, body: %s", updateRec.Code, http.StatusOK, updateRec.Body.String())
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/feature-flags/"+flag.ID+"/notification-channels/"+channel.ID, nil)
	deleteReq.AddCookie(cookie)
	deleteRec := httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("delete notification channel status = %d, want %d, body: %s", deleteRec.Code, http.StatusNoContent, deleteRec.Body.String())
	}
}

// TestNewRouter_NotificationChannelRoutes_RequirePermission confirms every
// notification-channel route is actually wired through RequirePermission
// (FeatureFlag: Edit for write routes, FeatureFlag: View for List) — an
// unprivileged user must get 403, not fall through unauthorized.
func TestNewRouter_NotificationChannelRoutes_RequirePermission(t *testing.T) {
	router, adminCookie, db := seedAdminAndEnv(t)

	createBody, _ := json.Marshal(map[string]string{"key": "checkout-flow", "name": "Checkout Flow", "type": model.FeatureFlagTypeBoolean})
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/feature-flags", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.AddCookie(adminCookie)
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)
	var flag struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(createRec.Body.Bytes(), &flag); err != nil {
		t.Fatalf("unmarshaling create response: %v", err)
	}

	cookie := seedUnprivilegedUser(t, db, router)

	channelBody, _ := json.Marshal(map[string]string{"channel_type": model.NotificationChannelTypeKafka, "destination": "flag-events"})
	channelReq := httptest.NewRequest(http.MethodPost, "/api/v1/feature-flags/"+flag.ID+"/notification-channels", bytes.NewReader(channelBody))
	channelReq.Header.Set("Content-Type", "application/json")
	channelReq.AddCookie(cookie)
	channelRec := httptest.NewRecorder()
	router.ServeHTTP(channelRec, channelReq)
	if channelRec.Code != http.StatusForbidden {
		t.Fatalf("create notification channel (unprivileged) status = %d, want %d, body: %s", channelRec.Code, http.StatusForbidden, channelRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/feature-flags/"+flag.ID+"/notification-channels", nil)
	listReq.AddCookie(cookie)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusForbidden {
		t.Fatalf("list notification channels (unprivileged) status = %d, want %d, body: %s", listRec.Code, http.StatusForbidden, listRec.Body.String())
	}

	updateBody, _ := json.Marshal(map[string]any{"enabled": false})
	updateReq := httptest.NewRequest(http.MethodPatch, "/api/v1/feature-flags/"+flag.ID+"/notification-channels/irrelevant-channel-id", bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.AddCookie(cookie)
	updateRec := httptest.NewRecorder()
	router.ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusForbidden {
		t.Fatalf("update notification channel (unprivileged) status = %d, want %d, body: %s", updateRec.Code, http.StatusForbidden, updateRec.Body.String())
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/feature-flags/"+flag.ID+"/notification-channels/irrelevant-channel-id", nil)
	deleteReq.AddCookie(cookie)
	deleteRec := httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusForbidden {
		t.Fatalf("delete notification channel (unprivileged) status = %d, want %d, body: %s", deleteRec.Code, http.StatusForbidden, deleteRec.Body.String())
	}
}

func TestNewRouter_KillswitchRoute(t *testing.T) {
	router, cookie, _ := seedAdminAndEnv(t)

	createBody, _ := json.Marshal(map[string]string{"key": "checkout-flow", "name": "Checkout Flow", "type": model.FeatureFlagTypeBoolean})
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/feature-flags", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.AddCookie(cookie)
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create feature flag status = %d, want %d, body: %s", createRec.Code, http.StatusCreated, createRec.Body.String())
	}
	var flag struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(createRec.Body.Bytes(), &flag); err != nil {
		t.Fatalf("unmarshaling create response: %v", err)
	}

	killswitchReq := httptest.NewRequest(http.MethodPost, "/api/v1/feature-flags/"+flag.ID+"/values/env-1/killswitch", nil)
	killswitchReq.AddCookie(cookie)
	killswitchRec := httptest.NewRecorder()
	router.ServeHTTP(killswitchRec, killswitchReq)
	if killswitchRec.Code != http.StatusOK {
		t.Fatalf("killswitch status = %d, want %d, body: %s", killswitchRec.Code, http.StatusOK, killswitchRec.Body.String())
	}

	var value struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.Unmarshal(killswitchRec.Body.Bytes(), &value); err != nil {
		t.Fatalf("unmarshaling killswitch response: %v", err)
	}
	if value.Enabled {
		t.Error("Enabled = true, want false after the killswitch fires")
	}
}

// TestNewRouter_KillswitchRoute_RequirePermission confirms the Killswitch
// route is wired through RequirePermission (FeatureFlagValue: Write, the
// same gate as the ordinary value Upsert route) — an unprivileged user must
// get 403.
func TestNewRouter_KillswitchRoute_RequirePermission(t *testing.T) {
	router, adminCookie, db := seedAdminAndEnv(t)

	createBody, _ := json.Marshal(map[string]string{"key": "checkout-flow", "name": "Checkout Flow", "type": model.FeatureFlagTypeBoolean})
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/feature-flags", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.AddCookie(adminCookie)
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)
	var flag struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(createRec.Body.Bytes(), &flag); err != nil {
		t.Fatalf("unmarshaling create response: %v", err)
	}

	cookie := seedUnprivilegedUser(t, db, router)

	killswitchReq := httptest.NewRequest(http.MethodPost, "/api/v1/feature-flags/"+flag.ID+"/values/env-1/killswitch", nil)
	killswitchReq.AddCookie(cookie)
	killswitchRec := httptest.NewRecorder()
	router.ServeHTTP(killswitchRec, killswitchReq)
	if killswitchRec.Code != http.StatusForbidden {
		t.Fatalf("killswitch (unprivileged) status = %d, want %d, body: %s", killswitchRec.Code, http.StatusForbidden, killswitchRec.Body.String())
	}
}

func TestNewRouter_ProtectedRoute_RequiresAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter(newTestDB(t), newTestConfig())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestNewRouter_LoginThenAccessProtectedRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newTestDB(t)

	// Seed a user directly through the service layer, bypassing HTTP,
	// since creating the very first user always requires an authenticated
	// actor except for this one bootstrap path (see UserService.Create).
	// Granted the seeded Administrador profile directly (bypassing the
	// setup wizard, which normally does this — see SetupHandler.Complete)
	// so this user can exercise the permission-gated routes below.
	userRepo := repository.NewUserRepository(db)
	users := service.NewUserService(userRepo, service.NewAuthzService(repository.NewAuthzRepository(db)))
	seeded, err := users.Create(t.Context(), nil, service.CreateUserInput{
		FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com", Password: "supersecret",
	})
	if err != nil {
		t.Fatalf("seeding user returned unexpected error: %v", err)
	}
	if err := repository.NewUserProfileRepository(db).ReplaceProfiles(t.Context(), seeded.ID, []string{model.AdministradorProfileID}, seeded.ID); err != nil {
		t.Fatalf("granting Administrador profile returned unexpected error: %v", err)
	}

	router := NewRouter(db, newTestConfig())

	loginBody, _ := json.Marshal(map[string]string{"email": "ada@example.com", "password": "supersecret"})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)

	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d, body: %s", loginRec.Code, http.StatusOK, loginRec.Body.String())
	}

	cookies := loginRec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("len(cookies) = %d, want 1", len(cookies))
	}

	meReq := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	meReq.AddCookie(cookies[0])
	meRec := httptest.NewRecorder()
	router.ServeHTTP(meRec, meReq)

	if meRec.Code != http.StatusOK {
		t.Fatalf("me status = %d, want %d, body: %s", meRec.Code, http.StatusOK, meRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	listReq.AddCookie(cookies[0])
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d, body: %s", listRec.Code, http.StatusOK, listRec.Body.String())
	}

	logoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/logout", nil)
	logoutReq.AddCookie(cookies[0])
	logoutRec := httptest.NewRecorder()
	router.ServeHTTP(logoutRec, logoutReq)

	if logoutRec.Code != http.StatusOK {
		t.Fatalf("logout status = %d, want %d, body: %s", logoutRec.Code, http.StatusOK, logoutRec.Body.String())
	}

	afterLogoutReq := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	afterLogoutReq.AddCookie(cookies[0])
	afterLogoutRec := httptest.NewRecorder()
	router.ServeHTTP(afterLogoutRec, afterLogoutReq)

	if afterLogoutRec.Code != http.StatusUnauthorized {
		t.Fatalf("me after logout status = %d, want %d", afterLogoutRec.Code, http.StatusUnauthorized)
	}
}
