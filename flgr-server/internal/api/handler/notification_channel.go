package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/NicolasCunha/flgr/flgr-server/internal/api/apierror"
	"github.com/NicolasCunha/flgr/flgr-server/internal/api/middleware"
	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/service"
)

// NotificationChannelHandler exposes the notification-channel CRUD
// sub-resource from docs/business/requirements/0006-feature-flag-killswitch.md.
// Every route is gated entirely by FeatureFlag: Edit, except List which
// only needs FeatureFlag: View, per the CRUD-implies-View rule every other
// resource follows — see router.go for the actual gating.
type NotificationChannelHandler struct {
	channels *service.NotificationChannelService
}

// NewNotificationChannelHandler returns a NotificationChannelHandler backed
// by channels.
func NewNotificationChannelHandler(channels *service.NotificationChannelService) *NotificationChannelHandler {
	return &NotificationChannelHandler{channels: channels}
}

type notificationChannelResponse struct {
	ID            string `json:"id"`
	FeatureFlagID string `json:"feature_flag_id"`
	ChannelType   string `json:"channel_type"`
	Destination   string `json:"destination"`
	Enabled       bool   `json:"enabled"`
	// SigningSecret is populated for HTTP channels only (omitted/"" for
	// Kafka) and is decrypted and returned on every read, not just
	// Create — per ADR-0009's explicit "viewable afterward" call-out,
	// which departs from the Service Key one-time-reveal convention (the
	// receiving system needs to know it to verify HMAC signatures).
	SigningSecret string    `json:"signing_secret,omitempty"`
	CreatedOn     time.Time `json:"created_on"`
	ModifiedOn    time.Time `json:"modified_on"`
}

func toNotificationChannelResponse(ch *model.NotificationChannel, secret string) notificationChannelResponse {
	return notificationChannelResponse{
		ID:            ch.ID,
		FeatureFlagID: ch.FeatureFlagID,
		ChannelType:   ch.ChannelType,
		Destination:   ch.Destination,
		Enabled:       ch.Enabled,
		SigningSecret: secret,
		CreatedOn:     ch.CreatedOn,
		ModifiedOn:    ch.ModifiedOn,
	}
}

type createNotificationChannelRequest struct {
	ChannelType string `json:"channel_type" binding:"required"`
	Destination string `json:"destination" binding:"required"`
}

// Create handles POST /api/v1/feature-flags/:id/notification-channels. For
// an HTTP channel, the response's signing_secret is the plaintext HMAC
// secret generated for it.
func (h *NotificationChannelHandler) Create(c *gin.Context) {
	var req createNotificationChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Write(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	actor := middleware.CurrentUser(c)
	ch, secret, err := h.channels.Create(c.Request.Context(), actor, c.Param("id"), service.CreateNotificationChannelInput{
		ChannelType: req.ChannelType,
		Destination: req.Destination,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toNotificationChannelResponse(ch, secret))
}

// List handles GET /api/v1/feature-flags/:id/notification-channels.
func (h *NotificationChannelHandler) List(c *gin.Context) {
	channels, secrets, err := h.channels.List(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}

	data := make([]notificationChannelResponse, len(channels))
	for i := range channels {
		data[i] = toNotificationChannelResponse(&channels[i], secrets[channels[i].ID])
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

type updateNotificationChannelRequest struct {
	Destination *string `json:"destination"`
	Enabled     *bool   `json:"enabled"`
}

// Update handles PATCH /api/v1/feature-flags/:id/notification-channels/:channelId.
// Regenerating the signing secret is out of scope for this phase — there is
// no field for it in the request.
func (h *NotificationChannelHandler) Update(c *gin.Context) {
	var req updateNotificationChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Write(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	actor := middleware.CurrentUser(c)
	ch, secret, err := h.channels.Update(c.Request.Context(), actor, c.Param("id"), c.Param("channelId"), service.UpdateNotificationChannelInput{
		Destination: req.Destination,
		Enabled:     req.Enabled,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, toNotificationChannelResponse(ch, secret))
}

// Delete handles DELETE /api/v1/feature-flags/:id/notification-channels/:channelId
// — a real delete, not a soft one: channel rows aren't referenced as an
// audit created_by/modified_by target anywhere, same reasoning as
// Environments/Profiles per ADR-0005.
func (h *NotificationChannelHandler) Delete(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	if err := h.channels.Delete(c.Request.Context(), actor, c.Param("id"), c.Param("channelId")); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
