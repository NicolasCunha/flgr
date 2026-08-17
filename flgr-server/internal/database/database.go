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

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	// A successful Ping immediately followed by a failing PRAGMA would
	// require the database to become unavailable mid-function; not
	// reachable in a normal test without a fault-injecting driver double.
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("enabling foreign keys: %w", err)
	}

	return db, nil
}
