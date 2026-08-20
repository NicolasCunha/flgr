package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/repository"
)

func fixedNow(t *testing.T) time.Time {
	t.Helper()
	ts := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	original := now
	now = func() time.Time { return ts }
	t.Cleanup(func() { now = original })
	return ts
}

func fixedID(t *testing.T, id string) {
	t.Helper()
	original := newID
	newID = func() string { return id }
	t.Cleanup(func() { newID = original })
}

func TestUserService_Create_Success(t *testing.T) {
	fixedNow(t)
	fixedID(t, "new-user-id")
	repo := new(mockUserRepository)
	repo.On("GetByEmail", mock.Anything, "ada@example.com").Return(nil, repository.ErrNotFound)
	repo.On("Create", mock.Anything, mock.MatchedBy(func(u *model.User) bool {
		return u.ID == "new-user-id" && u.Email == "ada@example.com" && u.Status == model.UserStatusActive
	})).Return(nil)

	svc := NewUserService(repo, new(mockPermissionChecker))
	actor := &model.User{ID: "actor-id"}

	u, err := svc.Create(context.Background(), actor, CreateUserInput{
		FirstName: "Ada", LastName: "Lovelace", Email: "Ada@Example.com ", Password: "supersecret",
	})
	if err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}
	if u.Email != "ada@example.com" {
		t.Errorf("Email = %q, want normalized %q", u.Email, "ada@example.com")
	}
	if u.PasswordHash == "" || u.PasswordHash == "supersecret" {
		t.Errorf("PasswordHash = %q, want a bcrypt hash", u.PasswordHash)
	}
	if u.CreatedByUserID == nil || *u.CreatedByUserID != "actor-id" {
		t.Errorf("CreatedByUserID = %v, want %q", u.CreatedByUserID, "actor-id")
	}
	repo.AssertExpectations(t)
}

func TestUserService_Create_NoActor_Bootstrap(t *testing.T) {
	fixedNow(t)
	fixedID(t, "bootstrap-id")
	repo := new(mockUserRepository)
	repo.On("GetByEmail", mock.Anything, "admin@example.com").Return(nil, repository.ErrNotFound)
	repo.On("Create", mock.Anything, mock.Anything).Return(nil)

	svc := NewUserService(repo, new(mockPermissionChecker))
	u, err := svc.Create(context.Background(), nil, CreateUserInput{
		FirstName: "Admin", LastName: "Bootstrap", Email: "admin@example.com", Password: "supersecret",
	})
	if err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}
	if u.CreatedByUserID != nil {
		t.Errorf("CreatedByUserID = %v, want nil for a system-bootstrapped user", u.CreatedByUserID)
	}
}

func TestUserService_Create_ValidationErrors(t *testing.T) {
	repo := new(mockUserRepository)
	svc := NewUserService(repo, new(mockPermissionChecker))

	tests := []struct {
		name  string
		input CreateUserInput
	}{
		{"missing first name", CreateUserInput{LastName: "Lovelace", Email: "a@example.com", Password: "supersecret"}},
		{"missing last name", CreateUserInput{FirstName: "Ada", Email: "a@example.com", Password: "supersecret"}},
		{"missing email", CreateUserInput{FirstName: "Ada", LastName: "Lovelace", Password: "supersecret"}},
		{"malformed email", CreateUserInput{FirstName: "Ada", LastName: "Lovelace", Email: "not-an-email", Password: "supersecret"}},
		{"password too short", CreateUserInput{FirstName: "Ada", LastName: "Lovelace", Email: "a@example.com", Password: "short"}},
		{"password too long for bcrypt", CreateUserInput{FirstName: "Ada", LastName: "Lovelace", Email: "a@example.com", Password: strings.Repeat("x", 73)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Create(context.Background(), nil, tt.input)
			if !errors.Is(err, ErrValidation) {
				t.Fatalf("Create() error = %v, want ErrValidation", err)
			}
		})
	}
	repo.AssertNotCalled(t, "Create")
}

