package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
)

func newTestFeatureFlagEnvironmentValue(featureFlagID, environmentID string) *model.FeatureFlagEnvironmentValue {
	now := time.Now().UTC().Truncate(time.Second)
	return &model.FeatureFlagEnvironmentValue{
		ID:            uuid.NewString(),
		FeatureFlagID: featureFlagID,
		EnvironmentID: environmentID,
		Enabled:       true,
		AuditFields:   model.AuditFields{CreatedOn: now, ModifiedOn: now},
	}
}

func seedFeatureFlagAndEnvironment(t *testing.T, db *sql.DB) (flagID, environmentID string) {
	t.Helper()
	ctx := context.Background()

	categoryID := developmentCategoryID(t, db)
	env := newTestEnvironment("value-repo-env", categoryID)
	if err := NewEnvironmentRepository(db).Create(ctx, env); err != nil {
		t.Fatalf("Create(env) returned unexpected error: %v", err)
	}

	flag := newTestFeatureFlag("value-repo-flag")
	if err := NewFeatureFlagRepository(db).Create(ctx, flag); err != nil {
		t.Fatalf("Create(flag) returned unexpected error: %v", err)
	}

	return flag.ID, env.ID
}

func TestFeatureFlagEnvironmentValueRepository_CreateAndGet(t *testing.T) {
	db := newTestDB(t)
	repo := NewFeatureFlagEnvironmentValueRepository(db)
	ctx := context.Background()

	flagID, envID := seedFeatureFlagAndEnvironment(t, db)

	v := newTestFeatureFlagEnvironmentValue(flagID, envID)
	if err := repo.Create(ctx, v); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	got, err := repo.GetByFlagAndEnvironment(ctx, flagID, envID)
	if err != nil {
		t.Fatalf("GetByFlagAndEnvironment() returned unexpected error: %v", err)
	}
	if got.ID != v.ID || !got.Enabled {
		t.Errorf("GetByFlagAndEnvironment() = %+v, want ID=%q Enabled=true", got, v.ID)
	}
}

func TestFeatureFlagEnvironmentValueRepository_Create_Duplicate(t *testing.T) {
	db := newTestDB(t)
	repo := NewFeatureFlagEnvironmentValueRepository(db)
	ctx := context.Background()

	flagID, envID := seedFeatureFlagAndEnvironment(t, db)

	if err := repo.Create(ctx, newTestFeatureFlagEnvironmentValue(flagID, envID)); err != nil {
		t.Fatalf("Create() (first) returned unexpected error: %v", err)
	}
	err := repo.Create(ctx, newTestFeatureFlagEnvironmentValue(flagID, envID))
	if !errors.Is(err, ErrFeatureFlagEnvironmentValueExists) {
		t.Fatalf("Create() (duplicate) error = %v, want ErrFeatureFlagEnvironmentValueExists", err)
	}
}

func TestFeatureFlagEnvironmentValueRepository_Create_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	flagID, envID := seedFeatureFlagAndEnvironment(t, db)
	repo := NewFeatureFlagEnvironmentValueRepository(db)
	_ = db.Close()

	err := repo.Create(context.Background(), newTestFeatureFlagEnvironmentValue(flagID, envID))
	if err == nil || errors.Is(err, ErrFeatureFlagEnvironmentValueExists) {
		t.Fatalf("Create() on closed database error = %v, want a plain non-nil error", err)
	}
}

func TestFeatureFlagEnvironmentValueRepository_GetByFlagAndEnvironment_NotFound(t *testing.T) {
	db := newTestDB(t)
	repo := NewFeatureFlagEnvironmentValueRepository(db)
	flagID, envID := seedFeatureFlagAndEnvironment(t, db)

	_, err := repo.GetByFlagAndEnvironment(context.Background(), flagID, envID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByFlagAndEnvironment() error = %v, want ErrNotFound", err)
	}
}

