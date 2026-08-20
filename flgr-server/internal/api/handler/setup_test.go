package handler

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/NicolasCunha/flgr/flgr-server/internal/repository"
	"github.com/NicolasCunha/flgr/flgr-server/internal/service"
)

func TestSetupHandler_Status_Needed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userService, _, db := newTestServicesWithDB(t)
	h := NewSetupHandler(userService, newTestUserAccessService(db))
	router := gin.New()
	router.GET("/setup", h.Status)

	rec := doRequest(router, http.MethodGet, "/setup", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp setupStatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if !resp.Needed {
		t.Error("Needed = false, want true when no users exist")
	}
}

func TestSetupHandler_Status_NotNeeded(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userService, _, db := newTestServicesWithDB(t)
	if _, err := userService.Create(t.Context(), nil, service.CreateUserInput{
		FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com", Password: "supersecret",
	}); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	h := NewSetupHandler(userService, newTestUserAccessService(db))
	router := gin.New()
	router.GET("/setup", h.Status)

	rec := doRequest(router, http.MethodGet, "/setup", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp setupStatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if resp.Needed {
		t.Error("Needed = true, want false once a user exists")
	}
}

func TestSetupHandler_Status_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newTestDB(t)
	userRepo := repository.NewUserRepository(db)
	userService := service.NewUserService(userRepo, service.NewAuthzService(repository.NewAuthzRepository(db)))
	access := newTestUserAccessService(db)
	_ = db.Close()

	h := NewSetupHandler(userService, access)
	router := gin.New()
	router.GET("/setup", h.Status)

	rec := doRequest(router, http.MethodGet, "/setup", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
}

func TestSetupHandler_Complete_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userService, _, db := newTestServicesWithDB(t)
	h := NewSetupHandler(userService, newTestUserAccessService(db))
	router := gin.New()
	router.POST("/setup", h.Complete)

	body, _ := json.Marshal(map[string]string{
		"first_name": "Ada", "last_name": "Lovelace", "email": "ada@example.com", "password": "supersecret",
	})
	rec := doRequest(router, http.MethodPost, "/setup", body, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var resp userResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if resp.Email != "ada@example.com" {
		t.Errorf("Email = %q, want %q", resp.Email, "ada@example.com")
	}

	// The new admin must actually have been granted the Administrador
	// profile — otherwise the wizard produces a User who can do nothing.
	ids, err := newTestUserAccessService(db).ProfileIDs(t.Context(), resp.ID)
	if err != nil {
		t.Fatalf("ProfileIDs() returned unexpected error: %v", err)
	}
	if len(ids) != 1 || ids[0] != "00000000-0000-0000-0000-000000000010" {
		t.Errorf("ProfileIDs() = %v, want the seeded Administrador profile", ids)
	}
}

func TestSetupHandler_Complete_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userService, _, db := newTestServicesWithDB(t)
	h := NewSetupHandler(userService, newTestUserAccessService(db))
	router := gin.New()
	router.POST("/setup", h.Complete)

	rec := doRequest(router, http.MethodPost, "/setup", []byte("not json"), nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestSetupHandler_Complete_AlreadyDone(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userService, _, db := newTestServicesWithDB(t)
	if _, err := userService.Create(t.Context(), nil, service.CreateUserInput{
		FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com", Password: "supersecret",
	}); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	h := NewSetupHandler(userService, newTestUserAccessService(db))
	router := gin.New()
	router.POST("/setup", h.Complete)

	body, _ := json.Marshal(map[string]string{
		"first_name": "Grace", "last_name": "Hopper", "email": "grace@example.com", "password": "supersecret",
	})
	rec := doRequest(router, http.MethodPost, "/setup", body, nil)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestSetupHandler_Complete_AccessServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userService, _, db := newTestServicesWithDB(t)
	access := newTestUserAccessService(db)
	h := NewSetupHandler(userService, access)
	router := gin.New()
	router.POST("/setup", h.Complete)

	// Defensive scenario: the seeded Administrador profile is somehow
	// gone (shouldn't happen — ProfileService.Delete guards it — but
	// exercises Complete's error path when the follow-up ReplaceProfiles
	// call fails, rather than silently creating a powerless admin).
	if _, err := db.Exec("DELETE FROM profiles WHERE id = '00000000-0000-0000-0000-000000000010'"); err != nil {
		t.Fatalf("removing seeded Administrador profile: %v", err)
	}

	body, _ := json.Marshal(map[string]string{
		"first_name": "Ada", "last_name": "Lovelace", "email": "ada@example.com", "password": "supersecret",
	})
	rec := doRequest(router, http.MethodPost, "/setup", body, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}
