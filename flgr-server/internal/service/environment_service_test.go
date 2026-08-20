package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/repository"
)

func newEnvironmentServiceMocks() (*mockEnvironmentRepository, *mockEnvironmentCategoryRepository) {
	return new(mockEnvironmentRepository), new(mockEnvironmentCategoryRepository)
}

func TestEnvironmentService_Create_ExistingCategory(t *testing.T) {
	fixedNow(t)
	fixedID(t, "env-1")
	environments, categories := newEnvironmentServiceMocks()
	categories.On("GetByName", mock.Anything, "Development").Return(&model.EnvironmentCategory{ID: "cat-dev"}, nil)
	environments.On("GetByName", mock.Anything, "app-dev").Return(nil, repository.ErrNotFound)
	environments.On("Create", mock.Anything, mock.MatchedBy(func(e *model.Environment) bool {
		return e.ID == "env-1" && e.Name == "app-dev" && e.EnvironmentCategoryID == "cat-dev"
	})).Return(nil)

	svc := NewEnvironmentService(environments, categories)
	actor := &model.User{ID: "actor-id"}

	e, err := svc.Create(context.Background(), actor, CreateEnvironmentInput{Name: "app-dev", CategoryName: "Development"})
	if err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}
	if e.EnvironmentCategoryID != "cat-dev" {
		t.Errorf("EnvironmentCategoryID = %q, want %q", e.EnvironmentCategoryID, "cat-dev")
	}
	categories.AssertNotCalled(t, "Create")
}

func TestEnvironmentService_Create_NewCategory(t *testing.T) {
	fixedNow(t)
	fixedID(t, "env-1")
	environments, categories := newEnvironmentServiceMocks()
	categories.On("GetByName", mock.Anything, "Sandbox").Return(nil, repository.ErrNotFound)
	categories.On("Create", mock.Anything, mock.MatchedBy(func(c *model.EnvironmentCategory) bool {
		return c.Name == "Sandbox" && !c.IsSystem
	})).Return(nil)
	environments.On("GetByName", mock.Anything, "app-sandbox").Return(nil, repository.ErrNotFound)
	environments.On("Create", mock.Anything, mock.Anything).Return(nil)

	svc := NewEnvironmentService(environments, categories)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Create(context.Background(), actor, CreateEnvironmentInput{Name: "app-sandbox", CategoryName: "Sandbox"})
	if err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}
}

func TestEnvironmentService_Create_NewCategory_RaceRefetches(t *testing.T) {
	fixedNow(t)
	fixedID(t, "env-1")
	environments, categories := newEnvironmentServiceMocks()
	categories.On("GetByName", mock.Anything, "Sandbox").Once().Return(nil, repository.ErrNotFound)
	categories.On("Create", mock.Anything, mock.Anything).Return(repository.ErrEnvironmentCategoryNameInUse)
	categories.On("GetByName", mock.Anything, "Sandbox").Once().Return(&model.EnvironmentCategory{ID: "cat-sandbox"}, nil)
	environments.On("GetByName", mock.Anything, "app-sandbox").Return(nil, repository.ErrNotFound)
	environments.On("Create", mock.Anything, mock.MatchedBy(func(e *model.Environment) bool {
		return e.EnvironmentCategoryID == "cat-sandbox"
	})).Return(nil)

	svc := NewEnvironmentService(environments, categories)
	actor := &model.User{ID: "actor-id"}

	e, err := svc.Create(context.Background(), actor, CreateEnvironmentInput{Name: "app-sandbox", CategoryName: "Sandbox"})
	if err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}
	if e.EnvironmentCategoryID != "cat-sandbox" {
		t.Errorf("EnvironmentCategoryID = %q, want %q", e.EnvironmentCategoryID, "cat-sandbox")
	}
}

func TestEnvironmentService_Create_NewCategory_RaceRefetchError(t *testing.T) {
	environments, categories := newEnvironmentServiceMocks()
	categories.On("GetByName", mock.Anything, "Sandbox").Once().Return(nil, repository.ErrNotFound)
	categories.On("Create", mock.Anything, mock.Anything).Return(repository.ErrEnvironmentCategoryNameInUse)
	categories.On("GetByName", mock.Anything, "Sandbox").Once().Return(nil, errors.New("db down"))

	svc := NewEnvironmentService(environments, categories)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Create(context.Background(), actor, CreateEnvironmentInput{Name: "app-sandbox", CategoryName: "Sandbox"})
	if err == nil {
		t.Fatal("Create() expected an error, got nil")
	}
	environments.AssertNotCalled(t, "Create")
}

