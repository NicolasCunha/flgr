package handler

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/NicolasCunha/flgr/flgr-server/internal/api/middleware"
	"github.com/NicolasCunha/flgr/flgr-server/internal/repository"
	"github.com/NicolasCunha/flgr/flgr-server/internal/service"
)

func TestAuthHandler_Login_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userService, authService := newTestServices(t)
	if _, err := userService.Create(t.Context(), nil, service.CreateUserInput{
		FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com", Password: "supersecret",
	}); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	h := NewAuthHandler(authService, false)
	router := gin.New()
	router.POST("/login", h.Login)

	body, _ := json.Marshal(map[string]string{"email": "ada@example.com", "password": "supersecret"})
	rec := doRequest(router, http.MethodPost, "/login", body, nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if len(rec.Result().Cookies()) != 1 {
		t.Fatalf("len(cookies) = %d, want 1", len(rec.Result().Cookies()))
	}
}

func TestAuthHandler_Login_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_, authService := newTestServices(t)
	h := NewAuthHandler(authService, false)
	router := gin.New()
	router.POST("/login", h.Login)

	rec := doRequest(router, http.MethodPost, "/login", []byte("not json"), nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_, authService := newTestServices(t)
	h := NewAuthHandler(authService, false)
	router := gin.New()
	router.POST("/login", h.Login)

	body, _ := json.Marshal(map[string]string{"email": "nobody@example.com", "password": "wrong"})
	rec := doRequest(router, http.MethodPost, "/login", body, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAuthHandler_Logout_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userService, authService := newTestServices(t)
	if _, err := userService.Create(t.Context(), nil, service.CreateUserInput{
		FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com", Password: "supersecret",
	}); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}
	_, token, err := authService.Login(t.Context(), "ada@example.com", "supersecret")
	if err != nil {
		t.Fatalf("Login() returned unexpected error: %v", err)
	}

	h := NewAuthHandler(authService, false)
	router := gin.New()
	router.POST("/logout", h.Logout)

	rec := doRequest(router, http.MethodPost, "/logout", nil, &http.Cookie{Name: middleware.SessionCookieName, Value: token})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestAuthHandler_Logout_NoCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_, authService := newTestServices(t)
	h := NewAuthHandler(authService, false)
	router := gin.New()
	router.POST("/logout", h.Logout)

	rec := doRequest(router, http.MethodPost, "/logout", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestAuthHandler_Logout_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newTestDB(t)
	userRepo := repository.NewUserRepository(db)
	authService := service.NewAuthService(userRepo, repository.NewSessionRepository(db), repository.NewLoginAttemptRepository(db))
	userService := service.NewUserService(userRepo, service.NewAuthzService(repository.NewAuthzRepository(db)))

	if _, err := userService.Create(t.Context(), nil, service.CreateUserInput{
		FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com", Password: "supersecret",
	}); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}
	_, token, err := authService.Login(t.Context(), "ada@example.com", "supersecret")
	if err != nil {
		t.Fatalf("Login() returned unexpected error: %v", err)
	}
	_ = db.Close()

	h := NewAuthHandler(authService, false)
	router := gin.New()
	router.POST("/logout", h.Logout)

	rec := doRequest(router, http.MethodPost, "/logout", nil, &http.Cookie{Name: middleware.SessionCookieName, Value: token})
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
}

func TestAuthHandler_Me(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userService, authService := newTestServices(t)
	if _, err := userService.Create(t.Context(), nil, service.CreateUserInput{
		FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com", Password: "supersecret",
	}); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}
	_, token, err := authService.Login(t.Context(), "ada@example.com", "supersecret")
	if err != nil {
		t.Fatalf("Login() returned unexpected error: %v", err)
	}

	h := NewAuthHandler(authService, false)
	router := gin.New()
	router.GET("/me", middleware.RequireAuth(authService), h.Me)

	rec := doRequest(router, http.MethodGet, "/me", nil, &http.Cookie{Name: middleware.SessionCookieName, Value: token})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp userResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if resp.Email != "ada@example.com" {
		t.Errorf("Email = %q, want %q", resp.Email, "ada@example.com")
	}
}
