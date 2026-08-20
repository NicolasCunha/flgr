package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/service"
)

// EnvironmentCategoryHandler exposes the environment category list from
// docs/business/requirements/0001-environment-management.md. Gated by
// the same "Environment: View" permission as environments themselves —
// 0003's permission catalog has no separate category permission.
type EnvironmentCategoryHandler struct {
	categories *service.EnvironmentCategoryService
}

// NewEnvironmentCategoryHandler returns an EnvironmentCategoryHandler
// backed by categories.
func NewEnvironmentCategoryHandler(categories *service.EnvironmentCategoryService) *EnvironmentCategoryHandler {
	return &EnvironmentCategoryHandler{categories: categories}
}

type environmentCategoryResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	IsSystem bool   `json:"is_system"`
}

func toEnvironmentCategoryResponse(c *model.EnvironmentCategory) environmentCategoryResponse {
	return environmentCategoryResponse{ID: c.ID, Name: c.Name, IsSystem: c.IsSystem}
}

// List handles GET /api/v1/environment-categories.
func (h *EnvironmentCategoryHandler) List(c *gin.Context) {
	categories, err := h.categories.List(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}

	data := make([]environmentCategoryResponse, len(categories))
	for i, cat := range categories {
		data[i] = toEnvironmentCategoryResponse(&cat)
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}
