package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/repository"
)

func newUserAccessServiceMocks() (*mockUserRepository, *mockUserProfileRepository, *mockUserPermissionRepository, *mockProfileRepository, *mockPermissionRepository) {
	return new(mockUserRepository), new(mockUserProfileRepository), new(mockUserPermissionRepository), new(mockProfileRepository), new(mockPermissionRepository)
}

func TestUserAccessService_ProfileIDs(t *testing.T) {
	users, userProfiles, userPerms, profiles, perms := newUserAccessServiceMocks()
	users.On("GetByID", mock.Anything, "u1").Return(&model.User{ID: "u1"}, nil)
	userProfiles.On("ListProfileIDs", mock.Anything, "u1").Return([]string{"p1"}, nil)

	svc := NewUserAccessService(users, userProfiles, userPerms, profiles, perms)
	ids, err := svc.ProfileIDs(context.Background(), "u1")
	if err != nil {
		t.Fatalf("ProfileIDs() returned unexpected error: %v", err)
	}
	if len(ids) != 1 || ids[0] != "p1" {
		t.Errorf("ProfileIDs() = %v, want [p1]", ids)
	}
}

func TestUserAccessService_ProfileIDs_UserNotFound(t *testing.T) {
	users, userProfiles, userPerms, profiles, perms := newUserAccessServiceMocks()
	users.On("GetByID", mock.Anything, "missing").Return(nil, repository.ErrNotFound)

	svc := NewUserAccessService(users, userProfiles, userPerms, profiles, perms)
	_, err := svc.ProfileIDs(context.Background(), "missing")
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("ProfileIDs() error = %v, want ErrUserNotFound", err)
	}
}

func TestUserAccessService_ProfileIDs_UserRepositoryError(t *testing.T) {
	users, userProfiles, userPerms, profiles, perms := newUserAccessServiceMocks()
	users.On("GetByID", mock.Anything, "u1").Return(nil, errors.New("db down"))

	svc := NewUserAccessService(users, userProfiles, userPerms, profiles, perms)
	_, err := svc.ProfileIDs(context.Background(), "u1")
	if err == nil || errors.Is(err, ErrUserNotFound) {
		t.Fatalf("ProfileIDs() error = %v, want a generic wrapped error", err)
	}
}

func TestUserAccessService_ProfileIDs_RepositoryError(t *testing.T) {
	users, userProfiles, userPerms, profiles, perms := newUserAccessServiceMocks()
	users.On("GetByID", mock.Anything, "u1").Return(&model.User{ID: "u1"}, nil)
	userProfiles.On("ListProfileIDs", mock.Anything, "u1").Return(nil, errors.New("db down"))

	svc := NewUserAccessService(users, userProfiles, userPerms, profiles, perms)
	if _, err := svc.ProfileIDs(context.Background(), "u1"); err == nil {
		t.Fatal("ProfileIDs() expected an error, got nil")
	}
}

func TestUserAccessService_ReplaceProfiles_NilActor(t *testing.T) {
	users, userProfiles, userPerms, profiles, perms := newUserAccessServiceMocks()
	svc := NewUserAccessService(users, userProfiles, userPerms, profiles, perms)

	err := svc.ReplaceProfiles(context.Background(), nil, "u1", []string{"p1"})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("ReplaceProfiles() error = %v, want ErrForbidden", err)
	}
}

func TestUserAccessService_ReplaceProfiles_UserNotFound(t *testing.T) {
	users, userProfiles, userPerms, profiles, perms := newUserAccessServiceMocks()
	users.On("GetByID", mock.Anything, "missing").Return(nil, repository.ErrNotFound)

	svc := NewUserAccessService(users, userProfiles, userPerms, profiles, perms)
	actor := &model.User{ID: "actor-id"}
	err := svc.ReplaceProfiles(context.Background(), actor, "missing", []string{"p1"})
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("ReplaceProfiles() error = %v, want ErrUserNotFound", err)
	}
}

func TestUserAccessService_ReplaceProfiles_UnknownProfile(t *testing.T) {
	users, userProfiles, userPerms, profiles, perms := newUserAccessServiceMocks()
	users.On("GetByID", mock.Anything, "u1").Return(&model.User{ID: "u1"}, nil)
	profiles.On("GetByID", mock.Anything, "does-not-exist").Return(nil, repository.ErrNotFound)

	svc := NewUserAccessService(users, userProfiles, userPerms, profiles, perms)
	actor := &model.User{ID: "actor-id"}
	err := svc.ReplaceProfiles(context.Background(), actor, "u1", []string{"does-not-exist"})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("ReplaceProfiles() error = %v, want ErrValidation", err)
	}
	userProfiles.AssertNotCalled(t, "ReplaceProfiles")
}

