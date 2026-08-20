package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/repository"
)

func TestAuthService_Login_Success(t *testing.T) {
	fixedNow(t)
	fixedID(t, "session-id")
	users := new(mockUserRepository)
	sessions := new(mockSessionRepository)
	attempts := new(mockLoginAttemptRepository)

	hash, err := hashPassword("correct-password")
	if err != nil {
		t.Fatalf("hashPassword() returned unexpected error: %v", err)
	}
	user := &model.User{ID: "u1", Email: "user@example.com", PasswordHash: hash, Status: model.UserStatusActive}

	attempts.On("GetByEmail", mock.Anything, "user@example.com").Return(nil, repository.ErrNotFound)
	users.On("GetByEmail", mock.Anything, "user@example.com").Return(user, nil)
	attempts.On("Reset", mock.Anything, "user@example.com").Return(nil)
	sessions.On("Create", mock.Anything, mock.MatchedBy(func(s *model.Session) bool {
		return s.UserID == "u1" && s.ID == "session-id"
	})).Return(nil)

	svc := NewAuthService(users, sessions, attempts)
	gotUser, token, err := svc.Login(context.Background(), " User@Example.com ", "correct-password")
	if err != nil {
		t.Fatalf("Login() returned unexpected error: %v", err)
	}
	if gotUser.ID != "u1" {
		t.Errorf("user.ID = %q, want %q", gotUser.ID, "u1")
	}
	if token == "" {
		t.Error("token is empty, want a generated token")
	}
}

func TestAuthService_Login_Locked(t *testing.T) {
	nowTs := fixedNow(t)
	users := new(mockUserRepository)
	sessions := new(mockSessionRepository)
	attempts := new(mockLoginAttemptRepository)

	future := nowTs.Add(5 * time.Minute)
	attempts.On("GetByEmail", mock.Anything, "locked@example.com").Return(&model.LoginAttempt{
		Email: "locked@example.com", FailedCount: 5, LockedUntil: &future,
	}, nil)

	svc := NewAuthService(users, sessions, attempts)
	_, _, err := svc.Login(context.Background(), "locked@example.com", "irrelevant")
	if !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("Login() error = %v, want ErrAccountLocked", err)
	}
	users.AssertNotCalled(t, "GetByEmail")
}

func TestAuthService_Login_LockExpired(t *testing.T) {
	nowTs := fixedNow(t)
	users := new(mockUserRepository)
	sessions := new(mockSessionRepository)
	attempts := new(mockLoginAttemptRepository)

	past := nowTs.Add(-5 * time.Minute)
	hash, _ := hashPassword("correct-password")
	user := &model.User{ID: "u1", Email: "was-locked@example.com", PasswordHash: hash, Status: model.UserStatusActive}

	attempts.On("GetByEmail", mock.Anything, "was-locked@example.com").Return(&model.LoginAttempt{
		Email: "was-locked@example.com", FailedCount: 5, LockedUntil: &past,
	}, nil)
	users.On("GetByEmail", mock.Anything, "was-locked@example.com").Return(user, nil)
	attempts.On("Reset", mock.Anything, "was-locked@example.com").Return(nil)
	sessions.On("Create", mock.Anything, mock.Anything).Return(nil)

	svc := NewAuthService(users, sessions, attempts)
	_, _, err := svc.Login(context.Background(), "was-locked@example.com", "correct-password")
	if err != nil {
		t.Fatalf("Login() returned unexpected error: %v", err)
	}
}

func TestAuthService_Login_AttemptsRepositoryError(t *testing.T) {
	users := new(mockUserRepository)
	sessions := new(mockSessionRepository)
	attempts := new(mockLoginAttemptRepository)
	attempts.On("GetByEmail", mock.Anything, "err@example.com").Return(nil, errors.New("db down"))

	svc := NewAuthService(users, sessions, attempts)
	_, _, err := svc.Login(context.Background(), "err@example.com", "irrelevant")
	if err == nil || errors.Is(err, ErrAccountLocked) {
		t.Fatalf("Login() error = %v, want a generic wrapped error", err)
	}
}

func TestAuthService_Login_UsersRepositoryError(t *testing.T) {
	users := new(mockUserRepository)
	sessions := new(mockSessionRepository)
	attempts := new(mockLoginAttemptRepository)
	attempts.On("GetByEmail", mock.Anything, "err@example.com").Return(nil, repository.ErrNotFound)
	users.On("GetByEmail", mock.Anything, "err@example.com").Return(nil, errors.New("db down"))

	svc := NewAuthService(users, sessions, attempts)
	_, _, err := svc.Login(context.Background(), "err@example.com", "irrelevant")
	if err == nil || errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want a generic wrapped error", err)
	}
}

