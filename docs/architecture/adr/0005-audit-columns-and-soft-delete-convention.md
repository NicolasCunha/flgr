# ADR-0005: Audit Columns & Soft-Delete Convention

## Summary

| Category | Status | Date | Author |
| -------- | ------ | ---- | ------ |
| Database | Accepted | 2026-08-16 | Nicolas Filipe Cunha |

**Supersedes:** N/A

**Superseded by:** N/A

See [Status Values](README.md#status-values) and [Categories](README.md#categories) in the ADR index for the allowed values in each field.

## Context

Every persisted record in flgr needs to carry who created and last modified it, and when — both for accountability and to support the audit-focused business requirements (e.g., [Feature Flag Audit Trail](../../business/requirements/0008-feature-flag-audit-trail.md)). A record can be created or modified either by a platform User acting through the UI, or by a Service Key acting through the API — so "who" is not always a User.

### Related Documents

| Document | Type | Description | Link |
| -------- | ---- | ----------- | ---- |
| Technology Stack | Architecture | Establishes SQLite as the database this convention applies to. | [0001-technology-stack.md](0001-technology-stack.md) |
| Business Requirements | Business | The requirements whose Data Model sections rely on this convention. | [README.md](../../business/requirements/README.md) |

## Decision

- Every table includes six audit columns: `created_by_user_id` (nullable FK → `users.id`), `created_by_service_key_id` (nullable FK → `service_keys.id`), `created_on` (timestamp, not null), `modified_by_user_id` (nullable FK → `users.id`), `modified_by_service_key_id` (nullable FK → `service_keys.id`), `modified_on` (timestamp, not null).
- Application code must ensure exactly one of `created_by_user_id` / `created_by_service_key_id` is set whenever a record is created, and likewise exactly one of `modified_by_user_id` / `modified_by_service_key_id` whenever it's modified. A `CHECK` constraint enforces this at the database level.
- The one exception is system-seeded rows inserted directly by a migration (e.g., the default "Administrador" profile, the permission catalog, seeded environment categories) — both actor columns may be left null there, since no User or Service Key performed the action.
- A row that is referenced by another row's `created_by`/`modified_by` columns must not be hard-deleted, since that would break that history. An entity that needs to support removal defines its own soft-delete mechanism (typically a `status` of Active/Inactive) in its own business requirement, instead of a physical `DELETE`. An entity with no such history dependency may still be hard-deleted — this is decided per entity, in its own requirement.
- Append-only audit-log tables (such as the Feature Flag Audit Trail) are exempt from carrying these six columns themselves — the log's own `actor` and `occurred_on` fields already serve that purpose, and a log entry is never modified after being written.

## Alternatives Considered

- **A single unifying `actors` table** wrapping both Users and Service Keys under one id space, with `created_by`/`modified_by` as a single FK to it — rejected for now, to avoid adding an abstraction layer before it's clearly needed. Can be revisited if the dual-column pattern proves unwieldy.
- **`created_by`/`created_on` only, no modification tracking** — rejected; knowing who last changed a record is necessary for the accountability this convention exists for.

## Consequences

- Every table's schema gains six columns; migrations and repository code must populate them consistently.
- A User or Service Key that has ever created or modified any record cannot be hard-deleted without breaking referential integrity — it must be deactivated instead.
- Every business requirement's Data Model section can reference this ADR instead of repeating these six columns in every table definition.

## References

N/A

## Author & Date

| Person | Role | Date | Description |
| -------- | ---- | ---- | ----------- |
| Nicolas Filipe Cunha | Founder | 2026-08-16 | Documented the audit columns and soft-delete convention. |
