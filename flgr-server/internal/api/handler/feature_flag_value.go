package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/NicolasCunha/flgr/flgr-server/internal/api/apierror"
	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/service"
)

// FeatureFlagValueHandler exposes the per-environment value sub-resource
// from docs/business/requirements/0005-feature-flag-management.md, gated
// by FeatureFlagValue: Read/Write — a separate permission surface from
// FeatureFlagHandler's FeatureFlag: Create/Edit/Remove/View (see
// service.FeatureFlagValueService's doc comment for why).
type FeatureFlagValueHandler struct {
	values *service.FeatureFlagValueService
}

// NewFeatureFlagValueHandler returns a FeatureFlagValueHandler backed by values.
func NewFeatureFlagValueHandler(values *service.FeatureFlagValueService) *FeatureFlagValueHandler {
	return &FeatureFlagValueHandler{values: values}
}

type featureFlagValueResponse struct {
	ID            string    `json:"id"`
	FeatureFlagID string    `json:"feature_flag_id"`
	EnvironmentID string    `json:"environment_id"`
	Enabled       bool      `json:"enabled"`
	Value         *string   `json:"value"`
	CreatedOn     time.Time `json:"created_on"`
	ModifiedOn    time.Time `json:"modified_on"`
}

func toFeatureFlagValueResponse(v *model.FeatureFlagEnvironmentValue) featureFlagValueResponse {
	return featureFlagValueResponse{
		ID:            v.ID,
		FeatureFlagID: v.FeatureFlagID,
		EnvironmentID: v.EnvironmentID,
		Enabled:       v.Enabled,
		Value:         v.Value,
		CreatedOn:     v.CreatedOn,
		ModifiedOn:    v.ModifiedOn,
	}
}

// List handles GET /api/v1/feature-flags/:id/values. Only configured
// (explicit-row) values are returned — an environment absent from the list
// evaluates as disabled, per 0005; not returned as a zero-value entry.
func (h *FeatureFlagValueHandler) List(c *gin.Context) {
	values, err := h.values.List(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}

	data := make([]featureFlagValueResponse, len(values))
	for i, v := range values {
		data[i] = toFeatureFlagValueResponse(&v)
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

type upsertFeatureFlagValueRequest struct {
	Enabled bool    `json:"enabled"`
	Value   *string `json:"value"`
}

// Upsert handles PATCH /api/v1/feature-flags/:id/values/:environmentId —
// sets whether the flag is enabled in that environment and, for non-Boolean
// types, its value; creates the row on first write, updates it thereafter.
func (h *FeatureFlagValueHandler) Upsert(c *gin.Context) {
	var req upsertFeatureFlagValueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Write(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	actor := actorFromContext(c)
	v, err := h.values.Upsert(c.Request.Context(), actor, c.Param("id"), c.Param("environmentId"), service.UpsertValueInput{
		Enabled: req.Enabled,
		Value:   req.Value,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, toFeatureFlagValueResponse(v))
}
