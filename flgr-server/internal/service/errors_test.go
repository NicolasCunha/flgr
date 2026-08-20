package service

import (
	"errors"
	"testing"
)

func TestValidationError_ErrorAndIs(t *testing.T) {
	err := validationErrorf("email is required")

	if got, want := err.Error(), "email is required"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
	if !errors.Is(err, ErrValidation) {
		t.Error("errors.Is(err, ErrValidation) = false, want true")
	}
}
