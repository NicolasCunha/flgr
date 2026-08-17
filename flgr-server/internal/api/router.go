// Package api builds the Gin engine and registers routes under /api/v1,
// per docs/architecture/adr/0007-api-design-conventions.md.
package api

import (
	"github.com/gin-gonic/gin"

	"github.com/NicolasCunha/flgr/flgr-server/internal/api/handler"
)

// NewRouter builds the Gin engine with all v1 routes registered.
func NewRouter() *gin.Engine {
	router := gin.Default()

	v1 := router.Group("/api/v1")
	v1.GET("/health", handler.Health)

	return router
}
