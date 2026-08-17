-- Initial schema for flgr.
-- Reflects docs/business/requirements/0001-0008 and ADR-0005, ADR-0006, ADR-0009.
--
-- Audit columns (created_by_user_id/created_by_service_key_id/created_on/
-- modified_by_user_id/modified_by_service_key_id/modified_on) follow
-- docs/architecture/adr/0005-audit-columns-and-soft-delete-convention.md:
-- exactly one of each *_by_user_id/*_by_service_key_id pair may be set,
-- or both may be null for rows seeded by this migration itself.
--
-- Tables explicitly exempted from the audit columns by their own requirement/ADR:
-- sessions, login_attempts (ADR-0006), feature_flag_audit_logs (0008),
-- notification_outbox (ADR-0009).

PRAGMA foreign_keys = ON;

-- 0002-user-management.md
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    email TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'Active',
    created_by_user_id TEXT REFERENCES users(id),
    created_by_service_key_id TEXT,
    created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified_by_user_id TEXT REFERENCES users(id),
    modified_by_service_key_id TEXT,
    modified_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (created_by_user_id IS NULL OR created_by_service_key_id IS NULL),
    CHECK (modified_by_user_id IS NULL OR modified_by_service_key_id IS NULL)
);
CREATE UNIQUE INDEX idx_users_email ON users(email);

-- 0004-service-key-management.md
CREATE TABLE service_keys (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    secret_hash TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'Active',
    can_read BOOLEAN NOT NULL DEFAULT 1,
    can_write BOOLEAN NOT NULL DEFAULT 0,
    created_by_user_id TEXT REFERENCES users(id),
    created_by_service_key_id TEXT REFERENCES service_keys(id),
    created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified_by_user_id TEXT REFERENCES users(id),
    modified_by_service_key_id TEXT REFERENCES service_keys(id),
    modified_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (created_by_user_id IS NULL OR created_by_service_key_id IS NULL),
    CHECK (modified_by_user_id IS NULL OR modified_by_service_key_id IS NULL)
);

-- Deferred FK from users -> service_keys, now that service_keys exists.
-- SQLite does not support ALTER TABLE ADD CONSTRAINT, so this is enforced
-- at the application layer for users.created_by_service_key_id /
-- users.modified_by_service_key_id instead of a declared FK.

-- 0001-environment-management.md
CREATE TABLE environment_categories (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    is_system BOOLEAN NOT NULL DEFAULT 0,
    created_by_user_id TEXT REFERENCES users(id),
    created_by_service_key_id TEXT REFERENCES service_keys(id),
    created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified_by_user_id TEXT REFERENCES users(id),
    modified_by_service_key_id TEXT REFERENCES service_keys(id),
    modified_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (created_by_user_id IS NULL OR created_by_service_key_id IS NULL),
    CHECK (modified_by_user_id IS NULL OR modified_by_service_key_id IS NULL)
);
CREATE UNIQUE INDEX idx_environment_categories_name ON environment_categories(name);

CREATE TABLE environments (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    environment_category_id TEXT NOT NULL REFERENCES environment_categories(id),
    created_by_user_id TEXT REFERENCES users(id),
    created_by_service_key_id TEXT REFERENCES service_keys(id),
    created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified_by_user_id TEXT REFERENCES users(id),
    modified_by_service_key_id TEXT REFERENCES service_keys(id),
    modified_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (created_by_user_id IS NULL OR created_by_service_key_id IS NULL),
    CHECK (modified_by_user_id IS NULL OR modified_by_service_key_id IS NULL)
);
CREATE UNIQUE INDEX idx_environments_name ON environments(name);

-- ADR-0006: Authentication & Session Strategy (exempt from standard audit columns)
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    token_hash TEXT NOT NULL,
    user_id TEXT NOT NULL REFERENCES users(id),
    issued_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    last_seen_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX idx_sessions_token_hash ON sessions(token_hash);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);

CREATE TABLE login_attempts (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL,
    failed_count INTEGER NOT NULL DEFAULT 0,
    first_failed_at TIMESTAMP,
    locked_until TIMESTAMP
);
CREATE UNIQUE INDEX idx_login_attempts_email ON login_attempts(email);

-- 0003-profile-and-permission-management.md
CREATE TABLE profiles (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    is_system BOOLEAN NOT NULL DEFAULT 0,
    created_by_user_id TEXT REFERENCES users(id),
    created_by_service_key_id TEXT REFERENCES service_keys(id),
    created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified_by_user_id TEXT REFERENCES users(id),
    modified_by_service_key_id TEXT REFERENCES service_keys(id),
    modified_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (created_by_user_id IS NULL OR created_by_service_key_id IS NULL),
    CHECK (modified_by_user_id IS NULL OR modified_by_service_key_id IS NULL)
);
CREATE UNIQUE INDEX idx_profiles_name ON profiles(name);

CREATE TABLE permissions (
    id TEXT PRIMARY KEY,
    resource TEXT NOT NULL,
    action TEXT NOT NULL,
    created_by_user_id TEXT REFERENCES users(id),
    created_by_service_key_id TEXT REFERENCES service_keys(id),
    created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified_by_user_id TEXT REFERENCES users(id),
    modified_by_service_key_id TEXT REFERENCES service_keys(id),
    modified_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (created_by_user_id IS NULL OR created_by_service_key_id IS NULL),
    CHECK (modified_by_user_id IS NULL OR modified_by_service_key_id IS NULL)
);
CREATE UNIQUE INDEX idx_permissions_resource_action ON permissions(resource, action);

CREATE TABLE profile_permissions (
    id TEXT PRIMARY KEY,
    profile_id TEXT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    permission_id TEXT NOT NULL REFERENCES permissions(id) ON DELETE RESTRICT,
    created_by_user_id TEXT REFERENCES users(id),
    created_by_service_key_id TEXT REFERENCES service_keys(id),
    created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified_by_user_id TEXT REFERENCES users(id),
    modified_by_service_key_id TEXT REFERENCES service_keys(id),
    modified_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (created_by_user_id IS NULL OR created_by_service_key_id IS NULL),
    CHECK (modified_by_user_id IS NULL OR modified_by_service_key_id IS NULL)
);
CREATE UNIQUE INDEX idx_profile_permissions_unique ON profile_permissions(profile_id, permission_id);

CREATE TABLE user_profiles (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    profile_id TEXT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    created_by_user_id TEXT REFERENCES users(id),
    created_by_service_key_id TEXT REFERENCES service_keys(id),
    created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified_by_user_id TEXT REFERENCES users(id),
    modified_by_service_key_id TEXT REFERENCES service_keys(id),
    modified_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (created_by_user_id IS NULL OR created_by_service_key_id IS NULL),
    CHECK (modified_by_user_id IS NULL OR modified_by_service_key_id IS NULL)
);
CREATE UNIQUE INDEX idx_user_profiles_unique ON user_profiles(user_id, profile_id);

CREATE TABLE user_permissions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    permission_id TEXT NOT NULL REFERENCES permissions(id) ON DELETE RESTRICT,
    created_by_user_id TEXT REFERENCES users(id),
    created_by_service_key_id TEXT REFERENCES service_keys(id),
    created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified_by_user_id TEXT REFERENCES users(id),
    modified_by_service_key_id TEXT REFERENCES service_keys(id),
    modified_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (created_by_user_id IS NULL OR created_by_service_key_id IS NULL),
    CHECK (modified_by_user_id IS NULL OR modified_by_service_key_id IS NULL)
);
CREATE UNIQUE INDEX idx_user_permissions_unique ON user_permissions(user_id, permission_id);

-- 0004-service-key-management.md
CREATE TABLE service_key_environments (
    id TEXT PRIMARY KEY,
    service_key_id TEXT NOT NULL REFERENCES service_keys(id) ON DELETE CASCADE,
    environment_id TEXT NOT NULL REFERENCES environments(id) ON DELETE RESTRICT,
    created_by_user_id TEXT REFERENCES users(id),
    created_by_service_key_id TEXT REFERENCES service_keys(id),
    created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified_by_user_id TEXT REFERENCES users(id),
    modified_by_service_key_id TEXT REFERENCES service_keys(id),
    modified_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (created_by_user_id IS NULL OR created_by_service_key_id IS NULL),
    CHECK (modified_by_user_id IS NULL OR modified_by_service_key_id IS NULL)
);
CREATE UNIQUE INDEX idx_service_key_environments_unique ON service_key_environments(service_key_id, environment_id);

-- 0005-feature-flag-management.md
CREATE TABLE feature_flags (
    id TEXT PRIMARY KEY,
    key TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    type TEXT NOT NULL,
    created_by_user_id TEXT REFERENCES users(id),
    created_by_service_key_id TEXT REFERENCES service_keys(id),
    created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified_by_user_id TEXT REFERENCES users(id),
    modified_by_service_key_id TEXT REFERENCES service_keys(id),
    modified_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (created_by_user_id IS NULL OR created_by_service_key_id IS NULL),
    CHECK (modified_by_user_id IS NULL OR modified_by_service_key_id IS NULL)
);
CREATE UNIQUE INDEX idx_feature_flags_key ON feature_flags(key);

CREATE TABLE feature_flag_environment_values (
    id TEXT PRIMARY KEY,
    feature_flag_id TEXT NOT NULL REFERENCES feature_flags(id) ON DELETE CASCADE,
    environment_id TEXT NOT NULL REFERENCES environments(id) ON DELETE RESTRICT,
    enabled BOOLEAN NOT NULL DEFAULT 0,
    value TEXT,
    created_by_user_id TEXT REFERENCES users(id),
    created_by_service_key_id TEXT REFERENCES service_keys(id),
    created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified_by_user_id TEXT REFERENCES users(id),
    modified_by_service_key_id TEXT REFERENCES service_keys(id),
    modified_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (created_by_user_id IS NULL OR created_by_service_key_id IS NULL),
    CHECK (modified_by_user_id IS NULL OR modified_by_service_key_id IS NULL)
);
CREATE UNIQUE INDEX idx_feature_flag_environment_values_unique ON feature_flag_environment_values(feature_flag_id, environment_id);

-- 0006-feature-flag-killswitch.md + ADR-0009 (signing_secret_encrypted for HTTP channels)
CREATE TABLE feature_flag_notification_channels (
    id TEXT PRIMARY KEY,
    feature_flag_id TEXT NOT NULL REFERENCES feature_flags(id) ON DELETE CASCADE,
    channel_type TEXT NOT NULL,
    destination TEXT NOT NULL,
    signing_secret_encrypted BLOB,
    enabled BOOLEAN NOT NULL DEFAULT 1,
    created_by_user_id TEXT REFERENCES users(id),
    created_by_service_key_id TEXT REFERENCES service_keys(id),
    created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified_by_user_id TEXT REFERENCES users(id),
    modified_by_service_key_id TEXT REFERENCES service_keys(id),
    modified_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (created_by_user_id IS NULL OR created_by_service_key_id IS NULL),
    CHECK (modified_by_user_id IS NULL OR modified_by_service_key_id IS NULL)
);
CREATE INDEX idx_feature_flag_notification_channels_flag ON feature_flag_notification_channels(feature_flag_id);

-- 0008-feature-flag-audit-trail.md (exempt from standard audit columns)
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

-- ADR-0009: Kafka & HTTP Webhook Notification Delivery (exempt from standard audit columns)
CREATE TABLE notification_outbox (
    id TEXT PRIMARY KEY,
    feature_flag_notification_channel_id TEXT NOT NULL REFERENCES feature_flag_notification_channels(id),
    payload TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'Pending',
    attempt_count INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMP,
    last_error TEXT,
    created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    delivered_on TIMESTAMP
);
CREATE INDEX idx_notification_outbox_status ON notification_outbox(status, next_attempt_at);

-- Seed data: system-protected rows, per ADR-0005 both actor columns are left null.

INSERT INTO environment_categories (id, name, is_system) VALUES
    ('00000000-0000-0000-0000-000000000001', 'Development', 1),
    ('00000000-0000-0000-0000-000000000002', 'Staging', 1),
    ('00000000-0000-0000-0000-000000000003', 'Production', 1);

INSERT INTO profiles (id, name, description, is_system) VALUES
    ('00000000-0000-0000-0000-000000000010', 'Administrador', 'Default profile holding every permission in the catalog. Protected per 0003-profile-and-permission-management.md.', 1);

INSERT INTO permissions (id, resource, action) VALUES
    ('environment:create', 'Environment', 'Create'),
    ('environment:edit', 'Environment', 'Edit'),
    ('environment:remove', 'Environment', 'Remove'),
    ('environment:view', 'Environment', 'View'),
    ('feature_flag:create', 'FeatureFlag', 'Create'),
    ('feature_flag:edit', 'FeatureFlag', 'Edit'),
    ('feature_flag:remove', 'FeatureFlag', 'Remove'),
    ('feature_flag:view', 'FeatureFlag', 'View'),
    ('feature_flag_value:read', 'FeatureFlagValue', 'Read'),
    ('feature_flag_value:write', 'FeatureFlagValue', 'Write'),
    ('service_key:create', 'ServiceKey', 'Create'),
    ('service_key:edit', 'ServiceKey', 'Edit'),
    ('service_key:remove', 'ServiceKey', 'Remove'),
    ('service_key:view', 'ServiceKey', 'View'),
    ('profile:create', 'Profile', 'Create'),
    ('profile:edit', 'Profile', 'Edit'),
    ('profile:remove', 'Profile', 'Remove'),
    ('profile:view', 'Profile', 'View'),
    ('user:create', 'User', 'Create'),
    ('user:edit', 'User', 'Edit'),
    ('user:remove', 'User', 'Remove'),
    ('user:view', 'User', 'View');

INSERT INTO profile_permissions (id, profile_id, permission_id)
SELECT lower(hex(randomblob(16))), '00000000-0000-0000-0000-000000000010', id FROM permissions;
