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

func TestEnvironmentCategoryHandler_List(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newTestDB(t)
	h := NewEnvironmentCategoryHandler(service.NewEnvironmentCategoryService(repository.NewEnvironmentCategoryRepository(db)))
	router := gin.New()
	router.GET("/environment-categories", h.List)

	rec := doRequest(router, http.MethodGet, "/environment-categories", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Data []environmentCategoryResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if len(resp.Data) != 3 {
		t.Errorf("len(Data) = %d, want 3 (the seeded categories)", len(resp.Data))
	}
}

func TestEnvironmentCategoryHandler_List_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newTestDB(t)
	h := NewEnvironmentCategoryHandler(service.NewEnvironmentCategoryService(repository.NewEnvironmentCategoryRepository(db)))
	_ = db.Close()

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/environment-categories", nil)

	h.List(c)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
}
