package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/repository"
)

func newFeatureFlagServiceMock() *mockFeatureFlagRepository {
	return new(mockFeatureFlagRepository)
}

func TestFeatureFlagService_Create_Success(t *testing.T) {
	fixedNow(t)
	fixedID(t, "flag-1")
	flags := newFeatureFlagServiceMock()
	flags.On("GetByKey", mock.Anything, "new-checkout-flow").Return(nil, repository.ErrNotFound)
	flags.On("Create", mock.Anything, mock.MatchedBy(func(f *model.FeatureFlag) bool {
		return f.ID == "flag-1" && f.Key == "new-checkout-flow" && f.Type == model.FeatureFlagTypeBoolean
	})).Return(nil)

	svc := NewFeatureFlagService(flags)
	actor := &model.User{ID: "actor-id"}

	f, err := svc.Create(context.Background(), actor, CreateFeatureFlagInput{
		Key: "new-checkout-flow", Name: "New Checkout Flow", Type: model.FeatureFlagTypeBoolean,
	})
	if err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}
	if f.Key != "new-checkout-flow" {
		t.Errorf("Key = %q, want %q", f.Key, "new-checkout-flow")
	}
}

func TestFeatureFlagService_Create_NilActor(t *testing.T) {
	flags := newFeatureFlagServiceMock()
	svc := NewFeatureFlagService(flags)

	_, err := svc.Create(context.Background(), nil, CreateFeatureFlagInput{Key: "x", Name: "x", Type: model.FeatureFlagTypeBoolean})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Create() error = %v, want ErrForbidden", err)
	}
}

func TestFeatureFlagService_Create_BlankKey(t *testing.T) {
	flags := newFeatureFlagServiceMock()
	svc := NewFeatureFlagService(flags)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Create(context.Background(), actor, CreateFeatureFlagInput{Key: "  ", Name: "x", Type: model.FeatureFlagTypeBoolean})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Create() error = %v, want ErrValidation", err)
	}
}

func TestFeatureFlagService_Create_InvalidKeyChars(t *testing.T) {
	flags := newFeatureFlagServiceMock()
	svc := NewFeatureFlagService(flags)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Create(context.Background(), actor, CreateFeatureFlagInput{Key: "not a valid key!", Name: "x", Type: model.FeatureFlagTypeBoolean})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Create() error = %v, want ErrValidation", err)
	}
}

func TestFeatureFlagService_Create_BlankName(t *testing.T) {
	flags := newFeatureFlagServiceMock()
	svc := NewFeatureFlagService(flags)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Create(context.Background(), actor, CreateFeatureFlagInput{Key: "valid-key", Name: "  ", Type: model.FeatureFlagTypeBoolean})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Create() error = %v, want ErrValidation", err)
	}
}

func TestFeatureFlagService_Create_InvalidType(t *testing.T) {
	flags := newFeatureFlagServiceMock()
	svc := NewFeatureFlagService(flags)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Create(context.Background(), actor, CreateFeatureFlagInput{Key: "valid-key", Name: "x", Type: "Float"})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Create() error = %v, want ErrValidation", err)
	}
	flags.AssertNotCalled(t, "GetByKey")
}

func TestFeatureFlagService_Create_KeyAlreadyInUse(t *testing.T) {
	flags := newFeatureFlagServiceMock()
	flags.On("GetByKey", mock.Anything, "taken").Return(&model.FeatureFlag{ID: "existing"}, nil)

	svc := NewFeatureFlagService(flags)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Create(context.Background(), actor, CreateFeatureFlagInput{Key: "taken", Name: "x", Type: model.FeatureFlagTypeBoolean})
	if !errors.Is(err, ErrFeatureFlagKeyInUse) {
		t.Fatalf("Create() error = %v, want ErrFeatureFlagKeyInUse", err)
	}
	flags.AssertNotCalled(t, "Create")
}

func TestFeatureFlagService_Create_KeyLookupError(t *testing.T) {
	flags := newFeatureFlagServiceMock()
	flags.On("GetByKey", mock.Anything, "x").Return(nil, errors.New("db down"))

	svc := NewFeatureFlagService(flags)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Create(context.Background(), actor, CreateFeatureFlagInput{Key: "x", Name: "x", Type: model.FeatureFlagTypeBoolean})
	if err == nil {
		t.Fatal("Create() expected an error, got nil")
	}
}

func TestFeatureFlagService_Create_RaceAtInsert(t *testing.T) {
	flags := newFeatureFlagServiceMock()
	flags.On("GetByKey", mock.Anything, "x").Return(nil, repository.ErrNotFound)
	flags.On("Create", mock.Anything, mock.Anything).Return(repository.ErrFeatureFlagKeyInUse)

	svc := NewFeatureFlagService(flags)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Create(context.Background(), actor, CreateFeatureFlagInput{Key: "x", Name: "x", Type: model.FeatureFlagTypeBoolean})
	if !errors.Is(err, ErrFeatureFlagKeyInUse) {
		t.Fatalf("Create() error = %v, want ErrFeatureFlagKeyInUse", err)
	}
}

