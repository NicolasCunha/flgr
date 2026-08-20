package middleware

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/repository"
	"github.com/NicolasCunha/flgr/flgr-server/internal/service"
)

// authzTestServices builds every service RequirePermission needs, all
// backed by the same database, plus the database itself for tests that
// need to grant a profile or otherwise mutate state directly.
func authzTestServices(t *testing.T) (*sql.DB, *service.AuthService, *service.UserService, *service.AuthzService) {
	t.Helper()
	db := newTestDB(t)
	userRepo := repository.NewUserRepository(db)
	authzService := service.NewAuthzService(repository.NewAuthzRepository(db))
	return db,
		service.NewAuthService(userRepo, repository.NewSessionRepository(db), repository.NewLoginAttemptRepository(db)),
		service.NewUserService(userRepo, authzService),
		authzService
}

func loginNewUser(t *testing.T, db *sql.DB, authService *service.AuthService, userService *service.UserService, email string) string {
	t.Helper()
	u, err := userService.Create(t.Context(), nil, service.CreateUserInput{
		FirstName: "Ada", LastName: "Lovelace", Email: email, Password: "supersecret",
	})
	if err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}
	_, token, err := authService.Login(t.Context(), email, "supersecret")
	if err != nil {
		t.Fatalf("Login() returned unexpected error: %v", err)
	}
	_ = db
	_ = u
	return token
}

func TestRequirePermission_Allowed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, authService, userService, authzService := authzTestServices(t)

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

	router := gin.New()
	router.GET("/protected", RequireAuth(authService), RequirePermission(authzService, model.ResourceUser, model.ActionCreate), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: token})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestRequirePermission_Forbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, authService, userService, authzService := authzTestServices(t)
	token := loginNewUser(t, db, authService, userService, "ada@example.com")

	router := gin.New()
	router.GET("/protected", RequireAuth(authService), RequirePermission(authzService, model.ResourceUser, model.ActionCreate), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: token})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

func TestRequirePermission_Unauthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_, _, _, authzService := authzTestServices(t)

	router := gin.New()
	router.GET("/protected", RequirePermission(authzService, model.ResourceUser, model.ActionCreate), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestRequirePermission_AuthzError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, authService, userService, authzService := authzTestServices(t)
	token := loginNewUser(t, db, authService, userService, "ada@example.com")

	router := gin.New()
	router.GET("/protected", RequireAuth(authService), RequirePermission(authzService, model.ResourceUser, model.ActionCreate), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: token})

	// Rename the permissions table out from under the authz query so it
	// fails, while sessions/users stay intact for RequireAuth to succeed
	// first (a DROP would violate the FK from profile_permissions).
	if _, err := db.Exec("ALTER TABLE permissions RENAME TO permissions_renamed"); err != nil {
		t.Fatalf("renaming permissions table: %v", err)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
}
