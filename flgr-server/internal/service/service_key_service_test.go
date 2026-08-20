package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/repository"
)

func newServiceKeyServiceMocks() (*mockServiceKeyRepository, *mockEnvironmentRepository, *mockServiceKeyEnvironmentRepository) {
	return new(mockServiceKeyRepository), new(mockEnvironmentRepository), new(mockServiceKeyEnvironmentRepository)
}

func TestServiceKeyService_Create_Success(t *testing.T) {
	fixedNow(t)
	fixedID(t, "sk-1")
	keys, environments, links := newServiceKeyServiceMocks()
	environments.On("GetByID", mock.Anything, "env-1").Return(&model.Environment{ID: "env-1"}, nil)
	keys.On("Create", mock.Anything, mock.MatchedBy(func(k *model.ServiceKey) bool {
		return k.ID == "sk-1" && k.Name == "checkout-service prod key" && k.Status == model.ServiceKeyStatusActive &&
			k.CanRead && !k.CanWrite && k.SecretHash != "" && k.CreatedByUserID != nil && *k.CreatedByUserID == "actor-id"
	})).Return(nil)
	links.On("ReplaceEnvironments", mock.Anything, "sk-1", []string{"env-1"}, "actor-id").Return(nil)

	svc := NewServiceKeyService(keys, environments, links)
	actor := &model.User{ID: "actor-id"}

	k, envIDs, secret, err := svc.Create(context.Background(), actor, CreateServiceKeyInput{
		Name: "checkout-service prod key", CanRead: true, EnvironmentIDs: []string{"env-1"},
	})
	if err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}
	if secret == "" {
		t.Error("Create() secret is empty, want a generated value")
	}
	if k.SecretHash == secret {
		t.Error("Create() stored the plaintext secret as its own hash")
	}
	if len(envIDs) != 1 || envIDs[0] != "env-1" {
		t.Errorf("Create() environment ids = %v, want [env-1]", envIDs)
	}
}

func TestServiceKeyService_Create_NilActor(t *testing.T) {
	keys, environments, links := newServiceKeyServiceMocks()
	svc := NewServiceKeyService(keys, environments, links)

	_, _, _, err := svc.Create(context.Background(), nil, CreateServiceKeyInput{Name: "x", CanRead: true, EnvironmentIDs: []string{"env-1"}})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Create() error = %v, want ErrForbidden", err)
	}
}

func TestServiceKeyService_Create_BlankName(t *testing.T) {
	keys, environments, links := newServiceKeyServiceMocks()
	svc := NewServiceKeyService(keys, environments, links)
	actor := &model.User{ID: "actor-id"}

	_, _, _, err := svc.Create(context.Background(), actor, CreateServiceKeyInput{Name: "  ", CanRead: true, EnvironmentIDs: []string{"env-1"}})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Create() error = %v, want ErrValidation", err)
	}
}

func TestServiceKeyService_Create_NeitherReadNorWrite(t *testing.T) {
	keys, environments, links := newServiceKeyServiceMocks()
	svc := NewServiceKeyService(keys, environments, links)
	actor := &model.User{ID: "actor-id"}

	_, _, _, err := svc.Create(context.Background(), actor, CreateServiceKeyInput{Name: "x", EnvironmentIDs: []string{"env-1"}})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Create() error = %v, want ErrValidation", err)
	}
}

func TestServiceKeyService_Create_NoEnvironments(t *testing.T) {
	keys, environments, links := newServiceKeyServiceMocks()
	svc := NewServiceKeyService(keys, environments, links)
	actor := &model.User{ID: "actor-id"}

	_, _, _, err := svc.Create(context.Background(), actor, CreateServiceKeyInput{Name: "x", CanRead: true})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Create() error = %v, want ErrValidation", err)
	}
}

