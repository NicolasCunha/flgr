package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
)

func newTestAuditLog(actorUserID string) *model.AuditLog {
	old := `{"enabled":true}`
	newVal := `{"enabled":false}`
	return &model.AuditLog{
		ID:          uuid.NewString(),
		EntityType:  model.AuditEntityTypeFeatureFlag,
		Action:      model.AuditActionKillswitchTriggered,
		ActorUserID: &actorUserID,
		Source:      model.AuditSourceAPI,
		OldValue:    &old,
		NewValue:    &newVal,
		OccurredOn:  time.Now().UTC().Truncate(time.Second),
	}
}

func TestAuditLogRepository_Create(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	userRepo := NewUserRepository(db)
	actor := newTestUser("audit-actor@example.com")
	if err := userRepo.Create(ctx, actor); err != nil {
		t.Fatalf("Create(actor) returned unexpected error: %v", err)
	}
	flag := newTestFeatureFlag("audit-repo-flag")
	if err := NewFeatureFlagRepository(db).Create(ctx, flag); err != nil {
		t.Fatalf("Create(flag) returned unexpected error: %v", err)
	}
	categoryID := developmentCategoryID(t, db)
	env := newTestEnvironment("audit-repo-env", categoryID)
	if err := NewEnvironmentRepository(db).Create(ctx, env); err != nil {
		t.Fatalf("Create(env) returned unexpected error: %v", err)
	}

	repo := NewAuditLogRepository(db)
	log := newTestAuditLog(actor.ID)
	link := &model.AuditLogFeatureFlag{FeatureFlagID: flag.ID, EnvironmentID: &env.ID}

	if err := repo.Create(ctx, log, link); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	// link.AuditLogID must be filled in with log.ID by Create, per the
	// interface's doc comment, so callers don't need to keep the two in
	// sync by hand.
	if link.AuditLogID != log.ID {
		t.Errorf("link.AuditLogID = %q, want %q", link.AuditLogID, log.ID)
	}

	var gotEntityType, gotAction, gotSource string
	var gotOld, gotNew sql.NullString
	row := db.QueryRow("SELECT entity_type, action, source, old_value, new_value FROM audit_log WHERE id = ?", log.ID)
	if err := row.Scan(&gotEntityType, &gotAction, &gotSource, &gotOld, &gotNew); err != nil {
		t.Fatalf("querying audit_log row: %v", err)
	}
	if gotEntityType != model.AuditEntityTypeFeatureFlag || gotAction != model.AuditActionKillswitchTriggered || gotSource != model.AuditSourceAPI {
		t.Errorf("audit_log row = (entity_type=%q, action=%q, source=%q), want (%q, %q, %q)",
			gotEntityType, gotAction, gotSource, model.AuditEntityTypeFeatureFlag, model.AuditActionKillswitchTriggered, model.AuditSourceAPI)
	}
	if !gotOld.Valid || gotOld.String != *log.OldValue || !gotNew.Valid || gotNew.String != *log.NewValue {
		t.Errorf("audit_log old/new = (%v, %v), want (%q, %q)", gotOld, gotNew, *log.OldValue, *log.NewValue)
	}

	var gotFlagID string
	var gotEnvID sql.NullString
	linkRow := db.QueryRow("SELECT feature_flag_id, environment_id FROM audit_log_feature_flag WHERE audit_log_id = ?", log.ID)
	if err := linkRow.Scan(&gotFlagID, &gotEnvID); err != nil {
		t.Fatalf("querying audit_log_feature_flag row: %v", err)
	}
	if gotFlagID != flag.ID || !gotEnvID.Valid || gotEnvID.String != env.ID {
		t.Errorf("audit_log_feature_flag row = (feature_flag_id=%q, environment_id=%v), want (%q, %q)", gotFlagID, gotEnvID, flag.ID, env.ID)
	}
}

func TestAuditLogRepository_Create_NilEnvironmentID(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	userRepo := NewUserRepository(db)
	actor := newTestUser("audit-actor-2@example.com")
	if err := userRepo.Create(ctx, actor); err != nil {
		t.Fatalf("Create(actor) returned unexpected error: %v", err)
	}
	flag := newTestFeatureFlag("audit-repo-flag-noenv")
	if err := NewFeatureFlagRepository(db).Create(ctx, flag); err != nil {
		t.Fatalf("Create(flag) returned unexpected error: %v", err)
	}

	repo := NewAuditLogRepository(db)
	log := newTestAuditLog(actor.ID)
	link := &model.AuditLogFeatureFlag{FeatureFlagID: flag.ID}

	if err := repo.Create(ctx, log, link); err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	var gotEnvID sql.NullString
	row := db.QueryRow("SELECT environment_id FROM audit_log_feature_flag WHERE audit_log_id = ?", log.ID)
	if err := row.Scan(&gotEnvID); err != nil {
		t.Fatalf("querying audit_log_feature_flag row: %v", err)
	}
	if gotEnvID.Valid {
		t.Errorf("environment_id = %v, want NULL for a flag-definition-level entry", gotEnvID)
	}
}

// TestAuditLogRepository_Create_LinkFailureRollsBackLogInsert exercises the
// transactional guarantee: if the second insert (audit_log_feature_flag)
// fails — here, an unknown feature_flag_id violating the FK — the first
// insert (audit_log) must not be left committed on its own. Confirmed by
// checking the audit_log row is absent afterward.
func TestAuditLogRepository_Create_LinkFailureRollsBackLogInsert(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	userRepo := NewUserRepository(db)
	actor := newTestUser("audit-actor-3@example.com")
	if err := userRepo.Create(ctx, actor); err != nil {
		t.Fatalf("Create(actor) returned unexpected error: %v", err)
	}

	repo := NewAuditLogRepository(db)
	log := newTestAuditLog(actor.ID)
	link := &model.AuditLogFeatureFlag{FeatureFlagID: "does-not-exist"}

	err := repo.Create(ctx, log, link)
	if err == nil {
		t.Fatal("Create() with an unknown feature_flag_id expected an error, got nil")
	}

	var count int
	row := db.QueryRow("SELECT COUNT(*) FROM audit_log WHERE id = ?", log.ID)
	if err := row.Scan(&count); err != nil {
		t.Fatalf("querying audit_log row count: %v", err)
	}
	if count != 0 {
		t.Errorf("audit_log rows for %q = %d, want 0 (rolled back)", log.ID, count)
	}
}

func TestAuditLogRepository_Create_ClosedDatabase(t *testing.T) {
	db := newTestDB(t)
	repo := NewAuditLogRepository(db)
	_ = db.Close()

	log := newTestAuditLog("irrelevant-actor")
	link := &model.AuditLogFeatureFlag{FeatureFlagID: "irrelevant-flag"}
	if err := repo.Create(context.Background(), log, link); err == nil {
		t.Fatal("Create() on closed database expected an error, got nil")
	}
}