func TestAuthService_Login_UnknownEmail_RecordsFailure(t *testing.T) {
	fixedNow(t)
	fixedID(t, "attempt-id")
	users := new(mockUserRepository)
	sessions := new(mockSessionRepository)
	attempts := new(mockLoginAttemptRepository)

	attempts.On("GetByEmail", mock.Anything, "unknown@example.com").Return(nil, repository.ErrNotFound)
	users.On("GetByEmail", mock.Anything, "unknown@example.com").Return(nil, repository.ErrNotFound)
	attempts.On("Upsert", mock.Anything, mock.MatchedBy(func(a *model.LoginAttempt) bool {
		return a.Email == "unknown@example.com" && a.FailedCount == 1 && a.LockedUntil == nil
	})).Return(nil)

	svc := NewAuthService(users, sessions, attempts)
	_, _, err := svc.Login(context.Background(), "unknown@example.com", "irrelevant")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want ErrInvalidCredentials", err)
	}
	attempts.AssertExpectations(t)
}

func TestAuthService_Login_WrongPassword_WithinWindow_Increments(t *testing.T) {
	nowTs := fixedNow(t)
	users := new(mockUserRepository)
	sessions := new(mockSessionRepository)
	attempts := new(mockLoginAttemptRepository)

	hash, _ := hashPassword("correct-password")
	user := &model.User{ID: "u1", Email: "wrong@example.com", PasswordHash: hash, Status: model.UserStatusActive}
	firstFailed := nowTs.Add(-1 * time.Minute)

	attempts.On("GetByEmail", mock.Anything, "wrong@example.com").Return(&model.LoginAttempt{
		ID: "existing-attempt", Email: "wrong@example.com", FailedCount: 3, FirstFailedAt: &firstFailed,
	}, nil)
	users.On("GetByEmail", mock.Anything, "wrong@example.com").Return(user, nil)
	attempts.On("Upsert", mock.Anything, mock.MatchedBy(func(a *model.LoginAttempt) bool {
		return a.ID == "existing-attempt" && a.FailedCount == 4 && a.LockedUntil == nil
	})).Return(nil)

	svc := NewAuthService(users, sessions, attempts)
	_, _, err := svc.Login(context.Background(), "wrong@example.com", "wrong-password")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want ErrInvalidCredentials", err)
	}
	attempts.AssertExpectations(t)
}

func TestAuthService_Login_WrongPassword_WindowExpired_Resets(t *testing.T) {
	nowTs := fixedNow(t)
	users := new(mockUserRepository)
	sessions := new(mockSessionRepository)
	attempts := new(mockLoginAttemptRepository)

	hash, _ := hashPassword("correct-password")
	user := &model.User{ID: "u1", Email: "stale@example.com", PasswordHash: hash, Status: model.UserStatusActive}
	firstFailed := nowTs.Add(-20 * time.Minute)

	attempts.On("GetByEmail", mock.Anything, "stale@example.com").Return(&model.LoginAttempt{
		ID: "existing-attempt", Email: "stale@example.com", FailedCount: 4, FirstFailedAt: &firstFailed,
	}, nil)
	users.On("GetByEmail", mock.Anything, "stale@example.com").Return(user, nil)
	attempts.On("Upsert", mock.Anything, mock.MatchedBy(func(a *model.LoginAttempt) bool {
		return a.ID == "existing-attempt" && a.FailedCount == 1
	})).Return(nil)

	svc := NewAuthService(users, sessions, attempts)
	_, _, err := svc.Login(context.Background(), "stale@example.com", "wrong-password")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want ErrInvalidCredentials", err)
	}
	attempts.AssertExpectations(t)
}

func TestAuthService_Login_ReachesLockThreshold(t *testing.T) {
	nowTs := fixedNow(t)
	users := new(mockUserRepository)
	sessions := new(mockSessionRepository)
	attempts := new(mockLoginAttemptRepository)

	hash, _ := hashPassword("correct-password")
	user := &model.User{ID: "u1", Email: "almost-locked@example.com", PasswordHash: hash, Status: model.UserStatusActive}
	firstFailed := nowTs.Add(-1 * time.Minute)

	attempts.On("GetByEmail", mock.Anything, "almost-locked@example.com").Return(&model.LoginAttempt{
		ID: "existing-attempt", Email: "almost-locked@example.com", FailedCount: 4, FirstFailedAt: &firstFailed,
	}, nil)
	users.On("GetByEmail", mock.Anything, "almost-locked@example.com").Return(user, nil)
	attempts.On("Upsert", mock.Anything, mock.MatchedBy(func(a *model.LoginAttempt) bool {
		return a.FailedCount == 5 && a.LockedUntil != nil && a.LockedUntil.Equal(nowTs.Add(lockDuration))
	})).Return(nil)

	svc := NewAuthService(users, sessions, attempts)
	_, _, err := svc.Login(context.Background(), "almost-locked@example.com", "wrong-password")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want ErrInvalidCredentials", err)
	}
	attempts.AssertExpectations(t)
}