func TestFeatureFlagService_Create_RepositoryError(t *testing.T) {
	flags := newFeatureFlagServiceMock()
	flags.On("GetByKey", mock.Anything, "x").Return(nil, repository.ErrNotFound)
	flags.On("Create", mock.Anything, mock.Anything).Return(errors.New("disk full"))

	svc := NewFeatureFlagService(flags)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Create(context.Background(), actor, CreateFeatureFlagInput{Key: "x", Name: "x", Type: model.FeatureFlagTypeBoolean})
	if err == nil || errors.Is(err, ErrFeatureFlagKeyInUse) {
		t.Fatalf("Create() error = %v, want a generic wrapped error", err)
	}
}

func TestFeatureFlagService_Get(t *testing.T) {
	flags := newFeatureFlagServiceMock()
	want := &model.FeatureFlag{ID: "f1", Key: "x"}
	flags.On("GetByID", mock.Anything, "f1").Return(want, nil)

	svc := NewFeatureFlagService(flags)
	got, err := svc.Get(context.Background(), "f1")
	if err != nil {
		t.Fatalf("Get() returned unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("Get() = %v, want %v", got, want)
	}
}

func TestFeatureFlagService_Get_NotFound(t *testing.T) {
	flags := newFeatureFlagServiceMock()
	flags.On("GetByID", mock.Anything, "missing").Return(nil, repository.ErrNotFound)

	svc := NewFeatureFlagService(flags)
	_, err := svc.Get(context.Background(), "missing")
	if !errors.Is(err, ErrFeatureFlagNotFound) {
		t.Fatalf("Get() error = %v, want ErrFeatureFlagNotFound", err)
	}
}

func TestFeatureFlagService_Get_RepositoryError(t *testing.T) {
	flags := newFeatureFlagServiceMock()
	flags.On("GetByID", mock.Anything, "f1").Return(nil, errors.New("db down"))

	svc := NewFeatureFlagService(flags)
	_, err := svc.Get(context.Background(), "f1")
	if err == nil || errors.Is(err, ErrFeatureFlagNotFound) {
		t.Fatalf("Get() error = %v, want a generic wrapped error", err)
	}
}

func TestFeatureFlagService_List_Defaults(t *testing.T) {
	flags := newFeatureFlagServiceMock()
	flags.On("List", mock.Anything, defaultPageSize, 0).Return([]model.FeatureFlag{{ID: "f1"}}, 1, nil)

	svc := NewFeatureFlagService(flags)
	got, total, err := svc.List(context.Background(), 0, 0)
	if err != nil {
		t.Fatalf("List() returned unexpected error: %v", err)
	}
	if total != 1 || len(got) != 1 {
		t.Errorf("List() = (%v, %d), want 1 flag, total 1", got, total)
	}
}

func TestFeatureFlagService_List_PageSizeCapped(t *testing.T) {
	flags := newFeatureFlagServiceMock()
	flags.On("List", mock.Anything, maxPageSize, maxPageSize*2).Return([]model.FeatureFlag{}, 0, nil)

	svc := NewFeatureFlagService(flags)
	if _, _, err := svc.List(context.Background(), 3, 1000); err != nil {
		t.Fatalf("List() returned unexpected error: %v", err)
	}
	flags.AssertExpectations(t)
}

func TestFeatureFlagService_List_RepositoryError(t *testing.T) {
	flags := newFeatureFlagServiceMock()
	flags.On("List", mock.Anything, defaultPageSize, 0).Return(nil, 0, errors.New("db down"))

	svc := NewFeatureFlagService(flags)
	if _, _, err := svc.List(context.Background(), 1, defaultPageSize); err == nil {
		t.Fatal("List() expected an error, got nil")
	}
}

func TestFeatureFlagService_Update_NilActor(t *testing.T) {
	flags := newFeatureFlagServiceMock()
	svc := NewFeatureFlagService(flags)

	_, err := svc.Update(context.Background(), nil, "f1", UpdateFeatureFlagInput{})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Update() error = %v, want ErrForbidden", err)
	}
}

func TestFeatureFlagService_Update_NotFound(t *testing.T) {
	flags := newFeatureFlagServiceMock()
	flags.On("GetByID", mock.Anything, "missing").Return(nil, repository.ErrNotFound)

	svc := NewFeatureFlagService(flags)
	actor := &model.User{ID: "actor-id"}
	_, err := svc.Update(context.Background(), actor, "missing", UpdateFeatureFlagInput{})
	if !errors.Is(err, ErrFeatureFlagNotFound) {
		t.Fatalf("Update() error = %v, want ErrFeatureFlagNotFound", err)
	}
}

func TestFeatureFlagService_Update_GetByIDError(t *testing.T) {
	flags := newFeatureFlagServiceMock()
	flags.On("GetByID", mock.Anything, "f1").Return(nil, errors.New("db down"))

	svc := NewFeatureFlagService(flags)
	actor := &model.User{ID: "actor-id"}
	_, err := svc.Update(context.Background(), actor, "f1", UpdateFeatureFlagInput{})
	if err == nil || errors.Is(err, ErrFeatureFlagNotFound) {
		t.Fatalf("Update() error = %v, want a generic wrapped error", err)
	}
}

