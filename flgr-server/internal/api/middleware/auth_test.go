package middleware

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/NicolasCunha/flgr/flgr-server/internal/database"
	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/repository"
	"github.com/NicolasCunha/flgr/flgr-server/internal/service"
)

// serviceKeyTestSetup bundles everything RequireServiceKeyAccess and the
// Bearer branch of RequireWriteAccess need: a real environment, a
// ServiceKeyService backed by the same db, and the admin *model.User
// (already granted Administrador, per grantAdministrador) used to create
// service keys.
type serviceKeyTestSetup struct {
	db          *sql.DB
	serviceKeys *service.ServiceKeyService
	environment *model.Environment
	admin       *model.User
}

func newServiceKeyTestSetup(t *testing.T) *serviceKeyTestSetup {
	t.Helper()
	db := newTestDB(t)

	userRepo := repository.NewUserRepository(db)
	authzService := service.NewAuthzService(repository.NewAuthzRepository(db))
	userService := service.NewUserService(userRepo, authzService)
	admin, err := userService.Create(t.Context(), nil, service.CreateUserInput{
		FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com", Password: "supersecret",
	})
	if err != nil {
		t.Fatalf("Create(admin) returned unexpected error: %v", err)
	}
	if err := repository.NewUserProfileRepository(db).ReplaceProfiles(t.Context(), admin.ID, []string{model.AdministradorProfileID}, admin.ID); err != nil {
		t.Fatalf("granting Administrador returned unexpected error: %v", err)
	}

	environmentService := service.NewEnvironmentService(
		repository.NewEnvironmentRepository(db),
		repository.NewEnvironmentCategoryRepository(db),
	)
	env, err := environmentService.Create(t.Context(), admin, service.CreateEnvironmentInput{Name: "prod", CategoryName: "Production"})
	if err != nil {
		t.Fatalf("seeding environment returned unexpected error: %v", err)
	}

	serviceKeys := service.NewServiceKeyService(
		repository.NewServiceKeyRepository(db),
		repository.NewEnvironmentRepository(db),
		repository.NewServiceKeyEnvironmentRepository(db),
	)

	return &serviceKeyTestSetup{db: db, serviceKeys: serviceKeys, environment: env, admin: admin}
}

// createServiceKey seeds a service key linked to setup.environment with the
// given capabilities, returning its plaintext secret.
func (setup *serviceKeyTestSetup) createServiceKey(t *testing.T, canRead, canWrite bool) string {
	t.Helper()
	_, _, secret, err := setup.serviceKeys.Create(t.Context(), setup.admin, service.CreateServiceKeyInput{
		Name:           "test key",
		CanRead:        canRead,
		CanWrite:       canWrite,
		EnvironmentIDs: []string{setup.environment.ID},
	})
	if err != nil {
		t.Fatalf("seeding service key returned unexpected error: %v", err)
	}
	return secret
}

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("database.Open() returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := database.Migrate(db, "../../../migrations"); err != nil {
		t.Fatalf("database.Migrate() returned unexpected error: %v", err)
	}

	return db
}

func newTestAuthService(t *testing.T) (*service.AuthService, *service.UserService) {
	t.Helper()
	db := newTestDB(t)
	userRepo := repository.NewUserRepository(db)
	return service.NewAuthService(userRepo, repository.NewSessionRepository(db), repository.NewLoginAttemptRepository(db)),
		service.NewUserService(userRepo, service.NewAuthzService(repository.NewAuthzRepository(db)))
}

