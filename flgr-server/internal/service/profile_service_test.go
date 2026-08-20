package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/repository"
)

func newProfileServiceMocks() (*mockProfileRepository, *mockProfilePermissionRepository, *mockPermissionRepository) {
	return new(mockProfileRepository), new(mockProfilePermissionRepository), new(mockPermissionRepository)
}

var testCatalog = []model.Permission{
	{ID: "user:view", Resource: "User", Action: "View"},
	{ID: "user:edit", Resource: "User", Action: "Edit"},
}

func TestProfileService_Create_Success(t *testing.T) {
	fixedNow(t)
	fixedID(t, "profile-1")
	profiles, profilePerms, permissions := newProfileServiceMocks()
	permissions.On("List", mock.Anything).Return(testCatalog, nil)
	profiles.On("GetByName", mock.Anything, "app-read").Return(nil, repository.ErrNotFound)
	profiles.On("Create", mock.Anything, mock.MatchedBy(func(p *model.Profile) bool {
		return p.ID == "profile-1" && p.Name == "app-read" && !p.IsSystem
	})).Return(nil)
	profilePerms.On("ReplacePermissions", mock.Anything, "profile-1", []string{"user:view"}, "actor-id").Return(nil)

	svc := NewProfileService(profiles, profilePerms, permissions)
	actor := &model.User{ID: "actor-id"}

	p, err := svc.Create(context.Background(), actor, CreateProfileInput{Name: "app-read", PermissionIDs: []string{"user:view"}})
	if err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}
	if p.Name != "app-read" {
		t.Errorf("Name = %q, want %q", p.Name, "app-read")
	}
}

func TestProfileService_Create_NoPermissions_SkipsReplace(t *testing.T) {
	fixedNow(t)
	fixedID(t, "profile-1")
	profiles, profilePerms, permissions := newProfileServiceMocks()
	permissions.On("List", mock.Anything).Return(testCatalog, nil)
	profiles.On("GetByName", mock.Anything, "empty-profile").Return(nil, repository.ErrNotFound)
	profiles.On("Create", mock.Anything, mock.Anything).Return(nil)

	svc := NewProfileService(profiles, profilePerms, permissions)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Create(context.Background(), actor, CreateProfileInput{Name: "empty-profile"})
	if err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}
	profilePerms.AssertNotCalled(t, "ReplacePermissions")
}

func TestProfileService_Create_NilActor(t *testing.T) {
	profiles, profilePerms, permissions := newProfileServiceMocks()
	svc := NewProfileService(profiles, profilePerms, permissions)

	_, err := svc.Create(context.Background(), nil, CreateProfileInput{Name: "x"})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Create() error = %v, want ErrForbidden", err)
	}
}

func TestProfileService_Create_BlankName(t *testing.T) {
	profiles, profilePerms, permissions := newProfileServiceMocks()
	svc := NewProfileService(profiles, profilePerms, permissions)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Create(context.Background(), actor, CreateProfileInput{Name: "  "})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Create() error = %v, want ErrValidation", err)
	}
}

func TestProfileService_Create_InvalidPermissionID(t *testing.T) {
	profiles, profilePerms, permissions := newProfileServiceMocks()
	permissions.On("List", mock.Anything).Return(testCatalog, nil)
	svc := NewProfileService(profiles, profilePerms, permissions)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Create(context.Background(), actor, CreateProfileInput{Name: "x", PermissionIDs: []string{"does-not-exist"}})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Create() error = %v, want ErrValidation", err)
	}
	profiles.AssertNotCalled(t, "Create")
}

func TestProfileService_Create_PermissionCatalogError(t *testing.T) {
	profiles, profilePerms, permissions := newProfileServiceMocks()
	permissions.On("List", mock.Anything).Return(nil, errors.New("db down"))
	svc := NewProfileService(profiles, profilePerms, permissions)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Create(context.Background(), actor, CreateProfileInput{Name: "x"})
	if err == nil {
		t.Fatal("Create() expected an error, got nil")
	}
}