func TestAuthService_Login_UpsertError(t *testing.T) {
	users := new(mockUserRepository)
	sessions := new(mockSessionRepository)
	attempts := new(mockLoginAttemptRepository)

	attempts.On("GetByEmail", mock.Anything, "err@example.com").Return(nil, repository.ErrNotFound)
	users.On("GetByEmail", mock.Anything, "err@example.com").Return(nil, repository.ErrNotFound)
	attempts.On("Upsert", mock.Anything, mock.Anything).Return(errors.New("db down"))

	svc := NewAuthService(users, sessions, attempts)
	_, _, err := svc.Login(context.Background(), "err@example.com", "irrelevant")
	if err == nil || errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want a generic wrapped error", err)
	}
}

func TestAuthService_Login_InactiveUser(t *testing.T) {
	users := new(mockUserRepository)
	sessions := new(mockSessionRepository)
	attempts := new(mockLoginAttemptRepository)

	hash, _ := hashPassword("correct-password")
	user := &model.User{ID: "u1", Email: "inactive@example.com", PasswordHash: hash, Status: model.UserStatusInactive}

	attempts.On("GetByEmail", mock.Anything, "inactive@example.com").Return(nil, repository.ErrNotFound)
	users.On("GetByEmail", mock.Anything, "inactive@example.com").Return(user, nil)

	svc := NewAuthService(users, sessions, attempts)
	_, _, err := svc.Login(context.Background(), "inactive@example.com", "correct-password")
	if !errors.Is(err, ErrUserInactive) {
		t.Fatalf("Login() error = %v, want ErrUserInactive", err)
	}
	attempts.AssertNotCalled(t, "Upsert")
}

func TestAuthService_Login_ResetError(t *testing.T) {
	users := new(mockUserRepository)
	sessions := new(mockSessionRepository)
	attempts := new(mockLoginAttemptRepository)

	hash, _ := hashPassword("correct-password")
	user := &model.User{ID: "u1", Email: "reset-err@example.com", PasswordHash: hash, Status: model.UserStatusActive}

	attempts.On("GetByEmail", mock.Anything, "reset-err@example.com").Return(nil, repository.ErrNotFound)
	users.On("GetByEmail", mock.Anything, "reset-err@example.com").Return(user, nil)
	attempts.On("Reset", mock.Anything, "reset-err@example.com").Return(errors.New("db down"))

	svc := NewAuthService(users, sessions, attempts)
	_, _, err := svc.Login(context.Background(), "reset-err@example.com", "correct-password")
	if err == nil {
		t.Fatal("Login() expected an error, got nil")
	}
}

func TestAuthService_Login_TokenGenerationError(t *testing.T) {
	users := new(mockUserRepository)
	sessions := new(mockSessionRepository)
	attempts := new(mockLoginAttemptRepository)

	hash, _ := hashPassword("correct-password")
	user := &model.User{ID: "u1", Email: "token-err@example.com", PasswordHash: hash, Status: model.UserStatusActive}

	attempts.On("GetByEmail", mock.Anything, "token-err@example.com").Return(nil, repository.ErrNotFound)
	users.On("GetByEmail", mock.Anything, "token-err@example.com").Return(user, nil)
	attempts.On("Reset", mock.Anything, "token-err@example.com").Return(nil)

	original := randRead
	randRead = func(b []byte) (int, error) { return 0, errors.New("no entropy") }
	t.Cleanup(func() { randRead = original })

	svc := NewAuthService(users, sessions, attempts)
	_, _, err := svc.Login(context.Background(), "token-err@example.com", "correct-password")
	if err == nil {
		t.Fatal("Login() expected an error, got nil")
	}
}

