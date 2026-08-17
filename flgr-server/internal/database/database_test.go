package database

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestOpen_Success(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() returned unexpected error: %v", err)
	}
	defer func() { _ = db.Close() }()
}

func TestOpen_SQLOpenError(t *testing.T) {
	original := sqlOpen
	sqlOpen = func(driverName, dataSourceName string) (*sql.DB, error) {
		return nil, errors.New("boom")
	}
	defer func() { sqlOpen = original }()

	if _, err := Open("irrelevant"); err == nil {
		t.Fatal("Open() expected error when sqlOpen fails, got nil")
	}
}

func TestOpen_InvalidPath(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "does-not-exist", "test.db")

	_, err := Open(dbPath)
	if err == nil {
		t.Fatal("Open() expected error for a path in a nonexistent directory, got nil")
	}
}

func TestMigrate(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() returned unexpected error: %v", err)
	}
	defer func() { _ = db.Close() }()

	if err := Migrate(db, "../../migrations"); err != nil {
		t.Fatalf("Migrate() returned unexpected error: %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM environment_categories WHERE is_system = 1").Scan(&count); err != nil {
		t.Fatalf("querying seeded categories: %v", err)
	}
	if count != 3 {
		t.Errorf("seeded environment_categories count = %d, want 3", count)
	}

	var profileCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM profile_permissions WHERE profile_id = '00000000-0000-0000-0000-000000000010'").Scan(&profileCount); err != nil {
		t.Fatalf("querying seeded Administrador permissions: %v", err)
	}
	if profileCount != 22 {
		t.Errorf("seeded Administrador permission count = %d, want 22", profileCount)
	}

	// Re-running Migrate should be a no-op, not an error.
	if err := Migrate(db, "../../migrations"); err != nil {
		t.Fatalf("Migrate() (second run) returned unexpected error: %v", err)
	}
}

func TestMigrate_MissingDirectory(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() returned unexpected error: %v", err)
	}
	defer func() { _ = db.Close() }()

	if err := Migrate(db, "does-not-exist"); err == nil {
		t.Fatal("Migrate() expected error for missing migrations directory, got nil")
	}
}

func TestMigrate_ClosedDatabase(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() returned unexpected error: %v", err)
	}
	_ = db.Close()

	if err := Migrate(db, "../../migrations"); err == nil {
		t.Fatal("Migrate() expected error on a closed database, got nil")
	}
}

func TestIsApplied_ClosedDatabase(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() returned unexpected error: %v", err)
	}
	if err := ensureSchemaMigrationsTable(db); err != nil {
		t.Fatalf("ensureSchemaMigrationsTable() returned unexpected error: %v", err)
	}
	_ = db.Close()

	if _, err := isApplied(db, "000001_initial_schema"); err == nil {
		t.Fatal("isApplied() expected error on a closed database, got nil")
	}
}

func TestApplyMigration_MissingFile(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() returned unexpected error: %v", err)
	}
	defer func() { _ = db.Close() }()
	if err := ensureSchemaMigrationsTable(db); err != nil {
		t.Fatalf("ensureSchemaMigrationsTable() returned unexpected error: %v", err)
	}

	missing := filepath.Join(t.TempDir(), "missing.up.sql")
	if err := applyMigration(db, missing, "000000_missing"); err == nil {
		t.Fatal("applyMigration() expected error for a missing file, got nil")
	}
}

func TestApplyMigration_InvalidSQL(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() returned unexpected error: %v", err)
	}
	defer func() { _ = db.Close() }()
	if err := ensureSchemaMigrationsTable(db); err != nil {
		t.Fatalf("ensureSchemaMigrationsTable() returned unexpected error: %v", err)
	}

	badFile := filepath.Join(t.TempDir(), "bad.up.sql")
	if err := os.WriteFile(badFile, []byte("THIS IS NOT VALID SQL;"), 0o644); err != nil {
		t.Fatalf("writing bad migration file: %v", err)
	}

	if err := applyMigration(db, badFile, "000000_bad"); err == nil {
		t.Fatal("applyMigration() expected error for invalid SQL, got nil")
	}
}

func TestApplyMigration_ClosedDatabase(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() returned unexpected error: %v", err)
	}
	if err := ensureSchemaMigrationsTable(db); err != nil {
		t.Fatalf("ensureSchemaMigrationsTable() returned unexpected error: %v", err)
	}

	validFile := filepath.Join(t.TempDir(), "ok.up.sql")
	if err := os.WriteFile(validFile, []byte("SELECT 1;"), 0o644); err != nil {
		t.Fatalf("writing migration file: %v", err)
	}
	_ = db.Close()

	if err := applyMigration(db, validFile, "000000_ok"); err == nil {
		t.Fatal("applyMigration() expected error on a closed database, got nil")
	}
}

func TestMigrate_IsAppliedError(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() returned unexpected error: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Pre-create an incompatible schema_migrations table (no "version"
	// column): Migrate's own CREATE TABLE IF NOT EXISTS becomes a no-op,
	// but isApplied's query against a missing column fails.
	if _, err := db.Exec("CREATE TABLE schema_migrations (bogus TEXT)"); err != nil {
		t.Fatalf("pre-creating incompatible schema_migrations table: %v", err)
	}

	migDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(migDir, "000001_x.up.sql"), []byte("SELECT 1;"), 0o644); err != nil {
		t.Fatalf("writing migration file: %v", err)
	}

	if err := Migrate(db, migDir); err == nil {
		t.Fatal("Migrate() expected error when isApplied fails, got nil")
	}
}

func TestMigrate_ApplyMigrationError(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() returned unexpected error: %v", err)
	}
	defer func() { _ = db.Close() }()

	migDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(migDir, "000001_bad.up.sql"), []byte("THIS IS NOT VALID SQL;"), 0o644); err != nil {
		t.Fatalf("writing migration file: %v", err)
	}

	if err := Migrate(db, migDir); err == nil {
		t.Fatal("Migrate() expected error when a migration file contains invalid SQL, got nil")
	}
}

func TestApplyMigration_DuplicateVersion(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() returned unexpected error: %v", err)
	}
	defer func() { _ = db.Close() }()
	if err := ensureSchemaMigrationsTable(db); err != nil {
		t.Fatalf("ensureSchemaMigrationsTable() returned unexpected error: %v", err)
	}

	// A harmless, re-runnable statement, so the second call fails on the
	// schema_migrations insert specifically, not on re-executing the SQL.
	okFile := filepath.Join(t.TempDir(), "ok.up.sql")
	if err := os.WriteFile(okFile, []byte("SELECT 1;"), 0o644); err != nil {
		t.Fatalf("writing migration file: %v", err)
	}

	if err := applyMigration(db, okFile, "000000_ok"); err != nil {
		t.Fatalf("applyMigration() (first run) returned unexpected error: %v", err)
	}
	if err := applyMigration(db, okFile, "000000_ok"); err == nil {
		t.Fatal("applyMigration() expected error for a duplicate version, got nil")
	}
}
