package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Migrate applies every "*.up.sql" file under migrationsDir, in filename
// order, that isn't already recorded in schema_migrations. It's safe to
// call repeatedly — already-applied migrations are skipped.
func Migrate(db *sql.DB, migrationsDir string) error {
	if err := ensureSchemaMigrationsTable(db); err != nil {
		return fmt.Errorf("creating schema_migrations table: %w", err)
	}

	files, err := upMigrationFiles(migrationsDir)
	if err != nil {
		return fmt.Errorf("listing migrations: %w", err)
	}

	for _, file := range files {
		version := strings.TrimSuffix(filepath.Base(file), ".up.sql")

		applied, err := isApplied(db, version)
		if err != nil {
			return fmt.Errorf("checking migration %s: %w", version, err)
		}
		if applied {
			continue
		}

		if err := applyMigration(db, file, version); err != nil {
			return fmt.Errorf("applying migration %s: %w", version, err)
		}
	}

	return nil
}

func ensureSchemaMigrationsTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

func upMigrationFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}
		files = append(files, filepath.Join(dir, entry.Name()))
	}

	sort.Strings(files)
	return files, nil
}

func isApplied(db *sql.DB, version string) (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(1) FROM schema_migrations WHERE version = ?", version).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func applyMigration(db *sql.DB, file, version string) error {
	contents, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(string(contents)); err != nil {
		return err
	}

	if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", version); err != nil {
		return err
	}

	return tx.Commit()
}
