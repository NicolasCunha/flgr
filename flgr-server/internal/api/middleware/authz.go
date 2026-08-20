package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/NicolasCunha/flgr/flgr-server/internal/api/apierror"
	"github.com/NicolasCunha/flgr/flgr-server/internal/service"
)

// RequirePermission aborts the request with 403 unless the authenticated
// user (set by RequireAuth, which must run first) has the given
// (resource, action) permission, per
// docs/business/requirements/0003-profile-and-permission-management.md.
//
// Only for routes with no "self" concept (e.g. creating or listing
// Users) — actions like GET/PATCH /users/:id, where a user may always
// act on their own record regardless of permissions, check authorization
// inside the service layer instead (see service.UserService.Get/.Update),
// since that can't be expressed as a static per-route check.
func RequirePermission(authz *service.AuthzService, resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := CurrentUser(c)
		if user == nil {
			apierror.Write(c, http.StatusUnauthorized, "unauthenticated", "authentication required")
			c.Abort()
			return
		}

		allowed, err := authz.HasPermission(c.Request.Context(), user.ID, resource, action)
		if err != nil {
			apierror.Write(c, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
			c.Abort()
			return
		}
		if !allowed {
			apierror.Write(c, http.StatusForbidden, "forbidden", "you do not have permission to perform this action")
			c.Abort()
			return
		}

		c.Next()
	}
}
