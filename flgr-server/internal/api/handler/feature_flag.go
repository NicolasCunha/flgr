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

// FeatureFlagHandler exposes docs/business/requirements/0005-feature-flag-management.md's
// flag-definition CRUD over HTTP. Every route is gated by route middleware
// (FeatureFlag: Create/Edit/Remove/View, see router.go) — flags have no
// "self" concept, unlike Users. See FeatureFlagValueHandler for the
// per-environment values sub-resource.
type FeatureFlagHandler struct {
	flags *service.FeatureFlagService
}

// NewFeatureFlagHandler returns a FeatureFlagHandler backed by flags.
func NewFeatureFlagHandler(flags *service.FeatureFlagService) *FeatureFlagHandler {
	return &FeatureFlagHandler{flags: flags}
}

type featureFlagResponse struct {
	ID          string    `json:"id"`
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	Type        string    `json:"type"`
	CreatedOn   time.Time `json:"created_on"`
	ModifiedOn  time.Time `json:"modified_on"`
}

func toFeatureFlagResponse(f *model.FeatureFlag) featureFlagResponse {
	return featureFlagResponse{
		ID:          f.ID,
		Key:         f.Key,
		Name:        f.Name,
		Description: f.Description,
		Type:        f.Type,
		CreatedOn:   f.CreatedOn,
		ModifiedOn:  f.ModifiedOn,
	}
}

type createFeatureFlagRequest struct {
	Key         string  `json:"key" binding:"required"`
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description"`
	Type        string  `json:"type" binding:"required"`
}

// Create handles POST /api/v1/feature-flags.
func (h *FeatureFlagHandler) Create(c *gin.Context) {
	var req createFeatureFlagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Write(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	actor := middleware.CurrentUser(c)
	f, err := h.flags.Create(c.Request.Context(), actor, service.CreateFeatureFlagInput{
		Key:         req.Key,
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toFeatureFlagResponse(f))
}

// Get handles GET /api/v1/feature-flags/:id.
func (h *FeatureFlagHandler) Get(c *gin.Context) {
	f, err := h.flags.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, toFeatureFlagResponse(f))
}

// List handles GET /api/v1/feature-flags, per the list envelope in
// ADR-0007.
func (h *FeatureFlagHandler) List(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 50)

	flags, total, err := h.flags.List(c.Request.Context(), page, pageSize)
	if err != nil {
		respondError(c, err)
		return
	}

	data := make([]featureFlagResponse, len(flags))
	for i, f := range flags {
		data[i] = toFeatureFlagResponse(&f)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       data,
		"pagination": paginationResponse{Page: page, PageSize: pageSize, Total: total},
	})
}

type updateFeatureFlagRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

// Update handles PATCH /api/v1/feature-flags/:id. Deliberately has no
// key/type fields to bind — both are immutable after creation, per 0005.
func (h *FeatureFlagHandler) Update(c *gin.Context) {
	var req updateFeatureFlagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Write(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	actor := middleware.CurrentUser(c)
	f, err := h.flags.Update(c.Request.Context(), actor, c.Param("id"), service.UpdateFeatureFlagInput{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, toFeatureFlagResponse(f))
}

// Delete handles DELETE /api/v1/feature-flags/:id — a real delete (not a
// soft one): a feature flag has no history dependency of its own, and its
// per-environment values cascade away with it, per 0005's Remove
// acceptance criterion.
func (h *FeatureFlagHandler) Delete(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	if err := h.flags.Delete(c.Request.Context(), actor, c.Param("id")); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusOK)
}
