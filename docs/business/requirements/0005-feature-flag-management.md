# 0005 - Feature Flag Management

## Summary

| Type | Status | Date | Author |
| ---- | ------ | ---- | ------ |
| Functional | Accepted | 2026-08-16 | Nicolas Filipe Cunha |

## Overview

A Feature Flag lets flgr control the availability or configuration of a feature in a consumer application without a deployment. A flag has a single, stable definition, and an independent value per environment — a flag can be enabled in `app-dev` and disabled in `app-prod` at the same time. This requirement covers creating and managing that definition and its per-environment values. Emergency disabling with notification is covered separately by [0006-feature-flag-killswitch.md](0006-feature-flag-killswitch.md); read access by consumer applications is covered by [0007-feature-flag-evaluation-api.md](0007-feature-flag-evaluation-api.md).

### Related Documents

| Document | Type | Description | Link |
| -------- | ---- | ----------- | ---- |
| Audit Columns & Soft-Delete Convention | Architecture | Defines the audit columns referenced in the Data Model below. | [0005-audit-columns-and-soft-delete-convention.md](../../architecture/adr/0005-audit-columns-and-soft-delete-convention.md) |
| Environment Management | Business | Defines the `Environment` entity a flag's values are scoped to. | [0001-environment-management.md](0001-environment-management.md) |
| Profile & Permission Management | Business | Defines the `Feature Flag` and `Feature Flag Value` permissions that gate this requirement's actions. | [0003-profile-and-permission-management.md](0003-profile-and-permission-management.md) |

## Content

Administrators with the `Feature Flag: Create` permission can create a flag with:

- **Key** — a unique, machine-readable identifier consumer applications use to reference the flag (e.g., `new-checkout-flow`). Restricted to letters, numbers, hyphens, and underscores.
- **Name** — a human-readable display name.
- **Description** — free text explaining what the flag controls.
- **Type** — `Boolean`, `String`, `Number`, or `JSON`. The type is fixed once the flag is created, so consumer applications can rely on a stable value shape.

Separately, a user with the `Feature Flag Value: Write` permission sets, per environment, whether the flag is enabled and — for non-`Boolean` types — what value it holds when enabled. A flag with no configured value for a given environment is treated as disabled by default; an explicit row does not need to exist for every environment up front.

Editing a flag's value through this requirement does **not** trigger a notification to consumer applications — only the Killswitch action does (see [0006-feature-flag-killswitch.md](0006-feature-flag-killswitch.md)).

## Data Model

In addition to the columns below, every table also includes the audit columns defined in [ADR-0005](../../architecture/adr/0005-audit-columns-and-soft-delete-convention.md).

### Table: `feature_flags`

| Column | Type | Nullable | Description |
| ------ | ---- | -------- | ----------- |
| id | UUID | No | Unique identifier |
| key | VARCHAR(150) | No | Machine-readable identifier, unique |
| name | VARCHAR(150) | No | Human-readable display name |
| description | TEXT | Yes | Free-text description |
| type | VARCHAR(20) | No | `Boolean`, `String`, `Number`, or `JSON`; immutable after creation |

**Primary Key:** `id`

**Foreign Keys:** N/A

**Indexes:**

| Name | Columns | Type | Description |
| ---- | ------- | ---- | ----------- |
| idx_feature_flags_key | key | UNIQUE | Enforces key uniqueness and speeds up evaluation lookups |

### Table: `feature_flag_environment_values`

| Column | Type | Nullable | Description |
| ------ | ---- | -------- | ----------- |
| id | UUID | No | Unique identifier |
| feature_flag_id | UUID | No | The flag this value belongs to |
| environment_id | UUID | No | The environment this value applies to |
| enabled | BOOLEAN | No | Whether the flag is on in this environment, default `false` |
| value | TEXT | Yes | The value served when `enabled = true`, interpreted per the flag's `type`; unused for `Boolean` flags |

**Primary Key:** `id`

**Foreign Keys:**

| Column | References | On Delete |
| ------ | ---------- | --------- |
| feature_flag_id | feature_flags(id) | CASCADE |
| environment_id | environments(id) | RESTRICT |

**Indexes:**

| Name | Columns | Type | Description |
| ---- | ------- | ---- | ----------- |
| idx_feature_flag_environment_values_unique | feature_flag_id, environment_id | UNIQUE | At most one value per flag per environment |

## Acceptance Criteria

- [ ] A user with the `Feature Flag: Create` permission can create a flag with a unique key, name, description, and type.
- [ ] Flag keys are restricted to letters, numbers, hyphens, and underscores, and are unique; creation with a duplicate or invalid key is rejected.
- [ ] A flag's `type` cannot be changed after creation.
- [ ] A user with the `Feature Flag: Edit` permission can update a flag's name and description (not its key or type).
- [ ] A user with the `Feature Flag: Remove` permission can remove a flag definition, along with its per-environment values.
- [ ] A user with the `Feature Flag Value: Write` permission can set, for a given flag and environment, whether it is enabled and — for non-`Boolean` types — its value.
- [ ] A flag with no configured value for a given environment evaluates as disabled in that environment.
- [ ] Setting a flag's value through this requirement's actions does not trigger any notification.
- [ ] A user with `Feature Flag: View` or `Feature Flag Value: Read` (or the actions that always imply them) can list flags and view their per-environment values.

## References

N/A

## Author & Date

| Person | Role | Date | Description |
| -------- | ---- | ---- | ----------- |
| Nicolas Filipe Cunha | Founder | 2026-08-16 | Created the Feature Flag Management requirement. |