func TestUserAccessService_ReplaceProfiles_ProfileCheckError(t *testing.T) {
	users, userProfiles, userPerms, profiles, perms := newUserAccessServiceMocks()
	users.On("GetByID", mock.Anything, "u1").Return(&model.User{ID: "u1"}, nil)
	profiles.On("GetByID", mock.Anything, "p1").Return(nil, errors.New("db down"))

	svc := NewUserAccessService(users, userProfiles, userPerms, profiles, perms)
	actor := &model.User{ID: "actor-id"}
	err := svc.ReplaceProfiles(context.Background(), actor, "u1", []string{"p1"})
	if err == nil {
		t.Fatal("ReplaceProfiles() expected an error, got nil")
	}
}

func TestUserAccessService_ReplaceProfiles_Success(t *testing.T) {
	users, userProfiles, userPerms, profiles, perms := newUserAccessServiceMocks()
	users.On("GetByID", mock.Anything, "u1").Return(&model.User{ID: "u1"}, nil)
	profiles.On("GetByID", mock.Anything, "p1").Return(&model.Profile{ID: "p1"}, nil)
	userProfiles.On("ReplaceProfiles", mock.Anything, "u1", []string{"p1"}, "actor-id").Return(nil)

	svc := NewUserAccessService(users, userProfiles, userPerms, profiles, perms)
	actor := &model.User{ID: "actor-id"}
	if err := svc.ReplaceProfiles(context.Background(), actor, "u1", []string{"p1"}); err != nil {
		t.Fatalf("ReplaceProfiles() returned unexpected error: %v", err)
	}
}

func TestUserAccessService_ReplaceProfiles_RepositoryError(t *testing.T) {
	users, userProfiles, userPerms, profiles, perms := newUserAccessServiceMocks()
	users.On("GetByID", mock.Anything, "u1").Return(&model.User{ID: "u1"}, nil)
	profiles.On("GetByID", mock.Anything, "p1").Return(&model.Profile{ID: "p1"}, nil)
	userProfiles.On("ReplaceProfiles", mock.Anything, "u1", []string{"p1"}, "actor-id").Return(errors.New("db down"))

	svc := NewUserAccessService(users, userProfiles, userPerms, profiles, perms)
	actor := &model.User{ID: "actor-id"}
	if err := svc.ReplaceProfiles(context.Background(), actor, "u1", []string{"p1"}); err == nil {
		t.Fatal("ReplaceProfiles() expected an error, got nil")
	}
}

func TestUserAccessService_DirectPermissionIDs(t *testing.T) {
	users, userProfiles, userPerms, profiles, perms := newUserAccessServiceMocks()
	users.On("GetByID", mock.Anything, "u1").Return(&model.User{ID: "u1"}, nil)
	userPerms.On("ListPermissionIDs", mock.Anything, "u1").Return([]string{"user:view"}, nil)

	svc := NewUserAccessService(users, userProfiles, userPerms, profiles, perms)
	ids, err := svc.DirectPermissionIDs(context.Background(), "u1")
	if err != nil {
		t.Fatalf("DirectPermissionIDs() returned unexpected error: %v", err)
	}
	if len(ids) != 1 || ids[0] != "user:view" {
		t.Errorf("DirectPermissionIDs() = %v, want [user:view]", ids)
	}
}

func TestUserAccessService_DirectPermissionIDs_UserNotFound(t *testing.T) {
	users, userProfiles, userPerms, profiles, perms := newUserAccessServiceMocks()
	users.On("GetByID", mock.Anything, "missing").Return(nil, repository.ErrNotFound)

	svc := NewUserAccessService(users, userProfiles, userPerms, profiles, perms)
	_, err := svc.DirectPermissionIDs(context.Background(), "missing")
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("DirectPermissionIDs() error = %v, want ErrUserNotFound", err)
	}
}

func TestUserAccessService_DirectPermissionIDs_RepositoryError(t *testing.T) {
	users, userProfiles, userPerms, profiles, perms := newUserAccessServiceMocks()
	users.On("GetByID", mock.Anything, "u1").Return(&model.User{ID: "u1"}, nil)
	userPerms.On("ListPermissionIDs", mock.Anything, "u1").Return(nil, errors.New("db down"))

	svc := NewUserAccessService(users, userProfiles, userPerms, profiles, perms)
	if _, err := svc.DirectPermissionIDs(context.Background(), "u1"); err == nil {
		t.Fatal("DirectPermissionIDs() expected an error, got nil")
	}
}

