DROP TABLE IF EXISTS audit_log_profile;
DROP TABLE IF EXISTS audit_log_service_key;
DROP TABLE IF EXISTS audit_log_environment;
DROP TABLE IF EXISTS audit_log_user;
DROP TABLE IF EXISTS audit_log_feature_flag;
DROP TABLE IF EXISTS audit_log;

CREATE TABLE feature_flag_audit_logs (
    id TEXT PRIMARY KEY,
    feature_flag_id TEXT NOT NULL REFERENCES feature_flags(id),
    environment_id TEXT REFERENCES environments(id),
    action TEXT NOT NULL,
    actor_user_id TEXT REFERENCES users(id),
    actor_service_key_id TEXT REFERENCES service_keys(id),
    source TEXT NOT NULL,
    old_value TEXT,
    new_value TEXT,
    occurred_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_feature_flag_audit_logs_flag ON feature_flag_audit_logs(feature_flag_id, occurred_on);