func TestUserService_Create_EmailAlreadyInUse_PreCheck(t *testing.T) {
	repo := new(mockUserRepository)
	repo.On("GetByEmail", mock.Anything, "taken@example.com").Return(&model.User{ID: "existing"}, nil)

	svc := NewUserService(repo, new(mockPermissionChecker))
	_, err := svc.Create(context.Background(), nil, CreateUserInput{
		FirstName: "Ada", LastName: "Lovelace", Email: "taken@example.com", Password: "supersecret",
	})
	if !errors.Is(err, ErrEmailInUse) {
		t.Fatalf("Create() error = %v, want ErrEmailInUse", err)
	}
	repo.AssertNotCalled(t, "Create")
}

func TestUserService_Create_EmailAlreadyInUse_RaceAtInsert(t *testing.T) {
	repo := new(mockUserRepository)
	repo.On("GetByEmail", mock.Anything, "race@example.com").Return(nil, repository.ErrNotFound)
	repo.On("Create", mock.Anything, mock.Anything).Return(repository.ErrEmailInUse)

	svc := NewUserService(repo, new(mockPermissionChecker))
	_, err := svc.Create(context.Background(), nil, CreateUserInput{
		FirstName: "Ada", LastName: "Lovelace", Email: "race@example.com", Password: "supersecret",
	})
	if !errors.Is(err, ErrEmailInUse) {
		t.Fatalf("Create() error = %v, want ErrEmailInUse", err)
	}
}

func TestUserService_Create_GetByEmailError(t *testing.T) {
	repo := new(mockUserRepository)
	repo.On("GetByEmail", mock.Anything, "boom@example.com").Return(nil, errors.New("db down"))

	svc := NewUserService(repo, new(mockPermissionChecker))
	_, err := svc.Create(context.Background(), nil, CreateUserInput{
		FirstName: "Ada", LastName: "Lovelace", Email: "boom@example.com", Password: "supersecret",
	})
	if err == nil || errors.Is(err, ErrEmailInUse) || errors.Is(err, ErrValidation) {
		t.Fatalf("Create() error = %v, want a generic wrapped error", err)
	}
}

func TestUserService_Create_RepositoryError(t *testing.T) {
	repo := new(mockUserRepository)
	repo.On("GetByEmail", mock.Anything, "err@example.com").Return(nil, repository.ErrNotFound)
	repo.On("Create", mock.Anything, mock.Anything).Return(errors.New("disk full"))

	svc := NewUserService(repo, new(mockPermissionChecker))
	_, err := svc.Create(context.Background(), nil, CreateUserInput{
		FirstName: "Ada", LastName: "Lovelace", Email: "err@example.com", Password: "supersecret",
	})
	if err == nil || errors.Is(err, ErrEmailInUse) {
		t.Fatalf("Create() error = %v, want a generic wrapped error", err)
	}
}

func TestUserService_Get_Self(t *testing.T) {
	repo := new(mockUserRepository)
	want := &model.User{ID: "u1", Email: "u1@example.com"}
	repo.On("GetByID", mock.Anything, "u1").Return(want, nil)

	svc := NewUserService(repo, new(mockPermissionChecker))
	actor := &model.User{ID: "u1"}
	got, err := svc.Get(context.Background(), actor, "u1")
	if err != nil {
		t.Fatalf("Get() returned unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("Get() = %v, want %v", got, want)
	}
}

func TestUserService_Get_NilActor(t *testing.T) {
	repo := new(mockUserRepository)
	svc := NewUserService(repo, new(mockPermissionChecker))

	_, err := svc.Get(context.Background(), nil, "u1")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Get() error = %v, want ErrForbidden", err)
	}
	repo.AssertNotCalled(t, "GetByID")
}

