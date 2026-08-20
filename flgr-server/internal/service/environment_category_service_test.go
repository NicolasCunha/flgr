package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
)

func TestEnvironmentCategoryService_List(t *testing.T) {
	repo := new(mockEnvironmentCategoryRepository)
	repo.On("ListAll", mock.Anything).Return([]model.EnvironmentCategory{{ID: "c1", Name: "Development", IsSystem: true}}, nil)

	svc := NewEnvironmentCategoryService(repo)
	categories, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() returned unexpected error: %v", err)
	}
	if len(categories) != 1 {
		t.Errorf("len(categories) = %d, want 1", len(categories))
	}
}

func TestEnvironmentCategoryService_List_RepositoryError(t *testing.T) {
	repo := new(mockEnvironmentCategoryRepository)
	repo.On("ListAll", mock.Anything).Return(nil, errors.New("db down"))

	svc := NewEnvironmentCategoryService(repo)
	if _, err := svc.List(context.Background()); err == nil {
		t.Fatal("List() expected an error, got nil")
	}
}