func TestProfileService_Create_NameAlreadyInUse(t *testing.T) {
	profiles, profilePerms, permissions := newProfileServiceMocks()
	permissions.On("List", mock.Anything).Return(testCatalog, nil)
	profiles.On("GetByName", mock.Anything, "taken").Return(&model.Profile{ID: "existing"}, nil)
	svc := NewProfileService(profiles, profilePerms, permissions)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Create(context.Background(), actor, CreateProfileInput{Name: "taken"})
	if !errors.Is(err, ErrProfileNameInUse) {
		t.Fatalf("Create() error = %v, want ErrProfileNameInUse", err)
	}
	profiles.AssertNotCalled(t, "Create")
}

func TestProfileService_Create_GetByNameError(t *testing.T) {
	profiles, profilePerms, permissions := newProfileServiceMocks()
	permissions.On("List", mock.Anything).Return(testCatalog, nil)
	profiles.On("GetByName", mock.Anything, "x").Return(nil, errors.New("db down"))
	svc := NewProfileService(profiles, profilePerms, permissions)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Create(context.Background(), actor, CreateProfileInput{Name: "x"})
	if err == nil || errors.Is(err, ErrProfileNameInUse) {
		t.Fatalf("Create() error = %v, want a generic wrapped error", err)
	}
}

func TestProfileService_Create_RaceAtInsert(t *testing.T) {
	profiles, profilePerms, permissions := newProfileServiceMocks()
	permissions.On("List", mock.Anything).Return(testCatalog, nil)
	profiles.On("GetByName", mock.Anything, "x").Return(nil, repository.ErrNotFound)
	profiles.On("Create", mock.Anything, mock.Anything).Return(repository.ErrProfileNameInUse)
	svc := NewProfileService(profiles, profilePerms, permissions)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Create(context.Background(), actor, CreateProfileInput{Name: "x"})
	if !errors.Is(err, ErrProfileNameInUse) {
		t.Fatalf("Create() error = %v, want ErrProfileNameInUse", err)
	}
}

func TestProfileService_Create_RepositoryError(t *testing.T) {
	profiles, profilePerms, permissions := newProfileServiceMocks()
	permissions.On("List", mock.Anything).Return(testCatalog, nil)
	profiles.On("GetByName", mock.Anything, "x").Return(nil, repository.ErrNotFound)
	profiles.On("Create", mock.Anything, mock.Anything).Return(errors.New("disk full"))
	svc := NewProfileService(profiles, profilePerms, permissions)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Create(context.Background(), actor, CreateProfileInput{Name: "x"})
	if err == nil || errors.Is(err, ErrProfileNameInUse) {
		t.Fatalf("Create() error = %v, want a generic wrapped error", err)
	}
}

func TestProfileService_Create_ReplacePermissionsError(t *testing.T) {
	fixedNow(t)
	fixedID(t, "profile-1")
	profiles, profilePerms, permissions := newProfileServiceMocks()
	permissions.On("List", mock.Anything).Return(testCatalog, nil)
	profiles.On("GetByName", mock.Anything, "x").Return(nil, repository.ErrNotFound)
	profiles.On("Create", mock.Anything, mock.Anything).Return(nil)
	profilePerms.On("ReplacePermissions", mock.Anything, "profile-1", mock.Anything, "actor-id").Return(errors.New("db down"))
	svc := NewProfileService(profiles, profilePerms, permissions)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Create(context.Background(), actor, CreateProfileInput{Name: "x", PermissionIDs: []string{"user:view"}})
	if err == nil {
		t.Fatal("Create() expected an error, got nil")
	}
}

func TestProfileService_Get(t *testing.T) {
	profiles, profilePerms, permissions := newProfileServiceMocks()
	want := &model.Profile{ID: "p1", Name: "app-read"}
	profiles.On("GetByID", mock.Anything, "p1").Return(want, nil)
	svc := NewProfileService(profiles, profilePerms, permissions)

	got, err := svc.Get(context.Background(), "p1")
	if err != nil {
		t.Fatalf("Get() returned unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("Get() = %v, want %v", got, want)
	}
}

func TestProfileService_Get_NotFound(t *testing.T) {
	profiles, profilePerms, permissions := newProfileServiceMocks()
	profiles.On("GetByID", mock.Anything, "missing").Return(nil, repository.ErrNotFound)
	svc := NewProfileService(profiles, profilePerms, permissions)

	_, err := svc.Get(context.Background(), "missing")
	if !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("Get() error = %v, want ErrProfileNotFound", err)
	}
}

