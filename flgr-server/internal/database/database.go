// Package database owns opening the SQLite connection and applying
// migrations. A pure-Go driver (modernc.org/sqlite) is used deliberately,
// to avoid a cgo/C-toolchain dependency in the build — the same reasoning
// docs/architecture/adr/0009-kafka-and-webhook-notification-delivery.md
// applied to the Kafka client choice.
package database

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Open opens the SQLite database at path and enables foreign key
// enforcement, which SQLite does not turn on by default per connection.
// sqlOpen is a seam over sql.Open so its (practically unreachable) error
// path can still be exercised in a test, per the exception clause in
// docs/architecture/adr/0004-testing-and-coverage-standards.md.
var sqlOpen = sql.Open

func Open(path string) (*sql.DB, error) {
	db, err := sqlOpen("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// SQLite (outside WAL mode) only ever allows one writer, and without a
	// busy_timeout a second connection hitting a lock fails immediately
	// with SQLITE_BUSY ("database is locked") instead of waiting — exactly
	// what surfaced as sporadic 401s on concurrent requests (e.g. a
	// session Touch write racing a GET on another table). database/sql's
	// default pool opens a new connection per concurrent caller, and a
	// PRAGMA set via one db.Exec call only applies to whichever connection
	// happens to run it, not to every connection the pool may later open —
	// so pinning the pool to a single connection is what makes the WAL/
	// busy_timeout/foreign_keys PRAGMAs below actually apply to every
	// query, not just the lucky first one. flgr's request volume doesn't
	// need real write concurrency; callers now just queue briefly instead
	// of erroring.
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	// A successful Ping immediately followed by a failing PRAGMA would
	// require the database to become unavailable mid-function; not
	// reachable in a normal test without a fault-injecting driver double.
	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		return nil, fmt.Errorf("enabling WAL journal mode: %w", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout = 5000"); err != nil {
		return nil, fmt.Errorf("setting busy timeout: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("enabling foreign keys: %w", err)
	}

	return db, nil
}
