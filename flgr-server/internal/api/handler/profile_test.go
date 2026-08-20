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

// setupProfileAuthed mirrors setupAuthed but for ProfileService-backed
// routes: one shared DB, an authenticated actor (also returned, for
// seeding fixtures directly through the service), and a single route.
func setupProfileAuthed(t *testing.T, method, path string, makeHandler func(*service.ProfileService) gin.HandlerFunc) (*gin.Engine, *http.Cookie, *service.ProfileService, *model.User) {
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

	profileService := service.NewProfileService(
		repository.NewProfileRepository(db),
		repository.NewProfilePermissionRepository(db),
		repository.NewPermissionRepository(db),
	)

	router := gin.New()
	router.Handle(method, path, middleware.RequireAuth(authService), makeHandler(profileService))

	return router, &http.Cookie{Name: middleware.SessionCookieName, Value: token}, profileService, actor
}

func TestProfileHandler_Create_Success(t *testing.T) {
	router, cookie, _, _ := setupProfileAuthed(t, http.MethodPost, "/profiles", func(ps *service.ProfileService) gin.HandlerFunc { return NewProfileHandler(ps).Create })

	body, _ := json.Marshal(map[string]any{"name": "app-read", "permission_ids": []string{"user:view"}})
	rec := doRequest(router, http.MethodPost, "/profiles", body, cookie)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var resp profileResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if resp.Name != "app-read" || resp.IsSystem {
		t.Errorf("resp = %+v, want Name=app-read IsSystem=false", resp)
	}
}

