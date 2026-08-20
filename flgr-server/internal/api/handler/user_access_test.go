package handler

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/NicolasCunha/flgr/flgr-server/internal/api/middleware"
	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/service"
)

// setupUserAccessAuthed mirrors setupAuthed but for UserAccessService-
// backed routes: one shared DB, an authenticated actor, a second "target"
// user to act on, and a single route.
func setupUserAccessAuthed(t *testing.T, method, path string, makeHandler func(*service.UserAccessService) gin.HandlerFunc) (*gin.Engine, *http.Cookie, *service.UserAccessService, *model.User) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	userService, authService, db := newTestServicesWithDB(t)
	if _, err := userService.Create(t.Context(), nil, service.CreateUserInput{
		FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com", Password: "supersecret",
	}); err != nil {
		t.Fatalf("Create(actor) returned unexpected error: %v", err)
	}
	target, err := userService.Create(t.Context(), nil, service.CreateUserInput{
		FirstName: "Bob", LastName: "Smith", Email: "bob@example.com", Password: "supersecret",
	})
	if err != nil {
		t.Fatalf("Create(target) returned unexpected error: %v", err)
	}
	_, token, err := authService.Login(t.Context(), "ada@example.com", "supersecret")
	if err != nil {
		t.Fatalf("Login() returned unexpected error: %v", err)
	}

	access := newTestUserAccessService(db)

	router := gin.New()
	router.Handle(method, path, middleware.RequireAuth(authService), makeHandler(access))

	return router, &http.Cookie{Name: middleware.SessionCookieName, Value: token}, access, target
}

func TestUserAccessHandler_GetProfiles(t *testing.T) {
	router, cookie, access, target := setupUserAccessAuthed(t, http.MethodGet, "/users/:id/profiles", func(a *service.UserAccessService) gin.HandlerFunc { return NewUserAccessHandler(a).GetProfiles })
	if err := access.ReplaceProfiles(t.Context(), target, target.ID, []string{model.AdministradorProfileID}); err != nil {
		t.Fatalf("ReplaceProfiles() returned unexpected error: %v", err)
	}

	rec := doRequest(router, http.MethodGet, "/users/"+target.ID+"/profiles", nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp struct {
		ProfileIDs []string `json:"profile_ids"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if len(resp.ProfileIDs) != 1 || resp.ProfileIDs[0] != model.AdministradorProfileID {
		t.Errorf("ProfileIDs = %v, want [%s]", resp.ProfileIDs, model.AdministradorProfileID)
	}
}

func TestUserAccessHandler_GetProfiles_NotFound(t *testing.T) {
	router, cookie, _, _ := setupUserAccessAuthed(t, http.MethodGet, "/users/:id/profiles", func(a *service.UserAccessService) gin.HandlerFunc { return NewUserAccessHandler(a).GetProfiles })

	rec := doRequest(router, http.MethodGet, "/users/does-not-exist/profiles", nil, cookie)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestUserAccessHandler_ReplaceProfiles(t *testing.T) {
	router, cookie, _, target := setupUserAccessAuthed(t, http.MethodPatch, "/users/:id/profiles", func(a *service.UserAccessService) gin.HandlerFunc { return NewUserAccessHandler(a).ReplaceProfiles })

	body, _ := json.Marshal(map[string][]string{"profile_ids": {model.AdministradorProfileID}})
	rec := doRequest(router, http.MethodPatch, "/users/"+target.ID+"/profiles", body, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestUserAccessHandler_ReplaceProfiles_BindError(t *testing.T) {
	router, cookie, _, target := setupUserAccessAuthed(t, http.MethodPatch, "/users/:id/profiles", func(a *service.UserAccessService) gin.HandlerFunc { return NewUserAccessHandler(a).ReplaceProfiles })

	rec := doRequest(router, http.MethodPatch, "/users/"+target.ID+"/profiles", []byte("not json"), cookie)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestUserAccessHandler_ReplaceProfiles_ServiceError(t *testing.T) {
	router, cookie, _, _ := setupUserAccessAuthed(t, http.MethodPatch, "/users/:id/profiles", func(a *service.UserAccessService) gin.HandlerFunc { return NewUserAccessHandler(a).ReplaceProfiles })

	body, _ := json.Marshal(map[string][]string{"profile_ids": {"does-not-exist"}})
	rec := doRequest(router, http.MethodPatch, "/users/does-not-exist/profiles", body, cookie)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestUserAccessHandler_GetDirectPermissions(t *testing.T) {
	router, cookie, access, target := setupUserAccessAuthed(t, http.MethodGet, "/users/:id/permissions/direct", func(a *service.UserAccessService) gin.HandlerFunc {
		return NewUserAccessHandler(a).GetDirectPermissions
	})
	if err := access.ReplaceDirectPermissions(t.Context(), target, target.ID, []string{"user:view"}); err != nil {
		t.Fatalf("ReplaceDirectPermissions() returned unexpected error: %v", err)
	}

	rec := doRequest(router, http.MethodGet, "/users/"+target.ID+"/permissions/direct", nil, cookie)
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

func TestUserAccessHandler_GetDirectPermissions_NotFound(t *testing.T) {
	router, cookie, _, _ := setupUserAccessAuthed(t, http.MethodGet, "/users/:id/permissions/direct", func(a *service.UserAccessService) gin.HandlerFunc {
		return NewUserAccessHandler(a).GetDirectPermissions
	})

	rec := doRequest(router, http.MethodGet, "/users/does-not-exist/permissions/direct", nil, cookie)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestUserAccessHandler_ReplaceDirectPermissions(t *testing.T) {
	router, cookie, _, target := setupUserAccessAuthed(t, http.MethodPatch, "/users/:id/permissions/direct", func(a *service.UserAccessService) gin.HandlerFunc {
		return NewUserAccessHandler(a).ReplaceDirectPermissions
	})

	body, _ := json.Marshal(map[string][]string{"permission_ids": {"user:view"}})
	rec := doRequest(router, http.MethodPatch, "/users/"+target.ID+"/permissions/direct", body, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestUserAccessHandler_ReplaceDirectPermissions_BindError(t *testing.T) {
	router, cookie, _, target := setupUserAccessAuthed(t, http.MethodPatch, "/users/:id/permissions/direct", func(a *service.UserAccessService) gin.HandlerFunc {
		return NewUserAccessHandler(a).ReplaceDirectPermissions
	})

	rec := doRequest(router, http.MethodPatch, "/users/"+target.ID+"/permissions/direct", []byte("not json"), cookie)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestUserAccessHandler_ReplaceDirectPermissions_ServiceError(t *testing.T) {
	router, cookie, _, _ := setupUserAccessAuthed(t, http.MethodPatch, "/users/:id/permissions/direct", func(a *service.UserAccessService) gin.HandlerFunc {
		return NewUserAccessHandler(a).ReplaceDirectPermissions
	})

	body, _ := json.Marshal(map[string][]string{"permission_ids": {"does-not-exist"}})
	rec := doRequest(router, http.MethodPatch, "/users/does-not-exist/permissions/direct", body, cookie)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}