func TestEnvironmentService_Create_CategoryRepositoryError(t *testing.T) {
	environments, categories := newEnvironmentServiceMocks()
	categories.On("Create", mock.Anything, mock.Anything).Return(errors.New("disk full"))
	categories.On("GetByName", mock.Anything, "Sandbox").Return(nil, repository.ErrNotFound)

	svc := NewEnvironmentService(environments, categories)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Create(context.Background(), actor, CreateEnvironmentInput{Name: "app-sandbox", CategoryName: "Sandbox"})
	if err == nil {
		t.Fatal("Create() expected an error, got nil")
	}
}

func TestEnvironmentService_Create_CategoryLookupError(t *testing.T) {
	environments, categories := newEnvironmentServiceMocks()
	categories.On("GetByName", mock.Anything, "Sandbox").Return(nil, errors.New("db down"))

	svc := NewEnvironmentService(environments, categories)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Create(context.Background(), actor, CreateEnvironmentInput{Name: "app-sandbox", CategoryName: "Sandbox"})
	if err == nil {
		t.Fatal("Create() expected an error, got nil")
	}
}

func TestEnvironmentService_Create_NilActor(t *testing.T) {
	environments, categories := newEnvironmentServiceMocks()
	svc := NewEnvironmentService(environments, categories)

	_, err := svc.Create(context.Background(), nil, CreateEnvironmentInput{Name: "x", CategoryName: "Development"})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Create() error = %v, want ErrForbidden", err)
	}
}

func TestEnvironmentService_Create_BlankName(t *testing.T) {
	environments, categories := newEnvironmentServiceMocks()
	svc := NewEnvironmentService(environments, categories)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Create(context.Background(), actor, CreateEnvironmentInput{Name: "  ", CategoryName: "Development"})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Create() error = %v, want ErrValidation", err)
	}
}

func TestEnvironmentService_Create_BlankCategory(t *testing.T) {
	environments, categories := newEnvironmentServiceMocks()
	svc := NewEnvironmentService(environments, categories)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Create(context.Background(), actor, CreateEnvironmentInput{Name: "x", CategoryName: "  "})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Create() error = %v, want ErrValidation", err)
	}
	environments.AssertNotCalled(t, "GetByName")
}

func TestEnvironmentService_Create_NameAlreadyInUse(t *testing.T) {
	environments, categories := newEnvironmentServiceMocks()
	categories.On("GetByName", mock.Anything, "Development").Return(&model.EnvironmentCategory{ID: "cat-dev"}, nil)
	environments.On("GetByName", mock.Anything, "taken").Return(&model.Environment{ID: "existing"}, nil)

	svc := NewEnvironmentService(environments, categories)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Create(context.Background(), actor, CreateEnvironmentInput{Name: "taken", CategoryName: "Development"})
	if !errors.Is(err, ErrEnvironmentNameInUse) {
		t.Fatalf("Create() error = %v, want ErrEnvironmentNameInUse", err)
	}
	environments.AssertNotCalled(t, "Create")
}

func TestEnvironmentService_Create_NameLookupError(t *testing.T) {
	environments, categories := newEnvironmentServiceMocks()
	categories.On("GetByName", mock.Anything, "Development").Return(&model.EnvironmentCategory{ID: "cat-dev"}, nil)
	environments.On("GetByName", mock.Anything, "x").Return(nil, errors.New("db down"))

	svc := NewEnvironmentService(environments, categories)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Create(context.Background(), actor, CreateEnvironmentInput{Name: "x", CategoryName: "Development"})
	if err == nil {
		t.Fatal("Create() expected an error, got nil")
	}
}