func TestServiceKeyService_Create_UnknownEnvironment(t *testing.T) {
	keys, environments, links := newServiceKeyServiceMocks()
	environments.On("GetByID", mock.Anything, "missing").Return(nil, repository.ErrNotFound)
	svc := NewServiceKeyService(keys, environments, links)
	actor := &model.User{ID: "actor-id"}

	_, _, _, err := svc.Create(context.Background(), actor, CreateServiceKeyInput{Name: "x", CanRead: true, EnvironmentIDs: []string{"missing"}})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Create() error = %v, want ErrValidation", err)
	}
	keys.AssertNotCalled(t, "Create")
}

func TestServiceKeyService_Create_EnvironmentLookupError(t *testing.T) {
	keys, environments, links := newServiceKeyServiceMocks()
	environments.On("GetByID", mock.Anything, "env-1").Return(nil, errors.New("db down"))
	svc := NewServiceKeyService(keys, environments, links)
	actor := &model.User{ID: "actor-id"}

	_, _, _, err := svc.Create(context.Background(), actor, CreateServiceKeyInput{Name: "x", CanRead: true, EnvironmentIDs: []string{"env-1"}})
	if err == nil || errors.Is(err, ErrValidation) {
		t.Fatalf("Create() error = %v, want a generic wrapped error", err)
	}
}

func TestServiceKeyService_Create_SecretGenerationError(t *testing.T) {
	original := randRead
	randRead = func(b []byte) (int, error) { return 0, errors.New("no entropy") }
	t.Cleanup(func() { randRead = original })

	keys, environments, links := newServiceKeyServiceMocks()
	environments.On("GetByID", mock.Anything, "env-1").Return(&model.Environment{ID: "env-1"}, nil)
	svc := NewServiceKeyService(keys, environments, links)
	actor := &model.User{ID: "actor-id"}

	_, _, _, err := svc.Create(context.Background(), actor, CreateServiceKeyInput{Name: "x", CanRead: true, EnvironmentIDs: []string{"env-1"}})
	if err == nil {
		t.Fatal("Create() expected an error, got nil")
	}
	keys.AssertNotCalled(t, "Create")
}

func TestServiceKeyService_Create_RepositoryError(t *testing.T) {
	keys, environments, links := newServiceKeyServiceMocks()
	environments.On("GetByID", mock.Anything, "env-1").Return(&model.Environment{ID: "env-1"}, nil)
	keys.On("Create", mock.Anything, mock.Anything).Return(errors.New("disk full"))
	svc := NewServiceKeyService(keys, environments, links)
	actor := &model.User{ID: "actor-id"}

	_, _, _, err := svc.Create(context.Background(), actor, CreateServiceKeyInput{Name: "x", CanRead: true, EnvironmentIDs: []string{"env-1"}})
	if err == nil {
		t.Fatal("Create() expected an error, got nil")
	}
	links.AssertNotCalled(t, "ReplaceEnvironments")
}

func TestServiceKeyService_Create_LinkError(t *testing.T) {
	keys, environments, links := newServiceKeyServiceMocks()
	environments.On("GetByID", mock.Anything, "env-1").Return(&model.Environment{ID: "env-1"}, nil)
	keys.On("Create", mock.Anything, mock.Anything).Return(nil)
	links.On("ReplaceEnvironments", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("disk full"))
	svc := NewServiceKeyService(keys, environments, links)
	actor := &model.User{ID: "actor-id"}

	_, _, _, err := svc.Create(context.Background(), actor, CreateServiceKeyInput{Name: "x", CanRead: true, EnvironmentIDs: []string{"env-1"}})
	if err == nil {
		t.Fatal("Create() expected an error, got nil")
	}
}

