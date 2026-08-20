package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/NicolasCunha/flgr/flgr-server/internal/api/middleware"
	"github.com/NicolasCunha/flgr/flgr-server/internal/repository"
	"github.com/NicolasCunha/flgr/flgr-server/internal/service"
)

func doRequest(router *gin.Engine, method, path string, body []byte, cookie *http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// setupAuthed is the common case: one shared DB, an actor created,
// granted the Administrador profile (so tests can exercise acting on
// another user without separately testing the authorization boundary —
// that's covered at the service layer, see user_service_test.go and
// profile_service_test.go), logged in, and a single route registered
// behind RequireAuth.
func setupAuthed(t *testing.T, method, path string, makeHandler func(*service.UserService) gin.HandlerFunc) (*gin.Engine, *http.Cookie, *service.UserService) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	userService, authService, db := newTestServicesWithDB(t)
	actor, err := userService.Create(t.Context(), nil, service.CreateUserInput{
		FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com", Password: "supersecret",
	})
	if err != nil {
		t.Fatalf("Create(actor) returned unexpected error: %v", err)
	}
	grantAdministrador(t, db, actor.ID)

	_, token, err := authService.Login(t.Context(), "ada@example.com", "supersecret")
	if err != nil {
		t.Fatalf("Login() returned unexpected error: %v", err)
	}

	router := gin.New()
	router.Handle(method, path, middleware.RequireAuth(authService), makeHandler(userService))

	return router, &http.Cookie{Name: middleware.SessionCookieName, Value: token}, userService
}

func TestUserHandler_Create_Success(t *testing.T) {
	router, cookie, _ := setupAuthed(t, http.MethodPost, "/users", func(us *service.UserService) gin.HandlerFunc { return NewUserHandler(us).Create })

	body, _ := json.Marshal(map[string]string{
		"first_name": "Grace", "last_name": "Hopper", "email": "grace@example.com", "password": "supersecret",
	})
	rec := doRequest(router, http.MethodPost, "/users", body, cookie)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var resp userResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if resp.Email != "grace@example.com" {
		t.Errorf("Email = %q, want %q", resp.Email, "grace@example.com")
	}
}

