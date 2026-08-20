// Package middleware holds Gin middleware shared across routes (auth,
// logging, recovery, etc.), per backend.md's project structure.
package middleware

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/NicolasCunha/flgr/flgr-server/internal/api/apierror"
	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/service"
)

// SessionCookieName is the cookie carrying the opaque session token, per
// docs/architecture/adr/0006-authentication-and-session-strategy.md.
const SessionCookieName = "flgr_session"

// bearerPrefix is the Authorization scheme RequireServiceKeyAccess and
// RequireWriteAccess expect for Service Key authentication, per ADR-0006.
const bearerPrefix = "Bearer "

// currentUserKey is the gin.Context key RequireAuth (and the session
// branch of RequireWriteAccess) stores the authenticated User under.
const currentUserKey = "current_user"

// currentServiceKeyKey / currentServiceKeyEnvironmentIDsKey are the
// gin.Context keys RequireServiceKeyAccess (and the Bearer branch of
// RequireWriteAccess) store the authenticated ServiceKey, and the
// environment ids it may access, under — the Service Key equivalent of
// currentUserKey.
const (
	currentServiceKeyKey               = "current_service_key"
	currentServiceKeyEnvironmentIDsKey = "current_service_key_environment_ids"
)

// RequireAuth resolves the session cookie via authService and aborts the
// request with 401 if it's missing or invalid, per ADR-0006 (a session
// must be unexpired and its user still Active, checked fresh on every
// request). On success, the authenticated user is available via CurrentUser.
func RequireAuth(authService *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := authenticateSession(c, authService)
		if !ok {
			c.Abort()
			return
		}
		c.Set(currentUserKey, user)
		c.Next()
	}
}

// authenticateSession is RequireAuth's cookie-lookup +
// AuthService.ValidateSession logic, factored out so RequireWriteAccess's
// session branch can reuse it verbatim instead of duplicating it. It
// writes the 401 response itself on failure, so a caller can just `return`
// (after c.Abort()) when ok is false.
func authenticateSession(c *gin.Context, authService *service.AuthService) (*model.User, bool) {
	token, err := c.Cookie(SessionCookieName)
	if err != nil || token == "" {
		log.Printf("auth: %s %s rejected — no %q cookie on the request", c.Request.Method, c.Request.URL.Path, SessionCookieName)
		apierror.Write(c, http.StatusUnauthorized, "unauthenticated", "authentication required")
		return nil, false
	}

	user, err := authService.ValidateSession(c.Request.Context(), token)
	if err != nil {
		// The specific reason (not found / expired / user inactive,
		// with timestamps) is logged inside ValidateSession itself —
		// this line just ties that reason to the request that hit it.
		log.Printf("auth: %s %s rejected — %v", c.Request.Method, c.Request.URL.Path, err)
		apierror.Write(c, http.StatusUnauthorized, "unauthenticated", "session is invalid or expired")
		return nil, false
	}

	return user, true
}

// CurrentUser returns the authenticated User set by RequireAuth (or the
// session branch of RequireWriteAccess), or nil if called outside of a
// route it guards.
func CurrentUser(c *gin.Context) *model.User {
	v, ok := c.Get(currentUserKey)
	if !ok {
		return nil
	}
	u, _ := v.(*model.User)
	return u
}

// CurrentServiceKey returns the authenticated ServiceKey set by
// RequireServiceKeyAccess (or the Bearer branch of RequireWriteAccess), or
// nil if called outside of a route it guards.
func CurrentServiceKey(c *gin.Context) *model.ServiceKey {
	v, ok := c.Get(currentServiceKeyKey)
	if !ok {
		return nil
	}
	k, _ := v.(*model.ServiceKey)
	return k
}

// CurrentServiceKeyEnvironmentIDs returns the environment ids the
// ServiceKey set by RequireServiceKeyAccess (or the Bearer branch of
// RequireWriteAccess) may access, or nil if called outside of a route it
// guards.
func CurrentServiceKeyEnvironmentIDs(c *gin.Context) []string {
	v, ok := c.Get(currentServiceKeyEnvironmentIDsKey)
	if !ok {
		return nil
	}
	ids, _ := v.([]string)
	return ids
}

