package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/NicolasCunha/flgr/flgr-server/internal/service"
)

// KillswitchHandler exposes the Killswitch trigger from
// docs/business/requirements/0006-feature-flag-killswitch.md, gated by the
// same FeatureFlagValue: Write permission as FeatureFlagValueHandler.Upsert
// (see router.go). It's a separate handler type — not a method on
// FeatureFlagValueHandler — because it's backed by a separate service
// (service.KillswitchService, which also touches notification channels and
// the audit log, not just the value repository FeatureFlagValueService
// wraps).
type KillswitchHandler struct {
	killswitch *service.KillswitchService
}

// NewKillswitchHandler returns a KillswitchHandler backed by killswitch.
func NewKillswitchHandler(killswitch *service.KillswitchService) *KillswitchHandler {
	return &KillswitchHandler{killswitch: killswitch}
}

// Trigger handles POST /api/v1/feature-flags/:id/values/:environmentId/killswitch
// — no request body, since it's an action rather than a data update. It
// responds with the same shape as FeatureFlagValueHandler.Upsert
// (toFeatureFlagValueResponse) so the frontend can update its UI the same
// way either action responds.
func (h *KillswitchHandler) Trigger(c *gin.Context) {
	actor := actorFromContext(c)
	v, err := h.killswitch.Trigger(c.Request.Context(), actor, c.Param("id"), c.Param("environmentId"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, toFeatureFlagValueResponse(v))
}