func TestFeatureFlagEnvironmentValueRepository_GetByFlagAndEnvironment_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewFeatureFlagEnvironmentValueRepository(db)
	_ = db.Close()

	_, err := repo.GetByFlagAndEnvironment(context.Background(), "irrelevant", "irrelevant")
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByFlagAndEnvironment() on closed database error = %v, want a non-nil, non-ErrNotFound error", err)
	}
}

func TestFeatureFlagEnvironmentValueRepository_ListByFlag(t *testing.T) {
	db := newTestDB(t)
	repo := NewFeatureFlagEnvironmentValueRepository(db)
	ctx := context.Background()

	flagID, envID := seedFeatureFlagAndEnvironment(t, db)
	if err := repo.Create(ctx, newTestFeatureFlagEnvironmentValue(flagID, envID)); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	values, err := repo.ListByFlag(ctx, flagID)
	if err != nil {
		t.Fatalf("ListByFlag() returned unexpected error: %v", err)
	}
	if len(values) != 1 || values[0].EnvironmentID != envID {
		t.Errorf("ListByFlag() = %+v, want one value for %q", values, envID)
	}
}

func TestFeatureFlagEnvironmentValueRepository_ListByFlag_Empty(t *testing.T) {
	db := newTestDB(t)
	repo := NewFeatureFlagEnvironmentValueRepository(db)
	flagID, _ := seedFeatureFlagAndEnvironment(t, db)

	values, err := repo.ListByFlag(context.Background(), flagID)
	if err != nil {
		t.Fatalf("ListByFlag() returned unexpected error: %v", err)
	}
	if len(values) != 0 {
		t.Errorf("ListByFlag() = %v, want empty", values)
	}
}

func TestFeatureFlagEnvironmentValueRepository_ListByFlag_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewFeatureFlagEnvironmentValueRepository(db)
	_ = db.Close()

	if _, err := repo.ListByFlag(context.Background(), "irrelevant"); err == nil {
		t.Fatal("ListByFlag() on closed database expected an error, got nil")
	}
}

func TestFeatureFlagEnvironmentValueRepository_Update(t *testing.T) {
	db := newTestDB(t)
	userRepo := NewUserRepository(db)
	repo := NewFeatureFlagEnvironmentValueRepository(db)
	ctx := context.Background()

	actor := newTestUser("value-editor@example.com")
	if err := userRepo.Create(ctx, actor); err != nil {
		t.Fatalf("Create(actor) returned unexpected error: %v", err)
	}
	flagID, envID := seedFeatureFlagAndEnvironment(t, db)

	v := newTestFeatureFlagEnvironmentValue(flagID, envID)
	v.Enabled = false
	if err := repo.Create(ctx, v); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	newValue := "on"
	v.Enabled = true
	v.Value = &newValue
	v.ModifiedByUserID = &actor.ID
	if err := repo.Update(ctx, v); err != nil {
		t.Fatalf("Update() returned unexpected error: %v", err)
	}

	got, err := repo.GetByFlagAndEnvironment(ctx, flagID, envID)
	if err != nil {
		t.Fatalf("GetByFlagAndEnvironment() returned unexpected error: %v", err)
	}
	if !got.Enabled || got.Value == nil || *got.Value != "on" {
		t.Errorf("GetByFlagAndEnvironment() after Update() = %+v, want Enabled=true Value=on", got)
	}
}

func TestFeatureFlagEnvironmentValueRepository_Update_NotFound(t *testing.T) {
	db := newTestDB(t)
	repo := NewFeatureFlagEnvironmentValueRepository(db)
	flagID, envID := seedFeatureFlagAndEnvironment(t, db)

	err := repo.Update(context.Background(), newTestFeatureFlagEnvironmentValue(flagID, envID))
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Update() error = %v, want ErrNotFound", err)
	}
}

func TestFeatureFlagEnvironmentValueRepository_Update_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	flagID, envID := seedFeatureFlagAndEnvironment(t, db)
	repo := NewFeatureFlagEnvironmentValueRepository(db)
	_ = db.Close()

	err := repo.Update(context.Background(), newTestFeatureFlagEnvironmentValue(flagID, envID))
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("Update() on closed database error = %v, want a plain non-nil error", err)
	}
}