func TestEnvironmentService_Create_RaceAtInsert(t *testing.T) {
	environments, categories := newEnvironmentServiceMocks()
	categories.On("GetByName", mock.Anything, "Development").Return(&model.EnvironmentCategory{ID: "cat-dev"}, nil)
	environments.On("GetByName", mock.Anything, "x").Return(nil, repository.ErrNotFound)
	environments.On("Create", mock.Anything, mock.Anything).Return(repository.ErrEnvironmentNameInUse)

	svc := NewEnvironmentService(environments, categories)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Create(context.Background(), actor, CreateEnvironmentInput{Name: "x", CategoryName: "Development"})
	if !errors.Is(err, ErrEnvironmentNameInUse) {
		t.Fatalf("Create() error = %v, want ErrEnvironmentNameInUse", err)
	}
}

func TestEnvironmentService_Create_RepositoryError(t *testing.T) {
	environments, categories := newEnvironmentServiceMocks()
	categories.On("GetByName", mock.Anything, "Development").Return(&model.EnvironmentCategory{ID: "cat-dev"}, nil)
	environments.On("GetByName", mock.Anything, "x").Return(nil, repository.ErrNotFound)
	environments.On("Create", mock.Anything, mock.Anything).Return(errors.New("disk full"))

	svc := NewEnvironmentService(environments, categories)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Create(context.Background(), actor, CreateEnvironmentInput{Name: "x", CategoryName: "Development"})
	if err == nil || errors.Is(err, ErrEnvironmentNameInUse) {
		t.Fatalf("Create() error = %v, want a generic wrapped error", err)
	}
}

func TestEnvironmentService_Get(t *testing.T) {
	environments, categories := newEnvironmentServiceMocks()
	want := &model.Environment{ID: "e1", Name: "app-dev"}
	environments.On("GetByID", mock.Anything, "e1").Return(want, nil)

	svc := NewEnvironmentService(environments, categories)
	got, err := svc.Get(context.Background(), "e1")
	if err != nil {
		t.Fatalf("Get() returned unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("Get() = %v, want %v", got, want)
	}
}

func TestEnvironmentService_Get_NotFound(t *testing.T) {
	environments, categories := newEnvironmentServiceMocks()
	environments.On("GetByID", mock.Anything, "missing").Return(nil, repository.ErrNotFound)

	svc := NewEnvironmentService(environments, categories)
	_, err := svc.Get(context.Background(), "missing")
	if !errors.Is(err, ErrEnvironmentNotFound) {
		t.Fatalf("Get() error = %v, want ErrEnvironmentNotFound", err)
	}
}

func TestEnvironmentService_Get_RepositoryError(t *testing.T) {
	environments, categories := newEnvironmentServiceMocks()
	environments.On("GetByID", mock.Anything, "e1").Return(nil, errors.New("db down"))

	svc := NewEnvironmentService(environments, categories)
	_, err := svc.Get(context.Background(), "e1")
	if err == nil || errors.Is(err, ErrEnvironmentNotFound) {
		t.Fatalf("Get() error = %v, want a generic wrapped error", err)
	}
}

func TestEnvironmentService_List_Defaults(t *testing.T) {
	environments, categories := newEnvironmentServiceMocks()
	environments.On("List", mock.Anything, defaultPageSize, 0).Return([]model.Environment{{ID: "e1"}}, 1, nil)

	svc := NewEnvironmentService(environments, categories)
	got, total, err := svc.List(context.Background(), 0, 0)
	if err != nil {
		t.Fatalf("List() returned unexpected error: %v", err)
	}
	if total != 1 || len(got) != 1 {
		t.Errorf("List() = (%v, %d), want 1 environment, total 1", got, total)
	}
}

func TestEnvironmentService_List_PageSizeCapped(t *testing.T) {
	environments, categories := newEnvironmentServiceMocks()
	environments.On("List", mock.Anything, maxPageSize, maxPageSize*2).Return([]model.Environment{}, 0, nil)

	svc := NewEnvironmentService(environments, categories)
	if _, _, err := svc.List(context.Background(), 3, 1000); err != nil {
		t.Fatalf("List() returned unexpected error: %v", err)
	}
	environments.AssertExpectations(t)
}

func TestEnvironmentService_List_RepositoryError(t *testing.T) {
	environments, categories := newEnvironmentServiceMocks()
	environments.On("List", mock.Anything, defaultPageSize, 0).Return(nil, 0, errors.New("db down"))

	svc := NewEnvironmentService(environments, categories)
	if _, _, err := svc.List(context.Background(), 1, defaultPageSize); err == nil {
		t.Fatal("List() expected an error, got nil")
	}
}

func TestEnvironmentService_Update_NilActor(t *testing.T) {
	environments, categories := newEnvironmentServiceMocks()
	svc := NewEnvironmentService(environments, categories)

	_, err := svc.Update(context.Background(), nil, "e1", UpdateEnvironmentInput{})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Update() error = %v, want ErrForbidden", err)
	}
}

