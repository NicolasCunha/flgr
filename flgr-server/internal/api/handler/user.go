package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/NicolasCunha/flgr/flgr-server/internal/api/apierror"
	"github.com/NicolasCunha/flgr/flgr-server/internal/api/middleware"
	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/service"
)

// UserHandler exposes docs/business/requirements/0002-user-management.md
// over HTTP, per docs/architecture/adr/0007-api-design-conventions.md.
type UserHandler struct {
	users *service.UserService
}

// NewUserHandler returns a UserHandler backed by users.
func NewUserHandler(users *service.UserService) *UserHandler {
	return &UserHandler{users: users}
}

type userResponse struct {
	ID         string    `json:"id"`
	FirstName  string    `json:"first_name"`
	LastName   string    `json:"last_name"`
	Email      string    `json:"email"`
	Status     string    `json:"status"`
	CreatedOn  time.Time `json:"created_on"`
	ModifiedOn time.Time `json:"modified_on"`
}

func toUserResponse(u *model.User) userResponse {
	return userResponse{
		ID:         u.ID,
		FirstName:  u.FirstName,
		LastName:   u.LastName,
		Email:      u.Email,
		Status:     u.Status,
		CreatedOn:  u.CreatedOn,
		ModifiedOn: u.ModifiedOn,
	}
}

type createUserRequest struct {
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
	Email     string `json:"email" binding:"required"`
	Password  string `json:"password" binding:"required"`
}

// Create handles POST /api/v1/users. The "User: Create" permission this
// requires is enforced by route middleware (see router.go), not here.
func (h *UserHandler) Create(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Write(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	actor := middleware.CurrentUser(c)
	u, err := h.users.Create(c.Request.Context(), actor, service.CreateUserInput{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		Password:  req.Password,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toUserResponse(u))
}

// Get handles GET /api/v1/users/:id.
func (h *UserHandler) Get(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	u, err := h.users.Get(c.Request.Context(), actor, c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, toUserResponse(u))
}

type paginationResponse struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

// List handles GET /api/v1/users, per the list envelope in ADR-0007. The
// "User: View" permission this requires is enforced by route middleware.
func (h *UserHandler) List(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 50)

	users, total, err := h.users.List(c.Request.Context(), page, pageSize)
	if err != nil {
		respondError(c, err)
		return
	}

	data := make([]userResponse, len(users))
	for i, u := range users {
		data[i] = toUserResponse(&u)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       data,
		"pagination": paginationResponse{Page: page, PageSize: pageSize, Total: total},
	})
}

func queryInt(c *gin.Context, key string, def int) int {
	raw := c.Query(key)
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return v
}

type updateUserRequest struct {
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
	Email     *string `json:"email"`
	Password  *string `json:"password"`
	Status    *string `json:"status"`
}

// Update handles PATCH /api/v1/users/:id. A user may always edit their
// own record; editing another's requires the "User: Edit" permission —
// both enforced inside service.UserService.Update, since routing can't
// express "self, or has this permission" as a static check.
func (h *UserHandler) Update(c *gin.Context) {
	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Write(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	actor := middleware.CurrentUser(c)
	u, err := h.users.Update(c.Request.Context(), actor, c.Param("id"), service.UpdateUserInput{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		Password:  req.Password,
		Status:    req.Status,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, toUserResponse(u))
}

// Deactivate handles DELETE /api/v1/users/:id — a soft delete (status set
// to Inactive), per docs/architecture/adr/0005-audit-columns-and-soft-delete-convention.md
// and the DELETE-maps-to-deactivation convention in ADR-0007. The
// "User: Remove" permission this requires is enforced by route middleware.
func (h *UserHandler) Deactivate(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	u, err := h.users.Deactivate(c.Request.Context(), actor, c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, toUserResponse(u))
}