func TestUserService_Get_Other_Allowed(t *testing.T) {
	repo := new(mockUserRepository)
	want := &model.User{ID: "u1", Email: "u1@example.com"}
	repo.On("GetByID", mock.Anything, "u1").Return(want, nil)
	authz := new(mockPermissionChecker)
	authz.On("HasPermission", mock.Anything, "admin-id", model.ResourceUser, model.ActionView).Return(true, nil)

	svc := NewUserService(repo, authz)
	actor := &model.User{ID: "admin-id"}
	got, err := svc.Get(context.Background(), actor, "u1")
	if err != nil {
		t.Fatalf("Get() returned unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("Get() = %v, want %v", got, want)
	}
}

func TestUserService_Get_Other_Forbidden(t *testing.T) {
	repo := new(mockUserRepository)
	authz := new(mockPermissionChecker)
	authz.On("HasPermission", mock.Anything, "admin-id", model.ResourceUser, model.ActionView).Return(false, nil)

	svc := NewUserService(repo, authz)
	actor := &model.User{ID: "admin-id"}
	_, err := svc.Get(context.Background(), actor, "u1")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Get() error = %v, want ErrForbidden", err)
	}
	repo.AssertNotCalled(t, "GetByID")
}

func TestUserService_Get_Other_AuthzError(t *testing.T) {
	repo := new(mockUserRepository)
	authz := new(mockPermissionChecker)
	authz.On("HasPermission", mock.Anything, "admin-id", model.ResourceUser, model.ActionView).Return(false, errors.New("db down"))

	svc := NewUserService(repo, authz)
	actor := &model.User{ID: "admin-id"}
	_, err := svc.Get(context.Background(), actor, "u1")
	if err == nil || errors.Is(err, ErrForbidden) {
		t.Fatalf("Get() error = %v, want a generic wrapped error", err)
	}
}

func TestUserService_Get_NotFound(t *testing.T) {
	repo := new(mockUserRepository)
	repo.On("GetByID", mock.Anything, "missing").Return(nil, repository.ErrNotFound)

	svc := NewUserService(repo, new(mockPermissionChecker))
	actor := &model.User{ID: "missing"}
	_, err := svc.Get(context.Background(), actor, "missing")
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("Get() error = %v, want ErrUserNotFound", err)
	}
}

func TestUserService_Get_RepositoryError(t *testing.T) {
	repo := new(mockUserRepository)
	repo.On("GetByID", mock.Anything, "u1").Return(nil, errors.New("db down"))

	svc := NewUserService(repo, new(mockPermissionChecker))
	actor := &model.User{ID: "u1"}
	_, err := svc.Get(context.Background(), actor, "u1")
	if err == nil || errors.Is(err, ErrUserNotFound) {
		t.Fatalf("Get() error = %v, want a generic wrapped error", err)
	}
}

func TestUserService_List_Defaults(t *testing.T) {
	repo := new(mockUserRepository)
	repo.On("List", mock.Anything, defaultPageSize, 0).Return([]model.User{{ID: "u1"}}, 1, nil)

	svc := NewUserService(repo, new(mockPermissionChecker))
	users, total, err := svc.List(context.Background(), 0, 0)
	if err != nil {
		t.Fatalf("List() returned unexpected error: %v", err)
	}
	if total != 1 || len(users) != 1 {
		t.Errorf("List() = (%v, %d), want 1 user, total 1", users, total)
	}
}

func TestUserService_List_PageSizeCapped(t *testing.T) {
	repo := new(mockUserRepository)
	repo.On("List", mock.Anything, maxPageSize, maxPageSize*2).Return([]model.User{}, 0, nil)

	svc := NewUserService(repo, new(mockPermissionChecker))
	if _, _, err := svc.List(context.Background(), 3, 1000); err != nil {
		t.Fatalf("List() returned unexpected error: %v", err)
	}
	repo.AssertExpectations(t)
}

func TestUserService_List_RepositoryError(t *testing.T) {
	repo := new(mockUserRepository)
	repo.On("List", mock.Anything, defaultPageSize, 0).Return(nil, 0, errors.New("db down"))

	svc := NewUserService(repo, new(mockPermissionChecker))
	if _, _, err := svc.List(context.Background(), 1, defaultPageSize); err == nil {
		t.Fatal("List() expected an error, got nil")
	}
}

