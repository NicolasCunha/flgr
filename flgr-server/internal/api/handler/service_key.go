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

// ServiceKeyHandler exposes docs/business/requirements/0004-service-key-management.md's
// service key CRUD over HTTP. Every route is gated by route middleware
// (ServiceKey: Create/Edit/Remove/View, see router.go) — service keys have
// no "self" concept, unlike Users.
type ServiceKeyHandler struct {
	serviceKeys *service.ServiceKeyService
}

// NewServiceKeyHandler returns a ServiceKeyHandler backed by serviceKeys.
func NewServiceKeyHandler(serviceKeys *service.ServiceKeyService) *ServiceKeyHandler {
	return &ServiceKeyHandler{serviceKeys: serviceKeys}
}

type serviceKeyResponse struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Status         string    `json:"status"`
	CanRead        bool      `json:"can_read"`
	CanWrite       bool      `json:"can_write"`
	EnvironmentIDs []string  `json:"environment_ids"`
	CreatedOn      time.Time `json:"created_on"`
	ModifiedOn     time.Time `json:"modified_on"`
}

func toServiceKeyResponse(k *model.ServiceKey, environmentIDs []string) serviceKeyResponse {
	return serviceKeyResponse{
		ID:             k.ID,
		Name:           k.Name,
		Status:         k.Status,
		CanRead:        k.CanRead,
		CanWrite:       k.CanWrite,
		EnvironmentIDs: environmentIDs,
		CreatedOn:      k.CreatedOn,
		ModifiedOn:     k.ModifiedOn,
	}
}

// createServiceKeyResponse embeds serviceKeyResponse plus the plaintext
// secret — present only in this one response, per 0004 ("shown in full
// only once, at creation time").
type createServiceKeyResponse struct {
	serviceKeyResponse
	Secret string `json:"secret"`
}

type createServiceKeyRequest struct {
	Name           string   `json:"name" binding:"required"`
	CanRead        *bool    `json:"can_read"`
	CanWrite       *bool    `json:"can_write"`
	EnvironmentIDs []string `json:"environment_ids" binding:"required"`
}

// Create handles POST /api/v1/service-keys. can_read/can_write default to
// the service_keys table's own defaults (true/false, per the Data Model in
// 0004) when omitted from the request.
func (h *ServiceKeyHandler) Create(c *gin.Context) {
	var req createServiceKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Write(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	canRead := true
	if req.CanRead != nil {
		canRead = *req.CanRead
	}
	canWrite := false
	if req.CanWrite != nil {
		canWrite = *req.CanWrite
	}

	actor := middleware.CurrentUser(c)
	k, envIDs, secret, err := h.serviceKeys.Create(c.Request.Context(), actor, service.CreateServiceKeyInput{
		Name:           req.Name,
		CanRead:        canRead,
		CanWrite:       canWrite,
		EnvironmentIDs: req.EnvironmentIDs,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, createServiceKeyResponse{
		serviceKeyResponse: toServiceKeyResponse(k, envIDs),
		Secret:             secret,
	})
}

// Get handles GET /api/v1/service-keys/:id.
func (h *ServiceKeyHandler) Get(c *gin.Context) {
	k, envIDs, err := h.serviceKeys.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, toServiceKeyResponse(k, envIDs))
}

// List handles GET /api/v1/service-keys, per the list envelope in
// ADR-0007.
func (h *ServiceKeyHandler) List(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 50)

	keys, envIDsByKey, total, err := h.serviceKeys.List(c.Request.Context(), page, pageSize)
	if err != nil {
		respondError(c, err)
		return
	}

	data := make([]serviceKeyResponse, len(keys))
	for i, k := range keys {
		data[i] = toServiceKeyResponse(&k, envIDsByKey[k.ID])
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       data,
		"pagination": paginationResponse{Page: page, PageSize: pageSize, Total: total},
	})
}

type updateServiceKeyRequest struct {
	Name           *string  `json:"name"`
	Status         *string  `json:"status"`
	CanRead        *bool    `json:"can_read"`
	CanWrite       *bool    `json:"can_write"`
	EnvironmentIDs []string `json:"environment_ids"`
}

// Update handles PATCH /api/v1/service-keys/:id.
func (h *ServiceKeyHandler) Update(c *gin.Context) {
	var req updateServiceKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Write(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	actor := middleware.CurrentUser(c)
	k, envIDs, err := h.serviceKeys.Update(c.Request.Context(), actor, c.Param("id"), service.UpdateServiceKeyInput{
		Name:           req.Name,
		Status:         req.Status,
		CanRead:        req.CanRead,
		CanWrite:       req.CanWrite,
		EnvironmentIDs: req.EnvironmentIDs,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, toServiceKeyResponse(k, envIDs))
}

// Deactivate handles DELETE /api/v1/service-keys/:id — a soft delete
// (status set to Inactive), per
// docs/architecture/adr/0005-audit-columns-and-soft-delete-convention.md
// and the DELETE-maps-to-deactivation convention in ADR-0007.
func (h *ServiceKeyHandler) Deactivate(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	k, envIDs, err := h.serviceKeys.Deactivate(c.Request.Context(), actor, c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, toServiceKeyResponse(k, envIDs))
}
