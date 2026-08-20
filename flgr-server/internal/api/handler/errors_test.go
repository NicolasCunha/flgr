package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/NicolasCunha/flgr/flgr-server/internal/service"
)

func TestRespondError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"validation", service.ErrValidation, http.StatusBadRequest, "validation_error"},
		{"email in use", service.ErrEmailInUse, http.StatusConflict, "conflict"},
		{"user not found", service.ErrUserNotFound, http.StatusNotFound, "not_found"},
		{"invalid credentials", service.ErrInvalidCredentials, http.StatusUnauthorized, "invalid_credentials"},
		{"account locked", service.ErrAccountLocked, http.StatusUnauthorized, "account_locked"},
		{"user inactive", service.ErrUserInactive, http.StatusUnauthorized, "user_inactive"},
		{"session invalid", service.ErrSessionInvalid, http.StatusUnauthorized, "unauthenticated"},
		{"forbidden", service.ErrForbidden, http.StatusForbidden, "forbidden"},
		{"environment in use", service.ErrEnvironmentInUse, http.StatusConflict, "conflict"},
		{"service key not found", service.ErrServiceKeyNotFound, http.StatusNotFound, "not_found"},
		{"feature flag not found", service.ErrFeatureFlagNotFound, http.StatusNotFound, "not_found"},
		{"feature flag key in use", service.ErrFeatureFlagKeyInUse, http.StatusConflict, "conflict"},
		{"feature flag in use", service.ErrFeatureFlagInUse, http.StatusConflict, "conflict"},
		{"unknown", errors.New("boom"), http.StatusInternalServerError, "internal_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)

			respondError(c, tt.err)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			var body struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("unmarshaling response body: %v", err)
			}
			if body.Error.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", body.Error.Code, tt.wantCode)
			}
		})
	}
}