func TestUserService_Update_Self(t *testing.T) {
	fixedNow(t)
	repo := new(mockUserRepository)
	existing := &model.User{ID: "self-id", FirstName: "Old", LastName: "Name", Email: "old@example.com", Status: model.UserStatusActive}
	repo.On("GetByID", mock.Anything, "self-id").Return(existing, nil)
	repo.On("Update", mock.Anything, mock.MatchedBy(func(u *model.User) bool {
		return u.FirstName == "New" && u.ModifiedByUserID != nil && *u.ModifiedByUserID == "self-id"
	})).Return(nil)

	svc := NewUserService(repo, new(mockPermissionChecker))
	actor := &model.User{ID: "self-id"}
	newFirst := "New"

	got, err := svc.Update(context.Background(), actor, "self-id", UpdateUserInput{FirstName: &newFirst})
	if err != nil {
		t.Fatalf("Update() returned unexpected error: %v", err)
	}
	if got.FirstName != "New" {
		t.Errorf("FirstName = %q, want %q", got.FirstName, "New")
	}
}

func TestUserService_Update_LastName(t *testing.T) {
	fixedNow(t)
	repo := new(mockUserRepository)
	existing := &model.User{ID: "self-id", FirstName: "Old", LastName: "Name", Email: "old@example.com", Status: model.UserStatusActive}
	repo.On("GetByID", mock.Anything, "self-id").Return(existing, nil)
	repo.On("Update", mock.Anything, mock.MatchedBy(func(u *model.User) bool {
		return u.LastName == "NewLastName"
	})).Return(nil)

	svc := NewUserService(repo, new(mockPermissionChecker))
	actor := &model.User{ID: "self-id"}
	newLast := "NewLastName"

	got, err := svc.Update(context.Background(), actor, "self-id", UpdateUserInput{LastName: &newLast})
	if err != nil {
		t.Fatalf("Update() returned unexpected error: %v", err)
	}
	if got.LastName != "NewLastName" {
		t.Errorf("LastName = %q, want %q", got.LastName, "NewLastName")
	}
}

func TestUserService_Update_SelfCannotChangeStatus(t *testing.T) {
	repo := new(mockUserRepository)
	existing := &model.User{ID: "self-id", FirstName: "A", LastName: "B", Email: "a@example.com", Status: model.UserStatusActive}
	repo.On("GetByID", mock.Anything, "self-id").Return(existing, nil)

	svc := NewUserService(repo, new(mockPermissionChecker))
	actor := &model.User{ID: "self-id"}
	inactive := model.UserStatusInactive

	_, err := svc.Update(context.Background(), actor, "self-id", UpdateUserInput{Status: &inactive})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Update() error = %v, want ErrForbidden", err)
	}
	repo.AssertNotCalled(t, "Update")
}

func TestUserService_Update_OtherUser_Status(t *testing.T) {
	fixedNow(t)
	repo := new(mockUserRepository)
	existing := &model.User{ID: "target-id", FirstName: "A", LastName: "B", Email: "a@example.com", Status: model.UserStatusActive}
	repo.On("GetByID", mock.Anything, "target-id").Return(existing, nil)
	repo.On("Update", mock.Anything, mock.MatchedBy(func(u *model.User) bool {
		return u.Status == model.UserStatusInactive
	})).Return(nil)

	svc := NewUserService(repo, allowingUserEditAuthz())
	actor := &model.User{ID: "admin-id"}
	inactive := model.UserStatusInactive

	got, err := svc.Update(context.Background(), actor, "target-id", UpdateUserInput{Status: &inactive})
	if err != nil {
		t.Fatalf("Update() returned unexpected error: %v", err)
	}
	if got.Status != model.UserStatusInactive {
		t.Errorf("Status = %q, want %q", got.Status, model.UserStatusInactive)
	}
}

