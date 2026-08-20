package repository

import (
	"context"
	"testing"
)

func TestPermissionRepository_List(t *testing.T) {
	repo := NewPermissionRepository(newTestDB(t))

	permissions, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("List() returned unexpected error: %v", err)
	}
	if len(permissions) != 22 {
		t.Fatalf("len(permissions) = %d, want 22 (the seeded catalog)", len(permissions))
	}

	var found bool
	for _, p := range permissions {
		if p.ID == "user:view" {
			found = true
			if p.Resource != "User" || p.Action != "View" {
				t.Errorf("user:view = %+v, want Resource=User Action=View", p)
			}
		}
	}
	if !found {
		t.Error("expected the seeded user:view permission to be present")
	}
}

func TestPermissionRepository_List_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewPermissionRepository(db)
	_ = db.Close()

	if _, err := repo.List(context.Background()); err == nil {
		t.Fatal("List() on closed database expected an error, got nil")
	}
}
