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

// setupFeatureFlagAuthed mirrors setupEnvironmentAuthed but for
// FeatureFlagService-backed routes.
func setupFeatureFlagAuthed(t *testing.T, method, path string, makeHandler func(*service.FeatureFlagService) gin.HandlerFunc) (router *gin.Engine, cookie *http.Cookie, flags *service.FeatureFlagService, actor *model.User) {
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

	router = gin.New()
	router.Handle(method, path, middleware.RequireAuth(authService), makeHandler(flagService))

	return router, &http.Cookie{Name: middleware.SessionCookieName, Value: token}, flagService, actor
}

func TestFeatureFlagHandler_Create_Success(t *testing.T) {
	router, cookie, _, _ := setupFeatureFlagAuthed(t, http.MethodPost, "/feature-flags", func(fs *service.FeatureFlagService) gin.HandlerFunc { return NewFeatureFlagHandler(fs).Create })

	body, _ := json.Marshal(map[string]string{"key": "new-checkout-flow", "name": "New Checkout Flow", "type": model.FeatureFlagTypeBoolean})
	rec := doRequest(router, http.MethodPost, "/feature-flags", body, cookie)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var resp featureFlagResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if resp.Key != "new-checkout-flow" || resp.Type != model.FeatureFlagTypeBoolean {
		t.Errorf("Key/Type = %q/%q, want new-checkout-flow/Boolean", resp.Key, resp.Type)
	}
}

func TestFeatureFlagHandler_Create_BindError(t *testing.T) {
	router, cookie, _, _ := setupFeatureFlagAuthed(t, http.MethodPost, "/feature-flags", func(fs *service.FeatureFlagService) gin.HandlerFunc { return NewFeatureFlagHandler(fs).Create })

	rec := doRequest(router, http.MethodPost, "/feature-flags", []byte("not json"), cookie)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestFeatureFlagHandler_Create_ServiceError(t *testing.T) {
	router, cookie, fs, actor := setupFeatureFlagAuthed(t, http.MethodPost, "/feature-flags", func(fs *service.FeatureFlagService) gin.HandlerFunc { return NewFeatureFlagHandler(fs).Create })
	if _, err := fs.Create(context.Background(), actor, service.CreateFeatureFlagInput{Key: "dup", Name: "x", Type: model.FeatureFlagTypeBoolean}); err != nil {
		t.Fatalf("seeding flag returned unexpected error: %v", err)
	}

	body, _ := json.Marshal(map[string]string{"key": "dup", "name": "y", "type": model.FeatureFlagTypeBoolean})
	rec := doRequest(router, http.MethodPost, "/feature-flags", body, cookie)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestFeatureFlagHandler_Get(t *testing.T) {
	router, cookie, fs, actor := setupFeatureFlagAuthed(t, http.MethodGet, "/feature-flags/:id", func(fs *service.FeatureFlagService) gin.HandlerFunc { return NewFeatureFlagHandler(fs).Get })
	f, err := fs.Create(context.Background(), actor, service.CreateFeatureFlagInput{Key: "x", Name: "x", Type: model.FeatureFlagTypeBoolean})
	if err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	rec := doRequest(router, http.MethodGet, "/feature-flags/"+f.ID, nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestFeatureFlagHandler_Get_NotFound(t *testing.T) {
	router, cookie, _, _ := setupFeatureFlagAuthed(t, http.MethodGet, "/feature-flags/:id", func(fs *service.FeatureFlagService) gin.HandlerFunc { return NewFeatureFlagHandler(fs).Get })

	rec := doRequest(router, http.MethodGet, "/feature-flags/does-not-exist", nil, cookie)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestFeatureFlagHandler_List(t *testing.T) {
	router, cookie, fs, actor := setupFeatureFlagAuthed(t, http.MethodGet, "/feature-flags", func(fs *service.FeatureFlagService) gin.HandlerFunc { return NewFeatureFlagHandler(fs).List })
	if _, err := fs.Create(context.Background(), actor, service.CreateFeatureFlagInput{Key: "x", Name: "x", Type: model.FeatureFlagTypeBoolean}); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	rec := doRequest(router, http.MethodGet, "/feature-flags?page=1&page_size=10", nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Data       []featureFlagResponse `json:"data"`
		Pagination paginationResponse    `json:"pagination"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Errorf("len(Data) = %d, want 1", len(resp.Data))
	}
}

func TestFeatureFlagHandler_List_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newTestDB(t)
	fs := service.NewFeatureFlagService(repository.NewFeatureFlagRepository(db))
	_ = db.Close()

	h := NewFeatureFlagHandler(fs)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/feature-flags", nil)

	h.List(c)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
}

func TestFeatureFlagHandler_Update(t *testing.T) {
	router, cookie, fs, actor := setupFeatureFlagAuthed(t, http.MethodPatch, "/feature-flags/:id", func(fs *service.FeatureFlagService) gin.HandlerFunc { return NewFeatureFlagHandler(fs).Update })
	f, err := fs.Create(context.Background(), actor, service.CreateFeatureFlagInput{Key: "x", Name: "old", Type: model.FeatureFlagTypeBoolean})
	if err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	body, _ := json.Marshal(map[string]string{"name": "new-name"})
	rec := doRequest(router, http.MethodPatch, "/feature-flags/"+f.ID, body, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp featureFlagResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if resp.Name != "new-name" {
		t.Errorf("Name = %q, want %q", resp.Name, "new-name")
	}
	if resp.Key != "x" {
		t.Errorf("Key = %q, want unchanged %q", resp.Key, "x")
	}
}

func TestFeatureFlagHandler_Update_BindError(t *testing.T) {
	router, cookie, _, _ := setupFeatureFlagAuthed(t, http.MethodPatch, "/feature-flags/:id", func(fs *service.FeatureFlagService) gin.HandlerFunc { return NewFeatureFlagHandler(fs).Update })

	rec := doRequest(router, http.MethodPatch, "/feature-flags/irrelevant", []byte("not json"), cookie)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestFeatureFlagHandler_Update_ServiceError(t *testing.T) {
	router, cookie, _, _ := setupFeatureFlagAuthed(t, http.MethodPatch, "/feature-flags/:id", func(fs *service.FeatureFlagService) gin.HandlerFunc { return NewFeatureFlagHandler(fs).Update })

	body, _ := json.Marshal(map[string]string{"name": "x"})
	rec := doRequest(router, http.MethodPatch, "/feature-flags/does-not-exist", body, cookie)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestFeatureFlagHandler_Delete(t *testing.T) {
	router, cookie, fs, actor := setupFeatureFlagAuthed(t, http.MethodDelete, "/feature-flags/:id", func(fs *service.FeatureFlagService) gin.HandlerFunc { return NewFeatureFlagHandler(fs).Delete })
	f, err := fs.Create(context.Background(), actor, service.CreateFeatureFlagInput{Key: "x", Name: "x", Type: model.FeatureFlagTypeBoolean})
	if err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	rec := doRequest(router, http.MethodDelete, "/feature-flags/"+f.ID, nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestFeatureFlagHandler_Delete_ServiceError(t *testing.T) {
	router, cookie, _, _ := setupFeatureFlagAuthed(t, http.MethodDelete, "/feature-flags/:id", func(fs *service.FeatureFlagService) gin.HandlerFunc { return NewFeatureFlagHandler(fs).Delete })

	rec := doRequest(router, http.MethodDelete, "/feature-flags/does-not-exist", nil, cookie)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
