# 0001 - Environment Management

## Summary

| Type | Status | Date | Author |
| ---- | ------ | ---- | ------ |
| Functional | Accepted | 2026-08-16 | Nicolas Filipe Cunha |

## Overview

An Environment represents a distinct stage of a consumer application's lifecycle (e.g., a development, staging, or production deployment) in which a feature flag can hold an independent value. Administrators create and manage environments; they are then used throughout the platform to scope feature flag values ([0005-feature-flag-management.md](0005-feature-flag-management.md)) and to restrict what a service key can access ([0004-service-key-management.md](0004-service-key-management.md)).

### Related Documents

| Document | Type | Description | Link |
| -------- | ---- | ----------- | ---- |
| Audit Columns & Soft-Delete Convention | Architecture | Defines the audit columns referenced in the Data Model below. | [0005-audit-columns-and-soft-delete-convention.md](../../architecture/adr/0005-audit-columns-and-soft-delete-convention.md) |
| Profile & Permission Management | Business | Defines the `Environment` permission that gates this requirement's actions. | [0003-profile-and-permission-management.md](0003-profile-and-permission-management.md) |

## Content

Administrators can create, edit, remove, and view environments. Each environment has:

- **Name** — a short identifier (e.g., `app-dev`), unique across the instance.
- **Description** — a free-text explanation of what the environment is for (e.g., "Development environment for the xpto application").
- **Category** — one of `Development`, `Staging`, or `Production`, or a custom value. When the user picks "Other" while creating or editing an environment, they type a custom category name; that name is added to the list of selectable categories for future environments (alongside `Other`, which always remains available to define another new one).

## Data Model

In addition to the columns below, every table also includes the audit columns defined in [ADR-0005](../../architecture/adr/0005-audit-columns-and-soft-delete-convention.md).

### Table: `environment_categories`

| Column | Type | Nullable | Description |
| ------ | ---- | -------- | ----------- |
| id | UUID | No | Unique identifier |
| name | VARCHAR(100) | No | Category name, unique (e.g., `Development`, `Staging`, `Production`, or a custom value) |
| is_system | BOOLEAN | No | `true` for the three seeded categories (`Development`, `Staging`, `Production`); `false` for a category created via "Other" |

**Primary Key:** `id`

**Foreign Keys:** N/A

**Indexes:**

| Name | Columns | Type | Description |
| ---- | ------- | ---- | ----------- |
| idx_environment_categories_name | name | UNIQUE | Prevents duplicate category names |

### Table: `environments`

| Column | Type | Nullable | Description |
| ------ | ---- | -------- | ----------- |
| id | UUID | No | Unique identifier |
| name | VARCHAR(100) | No | Environment name, unique |
| description | TEXT | Yes | Free-text description |
| environment_category_id | UUID | No | The environment's category |

**Primary Key:** `id`

**Foreign Keys:**

| Column | References | On Delete |
| ------ | ---------- | --------- |
| environment_category_id | environment_categories(id) | RESTRICT |

**Indexes:**

| Name | Columns | Type | Description |
| ---- | ------- | ---- | ----------- |
| idx_environments_name | name | UNIQUE | Prevents duplicate environment names |

## Acceptance Criteria

- [ ] A user with the `Environment: Create` permission can create an environment with a name, description, and category.
- [ ] Environment names are unique; attempting to create or rename to a name already in use is rejected.
- [ ] The category selector offers `Development`, `Staging`, `Production`, every previously created custom category, and `Other`.
- [ ] Selecting `Other` requires typing a new category name, which is persisted as a new, reusable, non-system category (`is_system = false`).
- [ ] The three seeded categories (`Development`, `Staging`, `Production`, `is_system = true`) cannot be edited or removed.
- [ ] A custom category (`is_system = false`) cannot be removed while any environment still references it.
- [ ] A user with the `Environment: Edit` permission can update an environment's name, description, or category.
- [ ] A user with the `Environment: Remove` permission can remove an environment, unless it is still referenced elsewhere (e.g., by a feature flag value or a service key), in which case removal is rejected.
- [ ] A user with the `Environment: View` permission (or `Create`/`Edit`/`Remove`, which always imply `View`) can list and view environments.

## References

N/A

## Author & Date

| Person | Role | Date | Description |
| -------- | ---- | ---- | ----------- |
| Nicolas Filipe Cunha | Founder | 2026-08-16 | Created the Environment Management requirement. |
