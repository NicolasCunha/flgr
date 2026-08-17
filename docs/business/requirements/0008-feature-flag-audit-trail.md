# 0008 - Feature Flag Audit Trail

## Summary

| Type | Status | Date | Author |
| ---- | ------ | ---- | ------ |
| Functional | Accepted | 2026-08-16 | Nicolas Filipe Cunha |

## Overview

Every change made to a feature flag — its definition or its per-environment value — is recorded in an append-only audit trail, regardless of whether it was made by a User through the UI or by a Service Key through the API. This is a dedicated log, distinct from the generic `created_by`/`modified_by` columns defined in [ADR-0005](../../architecture/adr/0005-audit-columns-and-soft-delete-convention.md): it keeps a full history of changes, not just the most recent one.

### Related Documents

| Document | Type | Description | Link |
| -------- | ---- | ----------- | ---- |
| Audit Columns & Soft-Delete Convention | Architecture | The generic per-table convention this requirement's log deliberately does not reuse (see Content below). | [0005-audit-columns-and-soft-delete-convention.md](../../architecture/adr/0005-audit-columns-and-soft-delete-convention.md) |
| Feature Flag Management | Business | Defines the create/edit/remove/value-change operations this log records. | [0005-feature-flag-management.md](0005-feature-flag-management.md) |
| Feature Flag Killswitch | Business | Defines the killswitch operation this log records. | [0006-feature-flag-killswitch.md](0006-feature-flag-killswitch.md) |

## Content

An entry is written whenever one of the following happens to a feature flag, whether triggered via the UI (by a User) or the API (by a Service Key):

- The flag's definition is created, edited, or removed ([0005-feature-flag-management.md](0005-feature-flag-management.md)).
- A per-environment value is changed ([0005-feature-flag-management.md](0005-feature-flag-management.md)).
- The Killswitch is triggered for an environment ([0006-feature-flag-killswitch.md](0006-feature-flag-killswitch.md)).

Each entry records what happened, who did it (a User or a Service Key), whether it came from the UI or the API, and the value before and after the change. Entries are immutable once written — there is no action to edit or delete an audit entry, including for administrators.

Read/evaluation calls (see [0007-feature-flag-evaluation-api.md](0007-feature-flag-evaluation-api.md)) are not recorded here.

### Viewing the Audit Trail

The audit trail is viewable in two places:

- A **global Audit Log screen**, listing entries across all flags, filterable by flag, environment, action, actor, and date range.
- A **History tab** on each flag's detail page, showing only that flag's entries, pre-filtered to it and offering the same remaining filters (environment, action, actor, date range).

Both views require the `Feature Flag: View` permission and list entries most recent first.

## Data Model

This table intentionally does **not** use the standard audit columns from [ADR-0005](../../architecture/adr/0005-audit-columns-and-soft-delete-convention.md) — its own `actor_*`/`source`/`occurred_on` columns already capture who/when, and an entry is never modified after being written, so `modified_by`/`modified_on` would not apply.

### Table: `feature_flag_audit_logs`

| Column | Type | Nullable | Description |
| ------ | ---- | -------- | ----------- |
| id | UUID | No | Unique identifier |
| feature_flag_id | UUID | No | The flag this entry is about |
| environment_id | UUID | Yes | The environment affected; `NULL` for flag-definition-level entries (create/edit/remove) |
| action | VARCHAR(30) | No | `FlagCreated`, `FlagEdited`, `FlagRemoved`, `ValueChanged`, or `KillswitchTriggered` |
| actor_user_id | UUID | Yes | The User who performed the action, if applicable |
| actor_service_key_id | UUID | Yes | The Service Key that performed the action, if applicable |
| source | VARCHAR(10) | No | `UI` or `API` |
| old_value | TEXT | Yes | The value before the change, if applicable |
| new_value | TEXT | Yes | The value after the change, if applicable |
| occurred_on | TIMESTAMP | No | When the action happened |

**Primary Key:** `id`

**Foreign Keys:**

| Column | References | On Delete |
| ------ | ---------- | --------- |
| feature_flag_id | feature_flags(id) | RESTRICT |
| environment_id | environments(id) | RESTRICT |
| actor_user_id | users(id) | RESTRICT |
| actor_service_key_id | service_keys(id) | RESTRICT |

**Indexes:**

| Name | Columns | Type | Description |
| ---- | ------- | ---- | ----------- |
| idx_feature_flag_audit_logs_flag | feature_flag_id, occurred_on | INDEX | Speeds up viewing a flag's full history in order |

## Acceptance Criteria

- [ ] Every create, edit, or removal of a flag's definition writes an entry with `environment_id = NULL`.
- [ ] Every per-environment value change writes an entry with the affected `environment_id`, `old_value`, and `new_value`.
- [ ] Every Killswitch trigger writes an entry with `action = KillswitchTriggered`, the affected `environment_id`, and the resulting `new_value`.
- [ ] Exactly one of `actor_user_id` / `actor_service_key_id` is set on every entry, matching who performed the action.
- [ ] `source` correctly reflects whether the action came from the UI or the API.
- [ ] No action exists to edit or delete an existing audit entry.
- [ ] Read/evaluation calls (see [0007-feature-flag-evaluation-api.md](0007-feature-flag-evaluation-api.md)) do not produce an entry.
- [ ] A user with the `Feature Flag: View` permission can access a global Audit Log screen listing entries across all flags, most recent first.
- [ ] The global Audit Log screen can be filtered by flag, environment, action, actor, and date range.
- [ ] A flag's detail page has a History tab showing only that flag's entries, most recent first, filterable by environment, action, actor, and date range.

## References

N/A

## Author & Date

| Person | Role | Date | Description |
| -------- | ---- | ---- | ----------- |
| Nicolas Filipe Cunha | Founder | 2026-08-16 | Created the Feature Flag Audit Trail requirement. |
