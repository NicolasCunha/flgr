package handler

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/NicolasCunha/flgr/flgr-server/internal/database"
	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/repository"
	"github.com/NicolasCunha/flgr/flgr-server/internal/service"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("database.Open() returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := database.Migrate(db, "../../../migrations"); err != nil {
		t.Fatalf("database.Migrate() returned unexpected error: %v", err)
	}

	return db
}

func newTestServices(t *testing.T) (*service.UserService, *service.AuthService) {
	t.Helper()
	userService, authService, _ := newTestServicesWithDB(t)
	return userService, authService
}

// newTestServicesWithDB is newTestServices plus the underlying *sql.DB,
// for tests that also need to grant a real permission (e.g. via
// grantAdministrador) rather than just authenticate.
func newTestServicesWithDB(t *testing.T) (*service.UserService, *service.AuthService, *sql.DB) {
	t.Helper()
	db := newTestDB(t)
	userRepo := repository.NewUserRepository(db)
	authzService := service.NewAuthzService(repository.NewAuthzRepository(db))
	return service.NewUserService(userRepo, authzService),
		service.NewAuthService(userRepo, repository.NewSessionRepository(db), repository.NewLoginAttemptRepository(db)),
		db
}

// newTestUserAccessService builds a UserAccessService backed by db.
func newTestUserAccessService(db *sql.DB) *service.UserAccessService {
	return service.NewUserAccessService(
		repository.NewUserRepository(db),
		repository.NewUserProfileRepository(db),
		repository.NewUserPermissionRepository(db),
		repository.NewProfileRepository(db),
		repository.NewPermissionRepository(db),
	)
}

// grantAdministrador assigns userID the seeded Administrador profile
// (full permission catalog), for tests exercising an actor that needs
// real authorization to act on another user or manage profiles.
func grantAdministrador(t *testing.T, db *sql.DB, userID string) {
	t.Helper()
	err := repository.NewUserProfileRepository(db).ReplaceProfiles(context.Background(), userID, []string{model.AdministradorProfileID}, userID)
	if err != nil {
		t.Fatalf("grantAdministrador() returned unexpected error: %v", err)
	}
}
