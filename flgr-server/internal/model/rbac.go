package model

// Resource and Action name the fixed permission catalog from
// docs/business/requirements/0003-profile-and-permission-management.md.
// Permissions are (resource, action) pairs; administrators grant or
// revoke them via profiles and users, but cannot create new ones.
const (
	ResourceEnvironment      = "Environment"
	ResourceFeatureFlag      = "FeatureFlag"
	ResourceFeatureFlagValue = "FeatureFlagValue"
	ResourceServiceKey       = "ServiceKey"
	ResourceProfile          = "Profile"
	ResourceUser             = "User"
)

const (
	ActionCreate = "Create"
	ActionEdit   = "Edit"
	ActionRemove = "Remove"
	ActionView   = "View"
	ActionRead   = "Read"
	ActionWrite  = "Write"
)

// Permission is one entry in the fixed, system-seeded catalog.
type Permission struct {
	ID       string
	Resource string
	Action   string
	AuditFields
}

// AdministradorProfileID is the seeded system profile's id (see
// migrations/000001_initial_schema.up.sql), holding the full permission
// catalog. cmd/server's setup-wizard bootstrap assigns it to the first
// User, so a fresh install actually has a working administrator — see
// service.UserService.CompleteSetup.
const AdministradorProfileID = "00000000-0000-0000-0000-000000000010"

// Profile is a named, reusable group of permissions. Exactly one system
// profile, Administrador, is seeded with the full catalog — see
// docs/business/requirements/0003-profile-and-permission-management.md.
type Profile struct {
	ID          string
	Name        string
	Description *string
	IsSystem    bool
	AuditFields
}
