package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/service"
)

// PermissionHandler exposes the fixed permission catalog from
// docs/business/requirements/0003-profile-and-permission-management.md.
// Only requires authentication (see router.go) — the catalog itself
// isn't sensitive, and every profile/user permission-editing screen
// needs it to render its options.
type PermissionHandler struct {
	permissions *service.PermissionService
}

// NewPermissionHandler returns a PermissionHandler backed by permissions.
func NewPermissionHandler(permissions *service.PermissionService) *PermissionHandler {
	return &PermissionHandler{permissions: permissions}
}

type permissionResponse struct {
	ID       string `json:"id"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

func toPermissionResponse(p *model.Permission) permissionResponse {
	return permissionResponse{ID: p.ID, Resource: p.Resource, Action: p.Action}
}

// List handles GET /api/v1/permissions.
func (h *PermissionHandler) List(c *gin.Context) {
	permissions, err := h.permissions.List(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}

	data := make([]permissionResponse, len(permissions))
	for i, p := range permissions {
		data[i] = toPermissionResponse(&p)
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}