func TestServiceKeyService_Get(t *testing.T) {
	keys, environments, links := newServiceKeyServiceMocks()
	want := &model.ServiceKey{ID: "sk-1", Name: "prod key"}
	keys.On("GetByID", mock.Anything, "sk-1").Return(want, nil)
	links.On("ListEnvironmentIDs", mock.Anything, "sk-1").Return([]string{"env-1"}, nil)

	svc := NewServiceKeyService(keys, environments, links)
	got, envIDs, err := svc.Get(context.Background(), "sk-1")
	if err != nil {
		t.Fatalf("Get() returned unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("Get() = %v, want %v", got, want)
	}
	if len(envIDs) != 1 || envIDs[0] != "env-1" {
		t.Errorf("Get() environment ids = %v, want [env-1]", envIDs)
	}
}

func TestServiceKeyService_Get_NotFound(t *testing.T) {
	keys, environments, links := newServiceKeyServiceMocks()
	keys.On("GetByID", mock.Anything, "missing").Return(nil, repository.ErrNotFound)

	svc := NewServiceKeyService(keys, environments, links)
	_, _, err := svc.Get(context.Background(), "missing")
	if !errors.Is(err, ErrServiceKeyNotFound) {
		t.Fatalf("Get() error = %v, want ErrServiceKeyNotFound", err)
	}
	links.AssertNotCalled(t, "ListEnvironmentIDs")
}

func TestServiceKeyService_Get_RepositoryError(t *testing.T) {
	keys, environments, links := newServiceKeyServiceMocks()
	keys.On("GetByID", mock.Anything, "sk-1").Return(nil, errors.New("db down"))

	svc := NewServiceKeyService(keys, environments, links)
	_, _, err := svc.Get(context.Background(), "sk-1")
	if err == nil || errors.Is(err, ErrServiceKeyNotFound) {
		t.Fatalf("Get() error = %v, want a generic wrapped error", err)
	}
}

func TestServiceKeyService_Get_EnvironmentIDsError(t *testing.T) {
	keys, environments, links := newServiceKeyServiceMocks()
	keys.On("GetByID", mock.Anything, "sk-1").Return(&model.ServiceKey{ID: "sk-1"}, nil)
	links.On("ListEnvironmentIDs", mock.Anything, "sk-1").Return(nil, errors.New("db down"))

	svc := NewServiceKeyService(keys, environments, links)
	_, _, err := svc.Get(context.Background(), "sk-1")
	if err == nil {
		t.Fatal("Get() expected an error, got nil")
	}
}

func TestServiceKeyService_List_Defaults(t *testing.T) {
	keys, environments, links := newServiceKeyServiceMocks()
	keys.On("List", mock.Anything, defaultPageSize, 0).Return([]model.ServiceKey{{ID: "sk-1"}}, 1, nil)
	links.On("ListEnvironmentIDs", mock.Anything, "sk-1").Return([]string{"env-1"}, nil)

	svc := NewServiceKeyService(keys, environments, links)
	got, envIDsByKey, total, err := svc.List(context.Background(), 0, 0)
	if err != nil {
		t.Fatalf("List() returned unexpected error: %v", err)
	}
	if total != 1 || len(got) != 1 {
		t.Errorf("List() = (%v, %d), want 1 service key, total 1", got, total)
	}
	if len(envIDsByKey["sk-1"]) != 1 {
		t.Errorf("List() environment ids for sk-1 = %v, want 1 entry", envIDsByKey["sk-1"])
	}
}

func TestServiceKeyService_List_PageSizeCapped(t *testing.T) {
	keys, environments, links := newServiceKeyServiceMocks()
	keys.On("List", mock.Anything, maxPageSize, maxPageSize*2).Return([]model.ServiceKey{}, 0, nil)

	svc := NewServiceKeyService(keys, environments, links)
	if _, _, _, err := svc.List(context.Background(), 3, 1000); err != nil {
		t.Fatalf("List() returned unexpected error: %v", err)
	}
	keys.AssertExpectations(t)
}

func TestServiceKeyService_List_RepositoryError(t *testing.T) {
	keys, environments, links := newServiceKeyServiceMocks()
	keys.On("List", mock.Anything, defaultPageSize, 0).Return(nil, 0, errors.New("db down"))

	svc := NewServiceKeyService(keys, environments, links)
	if _, _, _, err := svc.List(context.Background(), 1, defaultPageSize); err == nil {
		t.Fatal("List() expected an error, got nil")
	}
}

func TestServiceKeyService_List_EnvironmentIDsError(t *testing.T) {
	keys, environments, links := newServiceKeyServiceMocks()
	keys.On("List", mock.Anything, defaultPageSize, 0).Return([]model.ServiceKey{{ID: "sk-1"}}, 1, nil)
	links.On("ListEnvironmentIDs", mock.Anything, "sk-1").Return(nil, errors.New("db down"))

	svc := NewServiceKeyService(keys, environments, links)
	if _, _, _, err := svc.List(context.Background(), 1, defaultPageSize); err == nil {
		t.Fatal("List() expected an error, got nil")
	}
}

func TestServiceKeyService_Update_NilActor(t *testing.T) {
	keys, environments, links := newServiceKeyServiceMocks()
	svc := NewServiceKeyService(keys, environments, links)

	_, _, err := svc.Update(context.Background(), nil, "sk-1", UpdateServiceKeyInput{})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Update() error = %v, want ErrForbidden", err)
	}
}