func TestFeatureFlagService_Update_NameAndDescription(t *testing.T) {
	fixedNow(t)
	flags := newFeatureFlagServiceMock()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1", Key: "x", Name: "old"}, nil)
	flags.On("Update", mock.Anything, mock.MatchedBy(func(f *model.FeatureFlag) bool {
		return f.Name == "new" && f.Description != nil && *f.Description == "desc" && f.Key == "x"
	})).Return(nil)

	svc := NewFeatureFlagService(flags)
	actor := &model.User{ID: "actor-id"}
	newName := "new"
	newDesc := "desc"

	got, err := svc.Update(context.Background(), actor, "f1", UpdateFeatureFlagInput{Name: &newName, Description: &newDesc})
	if err != nil {
		t.Fatalf("Update() returned unexpected error: %v", err)
	}
	if got.Name != "new" {
		t.Errorf("Name = %q, want %q", got.Name, "new")
	}
}

func TestFeatureFlagService_Update_BlankName(t *testing.T) {
	flags := newFeatureFlagServiceMock()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1", Key: "x", Name: "old"}, nil)

	svc := NewFeatureFlagService(flags)
	actor := &model.User{ID: "actor-id"}
	blank := "   "

	_, err := svc.Update(context.Background(), actor, "f1", UpdateFeatureFlagInput{Name: &blank})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Update() error = %v, want ErrValidation", err)
	}
}

func TestFeatureFlagService_Update_NotFoundRace(t *testing.T) {
	flags := newFeatureFlagServiceMock()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1", Key: "x", Name: "old"}, nil)
	flags.On("Update", mock.Anything, mock.Anything).Return(repository.ErrNotFound)

	svc := NewFeatureFlagService(flags)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Update(context.Background(), actor, "f1", UpdateFeatureFlagInput{})
	if !errors.Is(err, ErrFeatureFlagNotFound) {
		t.Fatalf("Update() error = %v, want ErrFeatureFlagNotFound", err)
	}
}

func TestFeatureFlagService_Update_RepositoryError(t *testing.T) {
	flags := newFeatureFlagServiceMock()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1", Key: "x", Name: "old"}, nil)
	flags.On("Update", mock.Anything, mock.Anything).Return(errors.New("disk full"))

	svc := NewFeatureFlagService(flags)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Update(context.Background(), actor, "f1", UpdateFeatureFlagInput{})
	if err == nil || errors.Is(err, ErrFeatureFlagNotFound) {
		t.Fatalf("Update() error = %v, want a generic wrapped error", err)
	}
}

func TestFeatureFlagService_Delete_NilActor(t *testing.T) {
	flags := newFeatureFlagServiceMock()
	svc := NewFeatureFlagService(flags)

	err := svc.Delete(context.Background(), nil, "f1")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Delete() error = %v, want ErrForbidden", err)
	}
}

func TestFeatureFlagService_Delete_Success(t *testing.T) {
	flags := newFeatureFlagServiceMock()
	flags.On("Delete", mock.Anything, "f1").Return(nil)

	svc := NewFeatureFlagService(flags)
	actor := &model.User{ID: "actor-id"}
	if err := svc.Delete(context.Background(), actor, "f1"); err != nil {
		t.Fatalf("Delete() returned unexpected error: %v", err)
	}
}

func TestFeatureFlagService_Delete_NotFound(t *testing.T) {
	flags := newFeatureFlagServiceMock()
	flags.On("Delete", mock.Anything, "missing").Return(repository.ErrNotFound)

	svc := NewFeatureFlagService(flags)
	actor := &model.User{ID: "actor-id"}
	err := svc.Delete(context.Background(), actor, "missing")
	if !errors.Is(err, ErrFeatureFlagNotFound) {
		t.Fatalf("Delete() error = %v, want ErrFeatureFlagNotFound", err)
	}
}

func TestFeatureFlagService_Delete_InUse(t *testing.T) {
	flags := newFeatureFlagServiceMock()
	flags.On("Delete", mock.Anything, "f1").Return(repository.ErrInUse)

	svc := NewFeatureFlagService(flags)
	actor := &model.User{ID: "actor-id"}
	err := svc.Delete(context.Background(), actor, "f1")
	if !errors.Is(err, ErrFeatureFlagInUse) {
		t.Fatalf("Delete() error = %v, want ErrFeatureFlagInUse", err)
	}
}

func TestFeatureFlagService_Delete_RepositoryError(t *testing.T) {
	flags := newFeatureFlagServiceMock()
	flags.On("Delete", mock.Anything, "f1").Return(errors.New("disk full"))

	svc := NewFeatureFlagService(flags)
	actor := &model.User{ID: "actor-id"}
	err := svc.Delete(context.Background(), actor, "f1")
	if err == nil || errors.Is(err, ErrFeatureFlagInUse) || errors.Is(err, ErrFeatureFlagNotFound) {
		t.Fatalf("Delete() error = %v, want a generic wrapped error", err)
	}
}
