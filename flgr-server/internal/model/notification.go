package model

import "time"

// Notification channel types, per
// docs/business/requirements/0006-feature-flag-killswitch.md.
const (
	NotificationChannelTypeKafka = "Kafka"
	NotificationChannelTypeHTTP  = "HTTP"
)

// NotificationChannel is a Kafka topic or HTTP webhook configured on a flag
// (independently of any particular environment) to receive a change event
// when the Killswitch fires, per 0006. SigningSecretEncrypted is set for
// HTTP channels only (nil for Kafka) and holds AES-256-GCM ciphertext per
// docs/architecture/adr/0010-application-configuration-and-secrets.md;
// encrypting/decrypting it is the service layer's job — this layer stores
// and retrieves it as opaque bytes.
type NotificationChannel struct {
	ID                     string
	FeatureFlagID          string
	ChannelType            string
	Destination            string
	SigningSecretEncrypted []byte
	Enabled                bool
	AuditFields
}

// Generic audit_log entity types, per
// docs/architecture/adr/0013-generic-audit-log-pattern.md. Only FeatureFlag
// is written to as of this phase (the Killswitch trigger path); the other
// values ADR-0013 reserves (User, Environment, ServiceKey, Profile) have no
// write path yet and are intentionally not defined here until a requirement
// needs them.
const AuditEntityTypeFeatureFlag = "FeatureFlag"

// Generic audit_log sources, per ADR-0013 and
// docs/business/requirements/0008-feature-flag-audit-trail.md — whether the
// action came from the UI or the API.
const (
	AuditSourceUI  = "UI"
	AuditSourceAPI = "API"
)

// AuditActionKillswitchTriggered is the action value an AuditLog entry gets
// when the Killswitch fires, per
// docs/business/requirements/0006-feature-flag-killswitch.md's acceptance
// criteria ("action = KillswitchTriggered"). The literal stored value is
// "Triggered", matching the enumerated action list in both
// docs/architecture/adr/0013-generic-audit-log-pattern.md ("Created,
// Updated, Removed, ValueChanged, Triggered") and 0008's own Data Model
// table ("Created, Edited, Removed, ValueChanged, or Triggered (killswitch)")
// — "KillswitchTriggered" appears only in 0008's prose AC, not in either
// table's enumerated value list. This phase only ever writes this one
// action; the flag create/edit/remove/value-change actions those same
// tables enumerate belong to a future phase's write path, and are not
// defined here to avoid guessing at a name two source docs already disagree
// on ("Updated" vs "Edited").
const AuditActionKillswitchTriggered = "Triggered"

// AuditLog is one immutable entry in the shared audit_log table, per
// ADR-0013. Exactly one of ActorUserID/ActorServiceKeyID is set. It has no
// AuditFields embed — unlike every other table in this codebase, audit_log
// is deliberately exempt from the standard audit columns (ADR-0005): its
// own actor_*/source/occurred_on columns already capture who/when, and a
// row is never modified after being written.
type AuditLog struct {
	ID                string
	EntityType        string
	Action            string
	ActorUserID       *string
	ActorServiceKeyID *string
	Source            string
	OldValue          *string
	NewValue          *string
	OccurredOn        time.Time
}

// AuditLogFeatureFlag is the audit_log_feature_flag link row identifying
// which flag (and, for a per-environment action like the Killswitch, which
// environment) one AuditLog entry is about, per ADR-0013's link-table
// pattern and 0008's Data Model. EnvironmentID is nil for flag-definition-
// level entries (create/edit/remove); the Killswitch trigger always sets it.
type AuditLogFeatureFlag struct {
	AuditLogID    string
	FeatureFlagID string
	EnvironmentID *string
}

// Notification outbox statuses, per
// docs/architecture/adr/0009-kafka-and-webhook-notification-delivery.md.
const (
	NotificationOutboxStatusPending   = "Pending"
	NotificationOutboxStatusDelivered = "Delivered"
	NotificationOutboxStatusFailed    = "Failed"
)

// NotificationOutboxEntry is one queued delivery of a change event to a
// single notification channel, per ADR-0009's outbox pattern: the
// Killswitch trigger writes one row per enabled channel in the same
// transaction as the value change and the audit log entry, then returns
// immediately — a background worker (a future phase, per ADR-0009) polls
// and delivers pending rows independently. Like AuditLog, it has no
// AuditFields embed: ADR-0009 explicitly calls it "a system-generated
// delivery record, not a user-editable entity," exempt from the standard
// audit columns.
type NotificationOutboxEntry struct {
	ID                               string
	FeatureFlagNotificationChannelID string
	Payload                          string
	Status                           string
	AttemptCount                     int
	NextAttemptAt                    *time.Time
	LastError                        *string
	CreatedOn                        time.Time
	DeliveredOn                      *time.Time
}
