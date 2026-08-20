package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/NicolasCunha/flgr/flgr-server/internal/api/apierror"
	"github.com/NicolasCunha/flgr/flgr-server/internal/api/middleware"
	"github.com/NicolasCunha/flgr/flgr-server/internal/service"
)

// AuthHandler exposes login/logout/me over HTTP, per
// docs/architecture/adr/0006-authentication-and-session-strategy.md.
type AuthHandler struct {
	auth         *service.AuthService
	secureCookie bool
}

// NewAuthHandler returns an AuthHandler backed by auth. secureCookie sets
// the session cookie's Secure attribute — false only for local HTTP
// development, per FLGR_SESSION_COOKIE_SECURE (docs/architecture/adr/0010).
func NewAuthHandler(auth *service.AuthService, secureCookie bool) *AuthHandler {
	return &AuthHandler{auth: auth, secureCookie: secureCookie}
}

type loginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login handles POST /api/v1/login.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Write(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	user, token, err := h.auth.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		respondError(c, err)
		return
	}

	h.setSessionCookie(c, token)
	c.JSON(http.StatusOK, toUserResponse(user))
}

// Logout handles POST /api/v1/logout.
func (h *AuthHandler) Logout(c *gin.Context) {
	token, err := c.Cookie(middleware.SessionCookieName)
	if err == nil && token != "" {
		if err := h.auth.Logout(c.Request.Context(), token); err != nil {
			respondError(c, err)
			return
		}
	}

	h.clearSessionCookie(c)
	c.Status(http.StatusOK)
}

// Me handles GET /api/v1/me, returning the caller's own user record.
func (h *AuthHandler) Me(c *gin.Context) {
	user := middleware.CurrentUser(c)
	c.JSON(http.StatusOK, toUserResponse(user))
}

func (h *AuthHandler) setSessionCookie(c *gin.Context, token string) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(middleware.SessionCookieName, token, 0, "/", "", h.secureCookie, true)
}

func (h *AuthHandler) clearSessionCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(middleware.SessionCookieName, "", -1, "/", "", h.secureCookie, true)
}