func TestServiceKeyService_Update_NotFound(t *testing.T) {
	keys, environments, links := newServiceKeyServiceMocks()
	keys.On("GetByID", mock.Anything, "missing").Return(nil, repository.ErrNotFound)

	svc := NewServiceKeyService(keys, environments, links)
	actor := &model.User{ID: "actor-id"}
	_, _, err := svc.Update(context.Background(), actor, "missing", UpdateServiceKeyInput{})
	if !errors.Is(err, ErrServiceKeyNotFound) {
		t.Fatalf("Update() error = %v, want ErrServiceKeyNotFound", err)
	}
}

func TestServiceKeyService_Update_GetByIDError(t *testing.T) {
	keys, environments, links := newServiceKeyServiceMocks()
	keys.On("GetByID", mock.Anything, "sk-1").Return(nil, errors.New("db down"))

	svc := NewServiceKeyService(keys, environments, links)
	actor := &model.User{ID: "actor-id"}
	_, _, err := svc.Update(context.Background(), actor, "sk-1", UpdateServiceKeyInput{})
	if err == nil || errors.Is(err, ErrServiceKeyNotFound) {
		t.Fatalf("Update() error = %v, want a generic wrapped error", err)
	}
}

func TestServiceKeyService_Update_NameStatusAndFlags(t *testing.T) {
	fixedNow(t)
	keys, environments, links := newServiceKeyServiceMocks()
	keys.On("GetByID", mock.Anything, "sk-1").Return(&model.ServiceKey{ID: "sk-1", Name: "old", Status: model.ServiceKeyStatusActive, CanRead: true}, nil)
	keys.On("Update", mock.Anything, mock.MatchedBy(func(k *model.ServiceKey) bool {
		return k.Name == "new" && k.Status == model.ServiceKeyStatusInactive && !k.CanRead && k.CanWrite
	})).Return(nil)
	links.On("ListEnvironmentIDs", mock.Anything, "sk-1").Return([]string{"env-1"}, nil)

	svc := NewServiceKeyService(keys, environments, links)
	actor := &model.User{ID: "actor-id"}
	newName := "new"
	newStatus := model.ServiceKeyStatusInactive
	canRead := false
	canWrite := true

	got, envIDs, err := svc.Update(context.Background(), actor, "sk-1", UpdateServiceKeyInput{Name: &newName, Status: &newStatus, CanRead: &canRead, CanWrite: &canWrite})
	if err != nil {
		t.Fatalf("Update() returned unexpected error: %v", err)
	}
	if got.Name != "new" || got.Status != model.ServiceKeyStatusInactive {
		t.Errorf("Update() = %+v, want Name=new Status=Inactive", got)
	}
	if len(envIDs) != 1 || envIDs[0] != "env-1" {
		t.Errorf("Update() environment ids = %v, want [env-1] (unchanged, re-fetched)", envIDs)
	}
	links.AssertNotCalled(t, "ReplaceEnvironments")
}

func TestServiceKeyService_Update_BlankName(t *testing.T) {
	keys, environments, links := newServiceKeyServiceMocks()
	keys.On("GetByID", mock.Anything, "sk-1").Return(&model.ServiceKey{ID: "sk-1", Name: "old", CanRead: true}, nil)

	svc := NewServiceKeyService(keys, environments, links)
	actor := &model.User{ID: "actor-id"}
	blank := "   "

	_, _, err := svc.Update(context.Background(), actor, "sk-1", UpdateServiceKeyInput{Name: &blank})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Update() error = %v, want ErrValidation", err)
	}
}

