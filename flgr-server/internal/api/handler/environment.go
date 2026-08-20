package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/NicolasCunha/flgr/flgr-server/internal/api/apierror"
	"github.com/NicolasCunha/flgr/flgr-server/internal/api/middleware"
	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/service"
)

// EnvironmentHandler exposes docs/business/requirements/0001-environment-management.md's
// Environment CRUD over HTTP. Every route is gated by route middleware
// (Environment: Create/Edit/Remove/View, see router.go) — environments
// have no "self" concept, unlike Users.
type EnvironmentHandler struct {
	environments *service.EnvironmentService
}

// NewEnvironmentHandler returns an EnvironmentHandler backed by environments.
func NewEnvironmentHandler(environments *service.EnvironmentService) *EnvironmentHandler {
	return &EnvironmentHandler{environments: environments}
}

type environmentResponse struct {
	ID                    string    `json:"id"`
	Name                  string    `json:"name"`
	Description           *string   `json:"description"`
	EnvironmentCategoryID string    `json:"environment_category_id"`
	CreatedOn             time.Time `json:"created_on"`
	ModifiedOn            time.Time `json:"modified_on"`
}

func toEnvironmentResponse(e *model.Environment) environmentResponse {
	return environmentResponse{
		ID:                    e.ID,
		Name:                  e.Name,
		Description:           e.Description,
		EnvironmentCategoryID: e.EnvironmentCategoryID,
		CreatedOn:             e.CreatedOn,
		ModifiedOn:            e.ModifiedOn,
	}
}

type createEnvironmentRequest struct {
	Name         string  `json:"name" binding:"required"`
	Description  *string `json:"description"`
	CategoryName string  `json:"category_name" binding:"required"`
}

// Create handles POST /api/v1/environments.
func (h *EnvironmentHandler) Create(c *gin.Context) {
	var req createEnvironmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Write(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	actor := middleware.CurrentUser(c)
	e, err := h.environments.Create(c.Request.Context(), actor, service.CreateEnvironmentInput{
		Name:         req.Name,
		Description:  req.Description,
		CategoryName: req.CategoryName,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toEnvironmentResponse(e))
}

// Get handles GET /api/v1/environments/:id.
func (h *EnvironmentHandler) Get(c *gin.Context) {
	e, err := h.environments.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, toEnvironmentResponse(e))
}

// List handles GET /api/v1/environments, per the list envelope in ADR-0007.
func (h *EnvironmentHandler) List(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 50)

	environments, total, err := h.environments.List(c.Request.Context(), page, pageSize)
	if err != nil {
		respondError(c, err)
		return
	}

	data := make([]environmentResponse, len(environments))
	for i, e := range environments {
		data[i] = toEnvironmentResponse(&e)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       data,
		"pagination": paginationResponse{Page: page, PageSize: pageSize, Total: total},
	})
}

type updateEnvironmentRequest struct {
	Name         *string `json:"name"`
	Description  *string `json:"description"`
	CategoryName *string `json:"category_name"`
}

// Update handles PATCH /api/v1/environments/:id.
func (h *EnvironmentHandler) Update(c *gin.Context) {
	var req updateEnvironmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Write(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	actor := middleware.CurrentUser(c)
	e, err := h.environments.Update(c.Request.Context(), actor, c.Param("id"), service.UpdateEnvironmentInput{
		Name:         req.Name,
		Description:  req.Description,
		CategoryName: req.CategoryName,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, toEnvironmentResponse(e))
}

// Delete handles DELETE /api/v1/environments/:id.
func (h *EnvironmentHandler) Delete(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	if err := h.environments.Delete(c.Request.Context(), actor, c.Param("id")); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusOK)
}