func TestProfileService_Get_RepositoryError(t *testing.T) {
	profiles, profilePerms, permissions := newProfileServiceMocks()
	profiles.On("GetByID", mock.Anything, "p1").Return(nil, errors.New("db down"))
	svc := NewProfileService(profiles, profilePerms, permissions)

	_, err := svc.Get(context.Background(), "p1")
	if err == nil || errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("Get() error = %v, want a generic wrapped error", err)
	}
}

func TestProfileService_List_Defaults(t *testing.T) {
	profiles, profilePerms, permissions := newProfileServiceMocks()
	profiles.On("List", mock.Anything, defaultPageSize, 0).Return([]model.Profile{{ID: "p1"}}, 1, nil)
	svc := NewProfileService(profiles, profilePerms, permissions)

	got, total, err := svc.List(context.Background(), 0, 0)
	if err != nil {
		t.Fatalf("List() returned unexpected error: %v", err)
	}
	if total != 1 || len(got) != 1 {
		t.Errorf("List() = (%v, %d), want 1 profile, total 1", got, total)
	}
}

func TestProfileService_List_PageSizeCapped(t *testing.T) {
	profiles, profilePerms, permissions := newProfileServiceMocks()
	profiles.On("List", mock.Anything, maxPageSize, maxPageSize*2).Return([]model.Profile{}, 0, nil)
	svc := NewProfileService(profiles, profilePerms, permissions)

	if _, _, err := svc.List(context.Background(), 3, 1000); err != nil {
		t.Fatalf("List() returned unexpected error: %v", err)
	}
	profiles.AssertExpectations(t)
}

func TestProfileService_List_RepositoryError(t *testing.T) {
	profiles, profilePerms, permissions := newProfileServiceMocks()
	profiles.On("List", mock.Anything, defaultPageSize, 0).Return(nil, 0, errors.New("db down"))
	svc := NewProfileService(profiles, profilePerms, permissions)

	if _, _, err := svc.List(context.Background(), 1, defaultPageSize); err == nil {
		t.Fatal("List() expected an error, got nil")
	}
}

func TestProfileService_PermissionIDs(t *testing.T) {
	profiles, profilePerms, permissions := newProfileServiceMocks()
	profiles.On("GetByID", mock.Anything, "p1").Return(&model.Profile{ID: "p1"}, nil)
	profilePerms.On("ListPermissionIDs", mock.Anything, "p1").Return([]string{"user:view"}, nil)
	svc := NewProfileService(profiles, profilePerms, permissions)

	ids, err := svc.PermissionIDs(context.Background(), "p1")
	if err != nil {
		t.Fatalf("PermissionIDs() returned unexpected error: %v", err)
	}
	if len(ids) != 1 || ids[0] != "user:view" {
		t.Errorf("PermissionIDs() = %v, want [user:view]", ids)
	}
}

func TestProfileService_PermissionIDs_ProfileNotFound(t *testing.T) {
	profiles, profilePerms, permissions := newProfileServiceMocks()
	profiles.On("GetByID", mock.Anything, "missing").Return(nil, repository.ErrNotFound)
	svc := NewProfileService(profiles, profilePerms, permissions)

	_, err := svc.PermissionIDs(context.Background(), "missing")
	if !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("PermissionIDs() error = %v, want ErrProfileNotFound", err)
	}
	profilePerms.AssertNotCalled(t, "ListPermissionIDs")
}

func TestProfileService_PermissionIDs_RepositoryError(t *testing.T) {
	profiles, profilePerms, permissions := newProfileServiceMocks()
	profiles.On("GetByID", mock.Anything, "p1").Return(&model.Profile{ID: "p1"}, nil)
	profilePerms.On("ListPermissionIDs", mock.Anything, "p1").Return(nil, errors.New("db down"))
	svc := NewProfileService(profiles, profilePerms, permissions)

	_, err := svc.PermissionIDs(context.Background(), "p1")
	if err == nil {
		t.Fatal("PermissionIDs() expected an error, got nil")
	}
}