func TestRequireAuth_MissingCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authService, _ := newTestAuthService(t)

	router := gin.New()
	router.GET("/protected", RequireAuth(authService), func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestRequireAuth_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authService, _ := newTestAuthService(t)

	router := gin.New()
	router.GET("/protected", RequireAuth(authService), func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "bogus-token"})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestRequireAuth_ValidSession_SetsCurrentUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authService, userService := newTestAuthService(t)

	if _, err := userService.Create(t.Context(), nil, service.CreateUserInput{
		FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com", Password: "supersecret",
	}); err != nil {
		t.Fatalf("Create(user) returned unexpected error: %v", err)
	}
	_, token, err := authService.Login(t.Context(), "ada@example.com", "supersecret")
	if err != nil {
		t.Fatalf("Login() returned unexpected error: %v", err)
	}

	var gotUser *model.User
	router := gin.New()
	router.GET("/protected", RequireAuth(authService), func(c *gin.Context) {
		gotUser = CurrentUser(c)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: token})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if gotUser == nil || gotUser.Email != "ada@example.com" {
		t.Fatalf("CurrentUser() = %v, want the logged-in user", gotUser)
	}
}

func TestCurrentUser_NotSet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	if got := CurrentUser(c); got != nil {
		t.Errorf("CurrentUser() = %v, want nil", got)
	}
}

func TestCurrentServiceKey_NotSet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	if got := CurrentServiceKey(c); got != nil {
		t.Errorf("CurrentServiceKey() = %v, want nil", got)
	}
}

func TestCurrentServiceKeyEnvironmentIDs_NotSet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	if got := CurrentServiceKeyEnvironmentIDs(c); got != nil {
		t.Errorf("CurrentServiceKeyEnvironmentIDs() = %v, want nil", got)
	}
}