func TestUserAccessService_ReplaceDirectPermissions_NilActor(t *testing.T) {
	users, userProfiles, userPerms, profiles, perms := newUserAccessServiceMocks()
	svc := NewUserAccessService(users, userProfiles, userPerms, profiles, perms)

	err := svc.ReplaceDirectPermissions(context.Background(), nil, "u1", []string{"user:view"})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("ReplaceDirectPermissions() error = %v, want ErrForbidden", err)
	}
}

func TestUserAccessService_ReplaceDirectPermissions_UserNotFound(t *testing.T) {
	users, userProfiles, userPerms, profiles, perms := newUserAccessServiceMocks()
	users.On("GetByID", mock.Anything, "missing").Return(nil, repository.ErrNotFound)

	svc := NewUserAccessService(users, userProfiles, userPerms, profiles, perms)
	actor := &model.User{ID: "actor-id"}
	err := svc.ReplaceDirectPermissions(context.Background(), actor, "missing", []string{"user:view"})
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("ReplaceDirectPermissions() error = %v, want ErrUserNotFound", err)
	}
}

func TestUserAccessService_ReplaceDirectPermissions_CatalogError(t *testing.T) {
	users, userProfiles, userPerms, profiles, perms := newUserAccessServiceMocks()
	users.On("GetByID", mock.Anything, "u1").Return(&model.User{ID: "u1"}, nil)
	perms.On("List", mock.Anything).Return(nil, errors.New("db down"))

	svc := NewUserAccessService(users, userProfiles, userPerms, profiles, perms)
	actor := &model.User{ID: "actor-id"}
	err := svc.ReplaceDirectPermissions(context.Background(), actor, "u1", []string{"user:view"})
	if err == nil {
		t.Fatal("ReplaceDirectPermissions() expected an error, got nil")
	}
}

func TestUserAccessService_ReplaceDirectPermissions_UnknownPermission(t *testing.T) {
	users, userProfiles, userPerms, profiles, perms := newUserAccessServiceMocks()
	users.On("GetByID", mock.Anything, "u1").Return(&model.User{ID: "u1"}, nil)
	perms.On("List", mock.Anything).Return(testCatalog, nil)

	svc := NewUserAccessService(users, userProfiles, userPerms, profiles, perms)
	actor := &model.User{ID: "actor-id"}
	err := svc.ReplaceDirectPermissions(context.Background(), actor, "u1", []string{"does-not-exist"})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("ReplaceDirectPermissions() error = %v, want ErrValidation", err)
	}
	userPerms.AssertNotCalled(t, "ReplacePermissions")
}

func TestUserAccessService_ReplaceDirectPermissions_Success(t *testing.T) {
	users, userProfiles, userPerms, profiles, perms := newUserAccessServiceMocks()
	users.On("GetByID", mock.Anything, "u1").Return(&model.User{ID: "u1"}, nil)
	perms.On("List", mock.Anything).Return(testCatalog, nil)
	userPerms.On("ReplacePermissions", mock.Anything, "u1", []string{"user:view"}, "actor-id").Return(nil)

	svc := NewUserAccessService(users, userProfiles, userPerms, profiles, perms)
	actor := &model.User{ID: "actor-id"}
	if err := svc.ReplaceDirectPermissions(context.Background(), actor, "u1", []string{"user:view"}); err != nil {
		t.Fatalf("ReplaceDirectPermissions() returned unexpected error: %v", err)
	}
}

func TestUserAccessService_ReplaceDirectPermissions_RepositoryError(t *testing.T) {
	users, userProfiles, userPerms, profiles, perms := newUserAccessServiceMocks()
	users.On("GetByID", mock.Anything, "u1").Return(&model.User{ID: "u1"}, nil)
	perms.On("List", mock.Anything).Return(testCatalog, nil)
	userPerms.On("ReplacePermissions", mock.Anything, "u1", []string{"user:view"}, "actor-id").Return(errors.New("db down"))

	svc := NewUserAccessService(users, userProfiles, userPerms, profiles, perms)
	actor := &model.User{ID: "actor-id"}
	if err := svc.ReplaceDirectPermissions(context.Background(), actor, "u1", []string{"user:view"}); err == nil {
		t.Fatal("ReplaceDirectPermissions() expected an error, got nil")
	}
}