func TestServiceKeyService_Update_InvalidStatus(t *testing.T) {
	keys, environments, links := newServiceKeyServiceMocks()
	keys.On("GetByID", mock.Anything, "sk-1").Return(&model.ServiceKey{ID: "sk-1", Name: "old", CanRead: true}, nil)

	svc := NewServiceKeyService(keys, environments, links)
	actor := &model.User{ID: "actor-id"}
	bad := "Deleted"

	_, _, err := svc.Update(context.Background(), actor, "sk-1", UpdateServiceKeyInput{Status: &bad})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Update() error = %v, want ErrValidation", err)
	}
}

func TestServiceKeyService_Update_NeitherReadNorWrite(t *testing.T) {
	keys, environments, links := newServiceKeyServiceMocks()
	keys.On("GetByID", mock.Anything, "sk-1").Return(&model.ServiceKey{ID: "sk-1", Name: "old", CanRead: true}, nil)

	svc := NewServiceKeyService(keys, environments, links)
	actor := &model.User{ID: "actor-id"}
	no := false

	_, _, err := svc.Update(context.Background(), actor, "sk-1", UpdateServiceKeyInput{CanRead: &no})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Update() error = %v, want ErrValidation", err)
	}
}

func TestServiceKeyService_Update_EmptyEnvironmentIDs(t *testing.T) {
	keys, environments, links := newServiceKeyServiceMocks()
	keys.On("GetByID", mock.Anything, "sk-1").Return(&model.ServiceKey{ID: "sk-1", Name: "old", CanRead: true}, nil)

	svc := NewServiceKeyService(keys, environments, links)
	actor := &model.User{ID: "actor-id"}

	_, _, err := svc.Update(context.Background(), actor, "sk-1", UpdateServiceKeyInput{EnvironmentIDs: []string{}})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Update() error = %v, want ErrValidation", err)
	}
}

func TestServiceKeyService_Update_UnknownEnvironment(t *testing.T) {
	keys, environments, links := newServiceKeyServiceMocks()
	keys.On("GetByID", mock.Anything, "sk-1").Return(&model.ServiceKey{ID: "sk-1", Name: "old", CanRead: true}, nil)
	environments.On("GetByID", mock.Anything, "missing").Return(nil, repository.ErrNotFound)

	svc := NewServiceKeyService(keys, environments, links)
	actor := &model.User{ID: "actor-id"}

	_, _, err := svc.Update(context.Background(), actor, "sk-1", UpdateServiceKeyInput{EnvironmentIDs: []string{"missing"}})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Update() error = %v, want ErrValidation", err)
	}
	keys.AssertNotCalled(t, "Update")
}

func TestServiceKeyService_Update_ReplaceEnvironments(t *testing.T) {
	fixedNow(t)
	keys, environments, links := newServiceKeyServiceMocks()
	keys.On("GetByID", mock.Anything, "sk-1").Return(&model.ServiceKey{ID: "sk-1", Name: "old", CanRead: true}, nil)
	environments.On("GetByID", mock.Anything, "env-2").Return(&model.Environment{ID: "env-2"}, nil)
	keys.On("Update", mock.Anything, mock.Anything).Return(nil)
	links.On("ReplaceEnvironments", mock.Anything, "sk-1", []string{"env-2"}, "actor-id").Return(nil)

	svc := NewServiceKeyService(keys, environments, links)
	actor := &model.User{ID: "actor-id"}

	_, envIDs, err := svc.Update(context.Background(), actor, "sk-1", UpdateServiceKeyInput{EnvironmentIDs: []string{"env-2"}})
	if err != nil {
		t.Fatalf("Update() returned unexpected error: %v", err)
	}
	if len(envIDs) != 1 || envIDs[0] != "env-2" {
		t.Errorf("Update() environment ids = %v, want [env-2]", envIDs)
	}
	links.AssertNotCalled(t, "ListEnvironmentIDs")
}