func TestUserService_Update_OtherUser_Forbidden(t *testing.T) {
	repo := new(mockUserRepository)
	existing := &model.User{ID: "target-id", FirstName: "A", LastName: "B", Email: "a@example.com", Status: model.UserStatusActive}
	repo.On("GetByID", mock.Anything, "target-id").Return(existing, nil)
	authz := new(mockPermissionChecker)
	authz.On("HasPermission", mock.Anything, "admin-id", model.ResourceUser, model.ActionEdit).Return(false, nil)

	svc := NewUserService(repo, authz)
	actor := &model.User{ID: "admin-id"}
	newFirst := "New"

	_, err := svc.Update(context.Background(), actor, "target-id", UpdateUserInput{FirstName: &newFirst})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Update() error = %v, want ErrForbidden", err)
	}
	repo.AssertNotCalled(t, "Update")
}

func TestUserService_Update_OtherUser_AuthzError(t *testing.T) {
	repo := new(mockUserRepository)
	existing := &model.User{ID: "target-id", FirstName: "A", LastName: "B", Email: "a@example.com", Status: model.UserStatusActive}
	repo.On("GetByID", mock.Anything, "target-id").Return(existing, nil)
	authz := new(mockPermissionChecker)
	authz.On("HasPermission", mock.Anything, "admin-id", model.ResourceUser, model.ActionEdit).Return(false, errors.New("db down"))

	svc := NewUserService(repo, authz)
	actor := &model.User{ID: "admin-id"}
	newFirst := "New"

	_, err := svc.Update(context.Background(), actor, "target-id", UpdateUserInput{FirstName: &newFirst})
	if err == nil || errors.Is(err, ErrForbidden) {
		t.Fatalf("Update() error = %v, want a generic wrapped error", err)
	}
}

func TestUserService_Update_InvalidStatus(t *testing.T) {
	repo := new(mockUserRepository)
	existing := &model.User{ID: "target-id", FirstName: "A", LastName: "B", Email: "a@example.com"}
	repo.On("GetByID", mock.Anything, "target-id").Return(existing, nil)

	svc := NewUserService(repo, allowingUserEditAuthz())
	actor := &model.User{ID: "admin-id"}
	bogus := "Bogus"

	_, err := svc.Update(context.Background(), actor, "target-id", UpdateUserInput{Status: &bogus})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Update() error = %v, want ErrValidation", err)
	}
}

func TestUserService_Update_NilActor(t *testing.T) {
	repo := new(mockUserRepository)
	svc := NewUserService(repo, new(mockPermissionChecker))

	_, err := svc.Update(context.Background(), nil, "target-id", UpdateUserInput{})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Update() error = %v, want ErrForbidden", err)
	}
	repo.AssertNotCalled(t, "GetByID")
}

func TestUserService_Update_NotFound(t *testing.T) {
	repo := new(mockUserRepository)
	repo.On("GetByID", mock.Anything, "missing").Return(nil, repository.ErrNotFound)

	svc := NewUserService(repo, new(mockPermissionChecker))
	actor := &model.User{ID: "admin-id"}
	_, err := svc.Update(context.Background(), actor, "missing", UpdateUserInput{})
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("Update() error = %v, want ErrUserNotFound", err)
	}
}

func TestUserService_Update_GetByIDError(t *testing.T) {
	repo := new(mockUserRepository)
	repo.On("GetByID", mock.Anything, "u1").Return(nil, errors.New("db down"))

	svc := NewUserService(repo, new(mockPermissionChecker))
	actor := &model.User{ID: "admin-id"}
	_, err := svc.Update(context.Background(), actor, "u1", UpdateUserInput{})
	if err == nil || errors.Is(err, ErrUserNotFound) {
		t.Fatalf("Update() error = %v, want a generic wrapped error", err)
	}
}

