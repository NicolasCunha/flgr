package repository

import (
	"context"
	"testing"
)

// TestReplaceAssociations_DeleteError exercises replaceAssociations'
// DELETE-step error branch directly (same package), which none of the
// three concrete repositories can trigger through their own public API
// without a closed database (a different branch, tested elsewhere).
func TestReplaceAssociations_DeleteError(t *testing.T) {
	db := newTestDB(t)

	err := replaceAssociations(context.Background(), db, "does_not_exist_table", "owner_id", "target_id", "owner", []string{"target"}, "actor")
	if err == nil {
		t.Fatal("replaceAssociations() against a nonexistent table expected an error, got nil")
	}
}

// TestListAssociatedIDs_QueryError exercises listAssociatedIDs' query
// error branch directly, for the same reason as above.
func TestListAssociatedIDs_QueryError(t *testing.T) {
	db := newTestDB(t)

	_, err := listAssociatedIDs(context.Background(), db, "does_not_exist_table", "owner_id", "target_id", "owner")
	if err == nil {
		t.Fatal("listAssociatedIDs() against a nonexistent table expected an error, got nil")
	}
}
