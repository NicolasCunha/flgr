-- Generic audit log pattern: docs/architecture/adr/0013-generic-audit-log-pattern.md.
--
-- Replaces feature_flag_audit_logs (000001) with a shared audit_log table plus one
-- thin link table per audited entity, so a future entity's audit trail only needs a
-- new link table, not a full parallel log table. audit_log and every audit_log_*
-- table are exempt from the standard audit columns (docs/architecture/adr/0005), for
-- the same reason feature_flag_audit_logs was: their own actor_*/source/occurred_on
-- columns already capture who/when, and a row is never modified after being written.
--
-- audit_log_user, audit_log_environment, audit_log_service_key, and
-- audit_log_profile are schema-only for now: no accepted requirement yet defines
-- what on those entities must be audited, so no application code writes to them.

DROP TABLE IF EXISTS feature_flag_audit_logs;

CREATE TABLE audit_log (
    id TEXT PRIMARY KEY,
    entity_type TEXT NOT NULL,
    action TEXT NOT NULL,
    actor_user_id TEXT REFERENCES users(id),
    actor_service_key_id TEXT REFERENCES service_keys(id),
    source TEXT NOT NULL,
    old_value TEXT,
    new_value TEXT,
    occurred_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (actor_user_id IS NULL OR actor_service_key_id IS NULL)
);
CREATE INDEX idx_audit_log_entity_type_occurred_on ON audit_log(entity_type, occurred_on);

-- 0008-feature-flag-audit-trail.md — the only entity with write logic wired up so far.
CREATE TABLE audit_log_feature_flag (
    audit_log_id TEXT PRIMARY KEY REFERENCES audit_log(id),
    feature_flag_id TEXT NOT NULL REFERENCES feature_flags(id),
    environment_id TEXT REFERENCES environments(id)
);
CREATE INDEX idx_audit_log_feature_flag_flag ON audit_log_feature_flag(feature_flag_id);

CREATE TABLE audit_log_user (
    audit_log_id TEXT PRIMARY KEY REFERENCES audit_log(id),
    user_id TEXT NOT NULL REFERENCES users(id)
);
CREATE INDEX idx_audit_log_user_user ON audit_log_user(user_id);

CREATE TABLE audit_log_environment (
    audit_log_id TEXT PRIMARY KEY REFERENCES audit_log(id),
    environment_id TEXT NOT NULL REFERENCES environments(id)
);
CREATE INDEX idx_audit_log_environment_environment ON audit_log_environment(environment_id);

CREATE TABLE audit_log_service_key (
    audit_log_id TEXT PRIMARY KEY REFERENCES audit_log(id),
    service_key_id TEXT NOT NULL REFERENCES service_keys(id)
);
CREATE INDEX idx_audit_log_service_key_service_key ON audit_log_service_key(service_key_id);

CREATE TABLE audit_log_profile (
    audit_log_id TEXT PRIMARY KEY REFERENCES audit_log(id),
    profile_id TEXT NOT NULL REFERENCES profiles(id)
);
CREATE INDEX idx_audit_log_profile_profile ON audit_log_profile(profile_id);
