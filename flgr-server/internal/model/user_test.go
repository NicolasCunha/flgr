package model

import "testing"

func TestUser_IsActive(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{UserStatusActive, true},
		{UserStatusInactive, false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			u := User{Status: tt.status}
			if got := u.IsActive(); got != tt.want {
				t.Errorf("IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}