func TestUserHandler_Create_BindError(t *testing.T) {
	router, cookie, _ := setupAuthed(t, http.MethodPost, "/users", func(us *service.UserService) gin.HandlerFunc { return NewUserHandler(us).Create })

	rec := doRequest(router, http.MethodPost, "/users", []byte("not json"), cookie)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestUserHandler_Create_ServiceError(t *testing.T) {
	router, cookie, _ := setupAuthed(t, http.MethodPost, "/users", func(us *service.UserService) gin.HandlerFunc { return NewUserHandler(us).Create })

	body, _ := json.Marshal(map[string]string{
		"first_name": "Ada", "last_name": "Lovelace", "email": "ada@example.com", "password": "supersecret",
	})
	rec := doRequest(router, http.MethodPost, "/users", body, cookie)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestUserHandler_Get(t *testing.T) {
	router, cookie, us := setupAuthed(t, http.MethodGet, "/users/:id", func(us *service.UserService) gin.HandlerFunc { return NewUserHandler(us).Get })

	target, err := us.Create(t.Context(), nil, service.CreateUserInput{
		FirstName: "Bob", LastName: "Smith", Email: "bob@example.com", Password: "supersecret",
	})
	if err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	rec := doRequest(router, http.MethodGet, "/users/"+target.ID, nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestUserHandler_Get_NotFound(t *testing.T) {
	router, cookie, _ := setupAuthed(t, http.MethodGet, "/users/:id", func(us *service.UserService) gin.HandlerFunc { return NewUserHandler(us).Get })

	rec := doRequest(router, http.MethodGet, "/users/does-not-exist", nil, cookie)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestUserHandler_List(t *testing.T) {
	router, cookie, _ := setupAuthed(t, http.MethodGet, "/users", func(us *service.UserService) gin.HandlerFunc { return NewUserHandler(us).List })

	rec := doRequest(router, http.MethodGet, "/users?page=1&page_size=10", nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Data       []userResponse     `json:"data"`
		Pagination paginationResponse `json:"pagination"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if resp.Pagination.Page != 1 || resp.Pagination.PageSize != 10 {
		t.Errorf("Pagination = %+v, want page=1 page_size=10", resp.Pagination)
	}
	if len(resp.Data) != 1 {
		t.Errorf("len(Data) = %d, want 1 (the seeded actor)", len(resp.Data))
	}
}

func TestUserHandler_List_DefaultsWhenQueryMissing(t *testing.T) {
	router, cookie, _ := setupAuthed(t, http.MethodGet, "/users", func(us *service.UserService) gin.HandlerFunc { return NewUserHandler(us).List })

	rec := doRequest(router, http.MethodGet, "/users", nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestUserHandler_List_InvalidQueryFallsBackToDefault(t *testing.T) {
	router, cookie, _ := setupAuthed(t, http.MethodGet, "/users", func(us *service.UserService) gin.HandlerFunc { return NewUserHandler(us).List })

	rec := doRequest(router, http.MethodGet, "/users?page=not-a-number", nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestUserHandler_List_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newTestDB(t)
	userRepo := repository.NewUserRepository(db)
	userService := service.NewUserService(userRepo, service.NewAuthzService(repository.NewAuthzRepository(db)))
	_ = db.Close()

	h := NewUserHandler(userService)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/users", nil)

	h.List(c)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
}

func TestUserHandler_Update(t *testing.T) {
	router, cookie, us := setupAuthed(t, http.MethodPatch, "/users/:id", func(us *service.UserService) gin.HandlerFunc { return NewUserHandler(us).Update })

	target, err := us.Create(t.Context(), nil, service.CreateUserInput{
		FirstName: "Bob", LastName: "Smith", Email: "bob@example.com", Password: "supersecret",
	})
	if err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	body, _ := json.Marshal(map[string]string{"first_name": "Robert"})
	rec := doRequest(router, http.MethodPatch, "/users/"+target.ID, body, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp userResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if resp.FirstName != "Robert" {
		t.Errorf("FirstName = %q, want %q", resp.FirstName, "Robert")
	}
}

func TestUserHandler_Update_BindError(t *testing.T) {
	router, cookie, _ := setupAuthed(t, http.MethodPatch, "/users/:id", func(us *service.UserService) gin.HandlerFunc { return NewUserHandler(us).Update })

	rec := doRequest(router, http.MethodPatch, "/users/irrelevant", []byte("not json"), cookie)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestUserHandler_Update_ServiceError(t *testing.T) {
	router, cookie, _ := setupAuthed(t, http.MethodPatch, "/users/:id", func(us *service.UserService) gin.HandlerFunc { return NewUserHandler(us).Update })

	body, _ := json.Marshal(map[string]string{"first_name": "Robert"})
	rec := doRequest(router, http.MethodPatch, "/users/does-not-exist", body, cookie)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestUserHandler_Deactivate(t *testing.T) {
	router, cookie, us := setupAuthed(t, http.MethodDelete, "/users/:id", func(us *service.UserService) gin.HandlerFunc { return NewUserHandler(us).Deactivate })

	target, err := us.Create(t.Context(), nil, service.CreateUserInput{
		FirstName: "Bob", LastName: "Smith", Email: "bob@example.com", Password: "supersecret",
	})
	if err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	rec := doRequest(router, http.MethodDelete, "/users/"+target.ID, nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp userResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if resp.Status != "Inactive" {
		t.Errorf("Status = %q, want %q", resp.Status, "Inactive")
	}
}

func TestUserHandler_Deactivate_ServiceError(t *testing.T) {
	router, cookie, _ := setupAuthed(t, http.MethodDelete, "/users/:id", func(us *service.UserService) gin.HandlerFunc { return NewUserHandler(us).Deactivate })

	rec := doRequest(router, http.MethodDelete, "/users/does-not-exist", nil, cookie)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