func TestAuthService_Login_SessionCreateError(t *testing.T) {
	users := new(mockUserRepository)
	sessions := new(mockSessionRepository)
	attempts := new(mockLoginAttemptRepository)

	hash, _ := hashPassword("correct-password")
	user := &model.User{ID: "u1", Email: "session-err@example.com", PasswordHash: hash, Status: model.UserStatusActive}

	attempts.On("GetByEmail", mock.Anything, "session-err@example.com").Return(nil, repository.ErrNotFound)
	users.On("GetByEmail", mock.Anything, "session-err@example.com").Return(user, nil)
	attempts.On("Reset", mock.Anything, "session-err@example.com").Return(nil)
	sessions.On("Create", mock.Anything, mock.Anything).Return(errors.New("db down"))

	svc := NewAuthService(users, sessions, attempts)
	_, _, err := svc.Login(context.Background(), "session-err@example.com", "correct-password")
	if err == nil {
		t.Fatal("Login() expected an error, got nil")
	}
}

func TestAuthService_Logout_Success(t *testing.T) {
	users := new(mockUserRepository)
	sessions := new(mockSessionRepository)
	attempts := new(mockLoginAttemptRepository)

	session := &model.Session{ID: "s1", TokenHash: hashToken("valid-token")}
	sessions.On("GetByTokenHash", mock.Anything, hashToken("valid-token")).Return(session, nil)
	sessions.On("Delete", mock.Anything, "s1").Return(nil)

	svc := NewAuthService(users, sessions, attempts)
	if err := svc.Logout(context.Background(), "valid-token"); err != nil {
		t.Fatalf("Logout() returned unexpected error: %v", err)
	}
}

func TestAuthService_Logout_AlreadyInvalid(t *testing.T) {
	users := new(mockUserRepository)
	sessions := new(mockSessionRepository)
	attempts := new(mockLoginAttemptRepository)
	sessions.On("GetByTokenHash", mock.Anything, mock.Anything).Return(nil, repository.ErrNotFound)

	svc := NewAuthService(users, sessions, attempts)
	if err := svc.Logout(context.Background(), "gone"); err != nil {
		t.Fatalf("Logout() returned unexpected error: %v", err)
	}
	sessions.AssertNotCalled(t, "Delete")
}

func TestAuthService_Logout_GetError(t *testing.T) {
	users := new(mockUserRepository)
	sessions := new(mockSessionRepository)
	attempts := new(mockLoginAttemptRepository)
	sessions.On("GetByTokenHash", mock.Anything, mock.Anything).Return(nil, errors.New("db down"))

	svc := NewAuthService(users, sessions, attempts)
	if err := svc.Logout(context.Background(), "token"); err == nil {
		t.Fatal("Logout() expected an error, got nil")
	}
}

func TestAuthService_Logout_DeleteError(t *testing.T) {
	users := new(mockUserRepository)
	sessions := new(mockSessionRepository)
	attempts := new(mockLoginAttemptRepository)
	session := &model.Session{ID: "s1"}
	sessions.On("GetByTokenHash", mock.Anything, mock.Anything).Return(session, nil)
	sessions.On("Delete", mock.Anything, "s1").Return(errors.New("db down"))

	svc := NewAuthService(users, sessions, attempts)
	if err := svc.Logout(context.Background(), "token"); err == nil {
		t.Fatal("Logout() expected an error, got nil")
	}
}

func TestAuthService_ValidateSession_Success(t *testing.T) {
	nowTs := fixedNow(t)
	users := new(mockUserRepository)
	sessions := new(mockSessionRepository)
	attempts := new(mockLoginAttemptRepository)

	session := &model.Session{ID: "s1", UserID: "u1", ExpiresAt: nowTs.Add(5 * time.Minute)}
	user := &model.User{ID: "u1", Status: model.UserStatusActive}
	sessions.On("GetByTokenHash", mock.Anything, mock.Anything).Return(session, nil)
	users.On("GetByID", mock.Anything, "u1").Return(user, nil)
	sessions.On("Touch", mock.Anything, "s1", nowTs, nowTs.Add(sessionInactivityWindow)).Return(nil)

	svc := NewAuthService(users, sessions, attempts)
	got, err := svc.ValidateSession(context.Background(), "token")
	if err != nil {
		t.Fatalf("ValidateSession() returned unexpected error: %v", err)
	}
	if got.ID != "u1" {
		t.Errorf("user.ID = %q, want %q", got.ID, "u1")
	}
}

func TestAuthService_ValidateSession_NotFound(t *testing.T) {
	users := new(mockUserRepository)
	sessions := new(mockSessionRepository)
	attempts := new(mockLoginAttemptRepository)
	sessions.On("GetByTokenHash", mock.Anything, mock.Anything).Return(nil, repository.ErrNotFound)

	svc := NewAuthService(users, sessions, attempts)
	_, err := svc.ValidateSession(context.Background(), "token")
	if !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("ValidateSession() error = %v, want ErrSessionInvalid", err)
	}
}

