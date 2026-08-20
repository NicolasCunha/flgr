// Package apierror writes the error envelope defined in
// docs/architecture/adr/0007-api-design-conventions.md, shared by
// handlers and middleware so the shape is defined in exactly one place.
package apierror

import "github.com/gin-gonic/gin"

// Write responds with the ADR-0007 error envelope:
//
//	{"error": {"code": "...", "message": "..."}}
func Write(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}
