// Package api builds the Gin engine and registers routes under /api/v1,
// per docs/architecture/adr/0007-api-design-conventions.md.
package api

import (
	"database/sql"
	"log"

	"github.com/gin-gonic/gin"

	"github.com/NicolasCunha/flgr/flgr-server/internal/api/handler"
	"github.com/NicolasCunha/flgr/flgr-server/internal/api/middleware"
	"github.com/NicolasCunha/flgr/flgr-server/internal/config"
	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/repository"
	"github.com/NicolasCunha/flgr/flgr-server/internal/service"
)

// NewRouter builds the Gin engine with all v1 routes registered, wiring
// the repository/service/handler layers per backend.md's Layering rule.
func NewRouter(db *sql.DB, cfg *config.Config) *gin.Engine {
	// FLGR_ENCRYPTION_KEY is validated for presence by config.Load, but not
	// for shape — decoding it into the 32-byte AES-256 key
	// NotificationChannelService needs happens here, failing fast (before
	// the server starts listening) if it's malformed, per ADR-0010's
	// fail-fast startup rule.
	encryptionKey, err := service.DecodeEncryptionKey(cfg.EncryptionKey)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	loginAttemptRepo := repository.NewLoginAttemptRepository(db)
	profileRepo := repository.NewProfileRepository(db)
	permissionRepo := repository.NewPermissionRepository(db)
	profilePermissionRepo := repository.NewProfilePermissionRepository(db)
	userProfileRepo := repository.NewUserProfileRepository(db)
	userPermissionRepo := repository.NewUserPermissionRepository(db)
	authzRepo := repository.NewAuthzRepository(db)
	environmentRepo := repository.NewEnvironmentRepository(db)
	environmentCategoryRepo := repository.NewEnvironmentCategoryRepository(db)
	serviceKeyRepo := repository.NewServiceKeyRepository(db)
	serviceKeyEnvironmentRepo := repository.NewServiceKeyEnvironmentRepository(db)
	featureFlagRepo := repository.NewFeatureFlagRepository(db)
	featureFlagEnvironmentValueRepo := repository.NewFeatureFlagEnvironmentValueRepository(db)
	notificationChannelRepo := repository.NewNotificationChannelRepository(db)
	auditLogRepo := repository.NewAuditLogRepository(db)
	notificationOutboxRepo := repository.NewNotificationOutboxRepository(db)

	authzService := service.NewAuthzService(authzRepo)
	userService := service.NewUserService(userRepo, authzService)
	authService := service.NewAuthService(userRepo, sessionRepo, loginAttemptRepo)
	profileService := service.NewProfileService(profileRepo, profilePermissionRepo, permissionRepo)
	permissionService := service.NewPermissionService(permissionRepo)
	userAccessService := service.NewUserAccessService(userRepo, userProfileRepo, userPermissionRepo, profileRepo, permissionRepo)
	environmentService := service.NewEnvironmentService(environmentRepo, environmentCategoryRepo)
	environmentCategoryService := service.NewEnvironmentCategoryService(environmentCategoryRepo)
	serviceKeyService := service.NewServiceKeyService(serviceKeyRepo, environmentRepo, serviceKeyEnvironmentRepo)
	featureFlagService := service.NewFeatureFlagService(featureFlagRepo)
	featureFlagValueService := service.NewFeatureFlagValueService(featureFlagRepo, environmentRepo, featureFlagEnvironmentValueRepo)
	notificationChannelService := service.NewNotificationChannelService(notificationChannelRepo, featureFlagRepo, encryptionKey)
	killswitchService := service.NewKillswitchService(featureFlagValueService, featureFlagRepo, environmentRepo, featureFlagEnvironmentValueRepo, notificationChannelRepo, auditLogRepo, notificationOutboxRepo)
	evaluationService := service.NewEvaluationService(featureFlagRepo, environmentRepo, featureFlagEnvironmentValueRepo)

	userHandler := handler.NewUserHandler(userService)
	authHandler := handler.NewAuthHandler(authService, cfg.SessionCookieSecure)
	setupHandler := handler.NewSetupHandler(userService, userAccessService)
	profileHandler := handler.NewProfileHandler(profileService)
	permissionHandler := handler.NewPermissionHandler(permissionService)
	userAccessHandler := handler.NewUserAccessHandler(userAccessService)
	environmentHandler := handler.NewEnvironmentHandler(environmentService)
	environmentCategoryHandler := handler.NewEnvironmentCategoryHandler(environmentCategoryService)
	serviceKeyHandler := handler.NewServiceKeyHandler(serviceKeyService)
	featureFlagHandler := handler.NewFeatureFlagHandler(featureFlagService)
	featureFlagValueHandler := handler.NewFeatureFlagValueHandler(featureFlagValueService)
	notificationChannelHandler := handler.NewNotificationChannelHandler(notificationChannelService)
	killswitchHandler := handler.NewKillswitchHandler(killswitchService)
	evaluationHandler := handler.NewEvaluationHandler(evaluationService)

	router := gin.Default()

	v1 := router.Group("/api/v1")
	v1.GET("/health", handler.Health)
	v1.GET("/setup", setupHandler.Status)
	v1.POST("/setup", setupHandler.Complete)
	v1.POST("/login", authHandler.Login)

	authorized := v1.Group("")
	authorized.Use(middleware.RequireAuth(authService))
	{
		authorized.POST("/logout", authHandler.Logout)
		authorized.GET("/me", authHandler.Me)

		authorized.GET("/permissions", permissionHandler.List)

		// Users: Get/Update have no static permission requirement — a
		// user may always act on their own record, and the "other user"
		// branch (User: View / User: Edit) is checked inside
		// service.UserService, which is why only Create/List/Deactivate
		// are gated by RequirePermission here (see their handler doc
		// comments).
		authorized.POST("/users", middleware.RequirePermission(authzService, model.ResourceUser, model.ActionCreate), userHandler.Create)
		authorized.GET("/users", middleware.RequirePermission(authzService, model.ResourceUser, model.ActionView), userHandler.List)
		authorized.GET("/users/:id", userHandler.Get)
		authorized.PATCH("/users/:id", userHandler.Update)
		authorized.DELETE("/users/:id", middleware.RequirePermission(authzService, model.ResourceUser, model.ActionRemove), userHandler.Deactivate)

		userEdit := authorized.Group("/users/:id")
		userEdit.Use(middleware.RequirePermission(authzService, model.ResourceUser, model.ActionEdit))
		{
			userEdit.GET("/profiles", userAccessHandler.GetProfiles)
			userEdit.PATCH("/profiles", userAccessHandler.ReplaceProfiles)
			userEdit.GET("/permissions/direct", userAccessHandler.GetDirectPermissions)
			userEdit.PATCH("/permissions/direct", userAccessHandler.ReplaceDirectPermissions)
		}

		authorized.POST("/profiles", middleware.RequirePermission(authzService, model.ResourceProfile, model.ActionCreate), profileHandler.Create)
		authorized.GET("/profiles", middleware.RequirePermission(authzService, model.ResourceProfile, model.ActionView), profileHandler.List)
		authorized.GET("/profiles/:id", middleware.RequirePermission(authzService, model.ResourceProfile, model.ActionView), profileHandler.Get)
		authorized.GET("/profiles/:id/permissions", middleware.RequirePermission(authzService, model.ResourceProfile, model.ActionView), profileHandler.PermissionIDs)
		authorized.PATCH("/profiles/:id", middleware.RequirePermission(authzService, model.ResourceProfile, model.ActionEdit), profileHandler.Update)
		authorized.DELETE("/profiles/:id", middleware.RequirePermission(authzService, model.ResourceProfile, model.ActionRemove), profileHandler.Delete)

		// environment-categories has no dedicated permission in 0003's
		// catalog — gated by Environment: View, same as environments.
		authorized.GET("/environment-categories", middleware.RequirePermission(authzService, model.ResourceEnvironment, model.ActionView), environmentCategoryHandler.List)

		authorized.POST("/environments", middleware.RequirePermission(authzService, model.ResourceEnvironment, model.ActionCreate), environmentHandler.Create)
		authorized.GET("/environments", middleware.RequirePermission(authzService, model.ResourceEnvironment, model.ActionView), environmentHandler.List)
		authorized.GET("/environments/:id", middleware.RequirePermission(authzService, model.ResourceEnvironment, model.ActionView), environmentHandler.Get)
		authorized.PATCH("/environments/:id", middleware.RequirePermission(authzService, model.ResourceEnvironment, model.ActionEdit), environmentHandler.Update)
		authorized.DELETE("/environments/:id", middleware.RequirePermission(authzService, model.ResourceEnvironment, model.ActionRemove), environmentHandler.Delete)

		authorized.POST("/service-keys", middleware.RequirePermission(authzService, model.ResourceServiceKey, model.ActionCreate), serviceKeyHandler.Create)
		authorized.GET("/service-keys", middleware.RequirePermission(authzService, model.ResourceServiceKey, model.ActionView), serviceKeyHandler.List)
		authorized.GET("/service-keys/:id", middleware.RequirePermission(authzService, model.ResourceServiceKey, model.ActionView), serviceKeyHandler.Get)
		authorized.PATCH("/service-keys/:id", middleware.RequirePermission(authzService, model.ResourceServiceKey, model.ActionEdit), serviceKeyHandler.Update)
		authorized.DELETE("/service-keys/:id", middleware.RequirePermission(authzService, model.ResourceServiceKey, model.ActionRemove), serviceKeyHandler.Deactivate)

		authorized.POST("/feature-flags", middleware.RequirePermission(authzService, model.ResourceFeatureFlag, model.ActionCreate), featureFlagHandler.Create)
		authorized.GET("/feature-flags", middleware.RequirePermission(authzService, model.ResourceFeatureFlag, model.ActionView), featureFlagHandler.List)
		authorized.GET("/feature-flags/:id", middleware.RequirePermission(authzService, model.ResourceFeatureFlag, model.ActionView), featureFlagHandler.Get)
		authorized.PATCH("/feature-flags/:id", middleware.RequirePermission(authzService, model.ResourceFeatureFlag, model.ActionEdit), featureFlagHandler.Update)
		authorized.DELETE("/feature-flags/:id", middleware.RequirePermission(authzService, model.ResourceFeatureFlag, model.ActionRemove), featureFlagHandler.Delete)

		// Per-environment values are a separate permission surface
		// (FeatureFlagValue: Read/Write) from the flag definition itself
		// (FeatureFlag: Create/Edit/Remove/View) — 0003's CRUD-implies-View
		// rule doesn't cover Read/Write, so these need their own gates.
		authorized.GET("/feature-flags/:id/values", middleware.RequirePermission(authzService, model.ResourceFeatureFlagValue, model.ActionRead), featureFlagValueHandler.List)

		// Notification channels are a flag-level (not per-environment)
		// sub-resource, gated entirely by FeatureFlag: Edit per 0006 — List
		// only needs FeatureFlag: View, the same CRUD-implies-View rule
		// every other resource follows.
		authorized.POST("/feature-flags/:id/notification-channels", middleware.RequirePermission(authzService, model.ResourceFeatureFlag, model.ActionEdit), notificationChannelHandler.Create)
		authorized.GET("/feature-flags/:id/notification-channels", middleware.RequirePermission(authzService, model.ResourceFeatureFlag, model.ActionView), notificationChannelHandler.List)
		authorized.PATCH("/feature-flags/:id/notification-channels/:channelId", middleware.RequirePermission(authzService, model.ResourceFeatureFlag, model.ActionEdit), notificationChannelHandler.Update)
		authorized.DELETE("/feature-flags/:id/notification-channels/:channelId", middleware.RequirePermission(authzService, model.ResourceFeatureFlag, model.ActionEdit), notificationChannelHandler.Delete)
	}

	// These routes live on v1 directly, not under authorized — authorized
	// unconditionally requires a session cookie (middleware.RequireAuth),
	// which would break the Bearer-token fallback a can_write Service Key
	// needs for the two write routes, and the evaluation route (read-only)
	// is Bearer-only to begin with, per
	// docs/business/requirements/0007-feature-flag-evaluation-api.md.
	//
	// The Killswitch is just another write to a flag's per-environment
	// value, so it reuses FeatureFlagValue: Write, per
	// docs/business/requirements/0006-feature-flag-killswitch.md.
	v1.PATCH("/feature-flags/:id/values/:environmentId", middleware.RequireWriteAccess(authService, authzService, serviceKeyService, model.ResourceFeatureFlagValue, model.ActionWrite), featureFlagValueHandler.Upsert)
	v1.POST("/feature-flags/:id/values/:environmentId/killswitch", middleware.RequireWriteAccess(authService, authzService, serviceKeyService, model.ResourceFeatureFlagValue, model.ActionWrite), killswitchHandler.Trigger)
	v1.GET("/environments/:environmentId/feature-flag-values", middleware.RequireServiceKeyAccess(serviceKeyService, "read"), evaluationHandler.Evaluate)

	return router
}
