package api

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/NicolasCunha/flgr/flgr-server/internal/config"
	"github.com/NicolasCunha/flgr/flgr-server/internal/database"
)

// newTestDB opens a fresh, fully-migrated SQLite database in a temp
// directory, mirroring internal/repository's testutil_test.go.
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

// testEncryptionKey is a valid base64-encoded 32-byte AES-256 key (all
// zero bytes — this is test-only, never a real secret) satisfying
// service.DecodeEncryptionKey, which NewRouter now calls fail-fast at
// startup per ADR-0010. Every router_test.go test goes through NewRouter,
// so newTestConfig must supply one or the whole package's tests abort via
// log.Fatalf inside NewRouter.
const testEncryptionKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="

func newTestConfig() *config.Config {
	return &config.Config{SessionCookieSecure: false, EncryptionKey: testEncryptionKey}
}