// bearerToken extracts the secret from an `Authorization: Bearer <secret>`
// header, per ADR-0006's Service Key authentication section. ok is false
// if the header is missing, malformed, or doesn't use the Bearer scheme.
func bearerToken(c *gin.Context) (string, bool) {
	header := c.GetHeader("Authorization")
	if !strings.HasPrefix(header, bearerPrefix) {
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, bearerPrefix))
	if token == "" {
		return "", false
	}
	return token, true
}

// containsString reports whether target is present in values.
func containsString(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}

// authenticateServiceKey is RequireServiceKeyAccess's Bearer-token +
// ServiceKeyService.Authenticate + capability/environment-scope logic,
// factored out so RequireWriteAccess's Bearer branch can reuse it verbatim
// instead of duplicating it. It writes the 401/403 response itself on
// failure, so a caller can just `return` (after c.Abort()) when ok is
// false.
func authenticateServiceKey(c *gin.Context, serviceKeyService *service.ServiceKeyService, capability string) (*model.ServiceKey, []string, bool) {
	secret, ok := bearerToken(c)
	if !ok {
		apierror.Write(c, http.StatusUnauthorized, "unauthenticated", "authentication required")
		return nil, nil, false
	}

	key, envIDs, err := serviceKeyService.Authenticate(c.Request.Context(), secret)
	if err != nil {
		apierror.Write(c, http.StatusUnauthorized, "unauthenticated", "service key is invalid or inactive")
		return nil, nil, false
	}

	switch capability {
	case "read":
		if !key.CanRead {
			apierror.Write(c, http.StatusForbidden, "forbidden", "you do not have permission to perform this action")
			return nil, nil, false
		}
	case "write":
		if !key.CanWrite {
			apierror.Write(c, http.StatusForbidden, "forbidden", "you do not have permission to perform this action")
			return nil, nil, false
		}
	}

	if !containsString(envIDs, c.Param("environmentId")) {
		apierror.Write(c, http.StatusForbidden, "forbidden", "you do not have permission to perform this action")
		return nil, nil, false
	}

	return key, envIDs, true
}

// RequireServiceKeyAccess authenticates the request via
// `Authorization: Bearer <secret>` against serviceKeyService, per
// docs/business/requirements/0007-feature-flag-evaluation-api.md. It
// aborts with 401 if the header is missing/malformed or the secret doesn't
// match an Active key, and with 403 if the key lacks the given capability
// ("read" or "write") or isn't linked to the environment named by the
// route's :environmentId param. On success, the key and its environment
// ids are available via CurrentServiceKey / CurrentServiceKeyEnvironmentIDs.
func RequireServiceKeyAccess(serviceKeyService *service.ServiceKeyService, capability string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key, envIDs, ok := authenticateServiceKey(c, serviceKeyService, capability)
		if !ok {
			c.Abort()
			return
		}
		c.Set(currentServiceKeyKey, key)
		c.Set(currentServiceKeyEnvironmentIDsKey, envIDs)
		c.Next()
	}
}

// RequireWriteAccess gates a write route (resource, action) that either a
// User (session cookie) or a Service Key with can_write (Bearer token) may
// call, per 0007 ("A key with can_write enabled may also use its
// credentials against the write actions"). If a session cookie is present
// on the request, it's authenticated the same way RequireAuth does, then
// checked against (resource, action) the same way RequirePermission does —
// 401/403 on failure, CurrentUser set on success. If no session cookie is
// present at all, it falls back to the same Bearer-token flow as
// RequireServiceKeyAccess with capability "write" — CurrentServiceKey /
// CurrentServiceKeyEnvironmentIDs set on success. If neither credential is
// present, 401.
func RequireWriteAccess(authService *service.AuthService, authzService *service.AuthzService, serviceKeyService *service.ServiceKeyService, resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if token, err := c.Cookie(SessionCookieName); err == nil && token != "" {
			user, ok := authenticateSession(c, authService)
			if !ok {
				c.Abort()
				return
			}

			allowed, err := authzService.HasPermission(c.Request.Context(), user.ID, resource, action)
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

			c.Set(currentUserKey, user)
			c.Next()
			return
		}

		key, envIDs, ok := authenticateServiceKey(c, serviceKeyService, "write")
		if !ok {
			c.Abort()
			return
		}
		c.Set(currentServiceKeyKey, key)
		c.Set(currentServiceKeyEnvironmentIDsKey, envIDs)
		c.Next()
	}
}