func TestUserService_Update_ValidationErrors(t *testing.T) {
	existing := &model.User{ID: "target-id", FirstName: "A", LastName: "B", Email: "a@example.com", Status: model.UserStatusActive}
	actor := &model.User{ID: "admin-id"}

	tests := []struct {
		name  string
		input UpdateUserInput
	}{
		{"blank first name", UpdateUserInput{FirstName: strPtr("  ")}},
		{"malformed email", UpdateUserInput{Email: strPtr("not-an-email")}},
		{"password too short", UpdateUserInput{Password: strPtr("short")}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockUserRepository)
			repo.On("GetByID", mock.Anything, "target-id").Return(&model.User{
				ID: existing.ID, FirstName: existing.FirstName, LastName: existing.LastName,
				Email: existing.Email, Status: existing.Status,
			}, nil)

			svc := NewUserService(repo, allowingUserEditAuthz())
			_, err := svc.Update(context.Background(), actor, "target-id", tt.input)
			if !errors.Is(err, ErrValidation) {
				t.Fatalf("Update() error = %v, want ErrValidation", err)
			}
			repo.AssertNotCalled(t, "Update")
		})
	}
}

func TestUserService_Update_EmailInUse(t *testing.T) {
	repo := new(mockUserRepository)
	repo.On("GetByID", mock.Anything, "target-id").Return(&model.User{
		ID: "target-id", FirstName: "A", LastName: "B", Email: "a@example.com", Status: model.UserStatusActive,
	}, nil)
	repo.On("Update", mock.Anything, mock.Anything).Return(repository.ErrEmailInUse)

	svc := NewUserService(repo, allowingUserEditAuthz())
	actor := &model.User{ID: "admin-id"}
	newEmail := "taken@example.com"

	_, err := svc.Update(context.Background(), actor, "target-id", UpdateUserInput{Email: &newEmail})
	if !errors.Is(err, ErrEmailInUse) {
		t.Fatalf("Update() error = %v, want ErrEmailInUse", err)
	}
}

func TestUserService_Update_RepositoryNotFoundRace(t *testing.T) {
	repo := new(mockUserRepository)
	repo.On("GetByID", mock.Anything, "target-id").Return(&model.User{
		ID: "target-id", FirstName: "A", LastName: "B", Email: "a@example.com", Status: model.UserStatusActive,
	}, nil)
	repo.On("Update", mock.Anything, mock.Anything).Return(repository.ErrNotFound)

	svc := NewUserService(repo, allowingUserEditAuthz())
	actor := &model.User{ID: "admin-id"}

	_, err := svc.Update(context.Background(), actor, "target-id", UpdateUserInput{})
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("Update() error = %v, want ErrUserNotFound", err)
	}
}

func TestUserService_Update_RepositoryGenericError(t *testing.T) {
	repo := new(mockUserRepository)
	repo.On("GetByID", mock.Anything, "target-id").Return(&model.User{
		ID: "target-id", FirstName: "A", LastName: "B", Email: "a@example.com", Status: model.UserStatusActive,
	}, nil)
	repo.On("Update", mock.Anything, mock.Anything).Return(errors.New("disk full"))

	svc := NewUserService(repo, allowingUserEditAuthz())
	actor := &model.User{ID: "admin-id"}

	_, err := svc.Update(context.Background(), actor, "target-id", UpdateUserInput{})
	if err == nil || errors.Is(err, ErrEmailInUse) || errors.Is(err, ErrUserNotFound) {
		t.Fatalf("Update() error = %v, want a generic wrapped error", err)
	}
}

func TestUserService_Update_PasswordChanged(t *testing.T) {
	fixedNow(t)
	repo := new(mockUserRepository)
	repo.On("GetByID", mock.Anything, "target-id").Return(&model.User{
		ID: "target-id", FirstName: "A", LastName: "B", Email: "a@example.com", Status: model.UserStatusActive, PasswordHash: "old-hash",
	}, nil)
	repo.On("Update", mock.Anything, mock.MatchedBy(func(u *model.User) bool {
		return u.PasswordHash != "old-hash" && u.PasswordHash != "newpassword123"
	})).Return(nil)

	svc := NewUserService(repo, new(mockPermissionChecker))
	actor := &model.User{ID: "target-id"}
	newPassword := "newpassword123"

	_, err := svc.Update(context.Background(), actor, "target-id", UpdateUserInput{Password: &newPassword})
	if err != nil {
		t.Fatalf("Update() returned unexpected error: %v", err)
	}
}

