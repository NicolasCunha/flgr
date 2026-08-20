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

// setupServiceKeyAuthed mirrors setupEnvironmentAuthed but for
// ServiceKeyService-backed routes. It also returns a real environment id
// (seeded via EnvironmentService), since 0004 requires every service key
// to be linked to at least one real environment.
func setupServiceKeyAuthed(t *testing.T, method, path string, makeHandler func(*service.ServiceKeyService) gin.HandlerFunc) (router *gin.Engine, cookie *http.Cookie, serviceKeys *service.ServiceKeyService, actor *model.User, environmentID string) {
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

	serviceKeyService := service.NewServiceKeyService(
		repository.NewServiceKeyRepository(db),
		repository.NewEnvironmentRepository(db),
		repository.NewServiceKeyEnvironmentRepository(db),
	)

	router = gin.New()
	router.Handle(method, path, middleware.RequireAuth(authService), makeHandler(serviceKeyService))

	return router, &http.Cookie{Name: middleware.SessionCookieName, Value: token}, serviceKeyService, actor, env.ID
}

func TestServiceKeyHandler_Create_Success(t *testing.T) {
	router, cookie, _, _, envID := setupServiceKeyAuthed(t, http.MethodPost, "/service-keys", func(ks *service.ServiceKeyService) gin.HandlerFunc { return NewServiceKeyHandler(ks).Create })

	body, _ := json.Marshal(map[string]any{"name": "checkout-service prod key", "can_read": true, "environment_ids": []string{envID}})
	rec := doRequest(router, http.MethodPost, "/service-keys", body, cookie)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var resp createServiceKeyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if resp.Name != "checkout-service prod key" {
		t.Errorf("Name = %q, want %q", resp.Name, "checkout-service prod key")
	}
	if resp.Secret == "" {
		t.Error("Secret is empty, want the generated plaintext secret")
	}
	if len(resp.EnvironmentIDs) != 1 || resp.EnvironmentIDs[0] != envID {
		t.Errorf("EnvironmentIDs = %v, want [%s]", resp.EnvironmentIDs, envID)
	}
	if !resp.CanRead || resp.CanWrite {
		t.Errorf("CanRead/CanWrite = %v/%v, want true/false", resp.CanRead, resp.CanWrite)
	}
}

func TestServiceKeyHandler_Create_DefaultsCanReadWrite(t *testing.T) {
	router, cookie, _, _, envID := setupServiceKeyAuthed(t, http.MethodPost, "/service-keys", func(ks *service.ServiceKeyService) gin.HandlerFunc { return NewServiceKeyHandler(ks).Create })

	body, _ := json.Marshal(map[string]any{"name": "default-flags key", "environment_ids": []string{envID}})
	rec := doRequest(router, http.MethodPost, "/service-keys", body, cookie)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var resp createServiceKeyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if !resp.CanRead || resp.CanWrite {
		t.Errorf("CanRead/CanWrite = %v/%v, want true/false (the service_keys table defaults)", resp.CanRead, resp.CanWrite)
	}
}

func TestServiceKeyHandler_Create_ExplicitCanWrite(t *testing.T) {
	router, cookie, _, _, envID := setupServiceKeyAuthed(t, http.MethodPost, "/service-keys", func(ks *service.ServiceKeyService) gin.HandlerFunc { return NewServiceKeyHandler(ks).Create })

	body, _ := json.Marshal(map[string]any{"name": "write-flags key", "can_read": false, "can_write": true, "environment_ids": []string{envID}})
	rec := doRequest(router, http.MethodPost, "/service-keys", body, cookie)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var resp createServiceKeyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if resp.CanRead || !resp.CanWrite {
		t.Errorf("CanRead/CanWrite = %v/%v, want false/true", resp.CanRead, resp.CanWrite)
	}
}

