package model

// EnvironmentCategory groups environments (e.g. Development, Staging,
// Production, or a custom value), per
// docs/business/requirements/0001-environment-management.md. The three
// system categories are seeded (IsSystem = true, immutable); a custom one
// is created implicitly the first time an environment picks a new name
// for it (see service.EnvironmentService.resolveCategory).
type EnvironmentCategory struct {
	ID       string
	Name     string
	IsSystem bool
	AuditFields
}

// Environment represents a distinct stage of a consumer application's
// lifecycle in which a feature flag can hold an independent value, per
// docs/business/requirements/0001-environment-management.md.
type Environment struct {
	ID                    string
	Name                  string
	Description           *string
	EnvironmentCategoryID string
	AuditFields
}
