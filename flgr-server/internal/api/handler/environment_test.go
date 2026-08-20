package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/NicolasCunha/flgr/flgr-server/internal/api/middleware"
	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/repository"
	"github.com/NicolasCunha/flgr/flgr-server/internal/service"
)

// setupEnvironmentAuthed mirrors setupProfileAuthed but for
// EnvironmentService-backed routes.
func setupEnvironmentAuthed(t *testing.T, method, path string, makeHandler func(*service.EnvironmentService) gin.HandlerFunc) (*gin.Engine, *http.Cookie, *service.EnvironmentService, *model.User) {
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

	router := gin.New()
	router.Handle(method, path, middleware.RequireAuth(authService), makeHandler(environmentService))

	return router, &http.Cookie{Name: middleware.SessionCookieName, Value: token}, environmentService, actor
}

func TestEnvironmentHandler_Create_Success(t *testing.T) {
	router, cookie, _, _ := setupEnvironmentAuthed(t, http.MethodPost, "/environments", func(es *service.EnvironmentService) gin.HandlerFunc { return NewEnvironmentHandler(es).Create })

	body, _ := json.Marshal(map[string]string{"name": "app-dev", "category_name": "Development"})
	rec := doRequest(router, http.MethodPost, "/environments", body, cookie)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var resp environmentResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if resp.Name != "app-dev" {
		t.Errorf("Name = %q, want %q", resp.Name, "app-dev")
	}
}

func TestEnvironmentHandler_Create_NewCategory(t *testing.T) {
	router, cookie, _, _ := setupEnvironmentAuthed(t, http.MethodPost, "/environments", func(es *service.EnvironmentService) gin.HandlerFunc { return NewEnvironmentHandler(es).Create })

	body, _ := json.Marshal(map[string]string{"name": "app-sandbox", "category_name": "Sandbox"})
	rec := doRequest(router, http.MethodPost, "/environments", body, cookie)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

func TestEnvironmentHandler_Create_BindError(t *testing.T) {
	router, cookie, _, _ := setupEnvironmentAuthed(t, http.MethodPost, "/environments", func(es *service.EnvironmentService) gin.HandlerFunc { return NewEnvironmentHandler(es).Create })

	rec := doRequest(router, http.MethodPost, "/environments", []byte("not json"), cookie)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestEnvironmentHandler_Create_ServiceError(t *testing.T) {
	router, cookie, es, actor := setupEnvironmentAuthed(t, http.MethodPost, "/environments", func(es *service.EnvironmentService) gin.HandlerFunc { return NewEnvironmentHandler(es).Create })
	if _, err := es.Create(context.Background(), actor, service.CreateEnvironmentInput{Name: "dup", CategoryName: "Development"}); err != nil {
		t.Fatalf("seeding environment returned unexpected error: %v", err)
	}

	body, _ := json.Marshal(map[string]string{"name": "dup", "category_name": "Development"})
	rec := doRequest(router, http.MethodPost, "/environments", body, cookie)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestEnvironmentHandler_Get(t *testing.T) {
	router, cookie, es, actor := setupEnvironmentAuthed(t, http.MethodGet, "/environments/:id", func(es *service.EnvironmentService) gin.HandlerFunc { return NewEnvironmentHandler(es).Get })
	e, err := es.Create(context.Background(), actor, service.CreateEnvironmentInput{Name: "app-dev", CategoryName: "Development"})
	if err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	rec := doRequest(router, http.MethodGet, "/environments/"+e.ID, nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestEnvironmentHandler_Get_NotFound(t *testing.T) {
	router, cookie, _, _ := setupEnvironmentAuthed(t, http.MethodGet, "/environments/:id", func(es *service.EnvironmentService) gin.HandlerFunc { return NewEnvironmentHandler(es).Get })

	rec := doRequest(router, http.MethodGet, "/environments/does-not-exist", nil, cookie)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestEnvironmentHandler_List(t *testing.T) {
	router, cookie, es, actor := setupEnvironmentAuthed(t, http.MethodGet, "/environments", func(es *service.EnvironmentService) gin.HandlerFunc { return NewEnvironmentHandler(es).List })
	if _, err := es.Create(context.Background(), actor, service.CreateEnvironmentInput{Name: "app-dev", CategoryName: "Development"}); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	rec := doRequest(router, http.MethodGet, "/environments?page=1&page_size=10", nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Data       []environmentResponse `json:"data"`
		Pagination paginationResponse    `json:"pagination"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Errorf("len(Data) = %d, want 1", len(resp.Data))
	}
}

func TestEnvironmentHandler_List_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newTestDB(t)
	es := service.NewEnvironmentService(
		repository.NewEnvironmentRepository(db),
		repository.NewEnvironmentCategoryRepository(db),
	)
	_ = db.Close()

	h := NewEnvironmentHandler(es)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/environments", nil)

	h.List(c)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
}

func TestEnvironmentHandler_Update(t *testing.T) {
	router, cookie, es, actor := setupEnvironmentAuthed(t, http.MethodPatch, "/environments/:id", func(es *service.EnvironmentService) gin.HandlerFunc { return NewEnvironmentHandler(es).Update })
	e, err := es.Create(context.Background(), actor, service.CreateEnvironmentInput{Name: "app-dev", CategoryName: "Development"})
	if err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	body, _ := json.Marshal(map[string]string{"name": "app-dev-renamed"})
	rec := doRequest(router, http.MethodPatch, "/environments/"+e.ID, body, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp environmentResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if resp.Name != "app-dev-renamed" {
		t.Errorf("Name = %q, want %q", resp.Name, "app-dev-renamed")
	}
}

func TestEnvironmentHandler_Update_BindError(t *testing.T) {
	router, cookie, _, _ := setupEnvironmentAuthed(t, http.MethodPatch, "/environments/:id", func(es *service.EnvironmentService) gin.HandlerFunc { return NewEnvironmentHandler(es).Update })

	rec := doRequest(router, http.MethodPatch, "/environments/irrelevant", []byte("not json"), cookie)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestEnvironmentHandler_Update_ServiceError(t *testing.T) {
	router, cookie, _, _ := setupEnvironmentAuthed(t, http.MethodPatch, "/environments/:id", func(es *service.EnvironmentService) gin.HandlerFunc { return NewEnvironmentHandler(es).Update })

	body, _ := json.Marshal(map[string]string{"name": "x"})
	rec := doRequest(router, http.MethodPatch, "/environments/does-not-exist", body, cookie)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestEnvironmentHandler_Delete(t *testing.T) {
	router, cookie, es, actor := setupEnvironmentAuthed(t, http.MethodDelete, "/environments/:id", func(es *service.EnvironmentService) gin.HandlerFunc { return NewEnvironmentHandler(es).Delete })
	e, err := es.Create(context.Background(), actor, service.CreateEnvironmentInput{Name: "app-dev", CategoryName: "Development"})
	if err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	rec := doRequest(router, http.MethodDelete, "/environments/"+e.ID, nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestEnvironmentHandler_Delete_ServiceError(t *testing.T) {
	router, cookie, _, _ := setupEnvironmentAuthed(t, http.MethodDelete, "/environments/:id", func(es *service.EnvironmentService) gin.HandlerFunc { return NewEnvironmentHandler(es).Delete })

	rec := doRequest(router, http.MethodDelete, "/environments/does-not-exist", nil, cookie)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