func TestServiceKeyHandler_Create_BindError(t *testing.T) {
	router, cookie, _, _, _ := setupServiceKeyAuthed(t, http.MethodPost, "/service-keys", func(ks *service.ServiceKeyService) gin.HandlerFunc { return NewServiceKeyHandler(ks).Create })

	rec := doRequest(router, http.MethodPost, "/service-keys", []byte("not json"), cookie)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestServiceKeyHandler_Create_ServiceError(t *testing.T) {
	router, cookie, _, _, _ := setupServiceKeyAuthed(t, http.MethodPost, "/service-keys", func(ks *service.ServiceKeyService) gin.HandlerFunc { return NewServiceKeyHandler(ks).Create })

	body, _ := json.Marshal(map[string]any{"name": "x", "environment_ids": []string{"does-not-exist"}})
	rec := doRequest(router, http.MethodPost, "/service-keys", body, cookie)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestServiceKeyHandler_Get(t *testing.T) {
	router, cookie, ks, actor, envID := setupServiceKeyAuthed(t, http.MethodGet, "/service-keys/:id", func(ks *service.ServiceKeyService) gin.HandlerFunc { return NewServiceKeyHandler(ks).Get })
	k, _, _, err := ks.Create(context.Background(), actor, service.CreateServiceKeyInput{Name: "x", CanRead: true, EnvironmentIDs: []string{envID}})
	if err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	rec := doRequest(router, http.MethodGet, "/service-keys/"+k.ID, nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp serviceKeyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if resp.ID != k.ID {
		t.Errorf("ID = %q, want %q", resp.ID, k.ID)
	}
}

func TestServiceKeyHandler_Get_NotFound(t *testing.T) {
	router, cookie, _, _, _ := setupServiceKeyAuthed(t, http.MethodGet, "/service-keys/:id", func(ks *service.ServiceKeyService) gin.HandlerFunc { return NewServiceKeyHandler(ks).Get })

	rec := doRequest(router, http.MethodGet, "/service-keys/does-not-exist", nil, cookie)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestServiceKeyHandler_List(t *testing.T) {
	router, cookie, ks, actor, envID := setupServiceKeyAuthed(t, http.MethodGet, "/service-keys", func(ks *service.ServiceKeyService) gin.HandlerFunc { return NewServiceKeyHandler(ks).List })
	if _, _, _, err := ks.Create(context.Background(), actor, service.CreateServiceKeyInput{Name: "x", CanRead: true, EnvironmentIDs: []string{envID}}); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	rec := doRequest(router, http.MethodGet, "/service-keys?page=1&page_size=10", nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Data       []serviceKeyResponse `json:"data"`
		Pagination paginationResponse   `json:"pagination"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Errorf("len(Data) = %d, want 1", len(resp.Data))
	}
	if len(resp.Data[0].EnvironmentIDs) != 1 || resp.Data[0].EnvironmentIDs[0] != envID {
		t.Errorf("Data[0].EnvironmentIDs = %v, want [%s]", resp.Data[0].EnvironmentIDs, envID)
	}
}

func TestServiceKeyHandler_List_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newTestDB(t)
	ks := service.NewServiceKeyService(
		repository.NewServiceKeyRepository(db),
		repository.NewEnvironmentRepository(db),
		repository.NewServiceKeyEnvironmentRepository(db),
	)
	_ = db.Close()

	h := NewServiceKeyHandler(ks)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/service-keys", nil)

	h.List(c)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
}

func TestServiceKeyHandler_Update(t *testing.T) {
	router, cookie, ks, actor, envID := setupServiceKeyAuthed(t, http.MethodPatch, "/service-keys/:id", func(ks *service.ServiceKeyService) gin.HandlerFunc { return NewServiceKeyHandler(ks).Update })
	k, _, _, err := ks.Create(context.Background(), actor, service.CreateServiceKeyInput{Name: "x", CanRead: true, EnvironmentIDs: []string{envID}})
	if err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	body, _ := json.Marshal(map[string]string{"name": "x-renamed"})
	rec := doRequest(router, http.MethodPatch, "/service-keys/"+k.ID, body, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp serviceKeyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if resp.Name != "x-renamed" {
		t.Errorf("Name = %q, want %q", resp.Name, "x-renamed")
	}
}

func TestServiceKeyHandler_Update_BindError(t *testing.T) {
	router, cookie, _, _, _ := setupServiceKeyAuthed(t, http.MethodPatch, "/service-keys/:id", func(ks *service.ServiceKeyService) gin.HandlerFunc { return NewServiceKeyHandler(ks).Update })

	rec := doRequest(router, http.MethodPatch, "/service-keys/irrelevant", []byte("not json"), cookie)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestServiceKeyHandler_Update_ServiceError(t *testing.T) {
	router, cookie, _, _, _ := setupServiceKeyAuthed(t, http.MethodPatch, "/service-keys/:id", func(ks *service.ServiceKeyService) gin.HandlerFunc { return NewServiceKeyHandler(ks).Update })

	body, _ := json.Marshal(map[string]string{"name": "x"})
	rec := doRequest(router, http.MethodPatch, "/service-keys/does-not-exist", body, cookie)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestServiceKeyHandler_Deactivate(t *testing.T) {
	router, cookie, ks, actor, envID := setupServiceKeyAuthed(t, http.MethodDelete, "/service-keys/:id", func(ks *service.ServiceKeyService) gin.HandlerFunc { return NewServiceKeyHandler(ks).Deactivate })
	k, _, _, err := ks.Create(context.Background(), actor, service.CreateServiceKeyInput{Name: "x", CanRead: true, EnvironmentIDs: []string{envID}})
	if err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	rec := doRequest(router, http.MethodDelete, "/service-keys/"+k.ID, nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp serviceKeyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if resp.Status != model.ServiceKeyStatusInactive {
		t.Errorf("Status = %q, want %q", resp.Status, model.ServiceKeyStatusInactive)
	}
}

func TestServiceKeyHandler_Deactivate_ServiceError(t *testing.T) {
	router, cookie, _, _, _ := setupServiceKeyAuthed(t, http.MethodDelete, "/service-keys/:id", func(ks *service.ServiceKeyService) gin.HandlerFunc { return NewServiceKeyHandler(ks).Deactivate })

	rec := doRequest(router, http.MethodDelete, "/service-keys/does-not-exist", nil, cookie)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
