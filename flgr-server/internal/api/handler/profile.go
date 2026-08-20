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

// ProfileHandler exposes docs/business/requirements/0003-profile-and-permission-management.md's
// Profile CRUD over HTTP. Every route is gated by route middleware
// (Profile: Create/Edit/Remove/View, see router.go) — profiles have no
// "self" concept, unlike Users.
type ProfileHandler struct {
	profiles *service.ProfileService
}

// NewProfileHandler returns a ProfileHandler backed by profiles.
func NewProfileHandler(profiles *service.ProfileService) *ProfileHandler {
	return &ProfileHandler{profiles: profiles}
}

type profileResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	IsSystem    bool      `json:"is_system"`
	CreatedOn   time.Time `json:"created_on"`
	ModifiedOn  time.Time `json:"modified_on"`
}

func toProfileResponse(p *model.Profile) profileResponse {
	return profileResponse{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		IsSystem:    p.IsSystem,
		CreatedOn:   p.CreatedOn,
		ModifiedOn:  p.ModifiedOn,
	}
}

type createProfileRequest struct {
	Name          string   `json:"name" binding:"required"`
	Description   *string  `json:"description"`
	PermissionIDs []string `json:"permission_ids"`
}

// Create handles POST /api/v1/profiles.
func (h *ProfileHandler) Create(c *gin.Context) {
	var req createProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Write(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	actor := middleware.CurrentUser(c)
	p, err := h.profiles.Create(c.Request.Context(), actor, service.CreateProfileInput{
		Name:          req.Name,
		Description:   req.Description,
		PermissionIDs: req.PermissionIDs,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toProfileResponse(p))
}

// Get handles GET /api/v1/profiles/:id.
func (h *ProfileHandler) Get(c *gin.Context) {
	p, err := h.profiles.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, toProfileResponse(p))
}

// List handles GET /api/v1/profiles, per the list envelope in ADR-0007.
func (h *ProfileHandler) List(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 50)

	profiles, total, err := h.profiles.List(c.Request.Context(), page, pageSize)
	if err != nil {
		respondError(c, err)
		return
	}

	data := make([]profileResponse, len(profiles))
	for i, p := range profiles {
		data[i] = toProfileResponse(&p)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       data,
		"pagination": paginationResponse{Page: page, PageSize: pageSize, Total: total},
	})
}

// PermissionIDs handles GET /api/v1/profiles/:id/permissions — the
// profile's directly-assigned permission ids (not resolved/effective;
// profiles don't inherit from anything else).
func (h *ProfileHandler) PermissionIDs(c *gin.Context) {
	ids, err := h.profiles.PermissionIDs(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"permission_ids": ids})
}

type updateProfileRequest struct {
	Name          *string   `json:"name"`
	Description   *string   `json:"description"`
	PermissionIDs *[]string `json:"permission_ids"`
}

// Update handles PATCH /api/v1/profiles/:id.
func (h *ProfileHandler) Update(c *gin.Context) {
	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Write(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	actor := middleware.CurrentUser(c)
	p, err := h.profiles.Update(c.Request.Context(), actor, c.Param("id"), service.UpdateProfileInput{
		Name:          req.Name,
		Description:   req.Description,
		PermissionIDs: req.PermissionIDs,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, toProfileResponse(p))
}

// Delete handles DELETE /api/v1/profiles/:id — a real, hard delete
// (profiles carry no created_by/modified_by references from other
// tables, so unlike Users they have no soft-delete requirement; see
// docs/architecture/adr/0005-audit-columns-and-soft-delete-convention.md).
func (h *ProfileHandler) Delete(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	if err := h.profiles.Delete(c.Request.Context(), actor, c.Param("id")); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusOK)
}
