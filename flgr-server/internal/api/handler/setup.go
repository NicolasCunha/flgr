package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/NicolasCunha/flgr/flgr-server/internal/api/apierror"
	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/service"
)

// SetupHandler exposes the first-run setup wizard: whether it's needed,
// and creating the first User. Both routes are public (no RequireAuth) —
// by definition, nobody can be authenticated yet on a fresh install. See
// UserService.SetupNeeded / .CompleteSetup for the one-time guard.
type SetupHandler struct {
	users  *service.UserService
	access *service.UserAccessService
}

// NewSetupHandler returns a SetupHandler backed by users and access.
// access is used to assign the new administrator the seeded Administrador
// profile (see Complete) — without it, the wizard would create a User
// with no permissions at all, unable to do anything once logged in.
func NewSetupHandler(users *service.UserService, access *service.UserAccessService) *SetupHandler {
	return &SetupHandler{users: users, access: access}
}

type setupStatusResponse struct {
	Needed bool `json:"needed"`
}

// Status handles GET /api/v1/setup — whether the frontend should show the
// setup wizard instead of the normal app.
func (h *SetupHandler) Status(c *gin.Context) {
	needed, err := h.users.SetupNeeded(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, setupStatusResponse{Needed: needed})
}

type completeSetupRequest struct {
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
	Email     string `json:"email" binding:"required"`
	Password  string `json:"password" binding:"required"`
}

// Complete handles POST /api/v1/setup — creates the first User and
// assigns them the seeded Administrador profile (model.AdministradorProfileID),
// so they can actually administer the instance once logged in. Fails
// with 409 if setup was already completed (see CompleteSetup).
func (h *SetupHandler) Complete(c *gin.Context) {
	var req completeSetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Write(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	u, err := h.users.CompleteSetup(c.Request.Context(), service.CreateUserInput{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		Password:  req.Password,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	// The new admin is their own actor here — nobody else exists yet to
	// have granted them this profile.
	if err := h.access.ReplaceProfiles(c.Request.Context(), u, u.ID, []string{model.AdministradorProfileID}); err != nil {
		respondError(c, fmt.Errorf("assigning administrator profile: %w", err))
		return
	}

	c.JSON(http.StatusCreated, toUserResponse(u))
}
