package service

import "github.com/NicolasCunha/flgr/flgr-server/internal/model"

// Actor identifies whichever credential authenticated a write: a User
// (session cookie, per docs/architecture/adr/0006-authentication-and-session-strategy.md)
// or a ServiceKey with can_write enabled (Bearer token, same ADR). It
// mirrors the dual created_by_user_id/created_by_service_key_id (and
// modified_by_*) column pattern used across the schema, per
// docs/architecture/adr/0005-audit-columns-and-soft-delete-convention.md,
// so a write method can set either audit pair straight from it without a
// type switch. Exactly one of UserID/ServiceKeyID is set for a real actor;
// the zero value (see IsZero) is "no authenticated actor", replacing the
// nil *model.User check write methods used before Service Keys could also
// write.
type Actor struct {
	UserID       *string
	ServiceKeyID *string
}

// ActorFromUser returns the Actor for an authenticated User.
func ActorFromUser(u *model.User) Actor {
	return Actor{UserID: &u.ID}
}

// ActorFromServiceKey returns the Actor for an authenticated ServiceKey.
func ActorFromServiceKey(k *model.ServiceKey) Actor {
	return Actor{ServiceKeyID: &k.ID}
}

// IsZero reports whether a has neither a User nor a ServiceKey set — the
// "no actor" case write methods reject with ErrForbidden.
func (a Actor) IsZero() bool {
	return a.UserID == nil && a.ServiceKeyID == nil
}
