package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/NicolasCunha/flgr/flgr-server/internal/repository"
	"github.com/NicolasCunha/flgr/flgr-server/internal/service"
)

func TestPermissionHandler_List(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newTestDB(t)
	h := NewPermissionHandler(service.NewPermissionService(repository.NewPermissionRepository(db)))
	router := gin.New()
	router.GET("/permissions", h.List)

	rec := doRequest(router, http.MethodGet, "/permissions", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Data []permissionResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if len(resp.Data) != 22 {
		t.Errorf("len(Data) = %d, want 22 (the seeded catalog)", len(resp.Data))
	}
}

func TestPermissionHandler_List_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newTestDB(t)
	h := NewPermissionHandler(service.NewPermissionService(repository.NewPermissionRepository(db)))
	_ = db.Close()

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/permissions", nil)

	h.List(c)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
}
