package model

import "time"

// AuditFields holds the six audit columns every table carries per
// docs/architecture/adr/0005-audit-columns-and-soft-delete-convention.md.
// Exactly one of CreatedByUserID/CreatedByServiceKeyID must be set, and
// likewise for the ModifiedBy pair — except for system-seeded rows, where
// both may be nil.
type AuditFields struct {
	CreatedByUserID        *string
	CreatedByServiceKeyID  *string
	CreatedOn              time.Time
	ModifiedByUserID       *string
	ModifiedByServiceKeyID *string
	ModifiedOn             time.Time
}
