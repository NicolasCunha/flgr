package model

const (
	ServiceKeyStatusActive   = "Active"
	ServiceKeyStatusInactive = "Inactive"
)

// ServiceKey is how a consumer application authenticates against flgr, per
// docs/business/requirements/0004-service-key-management.md. It is not tied
// to a User or a Profile — CanRead/CanWrite are the key's own access grant,
// independent of any User's permissions, and its environment scope (see
// repository.ServiceKeyEnvironmentRepository) is managed separately.
type ServiceKey struct {
	ID         string
	Name       string
	SecretHash string
	Status     string
	CanRead    bool
	CanWrite   bool
	AuditFields
}

// IsActive reports whether the key's Status is Active — a key that is
// Inactive must be rejected outright, per
// docs/business/requirements/0007-feature-flag-evaluation-api.md.
func (k *ServiceKey) IsActive() bool {
	return k.Status == ServiceKeyStatusActive
}