func TestProfileService_Update_NilActor(t *testing.T) {
	profiles, profilePerms, permissions := newProfileServiceMocks()
	svc := NewProfileService(profiles, profilePerms, permissions)

	_, err := svc.Update(context.Background(), nil, "p1", UpdateProfileInput{})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Update() error = %v, want ErrForbidden", err)
	}
}

func TestProfileService_Update_NotFound(t *testing.T) {
	profiles, profilePerms, permissions := newProfileServiceMocks()
	profiles.On("GetByID", mock.Anything, "missing").Return(nil, repository.ErrNotFound)
	svc := NewProfileService(profiles, profilePerms, permissions)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Update(context.Background(), actor, "missing", UpdateProfileInput{})
	if !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("Update() error = %v, want ErrProfileNotFound", err)
	}
}

func TestProfileService_Update_GetByIDError(t *testing.T) {
	profiles, profilePerms, permissions := newProfileServiceMocks()
	profiles.On("GetByID", mock.Anything, "p1").Return(nil, errors.New("db down"))
	svc := NewProfileService(profiles, profilePerms, permissions)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Update(context.Background(), actor, "p1", UpdateProfileInput{})
	if err == nil || errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("Update() error = %v, want a generic wrapped error", err)
	}
}

func TestProfileService_Update_SystemProfile_CannotEditPermissions(t *testing.T) {
	profiles, profilePerms, permissions := newProfileServiceMocks()
	profiles.On("GetByID", mock.Anything, "admin-profile").Return(&model.Profile{ID: "admin-profile", Name: "Administrador", IsSystem: true}, nil)
	svc := NewProfileService(profiles, profilePerms, permissions)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Update(context.Background(), actor, "admin-profile", UpdateProfileInput{PermissionIDs: &[]string{"user:view"}})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Update() error = %v, want ErrForbidden", err)
	}
	profiles.AssertNotCalled(t, "Update")
}

func TestProfileService_Update_SystemProfile_NameChangeAllowed(t *testing.T) {
	fixedNow(t)
	profiles, profilePerms, permissions := newProfileServiceMocks()
	profiles.On("GetByID", mock.Anything, "admin-profile").Return(&model.Profile{ID: "admin-profile", Name: "Administrador", IsSystem: true}, nil)
	profiles.On("Update", mock.Anything, mock.MatchedBy(func(p *model.Profile) bool {
		return p.Name == "Renamed"
	})).Return(nil)
	svc := NewProfileService(profiles, profilePerms, permissions)
	actor := &model.User{ID: "actor-id"}
	newName := "Renamed"

	_, err := svc.Update(context.Background(), actor, "admin-profile", UpdateProfileInput{Name: &newName})
	if err != nil {
		t.Fatalf("Update() returned unexpected error: %v", err)
	}
}

func TestProfileService_Update_InvalidPermissionID(t *testing.T) {
	profiles, profilePerms, permissions := newProfileServiceMocks()
	profiles.On("GetByID", mock.Anything, "p1").Return(&model.Profile{ID: "p1", Name: "x"}, nil)
	permissions.On("List", mock.Anything).Return(testCatalog, nil)
	svc := NewProfileService(profiles, profilePerms, permissions)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Update(context.Background(), actor, "p1", UpdateProfileInput{PermissionIDs: &[]string{"does-not-exist"}})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Update() error = %v, want ErrValidation", err)
	}
	profiles.AssertNotCalled(t, "Update")
}

func TestProfileService_Update_BlankName(t *testing.T) {
	profiles, profilePerms, permissions := newProfileServiceMocks()
	profiles.On("GetByID", mock.Anything, "p1").Return(&model.Profile{ID: "p1", Name: "x"}, nil)
	svc := NewProfileService(profiles, profilePerms, permissions)
	actor := &model.User{ID: "actor-id"}
	blank := "   "

	_, err := svc.Update(context.Background(), actor, "p1", UpdateProfileInput{Name: &blank})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Update() error = %v, want ErrValidation", err)
	}
}