func TestEnvironmentService_Update_NotFound(t *testing.T) {
	environments, categories := newEnvironmentServiceMocks()
	environments.On("GetByID", mock.Anything, "missing").Return(nil, repository.ErrNotFound)

	svc := NewEnvironmentService(environments, categories)
	actor := &model.User{ID: "actor-id"}
	_, err := svc.Update(context.Background(), actor, "missing", UpdateEnvironmentInput{})
	if !errors.Is(err, ErrEnvironmentNotFound) {
		t.Fatalf("Update() error = %v, want ErrEnvironmentNotFound", err)
	}
}

func TestEnvironmentService_Update_GetByIDError(t *testing.T) {
	environments, categories := newEnvironmentServiceMocks()
	environments.On("GetByID", mock.Anything, "e1").Return(nil, errors.New("db down"))

	svc := NewEnvironmentService(environments, categories)
	actor := &model.User{ID: "actor-id"}
	_, err := svc.Update(context.Background(), actor, "e1", UpdateEnvironmentInput{})
	if err == nil || errors.Is(err, ErrEnvironmentNotFound) {
		t.Fatalf("Update() error = %v, want a generic wrapped error", err)
	}
}

func TestEnvironmentService_Update_NameDescriptionAndCategory(t *testing.T) {
	fixedNow(t)
	environments, categories := newEnvironmentServiceMocks()
	environments.On("GetByID", mock.Anything, "e1").Return(&model.Environment{ID: "e1", Name: "old", EnvironmentCategoryID: "cat-dev"}, nil)
	categories.On("GetByName", mock.Anything, "Staging").Return(&model.EnvironmentCategory{ID: "cat-staging"}, nil)
	environments.On("Update", mock.Anything, mock.MatchedBy(func(e *model.Environment) bool {
		return e.Name == "new" && e.EnvironmentCategoryID == "cat-staging" && e.Description != nil && *e.Description == "desc"
	})).Return(nil)

	svc := NewEnvironmentService(environments, categories)
	actor := &model.User{ID: "actor-id"}
	newName := "new"
	newDesc := "desc"
	newCategory := "Staging"

	got, err := svc.Update(context.Background(), actor, "e1", UpdateEnvironmentInput{Name: &newName, Description: &newDesc, CategoryName: &newCategory})
	if err != nil {
		t.Fatalf("Update() returned unexpected error: %v", err)
	}
	if got.Name != "new" {
		t.Errorf("Name = %q, want %q", got.Name, "new")
	}
}

func TestEnvironmentService_Update_CategoryError(t *testing.T) {
	environments, categories := newEnvironmentServiceMocks()
	environments.On("GetByID", mock.Anything, "e1").Return(&model.Environment{ID: "e1", Name: "old"}, nil)
	categories.On("GetByName", mock.Anything, "Bad").Return(nil, errors.New("db down"))

	svc := NewEnvironmentService(environments, categories)
	actor := &model.User{ID: "actor-id"}
	newCategory := "Bad"

	_, err := svc.Update(context.Background(), actor, "e1", UpdateEnvironmentInput{CategoryName: &newCategory})
	if err == nil {
		t.Fatal("Update() expected an error, got nil")
	}
}

func TestEnvironmentService_Update_BlankName(t *testing.T) {
	environments, categories := newEnvironmentServiceMocks()
	environments.On("GetByID", mock.Anything, "e1").Return(&model.Environment{ID: "e1", Name: "old"}, nil)

	svc := NewEnvironmentService(environments, categories)
	actor := &model.User{ID: "actor-id"}
	blank := "   "

	_, err := svc.Update(context.Background(), actor, "e1", UpdateEnvironmentInput{Name: &blank})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Update() error = %v, want ErrValidation", err)
	}
}