func TestUserService_Update_PasswordTooLongForBcrypt(t *testing.T) {
	repo := new(mockUserRepository)
	repo.On("GetByID", mock.Anything, "target-id").Return(&model.User{
		ID: "target-id", FirstName: "A", LastName: "B", Email: "a@example.com", Status: model.UserStatusActive,
	}, nil)

	svc := NewUserService(repo, new(mockPermissionChecker))
	actor := &model.User{ID: "target-id"}
	tooLong := strings.Repeat("x", 73)

	_, err := svc.Update(context.Background(), actor, "target-id", UpdateUserInput{Password: &tooLong})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Update() error = %v, want ErrValidation", err)
	}
}

func TestUserService_Deactivate_NilActor(t *testing.T) {
	repo := new(mockUserRepository)
	svc := NewUserService(repo, new(mockPermissionChecker))

	_, err := svc.Deactivate(context.Background(), nil, "target-id")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Deactivate() error = %v, want ErrForbidden", err)
	}
}

func TestUserService_Deactivate_Success(t *testing.T) {
	fixedNow(t)
	repo := new(mockUserRepository)
	repo.On("GetByID", mock.Anything, "target-id").Return(&model.User{
		ID: "target-id", FirstName: "A", LastName: "B", Email: "a@example.com", Status: model.UserStatusActive,
	}, nil)
	repo.On("Update", mock.Anything, mock.MatchedBy(func(u *model.User) bool {
		return u.Status == model.UserStatusInactive
	})).Return(nil)

	svc := NewUserService(repo, new(mockPermissionChecker))
	actor := &model.User{ID: "admin-id"}

	got, err := svc.Deactivate(context.Background(), actor, "target-id")
	if err != nil {
		t.Fatalf("Deactivate() returned unexpected error: %v", err)
	}
	if got.Status != model.UserStatusInactive {
		t.Errorf("Status = %q, want %q", got.Status, model.UserStatusInactive)
	}
}

func TestUserService_Deactivate_NotFound(t *testing.T) {
	repo := new(mockUserRepository)
	repo.On("GetByID", mock.Anything, "missing").Return(nil, repository.ErrNotFound)
	svc := NewUserService(repo, new(mockPermissionChecker))
	actor := &model.User{ID: "admin-id"}

	_, err := svc.Deactivate(context.Background(), actor, "missing")
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("Deactivate() error = %v, want ErrUserNotFound", err)
	}
}

func TestUserService_Deactivate_GetByIDError(t *testing.T) {
	repo := new(mockUserRepository)
	repo.On("GetByID", mock.Anything, "target-id").Return(nil, errors.New("db down"))
	svc := NewUserService(repo, new(mockPermissionChecker))
	actor := &model.User{ID: "admin-id"}

	_, err := svc.Deactivate(context.Background(), actor, "target-id")
	if err == nil || errors.Is(err, ErrUserNotFound) {
		t.Fatalf("Deactivate() error = %v, want a generic wrapped error", err)
	}
}

func TestUserService_SetupNeeded_True(t *testing.T) {
	repo := new(mockUserRepository)
	repo.On("List", mock.Anything, 1, 0).Return([]model.User{}, 0, nil)

	svc := NewUserService(repo, new(mockPermissionChecker))
	needed, err := svc.SetupNeeded(context.Background())
	if err != nil {
		t.Fatalf("SetupNeeded() returned unexpected error: %v", err)
	}
	if !needed {
		t.Error("SetupNeeded() = false, want true when no users exist")
	}
}

