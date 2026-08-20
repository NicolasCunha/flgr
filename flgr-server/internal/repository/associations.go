package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

// listAssociatedIDs returns targetColumn values from table for a given
// ownerID — the shared query shape behind ListProfileIDs/ListPermissionIDs
// on the three many-to-many repositories below. table/ownerColumn/
// targetColumn are always Go string literals from this package's own
// callers, never request data, so building the query with fmt.Sprintf is
// safe here (no injection surface).
func listAssociatedIDs(ctx context.Context, db *sql.DB, table, ownerColumn, targetColumn, ownerID string) ([]string, error) {
	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s = ?", targetColumn, table, ownerColumn)
	rows, err := db.QueryContext(ctx, query, ownerID)
	if err != nil {
		return nil, fmt.Errorf("listing %s: %w", table, err)
	}
	defer func() { _ = rows.Close() }()

	// A Scan failure here, or rows.Err() below, isn't reachable in a test
	// without a fault-injecting driver double, per the exception clause in
	// docs/architecture/adr/0004-testing-and-coverage-standards.md — a
	// single TEXT column scanning into a string can't fail with real data.
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning %s: %w", table, err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("listing %s: %w", table, err)
	}

	return ids, nil
}

// replaceAssociations atomically clears every table row for ownerID and
// inserts one fresh row per targetIDs, stamping actorUserID as both
// created_by_user_id and modified_by_user_id (these rows are always
// written by an authenticated User in this codebase's current phase, so
// the created_by_service_key_id/modified_by_service_key_id pairing from
// docs/architecture/adr/0005-audit-columns-and-soft-delete-convention.md
// is left null, same as every other write path today).
func replaceAssociations(ctx context.Context, db *sql.DB, table, ownerColumn, targetColumn, ownerID string, targetIDs []string, actorUserID string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning %s replacement: %w", table, err)
	}
	defer func() { _ = tx.Rollback() }()

	deleteQuery := fmt.Sprintf("DELETE FROM %s WHERE %s = ?", table, ownerColumn)
	if _, err := tx.ExecContext(ctx, deleteQuery, ownerID); err != nil {
		return fmt.Errorf("clearing %s: %w", table, err)
	}

	insertQuery := fmt.Sprintf(
		"INSERT INTO %s (id, %s, %s, created_by_user_id, modified_by_user_id) VALUES (?, ?, ?, ?, ?)",
		table, ownerColumn, targetColumn,
	)
	for _, targetID := range targetIDs {
		if _, err := tx.ExecContext(ctx, insertQuery, uuid.NewString(), ownerID, targetID, actorUserID, actorUserID); err != nil {
			return fmt.Errorf("inserting into %s: %w", table, err)
		}
	}

	// SQLite enforces constraints at each statement, not deferred to
	// COMMIT, so this failing isn't reachable in a test without a fault-
	// injecting driver double, per the exception clause in ADR-0004.
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing %s replacement: %w", table, err)
	}
	return nil
}
