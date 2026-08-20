package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
)

func TestPermissionService_List(t *testing.T) {
	repo := new(mockPermissionRepository)
	repo.On("List", mock.Anything).Return(testCatalog, nil)

	svc := NewPermissionService(repo)
	perms, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() returned unexpected error: %v", err)
	}
	if len(perms) != len(testCatalog) {
		t.Errorf("len(perms) = %d, want %d", len(perms), len(testCatalog))
	}
}

func TestPermissionService_List_RepositoryError(t *testing.T) {
	repo := new(mockPermissionRepository)
	repo.On("List", mock.Anything).Return(nil, errors.New("db down"))

	svc := NewPermissionService(repo)
	if _, err := svc.List(context.Background()); err == nil {
		t.Fatal("List() expected an error, got nil")
	}
}
