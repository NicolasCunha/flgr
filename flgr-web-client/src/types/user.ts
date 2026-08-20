// Shared User shape, per docs/business/requirements/0002-user-management.md.
// camelCase — the snake_case <-> camelCase conversion happens once, at the
// RTK Query boundary (docs/architecture/adr/0007-api-design-conventions.md).
export interface User {
  id: string
  firstName: string
  lastName: string
  email: string
  status: string
  createdOn: string
  modifiedOn: string
}
