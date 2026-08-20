package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/NicolasCunha/flgr/flgr-server/internal/service"
)

// EvaluationHandler exposes the read (pull) path from
// docs/business/requirements/0007-feature-flag-evaluation-api.md, reached
// via Service Key (Bearer) authentication — see
// middleware.RequireServiceKeyAccess — rather than a session, and gated by
// ServiceKey.CanRead instead of a User permission.
type EvaluationHandler struct {
	evaluations *service.EvaluationService
}

// NewEvaluationHandler returns an EvaluationHandler backed by evaluations.
func NewEvaluationHandler(evaluations *service.EvaluationService) *EvaluationHandler {
	return &EvaluationHandler{evaluations: evaluations}
}

type evaluatedFlagResponse struct {
	Key     string  `json:"key"`
	Enabled bool    `json:"enabled"`
	Value   *string `json:"value"`
}

func toEvaluatedFlagResponse(f service.EvaluatedFlag) evaluatedFlagResponse {
	return evaluatedFlagResponse{Key: f.Key, Enabled: f.Enabled, Value: f.Value}
}

// parseKeys splits a comma-separated ?keys= query param into its
// individual, trimmed, non-empty entries. It returns nil (not an empty
// slice) when the param is absent or resolves to no entries, so
// EvaluationService.Evaluate can tell "no filter" apart from "filter to
// nothing".
func parseKeys(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	keys := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		keys = append(keys, p)
	}
	if len(keys) == 0 {
		return nil
	}
	return keys
}

// Evaluate handles GET /api/v1/environments/:environmentId/feature-flag-values
// — returns the evaluated state (per EvaluationService.Evaluate) of every
// flag named by the optional ?keys= filter, or every flag if it's absent.
func (h *EvaluationHandler) Evaluate(c *gin.Context) {
	keys := parseKeys(c.Query("keys"))

	flags, err := h.evaluations.Evaluate(c.Request.Context(), c.Param("environmentId"), keys)
	if err != nil {
		respondError(c, err)
		return
	}

	data := make([]evaluatedFlagResponse, len(flags))
	for i, f := range flags {
		data[i] = toEvaluatedFlagResponse(f)
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}
