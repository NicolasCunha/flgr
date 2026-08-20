package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/NicolasCunha/flgr/flgr-server/internal/api/apierror"
	"github.com/NicolasCunha/flgr/flgr-server/internal/api/middleware"
	"github.com/NicolasCunha/flgr/flgr-server/internal/service"
)

// actorFromContext resolves the service.Actor for the authenticated
// caller on a route guarded by middleware.RequireWriteAccess — a User
// (session cookie) or a ServiceKey (Bearer token), whichever is set. It
// returns the zero Actor (service.Actor{}) if neither is set, which every
// Actor-accepting service method rejects with service.ErrForbidden.
func actorFromContext(c *gin.Context) service.Actor {
	if u := middleware.CurrentUser(c); u != nil {
		return service.ActorFromUser(u)
	}
	if k := middleware.CurrentServiceKey(c); k != nil {
		return service.ActorFromServiceKey(k)
	}
	return service.Actor{}
}

// respondError maps a service-layer error to the ADR-0007 error envelope
// and HTTP status code, per backend.md's "translate errors to HTTP status
// codes in one place" convention — handlers call this instead of mapping
// errors ad hoc.
func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrValidation):
		apierror.Write(c, http.StatusBadRequest, "validation_error", err.Error())
	case errors.Is(err, service.ErrEmailInUse):
		apierror.Write(c, http.StatusConflict, "conflict", "email is already in use")
	case errors.Is(err, service.ErrSetupAlreadyComplete):
		apierror.Write(c, http.StatusConflict, "conflict", "initial setup has already been completed")
	case errors.Is(err, service.ErrUserNotFound):
		apierror.Write(c, http.StatusNotFound, "not_found", "user not found")
	case errors.Is(err, service.ErrProfileNotFound):
		apierror.Write(c, http.StatusNotFound, "not_found", "profile not found")
	case errors.Is(err, service.ErrProfileNameInUse):
		apierror.Write(c, http.StatusConflict, "conflict", "profile name is already in use")
	case errors.Is(err, service.ErrLastFullCatalogProfile):
		apierror.Write(c, http.StatusConflict, "conflict", "cannot remove the only profile holding the full permission catalog")
	case errors.Is(err, service.ErrEnvironmentNotFound):
		apierror.Write(c, http.StatusNotFound, "not_found", "environment not found")
	case errors.Is(err, service.ErrEnvironmentNameInUse):
		apierror.Write(c, http.StatusConflict, "conflict", "environment name is already in use")
	case errors.Is(err, service.ErrEnvironmentInUse):
		apierror.Write(c, http.StatusConflict, "conflict", "environment is still in use")
	case errors.Is(err, service.ErrServiceKeyNotFound):
		apierror.Write(c, http.StatusNotFound, "not_found", "service key not found")
	case errors.Is(err, service.ErrFeatureFlagNotFound):
		apierror.Write(c, http.StatusNotFound, "not_found", "feature flag not found")
	case errors.Is(err, service.ErrFeatureFlagKeyInUse):
		apierror.Write(c, http.StatusConflict, "conflict", "feature flag key is already in use")
	case errors.Is(err, service.ErrFeatureFlagInUse):
		apierror.Write(c, http.StatusConflict, "conflict", "feature flag is still in use")
	case errors.Is(err, service.ErrNotificationChannelNotFound):
		apierror.Write(c, http.StatusNotFound, "not_found", "notification channel not found")
	case errors.Is(err, service.ErrNotificationChannelInUse):
		apierror.Write(c, http.StatusConflict, "conflict", "notification channel is still in use")
	case errors.Is(err, service.ErrInvalidCredentials):
		apierror.Write(c, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
	case errors.Is(err, service.ErrAccountLocked):
		apierror.Write(c, http.StatusUnauthorized, "account_locked", "too many failed login attempts; try again later")
	case errors.Is(err, service.ErrUserInactive):
		apierror.Write(c, http.StatusUnauthorized, "user_inactive", "this account is inactive")
	case errors.Is(err, service.ErrSessionInvalid):
		apierror.Write(c, http.StatusUnauthorized, "unauthenticated", "session is invalid or expired")
	case errors.Is(err, service.ErrForbidden):
		apierror.Write(c, http.StatusForbidden, "forbidden", "you do not have permission to perform this action")
	default:
		apierror.Write(c, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