func TestUserService_SetupNeeded_False(t *testing.T) {
	repo := new(mockUserRepository)
	repo.On("List", mock.Anything, 1, 0).Return([]model.User{{ID: "existing"}}, 1, nil)

	svc := NewUserService(repo, new(mockPermissionChecker))
	needed, err := svc.SetupNeeded(context.Background())
	if err != nil {
		t.Fatalf("SetupNeeded() returned unexpected error: %v", err)
	}
	if needed {
		t.Error("SetupNeeded() = true, want false when a user already exists")
	}
}

func TestUserService_SetupNeeded_RepositoryError(t *testing.T) {
	repo := new(mockUserRepository)
	repo.On("List", mock.Anything, 1, 0).Return(nil, 0, errors.New("db down"))

	svc := NewUserService(repo, new(mockPermissionChecker))
	if _, err := svc.SetupNeeded(context.Background()); err == nil {
		t.Fatal("SetupNeeded() expected an error, got nil")
	}
}

func TestUserService_CompleteSetup_AlreadyComplete(t *testing.T) {
	repo := new(mockUserRepository)
	repo.On("List", mock.Anything, 1, 0).Return([]model.User{{ID: "existing"}}, 1, nil)

	svc := NewUserService(repo, new(mockPermissionChecker))
	_, err := svc.CompleteSetup(context.Background(), CreateUserInput{
		FirstName: "Ada", LastName: "Lovelace", Email: "admin@example.com", Password: "supersecret",
	})
	if !errors.Is(err, ErrSetupAlreadyComplete) {
		t.Fatalf("CompleteSetup() error = %v, want ErrSetupAlreadyComplete", err)
	}
	repo.AssertNotCalled(t, "Create")
}

func TestUserService_CompleteSetup_SetupNeededError(t *testing.T) {
	repo := new(mockUserRepository)
	repo.On("List", mock.Anything, 1, 0).Return(nil, 0, errors.New("db down"))

	svc := NewUserService(repo, new(mockPermissionChecker))
	_, err := svc.CompleteSetup(context.Background(), CreateUserInput{
		FirstName: "Ada", LastName: "Lovelace", Email: "admin@example.com", Password: "supersecret",
	})
	if err == nil || errors.Is(err, ErrSetupAlreadyComplete) {
		t.Fatalf("CompleteSetup() error = %v, want a generic wrapped error", err)
	}
}

func TestUserService_CompleteSetup_Creates(t *testing.T) {
	fixedNow(t)
	fixedID(t, "first-user-id")
	repo := new(mockUserRepository)
	repo.On("List", mock.Anything, 1, 0).Return([]model.User{}, 0, nil)
	repo.On("GetByEmail", mock.Anything, "admin@example.com").Return(nil, repository.ErrNotFound)
	repo.On("Create", mock.Anything, mock.MatchedBy(func(u *model.User) bool {
		return u.Email == "admin@example.com" && u.CreatedByUserID == nil
	})).Return(nil)

	svc := NewUserService(repo, new(mockPermissionChecker))
	u, err := svc.CompleteSetup(context.Background(), CreateUserInput{
		FirstName: "Ada", LastName: "Lovelace", Email: "admin@example.com", Password: "supersecret",
	})
	if err != nil {
		t.Fatalf("CompleteSetup() returned unexpected error: %v", err)
	}
	if u == nil || u.Email != "admin@example.com" {
		t.Fatalf("CompleteSetup() = %v, want the created admin user", u)
	}
}

func strPtr(s string) *string { return &s }

// allowingUserEditAuthz returns a mockPermissionChecker that grants
// User:Edit to any actor — used by Update tests exercising the "other
// user" branch where authorization itself isn't what's under test.
func allowingUserEditAuthz() *mockPermissionChecker {
	m := new(mockPermissionChecker)
	m.On("HasPermission", mock.Anything, mock.Anything, model.ResourceUser, model.ActionEdit).Return(true, nil)
	return m
}