func TestServiceKeyService_Update_ReplaceEnvironmentsError(t *testing.T) {
	fixedNow(t)
	keys, environments, links := newServiceKeyServiceMocks()
	keys.On("GetByID", mock.Anything, "sk-1").Return(&model.ServiceKey{ID: "sk-1", Name: "old", CanRead: true}, nil)
	environments.On("GetByID", mock.Anything, "env-2").Return(&model.Environment{ID: "env-2"}, nil)
	keys.On("Update", mock.Anything, mock.Anything).Return(nil)
	links.On("ReplaceEnvironments", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("disk full"))

	svc := NewServiceKeyService(keys, environments, links)
	actor := &model.User{ID: "actor-id"}

	_, _, err := svc.Update(context.Background(), actor, "sk-1", UpdateServiceKeyInput{EnvironmentIDs: []string{"env-2"}})
	if err == nil {
		t.Fatal("Update() expected an error, got nil")
	}
}

func TestServiceKeyService_Update_EnvironmentIDsRefetchError(t *testing.T) {
	fixedNow(t)
	keys, environments, links := newServiceKeyServiceMocks()
	keys.On("GetByID", mock.Anything, "sk-1").Return(&model.ServiceKey{ID: "sk-1", Name: "old", CanRead: true}, nil)
	keys.On("Update", mock.Anything, mock.Anything).Return(nil)
	links.On("ListEnvironmentIDs", mock.Anything, "sk-1").Return(nil, errors.New("db down"))

	svc := NewServiceKeyService(keys, environments, links)
	actor := &model.User{ID: "actor-id"}

	_, _, err := svc.Update(context.Background(), actor, "sk-1", UpdateServiceKeyInput{})
	if err == nil {
		t.Fatal("Update() expected an error, got nil")
	}
}

func TestServiceKeyService_Update_NotFoundRace(t *testing.T) {
	keys, environments, links := newServiceKeyServiceMocks()
	keys.On("GetByID", mock.Anything, "sk-1").Return(&model.ServiceKey{ID: "sk-1", Name: "old", CanRead: true}, nil)
	keys.On("Update", mock.Anything, mock.Anything).Return(repository.ErrNotFound)

	svc := NewServiceKeyService(keys, environments, links)
	actor := &model.User{ID: "actor-id"}

	_, _, err := svc.Update(context.Background(), actor, "sk-1", UpdateServiceKeyInput{})
	if !errors.Is(err, ErrServiceKeyNotFound) {
		t.Fatalf("Update() error = %v, want ErrServiceKeyNotFound", err)
	}
}

func TestServiceKeyService_Update_RepositoryError(t *testing.T) {
	keys, environments, links := newServiceKeyServiceMocks()
	keys.On("GetByID", mock.Anything, "sk-1").Return(&model.ServiceKey{ID: "sk-1", Name: "old", CanRead: true}, nil)
	keys.On("Update", mock.Anything, mock.Anything).Return(errors.New("disk full"))

	svc := NewServiceKeyService(keys, environments, links)
	actor := &model.User{ID: "actor-id"}

	_, _, err := svc.Update(context.Background(), actor, "sk-1", UpdateServiceKeyInput{})
	if err == nil || errors.Is(err, ErrServiceKeyNotFound) {
		t.Fatalf("Update() error = %v, want a generic wrapped error", err)
	}
}

func TestServiceKeyService_Authenticate_UnknownSecret(t *testing.T) {
	keys, environments, links := newServiceKeyServiceMocks()
	keys.On("GetBySecretHash", mock.Anything, mock.Anything).Return(nil, repository.ErrNotFound)

	svc := NewServiceKeyService(keys, environments, links)
	_, _, err := svc.Authenticate(context.Background(), "bogus-secret")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Authenticate() error = %v, want ErrForbidden", err)
	}
	links.AssertNotCalled(t, "ListEnvironmentIDs")
}

