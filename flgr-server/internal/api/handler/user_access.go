package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/NicolasCunha/flgr/flgr-server/internal/api/apierror"
	"github.com/NicolasCunha/flgr/flgr-server/internal/api/middleware"
	"github.com/NicolasCunha/flgr/flgr-server/internal/service"
)

// UserAccessHandler exposes a User's profile memberships and direct
// permission grants over HTTP, per
// docs/business/requirements/0003-profile-and-permission-management.md.
// Every route is gated by the "User: Edit" permission via route
// middleware (see router.go) — there's no self-service exception here,
// unlike UserHandler.Get/.Update.
type UserAccessHandler struct {
	access *service.UserAccessService
}

// NewUserAccessHandler returns a UserAccessHandler backed by access.
func NewUserAccessHandler(access *service.UserAccessService) *UserAccessHandler {
	return &UserAccessHandler{access: access}
}

// GetProfiles handles GET /api/v1/users/:id/profiles.
func (h *UserAccessHandler) GetProfiles(c *gin.Context) {
	ids, err := h.access.ProfileIDs(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"profile_ids": ids})
}

type replaceProfilesRequest struct {
	ProfileIDs []string `json:"profile_ids"`
}

// ReplaceProfiles handles PATCH /api/v1/users/:id/profiles — replaces
// the user's entire set of profile memberships.
func (h *UserAccessHandler) ReplaceProfiles(c *gin.Context) {
	var req replaceProfilesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Write(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	actor := middleware.CurrentUser(c)
	if err := h.access.ReplaceProfiles(c.Request.Context(), actor, c.Param("id"), req.ProfileIDs); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusOK)
}

// GetDirectPermissions handles GET /api/v1/users/:id/permissions/direct.
func (h *UserAccessHandler) GetDirectPermissions(c *gin.Context) {
	ids, err := h.access.DirectPermissionIDs(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"permission_ids": ids})
}

type replaceDirectPermissionsRequest struct {
	PermissionIDs []string `json:"permission_ids"`
}

// ReplaceDirectPermissions handles PATCH /api/v1/users/:id/permissions/direct
// — replaces the user's entire set of direct (profile-bypassing) grants.
func (h *UserAccessHandler) ReplaceDirectPermissions(c *gin.Context) {
	var req replaceDirectPermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Write(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	actor := middleware.CurrentUser(c)
	if err := h.access.ReplaceDirectPermissions(c.Request.Context(), actor, c.Param("id"), req.PermissionIDs); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusOK)
}
