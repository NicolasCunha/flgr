package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
)

func TestAuthzService_EffectivePermissions(t *testing.T) {
	repo := new(mockAuthzRepository)
	want := []model.Permission{{ID: "user:view", Resource: "User", Action: "View"}}
	repo.On("EffectivePermissions", mock.Anything, "u1").Return(want, nil)

	svc := NewAuthzService(repo)
	got, err := svc.EffectivePermissions(context.Background(), "u1")
	if err != nil {
		t.Fatalf("EffectivePermissions() returned unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != "user:view" {
		t.Errorf("EffectivePermissions() = %v, want %v", got, want)
	}
}

func TestAuthzService_EffectivePermissions_RepositoryError(t *testing.T) {
	repo := new(mockAuthzRepository)
	repo.On("EffectivePermissions", mock.Anything, "u1").Return(nil, errors.New("db down"))

	svc := NewAuthzService(repo)
	if _, err := svc.EffectivePermissions(context.Background(), "u1"); err == nil {
		t.Fatal("EffectivePermissions() expected an error, got nil")
	}
}

func TestAuthzService_HasPermission(t *testing.T) {
	tests := []struct {
		name     string
		perms    []model.Permission
		resource string
		action   string
		want     bool
	}{
		{
			name:     "exact match",
			perms:    []model.Permission{{ID: "user:view", Resource: "User", Action: "View"}},
			resource: "User", action: "View", want: true,
		},
		{
			name:     "no match at all",
			perms:    []model.Permission{{ID: "user:view", Resource: "User", Action: "View"}},
			resource: "Environment", action: "View", want: false,
		},
		{
			name:     "create implies view",
			perms:    []model.Permission{{ID: "user:create", Resource: "User", Action: "Create"}},
			resource: "User", action: "View", want: true,
		},
		{
			name:     "edit implies view",
			perms:    []model.Permission{{ID: "user:edit", Resource: "User", Action: "Edit"}},
			resource: "User", action: "View", want: true,
		},
		{
			name:     "remove implies view",
			perms:    []model.Permission{{ID: "user:remove", Resource: "User", Action: "Remove"}},
			resource: "User", action: "View", want: true,
		},
		{
			name:     "view does not imply create",
			perms:    []model.Permission{{ID: "user:view", Resource: "User", Action: "View"}},
			resource: "User", action: "Create", want: false,
		},
		{
			name:     "no permissions at all",
			perms:    nil,
			resource: "User", action: "View", want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockAuthzRepository)
			repo.On("EffectivePermissions", mock.Anything, "u1").Return(tt.perms, nil)

			svc := NewAuthzService(repo)
			got, err := svc.HasPermission(context.Background(), "u1", tt.resource, tt.action)
			if err != nil {
				t.Fatalf("HasPermission() returned unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("HasPermission(%s, %s) = %v, want %v", tt.resource, tt.action, got, tt.want)
			}
		})
	}
}

func TestAuthzService_HasPermission_RepositoryError(t *testing.T) {
	repo := new(mockAuthzRepository)
	repo.On("EffectivePermissions", mock.Anything, "u1").Return(nil, errors.New("db down"))

	svc := NewAuthzService(repo)
	if _, err := svc.HasPermission(context.Background(), "u1", "User", "View"); err == nil {
		t.Fatal("HasPermission() expected an error, got nil")
	}
}
