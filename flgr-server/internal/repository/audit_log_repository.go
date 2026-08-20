package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
)

// AuditLogRepository is the data access seam for the shared audit_log table
// plus its feature-flag link table, per
// docs/architecture/adr/0013-generic-audit-log-pattern.md and
// docs/business/requirements/0008-feature-flag-audit-trail.md. It only
// writes — this phase (the Killswitch trigger) has no read/list AC yet;
// listing/filtering the audit trail is 0008's own future phase.
//
// audit_log_feature_flag is the only link table with write logic today
// (per migration 000002's own comment), so this is the only Create this
// repository needs; a future entity's audit trail would add its own
// AuditLog*Repository following the same shape rather than growing this
// one's signature.
type AuditLogRepository interface {
	// Create writes log and its feature-flag link row in the same
	// transaction. link.AuditLogID is overwritten with log.ID before the
	// insert, so callers don't need to keep the two in sync by hand.
	Create(ctx context.Context, log *model.AuditLog, link *model.AuditLogFeatureFlag) error
}

type auditLogRepository struct {
	db *sql.DB
}

// NewAuditLogRepository returns an AuditLogRepository backed by db.
func NewAuditLogRepository(db *sql.DB) AuditLogRepository {
	return &auditLogRepository{db: db}
}

func (r *auditLogRepository) Create(ctx context.Context, log *model.AuditLog, link *model.AuditLogFeatureFlag) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning audit log write: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO audit_log (
			id, entity_type, action, actor_user_id, actor_service_key_id,
			source, old_value, new_value, occurred_on
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		log.ID, log.EntityType, log.Action,
		nullString(log.ActorUserID), nullString(log.ActorServiceKeyID),
		log.Source, nullString(log.OldValue), nullString(log.NewValue), log.OccurredOn,
	)
	if err != nil {
		return fmt.Errorf("inserting audit log: %w", err)
	}

	link.AuditLogID = log.ID
	_, err = tx.ExecContext(ctx, `
		INSERT INTO audit_log_feature_flag (audit_log_id, feature_flag_id, environment_id)
		VALUES (?, ?, ?)`,
		link.AuditLogID, link.FeatureFlagID, nullString(link.EnvironmentID),
	)
	if err != nil {
		return fmt.Errorf("inserting audit log feature flag link: %w", err)
	}

	// SQLite enforces constraints at each statement, not deferred to
	// COMMIT, so this failing isn't reachable in a test without a fault-
	// injecting driver double, per the exception clause in
	// docs/architecture/adr/0004-testing-and-coverage-standards.md.
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing audit log write: %w", err)
	}
	return nil
}