func TestProfileService_Update_NameAndDescriptionAndPermissions(t *testing.T) {
	fixedNow(t)
	profiles, profilePerms, permissions := newProfileServiceMocks()
	profiles.On("GetByID", mock.Anything, "p1").Return(&model.Profile{ID: "p1", Name: "old"}, nil)
	permissions.On("List", mock.Anything).Return(testCatalog, nil)
	profiles.On("Update", mock.Anything, mock.MatchedBy(func(p *model.Profile) bool {
		return p.Name == "new" && p.Description != nil && *p.Description == "desc"
	})).Return(nil)
	profilePerms.On("ReplacePermissions", mock.Anything, "p1", []string{"user:view"}, "actor-id").Return(nil)

	svc := NewProfileService(profiles, profilePerms, permissions)
	actor := &model.User{ID: "actor-id"}
	newName := "new"
	newDesc := "desc"

	got, err := svc.Update(context.Background(), actor, "p1", UpdateProfileInput{
		Name: &newName, Description: &newDesc, PermissionIDs: &[]string{"user:view"},
	})
	if err != nil {
		t.Fatalf("Update() returned unexpected error: %v", err)
	}
	if got.Name != "new" {
		t.Errorf("Name = %q, want %q", got.Name, "new")
	}
}

func TestProfileService_Update_NameInUse(t *testing.T) {
	profiles, profilePerms, permissions := newProfileServiceMocks()
	profiles.On("GetByID", mock.Anything, "p1").Return(&model.Profile{ID: "p1", Name: "old"}, nil)
	profiles.On("Update", mock.Anything, mock.Anything).Return(repository.ErrProfileNameInUse)
	svc := NewProfileService(profiles, profilePerms, permissions)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Update(context.Background(), actor, "p1", UpdateProfileInput{})
	if !errors.Is(err, ErrProfileNameInUse) {
		t.Fatalf("Update() error = %v, want ErrProfileNameInUse", err)
	}
}

func TestProfileService_Update_NotFoundRace(t *testing.T) {
	profiles, profilePerms, permissions := newProfileServiceMocks()
	profiles.On("GetByID", mock.Anything, "p1").Return(&model.Profile{ID: "p1", Name: "old"}, nil)
	profiles.On("Update", mock.Anything, mock.Anything).Return(repository.ErrNotFound)
	svc := NewProfileService(profiles, profilePerms, permissions)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Update(context.Background(), actor, "p1", UpdateProfileInput{})
	if !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("Update() error = %v, want ErrProfileNotFound", err)
	}
}

func TestProfileService_Update_RepositoryError(t *testing.T) {
	profiles, profilePerms, permissions := newProfileServiceMocks()
	profiles.On("GetByID", mock.Anything, "p1").Return(&model.Profile{ID: "p1", Name: "old"}, nil)
	profiles.On("Update", mock.Anything, mock.Anything).Return(errors.New("disk full"))
	svc := NewProfileService(profiles, profilePerms, permissions)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Update(context.Background(), actor, "p1", UpdateProfileInput{})
	if err == nil || errors.Is(err, ErrProfileNameInUse) || errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("Update() error = %v, want a generic wrapped error", err)
	}
}

func TestProfileService_Update_ReplacePermissionsError(t *testing.T) {
	fixedNow(t)
	profiles, profilePerms, permissions := newProfileServiceMocks()
	profiles.On("GetByID", mock.Anything, "p1").Return(&model.Profile{ID: "p1", Name: "old"}, nil)
	permissions.On("List", mock.Anything).Return(testCatalog, nil)
	profiles.On("Update", mock.Anything, mock.Anything).Return(nil)
	profilePerms.On("ReplacePermissions", mock.Anything, "p1", mock.Anything, "actor-id").Return(errors.New("db down"))
	svc := NewProfileService(profiles, profilePerms, permissions)
	actor := &model.User{ID: "actor-id"}

	_, err := svc.Update(context.Background(), actor, "p1", UpdateProfileInput{PermissionIDs: &[]string{"user:view"}})
	if err == nil {
		t.Fatal("Update() expected an error, got nil")
	}
}

func TestProfileService_Delete_NilActor(t *testing.T) {
	profiles, profilePerms, permissions := newProfileServiceMocks()
	svc := NewProfileService(profiles, profilePerms, permissions)

	err := svc.Delete(context.Background(), nil, "p1")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Delete() error = %v, want ErrForbidden", err)
	}
}

