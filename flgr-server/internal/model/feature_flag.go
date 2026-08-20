package model

const (
	FeatureFlagTypeBoolean = "Boolean"
	FeatureFlagTypeString  = "String"
	FeatureFlagTypeNumber  = "Number"
	FeatureFlagTypeJSON    = "JSON"
)

// FeatureFlag is a single, stable definition controlling the availability
// or configuration of a feature in a consumer application, per
// docs/business/requirements/0005-feature-flag-management.md. Type is
// fixed at creation and never changes — see FeatureFlagEnvironmentValue for
// the per-environment values a flag actually evaluates to.
type FeatureFlag struct {
	ID          string
	Key         string
	Name        string
	Description *string
	Type        string
	AuditFields
}

// FeatureFlagEnvironmentValue is a flag's independent on/off state (and,
// for non-Boolean types, its served value) in one Environment, per 0005.
// A flag with no row for a given environment evaluates as disabled there —
// rows are created on demand by FeatureFlagValueService.Upsert, not
// pre-populated for every environment.
type FeatureFlagEnvironmentValue struct {
	ID            string
	FeatureFlagID string
	EnvironmentID string
	Enabled       bool
	Value         *string
	AuditFields
}
