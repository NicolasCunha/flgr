package repository

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/NicolasCunha/flgr/flgr-server/internal/database"
)

// newTestDB opens a fresh, fully-migrated SQLite database in a temp
// directory, per backend.md's "repository layer tests run against a real
// SQLite database" convention.
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("database.Open() returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := database.Migrate(db, "../../migrations"); err != nil {
		t.Fatalf("database.Migrate() returned unexpected error: %v", err)
	}

	return db
}