func TestProfileHandler_Create_BindError(t *testing.T) {
	router, cookie, _, _ := setupProfileAuthed(t, http.MethodPost, "/profiles", func(ps *service.ProfileService) gin.HandlerFunc { return NewProfileHandler(ps).Create })

	rec := doRequest(router, http.MethodPost, "/profiles", []byte("not json"), cookie)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestProfileHandler_Create_ServiceError(t *testing.T) {
	router, cookie, ps, actor := setupProfileAuthed(t, http.MethodPost, "/profiles", func(ps *service.ProfileService) gin.HandlerFunc { return NewProfileHandler(ps).Create })
	if _, err := ps.Create(context.Background(), actor, service.CreateProfileInput{Name: "dup"}); err != nil {
		t.Fatalf("seeding profile returned unexpected error: %v", err)
	}

	body, _ := json.Marshal(map[string]any{"name": "dup"})
	rec := doRequest(router, http.MethodPost, "/profiles", body, cookie)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestProfileHandler_Get(t *testing.T) {
	router, cookie, ps, actor := setupProfileAuthed(t, http.MethodGet, "/profiles/:id", func(ps *service.ProfileService) gin.HandlerFunc { return NewProfileHandler(ps).Get })
	p, err := ps.Create(context.Background(), actor, service.CreateProfileInput{Name: "app-read"})
	if err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	rec := doRequest(router, http.MethodGet, "/profiles/"+p.ID, nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestProfileHandler_Get_NotFound(t *testing.T) {
	router, cookie, _, _ := setupProfileAuthed(t, http.MethodGet, "/profiles/:id", func(ps *service.ProfileService) gin.HandlerFunc { return NewProfileHandler(ps).Get })

	rec := doRequest(router, http.MethodGet, "/profiles/does-not-exist", nil, cookie)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestProfileHandler_List(t *testing.T) {
	router, cookie, ps, actor := setupProfileAuthed(t, http.MethodGet, "/profiles", func(ps *service.ProfileService) gin.HandlerFunc { return NewProfileHandler(ps).List })
	if _, err := ps.Create(context.Background(), actor, service.CreateProfileInput{Name: "app-read"}); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	rec := doRequest(router, http.MethodGet, "/profiles?page=1&page_size=10", nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Data       []profileResponse  `json:"data"`
		Pagination paginationResponse `json:"pagination"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	// The seeded Administrador profile + the one just created.
	if len(resp.Data) != 2 {
		t.Errorf("len(Data) = %d, want 2", len(resp.Data))
	}
}

func TestProfileHandler_List_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newTestDB(t)
	ps := service.NewProfileService(
		repository.NewProfileRepository(db),
		repository.NewProfilePermissionRepository(db),
		repository.NewPermissionRepository(db),
	)
	_ = db.Close()

	h := NewProfileHandler(ps)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/profiles", nil)

	h.List(c)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
}

func TestProfileHandler_Delete_LastFullCatalogProfile(t *testing.T) {
	router, cookie, _, _ := setupProfileAuthed(t, http.MethodDelete, "/profiles/:id", func(ps *service.ProfileService) gin.HandlerFunc { return NewProfileHandler(ps).Delete })

	rec := doRequest(router, http.MethodDelete, "/profiles/"+model.AdministradorProfileID, nil, cookie)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestProfileHandler_PermissionIDs(t *testing.T) {
	router, cookie, ps, actor := setupProfileAuthed(t, http.MethodGet, "/profiles/:id/permissions", func(ps *service.ProfileService) gin.HandlerFunc { return NewProfileHandler(ps).PermissionIDs })
	p, err := ps.Create(context.Background(), actor, service.CreateProfileInput{Name: "app-read", PermissionIDs: []string{"user:view"}})
	if err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	rec := doRequest(router, http.MethodGet, "/profiles/"+p.ID+"/permissions", nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp struct {
		PermissionIDs []string `json:"permission_ids"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if len(resp.PermissionIDs) != 1 || resp.PermissionIDs[0] != "user:view" {
		t.Errorf("PermissionIDs = %v, want [user:view]", resp.PermissionIDs)
	}
}

func TestProfileHandler_PermissionIDs_NotFound(t *testing.T) {
	router, cookie, _, _ := setupProfileAuthed(t, http.MethodGet, "/profiles/:id/permissions", func(ps *service.ProfileService) gin.HandlerFunc { return NewProfileHandler(ps).PermissionIDs })

	rec := doRequest(router, http.MethodGet, "/profiles/does-not-exist/permissions", nil, cookie)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestProfileHandler_Update(t *testing.T) {
	router, cookie, ps, actor := setupProfileAuthed(t, http.MethodPatch, "/profiles/:id", func(ps *service.ProfileService) gin.HandlerFunc { return NewProfileHandler(ps).Update })
	p, err := ps.Create(context.Background(), actor, service.CreateProfileInput{Name: "app-read"})
	if err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	body, _ := json.Marshal(map[string]string{"name": "app-read-renamed"})
	rec := doRequest(router, http.MethodPatch, "/profiles/"+p.ID, body, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp profileResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if resp.Name != "app-read-renamed" {
		t.Errorf("Name = %q, want %q", resp.Name, "app-read-renamed")
	}
}

func TestProfileHandler_Update_BindError(t *testing.T) {
	router, cookie, _, _ := setupProfileAuthed(t, http.MethodPatch, "/profiles/:id", func(ps *service.ProfileService) gin.HandlerFunc { return NewProfileHandler(ps).Update })

	rec := doRequest(router, http.MethodPatch, "/profiles/irrelevant", []byte("not json"), cookie)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestProfileHandler_Update_ServiceError(t *testing.T) {
	router, cookie, _, _ := setupProfileAuthed(t, http.MethodPatch, "/profiles/:id", func(ps *service.ProfileService) gin.HandlerFunc { return NewProfileHandler(ps).Update })

	body, _ := json.Marshal(map[string]string{"name": "x"})
	rec := doRequest(router, http.MethodPatch, "/profiles/does-not-exist", body, cookie)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestProfileHandler_Delete(t *testing.T) {
	router, cookie, ps, actor := setupProfileAuthed(t, http.MethodDelete, "/profiles/:id", func(ps *service.ProfileService) gin.HandlerFunc { return NewProfileHandler(ps).Delete })
	p, err := ps.Create(context.Background(), actor, service.CreateProfileInput{Name: "app-read"})
	if err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	rec := doRequest(router, http.MethodDelete, "/profiles/"+p.ID, nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestProfileHandler_Delete_ServiceError(t *testing.T) {
	router, cookie, _, _ := setupProfileAuthed(t, http.MethodDelete, "/profiles/:id", func(ps *service.ProfileService) gin.HandlerFunc { return NewProfileHandler(ps).Delete })

	rec := doRequest(router, http.MethodDelete, "/profiles/does-not-exist", nil, cookie)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