func TestProfileService_Delete_NotFound(t *testing.T) {
	profiles, profilePerms, permissions := newProfileServiceMocks()
	profiles.On("GetByID", mock.Anything, "missing").Return(nil, repository.ErrNotFound)
	svc := NewProfileService(profiles, profilePerms, permissions)
	actor := &model.User{ID: "actor-id"}

	err := svc.Delete(context.Background(), actor, "missing")
	if !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("Delete() error = %v, want ErrProfileNotFound", err)
	}
}

func TestProfileService_Delete_GetByIDError(t *testing.T) {
	profiles, profilePerms, permissions := newProfileServiceMocks()
	profiles.On("GetByID", mock.Anything, "p1").Return(nil, errors.New("db down"))
	svc := NewProfileService(profiles, profilePerms, permissions)
	actor := &model.User{ID: "actor-id"}

	err := svc.Delete(context.Background(), actor, "p1")
	if err == nil || errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("Delete() error = %v, want a generic wrapped error", err)
	}
}

func TestProfileService_Delete_NonSystemProfile_AlwaysAllowed(t *testing.T) {
	profiles, profilePerms, permissions := newProfileServiceMocks()
	profiles.On("GetByID", mock.Anything, "p1").Return(&model.Profile{ID: "p1", IsSystem: false}, nil)
	profiles.On("Delete", mock.Anything, "p1").Return(nil)
	svc := NewProfileService(profiles, profilePerms, permissions)
	actor := &model.User{ID: "actor-id"}

	if err := svc.Delete(context.Background(), actor, "p1"); err != nil {
		t.Fatalf("Delete() returned unexpected error: %v", err)
	}
	permissions.AssertNotCalled(t, "List")
}

func TestProfileService_Delete_SystemProfile_AnotherHoldsFullCatalog(t *testing.T) {
	profiles, profilePerms, permissions := newProfileServiceMocks()
	profiles.On("GetByID", mock.Anything, "admin-profile").Return(&model.Profile{ID: "admin-profile", IsSystem: true}, nil)
	permissions.On("List", mock.Anything).Return(testCatalog, nil)
	profiles.On("ListAll", mock.Anything).Return([]model.Profile{
		{ID: "admin-profile"}, {ID: "backup-admin"},
	}, nil)
	profilePerms.On("ListPermissionIDs", mock.Anything, "backup-admin").Return([]string{"user:view", "user:edit"}, nil)
	profiles.On("Delete", mock.Anything, "admin-profile").Return(nil)

	svc := NewProfileService(profiles, profilePerms, permissions)
	actor := &model.User{ID: "actor-id"}

	if err := svc.Delete(context.Background(), actor, "admin-profile"); err != nil {
		t.Fatalf("Delete() returned unexpected error: %v", err)
	}
}

func TestProfileService_Delete_SystemProfile_NoOtherFullCatalogProfile(t *testing.T) {
	profiles, profilePerms, permissions := newProfileServiceMocks()
	profiles.On("GetByID", mock.Anything, "admin-profile").Return(&model.Profile{ID: "admin-profile", IsSystem: true}, nil)
	permissions.On("List", mock.Anything).Return(testCatalog, nil)
	profiles.On("ListAll", mock.Anything).Return([]model.Profile{
		{ID: "admin-profile"}, {ID: "partial-profile"},
	}, nil)
	profilePerms.On("ListPermissionIDs", mock.Anything, "partial-profile").Return([]string{"user:view"}, nil)

	svc := NewProfileService(profiles, profilePerms, permissions)
	actor := &model.User{ID: "actor-id"}

	err := svc.Delete(context.Background(), actor, "admin-profile")
	if !errors.Is(err, ErrLastFullCatalogProfile) {
		t.Fatalf("Delete() error = %v, want ErrLastFullCatalogProfile", err)
	}
	profiles.AssertNotCalled(t, "Delete")
}