func TestRequireServiceKeyAccess_MissingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setup := newServiceKeyTestSetup(t)

	router := gin.New()
	router.GET("/environments/:environmentId/feature-flag-values", RequireServiceKeyAccess(setup.serviceKeys, "read"), func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/environments/"+setup.environment.ID+"/feature-flag-values", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestRequireServiceKeyAccess_MalformedHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setup := newServiceKeyTestSetup(t)

	router := gin.New()
	router.GET("/environments/:environmentId/feature-flag-values", RequireServiceKeyAccess(setup.serviceKeys, "read"), func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/environments/"+setup.environment.ID+"/feature-flag-values", nil)
	req.Header.Set("Authorization", "Basic bogus-scheme")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestRequireServiceKeyAccess_EmptyBearerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setup := newServiceKeyTestSetup(t)

	router := gin.New()
	router.GET("/environments/:environmentId/feature-flag-values", RequireServiceKeyAccess(setup.serviceKeys, "read"), func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/environments/"+setup.environment.ID+"/feature-flag-values", nil)
	req.Header.Set("Authorization", "Bearer ")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestRequireServiceKeyAccess_UnknownSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setup := newServiceKeyTestSetup(t)

	router := gin.New()
	router.GET("/environments/:environmentId/feature-flag-values", RequireServiceKeyAccess(setup.serviceKeys, "read"), func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/environments/"+setup.environment.ID+"/feature-flag-values", nil)
	req.Header.Set("Authorization", "Bearer bogus-secret")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestRequireServiceKeyAccess_InactiveKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setup := newServiceKeyTestSetup(t)
	secret := setup.createServiceKey(t, true, false)

	// Deactivate the key straight through the repository (no ServiceKeyService
	// call needed) — status is the only field that matters for this test.
	if _, err := setup.db.Exec("UPDATE service_keys SET status = ?", model.ServiceKeyStatusInactive); err != nil {
		t.Fatalf("deactivating service key: %v", err)
	}

	router := gin.New()
	router.GET("/environments/:environmentId/feature-flag-values", RequireServiceKeyAccess(setup.serviceKeys, "read"), func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/environments/"+setup.environment.ID+"/feature-flag-values", nil)
	req.Header.Set("Authorization", "Bearer "+secret)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestRequireServiceKeyAccess_WrongCapability_Read(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setup := newServiceKeyTestSetup(t)
	secret := setup.createServiceKey(t, false, true) // CanWrite only, not CanRead

	router := gin.New()
	router.GET("/environments/:environmentId/feature-flag-values", RequireServiceKeyAccess(setup.serviceKeys, "read"), func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/environments/"+setup.environment.ID+"/feature-flag-values", nil)
	req.Header.Set("Authorization", "Bearer "+secret)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

func TestRequireServiceKeyAccess_WrongCapability(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setup := newServiceKeyTestSetup(t)
	secret := setup.createServiceKey(t, true, false) // CanRead only

	router := gin.New()
	router.POST("/environments/:environmentId/feature-flag-values", RequireServiceKeyAccess(setup.serviceKeys, "write"), func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodPost, "/environments/"+setup.environment.ID+"/feature-flag-values", nil)
	req.Header.Set("Authorization", "Bearer "+secret)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

func TestRequireServiceKeyAccess_EnvironmentOutOfScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setup := newServiceKeyTestSetup(t)
	secret := setup.createServiceKey(t, true, false)

	router := gin.New()
	router.GET("/environments/:environmentId/feature-flag-values", RequireServiceKeyAccess(setup.serviceKeys, "read"), func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/environments/some-other-environment/feature-flag-values", nil)
	req.Header.Set("Authorization", "Bearer "+secret)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

func TestRequireServiceKeyAccess_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setup := newServiceKeyTestSetup(t)
	secret := setup.createServiceKey(t, true, false)

	var gotKey *model.ServiceKey
	var gotEnvIDs []string
	router := gin.New()
	router.GET("/environments/:environmentId/feature-flag-values", RequireServiceKeyAccess(setup.serviceKeys, "read"), func(c *gin.Context) {
		gotKey = CurrentServiceKey(c)
		gotEnvIDs = CurrentServiceKeyEnvironmentIDs(c)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/environments/"+setup.environment.ID+"/feature-flag-values", nil)
	req.Header.Set("Authorization", "Bearer "+secret)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if gotKey == nil {
		t.Fatal("CurrentServiceKey() = nil, want the authenticated key")
	}
	if len(gotEnvIDs) != 1 || gotEnvIDs[0] != setup.environment.ID {
		t.Errorf("CurrentServiceKeyEnvironmentIDs() = %v, want [%s]", gotEnvIDs, setup.environment.ID)
	}
}

func TestRequireWriteAccess_SessionAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, authService, userService, authzService := authzTestServices(t)
	serviceKeys := service.NewServiceKeyService(
		repository.NewServiceKeyRepository(db),
		repository.NewEnvironmentRepository(db),
		repository.NewServiceKeyEnvironmentRepository(db),
	)

	u, err := userService.Create(t.Context(), nil, service.CreateUserInput{
		FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com", Password: "supersecret",
	})
	if err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}
	if err := repository.NewUserProfileRepository(db).ReplaceProfiles(t.Context(), u.ID, []string{model.AdministradorProfileID}, u.ID); err != nil {
		t.Fatalf("granting Administrador returned unexpected error: %v", err)
	}
	_, token, err := authService.Login(t.Context(), "ada@example.com", "supersecret")
	if err != nil {
		t.Fatalf("Login() returned unexpected error: %v", err)
	}

	var gotUser *model.User
	router := gin.New()
	router.PATCH("/feature-flags/:id/values/:environmentId", RequireWriteAccess(authService, authzService, serviceKeys, model.ResourceFeatureFlagValue, model.ActionWrite), func(c *gin.Context) {
		gotUser = CurrentUser(c)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPatch, "/feature-flags/f1/values/env-1", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: token})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if gotUser == nil || gotUser.Email != "ada@example.com" {
		t.Fatalf("CurrentUser() = %v, want the logged-in user", gotUser)
	}
}

func TestRequireWriteAccess_SessionInvalid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, _, _, authzService := authzTestServices(t)
	authService := service.NewAuthService(repository.NewUserRepository(db), repository.NewSessionRepository(db), repository.NewLoginAttemptRepository(db))
	serviceKeys := service.NewServiceKeyService(
		repository.NewServiceKeyRepository(db),
		repository.NewEnvironmentRepository(db),
		repository.NewServiceKeyEnvironmentRepository(db),
	)

	router := gin.New()
	router.PATCH("/feature-flags/:id/values/:environmentId", RequireWriteAccess(authService, authzService, serviceKeys, model.ResourceFeatureFlagValue, model.ActionWrite), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPatch, "/feature-flags/f1/values/env-1", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "bogus-token"})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestRequireWriteAccess_SessionForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, authService, userService, authzService := authzTestServices(t)
	serviceKeys := service.NewServiceKeyService(
		repository.NewServiceKeyRepository(db),
		repository.NewEnvironmentRepository(db),
		repository.NewServiceKeyEnvironmentRepository(db),
	)
	token := loginNewUser(t, db, authService, userService, "ada@example.com") // no profile granted

	router := gin.New()
	router.PATCH("/feature-flags/:id/values/:environmentId", RequireWriteAccess(authService, authzService, serviceKeys, model.ResourceFeatureFlagValue, model.ActionWrite), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPatch, "/feature-flags/f1/values/env-1", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: token})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

func TestRequireWriteAccess_SessionAuthzError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, authService, userService, authzService := authzTestServices(t)
	serviceKeys := service.NewServiceKeyService(
		repository.NewServiceKeyRepository(db),
		repository.NewEnvironmentRepository(db),
		repository.NewServiceKeyEnvironmentRepository(db),
	)
	token := loginNewUser(t, db, authService, userService, "ada@example.com")

	router := gin.New()
	router.PATCH("/feature-flags/:id/values/:environmentId", RequireWriteAccess(authService, authzService, serviceKeys, model.ResourceFeatureFlagValue, model.ActionWrite), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPatch, "/feature-flags/f1/values/env-1", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: token})

	if _, err := db.Exec("ALTER TABLE permissions RENAME TO permissions_renamed"); err != nil {
		t.Fatalf("renaming permissions table: %v", err)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
}

func TestRequireWriteAccess_BearerFallback_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setup := newServiceKeyTestSetup(t)
	secret := setup.createServiceKey(t, false, true) // CanWrite only
	authService := service.NewAuthService(repository.NewUserRepository(setup.db), repository.NewSessionRepository(setup.db), repository.NewLoginAttemptRepository(setup.db))
	authzService := service.NewAuthzService(repository.NewAuthzRepository(setup.db))

	var gotKey *model.ServiceKey
	router := gin.New()
	router.PATCH("/feature-flags/:id/values/:environmentId", RequireWriteAccess(authService, authzService, setup.serviceKeys, model.ResourceFeatureFlagValue, model.ActionWrite), func(c *gin.Context) {
		gotKey = CurrentServiceKey(c)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPatch, "/feature-flags/f1/values/"+setup.environment.ID, nil)
	req.Header.Set("Authorization", "Bearer "+secret)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if gotKey == nil {
		t.Fatal("CurrentServiceKey() = nil, want the authenticated key")
	}
}

func TestRequireWriteAccess_BearerFallback_CanWriteFalse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setup := newServiceKeyTestSetup(t)
	secret := setup.createServiceKey(t, true, false) // CanRead only, not CanWrite
	authService := service.NewAuthService(repository.NewUserRepository(setup.db), repository.NewSessionRepository(setup.db), repository.NewLoginAttemptRepository(setup.db))
	authzService := service.NewAuthzService(repository.NewAuthzRepository(setup.db))

	router := gin.New()
	router.PATCH("/feature-flags/:id/values/:environmentId", RequireWriteAccess(authService, authzService, setup.serviceKeys, model.ResourceFeatureFlagValue, model.ActionWrite), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPatch, "/feature-flags/f1/values/"+setup.environment.ID, nil)
	req.Header.Set("Authorization", "Bearer "+secret)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

func TestRequireWriteAccess_NoCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, authService, _, authzService := authzTestServices(t)
	serviceKeys := service.NewServiceKeyService(
		repository.NewServiceKeyRepository(db),
		repository.NewEnvironmentRepository(db),
		repository.NewServiceKeyEnvironmentRepository(db),
	)

	router := gin.New()
	router.PATCH("/feature-flags/:id/values/:environmentId", RequireWriteAccess(authService, authzService, serviceKeys, model.ResourceFeatureFlagValue, model.ActionWrite), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPatch, "/feature-flags/f1/values/env-1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}