func TestServiceKeyService_Authenticate_RepositoryError(t *testing.T) {
	keys, environments, links := newServiceKeyServiceMocks()
	keys.On("GetBySecretHash", mock.Anything, mock.Anything).Return(nil, errors.New("db down"))

	svc := NewServiceKeyService(keys, environments, links)
	_, _, err := svc.Authenticate(context.Background(), "some-secret")
	if err == nil || errors.Is(err, ErrForbidden) {
		t.Fatalf("Authenticate() error = %v, want a generic wrapped error", err)
	}
}

func TestServiceKeyService_Authenticate_InactiveKey(t *testing.T) {
	keys, environments, links := newServiceKeyServiceMocks()
	keys.On("GetBySecretHash", mock.Anything, mock.Anything).Return(&model.ServiceKey{ID: "sk-1", Status: model.ServiceKeyStatusInactive}, nil)

	svc := NewServiceKeyService(keys, environments, links)
	_, _, err := svc.Authenticate(context.Background(), "some-secret")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Authenticate() error = %v, want ErrForbidden", err)
	}
	links.AssertNotCalled(t, "ListEnvironmentIDs")
}

func TestServiceKeyService_Authenticate_ActiveKey(t *testing.T) {
	keys, environments, links := newServiceKeyServiceMocks()
	keys.On("GetBySecretHash", mock.Anything, hashToken("correct-secret")).Return(&model.ServiceKey{ID: "sk-1", Status: model.ServiceKeyStatusActive, CanRead: true}, nil)
	links.On("ListEnvironmentIDs", mock.Anything, "sk-1").Return([]string{"env-1", "env-2"}, nil)

	svc := NewServiceKeyService(keys, environments, links)
	k, envIDs, err := svc.Authenticate(context.Background(), "correct-secret")
	if err != nil {
		t.Fatalf("Authenticate() returned unexpected error: %v", err)
	}
	if k == nil || k.ID != "sk-1" {
		t.Errorf("Authenticate() key = %v, want sk-1", k)
	}
	if len(envIDs) != 2 || envIDs[0] != "env-1" || envIDs[1] != "env-2" {
		t.Errorf("Authenticate() environment ids = %v, want [env-1 env-2]", envIDs)
	}
}

func TestServiceKeyService_Authenticate_EnvironmentIDsError(t *testing.T) {
	keys, environments, links := newServiceKeyServiceMocks()
	keys.On("GetBySecretHash", mock.Anything, mock.Anything).Return(&model.ServiceKey{ID: "sk-1", Status: model.ServiceKeyStatusActive, CanRead: true}, nil)
	links.On("ListEnvironmentIDs", mock.Anything, "sk-1").Return(nil, errors.New("db down"))

	svc := NewServiceKeyService(keys, environments, links)
	_, _, err := svc.Authenticate(context.Background(), "correct-secret")
	if err == nil {
		t.Fatal("Authenticate() expected an error, got nil")
	}
}

func TestServiceKeyService_Deactivate(t *testing.T) {
	fixedNow(t)
	keys, environments, links := newServiceKeyServiceMocks()
	keys.On("GetByID", mock.Anything, "sk-1").Return(&model.ServiceKey{ID: "sk-1", Name: "prod key", Status: model.ServiceKeyStatusActive, CanRead: true}, nil)
	keys.On("Update", mock.Anything, mock.MatchedBy(func(k *model.ServiceKey) bool {
		return k.Status == model.ServiceKeyStatusInactive
	})).Return(nil)
	links.On("ListEnvironmentIDs", mock.Anything, "sk-1").Return([]string{"env-1"}, nil)

	svc := NewServiceKeyService(keys, environments, links)
	actor := &model.User{ID: "actor-id"}

	got, envIDs, err := svc.Deactivate(context.Background(), actor, "sk-1")
	if err != nil {
		t.Fatalf("Deactivate() returned unexpected error: %v", err)
	}
	if got.Status != model.ServiceKeyStatusInactive {
		t.Errorf("Status = %q, want %q", got.Status, model.ServiceKeyStatusInactive)
	}
	if len(envIDs) != 1 || envIDs[0] != "env-1" {
		t.Errorf("Deactivate() environment ids = %v, want [env-1]", envIDs)
	}
}
