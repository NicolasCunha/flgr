# 0004 - Service Key Management

## Summary

| Type | Status | Date | Author |
| ---- | ------ | ---- | ------ |
| Functional | Accepted | 2026-08-16 | Nicolas Filipe Cunha |

## Overview

A Service Key is how a consumer application authenticates against flgr — it is not tied to a human User or to a Profile. Administrators create service keys, scope each one to the environment(s) it may access, and decide whether it can read flag values, write them, or both. This requirement covers managing service keys; how a service key is actually used to read or change flag values is covered by [0007-feature-flag-evaluation-api.md](0007-feature-flag-evaluation-api.md).

### Related Documents

| Document | Type | Description | Link |
| -------- | ---- | ----------- | ---- |
| Audit Columns & Soft-Delete Convention | Architecture | Defines the audit columns referenced in the Data Model below, and the soft-delete/deactivation principle applied to `status` here. | [0005-audit-columns-and-soft-delete-convention.md](../../architecture/adr/0005-audit-columns-and-soft-delete-convention.md) |
| Environment Management | Business | Defines the `Environment` entity a service key is linked to. | [0001-environment-management.md](0001-environment-management.md) |
| Profile & Permission Management | Business | Defines the `Service Key` permission that gates this requirement's actions (managing service keys is a User/Profile concern; a service key's own read/write access is separate — see Content below). | [0003-profile-and-permission-management.md](0003-profile-and-permission-management.md) |

## Content

Administrators with the `Service Key: Create` permission can create a service key. A service key has:

- **Name** — a human-readable label (e.g., "checkout-service prod key").
- **Secret** — a generated credential; it is shown in full only once, at creation time, and stored thereafter only as a hash (never retrievable again, mirroring how user passwords are handled).
- **Status** — `Active` or `Inactive`. Consumer applications can only authenticate with an `Active` key.
- **Environments** — one or more environments the key is allowed to access. Requests scoped to an environment the key isn't linked to are rejected.
- **Access** — `can_read` and/or `can_write`, independent of each other and of the requesting User's own permissions (a service key is not assigned a Profile). At least one of the two must be enabled.

## Data Model

In addition to the columns below, every table also includes the audit columns defined in [ADR-0005](../../architecture/adr/0005-audit-columns-and-soft-delete-convention.md).

### Table: `service_keys`

| Column | Type | Nullable | Description |
| ------ | ---- | -------- | ----------- |
| id | UUID | No | Unique identifier |
| name | VARCHAR(150) | No | Human-readable label |
| secret_hash | VARCHAR(255) | No | Hash of the generated secret; the plaintext is never stored |
| status | VARCHAR(20) | No | `Active` or `Inactive`, default `Active` |
| can_read | BOOLEAN | No | Whether the key can read flag values, default `true` |
| can_write | BOOLEAN | No | Whether the key can write flag values, default `false` |

**Primary Key:** `id`

**Foreign Keys:** N/A

**Indexes:** N/A

### Table: `service_key_environments`

| Column | Type | Nullable | Description |
| ------ | ---- | -------- | ----------- |
| id | UUID | No | Unique identifier |
| service_key_id | UUID | No | The service key being scoped |
| environment_id | UUID | No | The environment the key may access |

**Primary Key:** `id`

**Foreign Keys:**

| Column | References | On Delete |
| ------ | ---------- | --------- |
| service_key_id | service_keys(id) | CASCADE |
| environment_id | environments(id) | RESTRICT |

**Indexes:**

| Name | Columns | Type | Description |
| ---- | ------- | ---- | ----------- |
| idx_service_key_environments_unique | service_key_id, environment_id | UNIQUE | Prevents linking the same environment to a key twice |

## Acceptance Criteria

- [ ] A user with the `Service Key: Create` permission can create a service key with a name, at least one linked environment, and at least one of `can_read`/`can_write` enabled.
- [ ] The generated secret is displayed in full exactly once, at creation; only its hash is ever stored or retrievable afterward.
- [ ] A request authenticated with an `Inactive` service key is rejected, regardless of its `can_read`/`can_write` settings.
- [ ] A request scoped to an environment the service key is not linked to is rejected.
- [ ] A user with the `Service Key: Edit` permission can rename a key, change its status, adjust its `can_read`/`can_write` flags, or change its linked environments.
- [ ] A user with the `Service Key: Remove` permission deactivates (`status = Inactive`) a service key rather than deleting the record, per [ADR-0005](../../architecture/adr/0005-audit-columns-and-soft-delete-convention.md).
- [ ] A user with the `Service Key: View` permission (or `Create`/`Edit`/`Remove`, which always imply `View`) can list and view service keys — the secret itself is never shown again, only its metadata.

## References

N/A

## Author & Date

| Person | Role | Date | Description |
| -------- | ---- | ---- | ----------- |
| Nicolas Filipe Cunha | Founder | 2026-08-16 | Created the Service Key Management requirement. |