func TestEnvironmentService_Update_NameInUse(t *testing.T) {
	environments, categories := newEnvironmentServiceMocks()
	environments.On("GetByID", mock.Anything, "e1").Return(&model.Environment{ID: "e1", Name: "old"}, nil)
	environments.On("Update", mock.Anything, mock.Anything).Return(repository.ErrEnvironmentNameInUse)

	svc := NewEnvironmentService(environments, categories)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Update(context.Background(), actor, "e1", UpdateEnvironmentInput{})
	if !errors.Is(err, ErrEnvironmentNameInUse) {
		t.Fatalf("Update() error = %v, want ErrEnvironmentNameInUse", err)
	}
}

func TestEnvironmentService_Update_NotFoundRace(t *testing.T) {
	environments, categories := newEnvironmentServiceMocks()
	environments.On("GetByID", mock.Anything, "e1").Return(&model.Environment{ID: "e1", Name: "old"}, nil)
	environments.On("Update", mock.Anything, mock.Anything).Return(repository.ErrNotFound)

	svc := NewEnvironmentService(environments, categories)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Update(context.Background(), actor, "e1", UpdateEnvironmentInput{})
	if !errors.Is(err, ErrEnvironmentNotFound) {
		t.Fatalf("Update() error = %v, want ErrEnvironmentNotFound", err)
	}
}

func TestEnvironmentService_Update_RepositoryError(t *testing.T) {
	environments, categories := newEnvironmentServiceMocks()
	environments.On("GetByID", mock.Anything, "e1").Return(&model.Environment{ID: "e1", Name: "old"}, nil)
	environments.On("Update", mock.Anything, mock.Anything).Return(errors.New("disk full"))

	svc := NewEnvironmentService(environments, categories)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Update(context.Background(), actor, "e1", UpdateEnvironmentInput{})
	if err == nil || errors.Is(err, ErrEnvironmentNameInUse) || errors.Is(err, ErrEnvironmentNotFound) {
		t.Fatalf("Update() error = %v, want a generic wrapped error", err)
	}
}

func TestEnvironmentService_Delete_NilActor(t *testing.T) {
	environments, categories := newEnvironmentServiceMocks()
	svc := NewEnvironmentService(environments, categories)

	err := svc.Delete(context.Background(), nil, "e1")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Delete() error = %v, want ErrForbidden", err)
	}
}

func TestEnvironmentService_Delete_Success(t *testing.T) {
	environments, categories := newEnvironmentServiceMocks()
	environments.On("Delete", mock.Anything, "e1").Return(nil)

	svc := NewEnvironmentService(environments, categories)
	actor := &model.User{ID: "actor-id"}
	if err := svc.Delete(context.Background(), actor, "e1"); err != nil {
		t.Fatalf("Delete() returned unexpected error: %v", err)
	}
}

func TestEnvironmentService_Delete_NotFound(t *testing.T) {
	environments, categories := newEnvironmentServiceMocks()
	environments.On("Delete", mock.Anything, "missing").Return(repository.ErrNotFound)

	svc := NewEnvironmentService(environments, categories)
	actor := &model.User{ID: "actor-id"}
	err := svc.Delete(context.Background(), actor, "missing")
	if !errors.Is(err, ErrEnvironmentNotFound) {
		t.Fatalf("Delete() error = %v, want ErrEnvironmentNotFound", err)
	}
}

func TestEnvironmentService_Delete_InUse(t *testing.T) {
	environments, categories := newEnvironmentServiceMocks()
	environments.On("Delete", mock.Anything, "e1").Return(repository.ErrInUse)

	svc := NewEnvironmentService(environments, categories)
	actor := &model.User{ID: "actor-id"}
	err := svc.Delete(context.Background(), actor, "e1")
	if !errors.Is(err, ErrEnvironmentInUse) {
		t.Fatalf("Delete() error = %v, want ErrEnvironmentInUse", err)
	}
}

func TestEnvironmentService_Delete_RepositoryError(t *testing.T) {
	environments, categories := newEnvironmentServiceMocks()
	environments.On("Delete", mock.Anything, "e1").Return(errors.New("disk full"))

	svc := NewEnvironmentService(environments, categories)
	actor := &model.User{ID: "actor-id"}
	err := svc.Delete(context.Background(), actor, "e1")
	if err == nil || errors.Is(err, ErrEnvironmentInUse) || errors.Is(err, ErrEnvironmentNotFound) {
		t.Fatalf("Delete() error = %v, want a generic wrapped error", err)
	}
}