func TestAuthService_ValidateSession_GetError(t *testing.T) {
	users := new(mockUserRepository)
	sessions := new(mockSessionRepository)
	attempts := new(mockLoginAttemptRepository)
	sessions.On("GetByTokenHash", mock.Anything, mock.Anything).Return(nil, errors.New("db down"))

	svc := NewAuthService(users, sessions, attempts)
	_, err := svc.ValidateSession(context.Background(), "token")
	if err == nil || errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("ValidateSession() error = %v, want a generic wrapped error", err)
	}
}

func TestAuthService_ValidateSession_Expired(t *testing.T) {
	nowTs := fixedNow(t)
	users := new(mockUserRepository)
	sessions := new(mockSessionRepository)
	attempts := new(mockLoginAttemptRepository)
	session := &model.Session{ID: "s1", UserID: "u1", ExpiresAt: nowTs.Add(-1 * time.Minute)}
	sessions.On("GetByTokenHash", mock.Anything, mock.Anything).Return(session, nil)

	svc := NewAuthService(users, sessions, attempts)
	_, err := svc.ValidateSession(context.Background(), "token")
	if !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("ValidateSession() error = %v, want ErrSessionInvalid", err)
	}
	users.AssertNotCalled(t, "GetByID")
}

func TestAuthService_ValidateSession_UserNotFound(t *testing.T) {
	nowTs := fixedNow(t)
	users := new(mockUserRepository)
	sessions := new(mockSessionRepository)
	attempts := new(mockLoginAttemptRepository)
	session := &model.Session{ID: "s1", UserID: "u1", ExpiresAt: nowTs.Add(5 * time.Minute)}
	sessions.On("GetByTokenHash", mock.Anything, mock.Anything).Return(session, nil)
	users.On("GetByID", mock.Anything, "u1").Return(nil, repository.ErrNotFound)

	svc := NewAuthService(users, sessions, attempts)
	_, err := svc.ValidateSession(context.Background(), "token")
	if !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("ValidateSession() error = %v, want ErrSessionInvalid", err)
	}
}

func TestAuthService_ValidateSession_UserRepositoryError(t *testing.T) {
	nowTs := fixedNow(t)
	users := new(mockUserRepository)
	sessions := new(mockSessionRepository)
	attempts := new(mockLoginAttemptRepository)
	session := &model.Session{ID: "s1", UserID: "u1", ExpiresAt: nowTs.Add(5 * time.Minute)}
	sessions.On("GetByTokenHash", mock.Anything, mock.Anything).Return(session, nil)
	users.On("GetByID", mock.Anything, "u1").Return(nil, errors.New("db down"))

	svc := NewAuthService(users, sessions, attempts)
	_, err := svc.ValidateSession(context.Background(), "token")
	if err == nil || errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("ValidateSession() error = %v, want a generic wrapped error", err)
	}
}

func TestAuthService_ValidateSession_InactiveUser(t *testing.T) {
	nowTs := fixedNow(t)
	users := new(mockUserRepository)
	sessions := new(mockSessionRepository)
	attempts := new(mockLoginAttemptRepository)
	session := &model.Session{ID: "s1", UserID: "u1", ExpiresAt: nowTs.Add(5 * time.Minute)}
	user := &model.User{ID: "u1", Status: model.UserStatusInactive}
	sessions.On("GetByTokenHash", mock.Anything, mock.Anything).Return(session, nil)
	users.On("GetByID", mock.Anything, "u1").Return(user, nil)

	svc := NewAuthService(users, sessions, attempts)
	_, err := svc.ValidateSession(context.Background(), "token")
	if !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("ValidateSession() error = %v, want ErrSessionInvalid", err)
	}
	sessions.AssertNotCalled(t, "Touch")
}

func TestAuthService_ValidateSession_TouchError(t *testing.T) {
	nowTs := fixedNow(t)
	users := new(mockUserRepository)
	sessions := new(mockSessionRepository)
	attempts := new(mockLoginAttemptRepository)
	session := &model.Session{ID: "s1", UserID: "u1", ExpiresAt: nowTs.Add(5 * time.Minute)}
	user := &model.User{ID: "u1", Status: model.UserStatusActive}
	sessions.On("GetByTokenHash", mock.Anything, mock.Anything).Return(session, nil)
	users.On("GetByID", mock.Anything, "u1").Return(user, nil)
	sessions.On("Touch", mock.Anything, "s1", mock.Anything, mock.Anything).Return(errors.New("db down"))

	svc := NewAuthService(users, sessions, attempts)
	_, err := svc.ValidateSession(context.Background(), "token")
	if err == nil {
		t.Fatal("ValidateSession() expected an error, got nil")
	}
}