func TestProfileService_Delete_AnotherProfileHoldsFullCatalog_PermissionsListError(t *testing.T) {
	profiles, profilePerms, permissions := newProfileServiceMocks()
	profiles.On("GetByID", mock.Anything, "admin-profile").Return(&model.Profile{ID: "admin-profile", IsSystem: true}, nil)
	permissions.On("List", mock.Anything).Return(nil, errors.New("db down"))

	svc := NewProfileService(profiles, profilePerms, permissions)
	actor := &model.User{ID: "actor-id"}

	err := svc.Delete(context.Background(), actor, "admin-profile")
	if err == nil {
		t.Fatal("Delete() expected an error, got nil")
	}
}

func TestProfileService_Delete_AnotherProfileHoldsFullCatalog_ListAllError(t *testing.T) {
	profiles, profilePerms, permissions := newProfileServiceMocks()
	profiles.On("GetByID", mock.Anything, "admin-profile").Return(&model.Profile{ID: "admin-profile", IsSystem: true}, nil)
	permissions.On("List", mock.Anything).Return(testCatalog, nil)
	profiles.On("ListAll", mock.Anything).Return(nil, errors.New("db down"))

	svc := NewProfileService(profiles, profilePerms, permissions)
	actor := &model.User{ID: "actor-id"}

	err := svc.Delete(context.Background(), actor, "admin-profile")
	if err == nil {
		t.Fatal("Delete() expected an error, got nil")
	}
}

func TestProfileService_Delete_AnotherProfileHoldsFullCatalog_ListPermissionIDsError(t *testing.T) {
	profiles, profilePerms, permissions := newProfileServiceMocks()
	profiles.On("GetByID", mock.Anything, "admin-profile").Return(&model.Profile{ID: "admin-profile", IsSystem: true}, nil)
	permissions.On("List", mock.Anything).Return(testCatalog, nil)
	profiles.On("ListAll", mock.Anything).Return([]model.Profile{{ID: "admin-profile"}, {ID: "other"}}, nil)
	profilePerms.On("ListPermissionIDs", mock.Anything, "other").Return(nil, errors.New("db down"))

	svc := NewProfileService(profiles, profilePerms, permissions)
	actor := &model.User{ID: "actor-id"}

	err := svc.Delete(context.Background(), actor, "admin-profile")
	if err == nil {
		t.Fatal("Delete() expected an error, got nil")
	}
}

func TestProfileService_Delete_NotFoundRace(t *testing.T) {
	profiles, profilePerms, permissions := newProfileServiceMocks()
	profiles.On("GetByID", mock.Anything, "p1").Return(&model.Profile{ID: "p1", IsSystem: false}, nil)
	profiles.On("Delete", mock.Anything, "p1").Return(repository.ErrNotFound)
	svc := NewProfileService(profiles, profilePerms, permissions)
	actor := &model.User{ID: "actor-id"}

	err := svc.Delete(context.Background(), actor, "p1")
	if !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("Delete() error = %v, want ErrProfileNotFound", err)
	}
}

func TestProfileService_Delete_RepositoryError(t *testing.T) {
	profiles, profilePerms, permissions := newProfileServiceMocks()
	profiles.On("GetByID", mock.Anything, "p1").Return(&model.Profile{ID: "p1", IsSystem: false}, nil)
	profiles.On("Delete", mock.Anything, "p1").Return(errors.New("disk full"))
	svc := NewProfileService(profiles, profilePerms, permissions)
	actor := &model.User{ID: "actor-id"}

	err := svc.Delete(context.Background(), actor, "p1")
	if err == nil || errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("Delete() error = %v, want a generic wrapped error", err)
	}
}

func TestIsFullCatalog(t *testing.T) {
	tests := []struct {
		name string
		ids  []string
		want bool
	}{
		{"exact match", []string{"user:view", "user:edit"}, true},
		{"different order still matches", []string{"user:edit", "user:view"}, true},
		{"missing one", []string{"user:view"}, false},
		{"same length but wrong id", []string{"user:view", "other-id"}, false},
		{"too many", []string{"user:view", "user:edit", "extra"}, false},
		{"empty", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isFullCatalog(tt.ids, testCatalog); got != tt.want {
				t.Errorf("isFullCatalog(%v) = %v, want %v", tt.ids, got, tt.want)
			}
		})
	}
}
