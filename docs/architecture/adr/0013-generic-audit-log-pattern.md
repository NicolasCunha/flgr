# ADR-0013: Generic Audit Log Pattern

## Summary

| Category | Status | Date | Author |
| -------- | ------ | ---- | ------ |
| Database | Accepted | 2026-08-17 | Nicolas Filipe Cunha |

**Supersedes:** N/A

**Superseded by:** N/A

See [Status Values](README.md#status-values) and [Categories](README.md#categories) in the ADR index for the allowed values in each field.

## Context

[0008-feature-flag-audit-trail.md](../../business/requirements/0008-feature-flag-audit-trail.md) originally defined a single, flag-specific `feature_flag_audit_logs` table: one row per change to a flag's definition or per-environment value, with its own `actor_*`/`source`/`occurred_on` columns (exempted from the standard audit columns in [ADR-0005](0005-audit-columns-and-soft-delete-convention.md), since it's append-only and never modified after being written).

flgr wants audit logging to be a capability the whole system can reuse, not something reinvented per entity — future requirements may need an audit trail for Users, Environments, Service Keys, or Profiles, each of which would otherwise need its own near-identical parallel table (same actor/source/occurred_on shape, different FK). This ADR extracts the reusable shape.

Only the feature flag write path is implemented under this pattern for now — no accepted requirement yet defines what on Users, Environments, Service Keys, or Profiles must be audited. This ADR still creates a link table for each of them now, so that when such a requirement lands, it only needs write logic, not a new table shape.

### Related Documents

| Document | Type | Description | Link |
| -------- | ---- | ----------- | ---- |
| Feature Flag Audit Trail | Business | The requirement this pattern now backs; its Data Model section points here. | [0008-feature-flag-audit-trail.md](../../business/requirements/0008-feature-flag-audit-trail.md) |
| Audit Columns & Soft-Delete Convention | Architecture | Establishes the standard audit columns this table remains exempt from, and the FK-integrity discipline this decision preserves. | [0005-audit-columns-and-soft-delete-convention.md](0005-audit-columns-and-soft-delete-convention.md) |

## Decision

- **`audit_log`** — one shared table for every audited entity: `id`, `entity_type` (e.g. `FeatureFlag`, `User`, `Environment`, `ServiceKey`, `Profile`), `action` (`Created`, `Updated`, `Removed`, `ValueChanged`, `Triggered` — generic verbs; the entity type already comes from `entity_type`/which link table has a row), `actor_user_id` / `actor_service_key_id` (exactly one set), `source` (`UI`/`API`), `old_value`/`new_value` (generic `TEXT`, format left to the caller), `occurred_on`.
- **One link table per audited entity** (`audit_log_feature_flag`, `audit_log_user`, `audit_log_environment`, `audit_log_service_key`, `audit_log_profile`), each just `audit_log_id` (PK, FK → `audit_log`) plus a real FK to the audited row, and any dimension specific to that entity (e.g. `audit_log_feature_flag.environment_id`, nullable for flag-definition-level entries). A new audited entity in the future adds one small link table, not a parallel log table.
- **`entity_type` is intentionally redundant** with "which link table has a row for this `audit_log_id`" — it exists purely so the global Audit Log screen ([0008](../../business/requirements/0008-feature-flag-audit-trail.md)) can filter/list by entity type directly on `audit_log`, without joining every link table to figure out what kind of entity each row is about. The link table remains the source of truth for the actual FK relationship.
- A per-flag History tab queries `audit_log_feature_flag` filtered by `feature_flag_id`, joined to `audit_log` for the rest of the fields and for `occurred_on` ordering.
- `audit_log` and all link tables remain exempt from the standard audit columns ([ADR-0005](0005-audit-columns-and-soft-delete-convention.md)), for the same reason `feature_flag_audit_logs` was: the `actor_*`/`source`/`occurred_on` columns already capture who/when, and a row is never modified after being written.

## Alternatives Considered

- **Keep the flag-specific `feature_flag_audit_logs` table** (original 0008 design) — rejected; doesn't generalize, would mean a full parallel table + index set every time another entity needs an audit trail.
- **Fully polymorphic single table** (`audit_log` with `entity_type` + `entity_id`, no link tables at all) — rejected; SQLite can't express a real `FOREIGN KEY` on a column whose target table depends on another column's value, so `entity_id` would have no referential integrity. That breaks the FK discipline [ADR-0005](0005-audit-columns-and-soft-delete-convention.md) established as the project-wide default.
- **Denormalize `occurred_on` onto each link table**, so the History tab's query doesn't need to join `audit_log` for ordering — rejected for now; adds a duplicated timestamp for a join cost that's acceptable at flgr's expected data volume (an admin tool, not a high-write telemetry pipeline). Revisit if profiling ever shows the join is a real bottleneck.

## Consequences

- `feature_flag_audit_logs` (from migration `000001`) is dropped and replaced by `audit_log` + `audit_log_feature_flag` in migration `000002`. `audit_log_user`, `audit_log_environment`, `audit_log_service_key`, and `audit_log_profile` are also created in `000002`, but unused — no application code writes to them until each entity's own audit requirement exists.
- Any future business requirement needing an audit trail for a new entity should reference this ADR and reuse the pattern (one link table) rather than re-deciding the shape.
- [0008](../../business/requirements/0008-feature-flag-audit-trail.md)'s Data Model section is updated to reference `audit_log` / `audit_log_feature_flag` instead of `feature_flag_audit_logs`; its Content and Acceptance Criteria are unaffected, since they describe behavior, not table shape.

## References

N/A

## Author & Date

| Person | Role | Date | Description |
| -------- | ---- | ---- | ----------- |
| Nicolas Filipe Cunha | Founder | 2026-08-17 | Documented the generic audit log pattern. |
