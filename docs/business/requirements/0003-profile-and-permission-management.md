# 0003 - Profile & Permission Management (RBAC)

## Summary

| Type | Status | Date | Author |
| ---- | ------ | ---- | ------ |
| Functional | Accepted | 2026-08-16 | Nicolas Filipe Cunha |

## Overview

flgr controls what each User can do through Profiles: named, reusable groups of permissions. A user can belong to more than one profile, and can also be granted permissions directly, bypassing profiles entirely. This requirement defines profiles, the permission catalog, how users are assigned to profiles, and the rules used to resolve a user's effective permissions.

### Related Documents

| Document | Type | Description | Link |
| -------- | ---- | ----------- | ---- |
| Audit Columns & Soft-Delete Convention | Architecture | Defines the audit columns referenced in the Data Model below. | [0005-audit-columns-and-soft-delete-convention.md](../../architecture/adr/0005-audit-columns-and-soft-delete-convention.md) |
| User Management | Business | Defines the `User` entity that profiles and permissions are assigned to. | [0002-user-management.md](0002-user-management.md) |

## Content

### Permission Catalog

Permissions are a fixed, system-defined catalog — administrators grant or revoke them via profiles and users, but cannot create new ones. Each permission is a `(resource, action)` pair:

| Resource | Actions |
| -------- | ------- |
| Environment | Create, Edit, Remove, View |
| Feature Flag | Create, Edit, Remove, View |
| Feature Flag Value | Read, Write |
| Service Key | Create, Edit, Remove, View |
| Profile | Create, Edit, Remove, View |
| User | Create, Edit, Remove, View |

Permissions apply globally across the whole instance — they are not scoped per environment. (A Service Key's own, separate read/write access is scoped per environment; see [0004-service-key-management.md](0004-service-key-management.md).)

**CRUD-implies-View rule:** whenever a user or profile is granted `Create`, `Edit`, or `Remove` for a resource, `View` for that same resource is always treated as granted too, whether or not it was explicitly assigned.

### Profiles

A Profile is a named group of permissions (e.g., `app-read`, `app-read-write`). Administrators create profiles, assign permissions to them, and assign users to them.

flgr always has exactly one system profile, **Administrador**, seeded with every permission in the catalog. It cannot be removed unless another profile also holds the full permission catalog — this guarantees the instance is never left without an administrator.

### Assigning Permissions

A user's access comes from two sources:

1. **Profiles** — every permission granted to any profile the user belongs to.
2. **Direct grants** — permissions assigned straight to the user, bypassing profiles.

### Resolving Effective Permissions

For a given user and a given permission:

- If the user belongs to more than one profile, and those profiles disagree on a permission, the most permissive answer wins (i.e., if any of the user's profiles grants it, the user has it).
- If a permission is granted directly to the user, that grant always wins, even if none of the user's profiles grant it (a profile can never take away what was explicitly given to the user directly).
- In short: a user has a permission if it's granted by *any* applicable source — a direct grant, or any profile they belong to. There is no explicit "deny" that overrides a "yes" from another source.

## Data Model

In addition to the columns below, every table also includes the audit columns defined in [ADR-0005](../../architecture/adr/0005-audit-columns-and-soft-delete-convention.md).

### Table: `profiles`

| Column | Type | Nullable | Description |
| ------ | ---- | -------- | ----------- |
| id | UUID | No | Unique identifier |
| name | VARCHAR(100) | No | Profile name, unique |
| description | TEXT | Yes | Free-text description |
| is_system | BOOLEAN | No | `true` only for the seeded `Administrador` profile |

**Primary Key:** `id`

**Indexes:**

| Name | Columns | Type | Description |
| ---- | ------- | ---- | ----------- |
| idx_profiles_name | name | UNIQUE | Prevents duplicate profile names |

### Table: `permissions`

System-seeded catalog; not directly created, edited, or removed by users.

| Column | Type | Nullable | Description |
| ------ | ---- | -------- | ----------- |
| id | UUID | No | Unique identifier |
| resource | VARCHAR(50) | No | e.g., `Environment`, `FeatureFlag`, `FeatureFlagValue`, `ServiceKey`, `Profile`, `User` |
| action | VARCHAR(20) | No | e.g., `Create`, `Edit`, `Remove`, `View`, `Read`, `Write` |

**Primary Key:** `id`

**Indexes:**

| Name | Columns | Type | Description |
| ---- | ------- | ---- | ----------- |
| idx_permissions_resource_action | resource, action | UNIQUE | Prevents duplicate catalog entries |

### Table: `profile_permissions`

| Column | Type | Nullable | Description |
| ------ | ---- | -------- | ----------- |
| id | UUID | No | Unique identifier |
| profile_id | UUID | No | The profile being granted the permission |
| permission_id | UUID | No | The permission being granted |

**Primary Key:** `id`

**Foreign Keys:**

| Column | References | On Delete |
| ------ | ---------- | --------- |
| profile_id | profiles(id) | CASCADE |
| permission_id | permissions(id) | RESTRICT |

**Indexes:**

| Name | Columns | Type | Description |
| ---- | ------- | ---- | ----------- |
| idx_profile_permissions_unique | profile_id, permission_id | UNIQUE | Prevents granting the same permission to a profile twice |

### Table: `user_profiles`

| Column | Type | Nullable | Description |
| ------ | ---- | -------- | ----------- |
| id | UUID | No | Unique identifier |
| user_id | UUID | No | The user being assigned |
| profile_id | UUID | No | The profile being assigned |

**Primary Key:** `id`

**Foreign Keys:**

| Column | References | On Delete |
| ------ | ---------- | --------- |
| user_id | users(id) | CASCADE |
| profile_id | profiles(id) | CASCADE |

**Indexes:**

| Name | Columns | Type | Description |
| ---- | ------- | ---- | ----------- |
| idx_user_profiles_unique | user_id, profile_id | UNIQUE | Prevents assigning the same profile to a user twice |

### Table: `user_permissions`

Direct, profile-bypassing grants.

| Column | Type | Nullable | Description |
| ------ | ---- | -------- | ----------- |
| id | UUID | No | Unique identifier |
| user_id | UUID | No | The user being granted the permission |
| permission_id | UUID | No | The permission being granted |

**Primary Key:** `id`

**Foreign Keys:**

| Column | References | On Delete |
| ------ | ---------- | --------- |
| user_id | users(id) | CASCADE |
| permission_id | permissions(id) | RESTRICT |

**Indexes:**

| Name | Columns | Type | Description |
| ---- | ------- | ---- | ----------- |
| idx_user_permissions_unique | user_id, permission_id | UNIQUE | Prevents granting the same permission to a user twice |

## Acceptance Criteria

- [ ] The `Administrador` profile exists by default (`is_system = true`) and holds every permission in the catalog.
- [ ] Removing the `Administrador` profile is rejected unless another profile currently holds the full permission catalog.
- [ ] A user with the `Profile: Create` permission can create a new, non-system profile and assign permissions to it.
- [ ] A user with the `Profile: Edit` permission can add or remove permissions on a non-system profile.
- [ ] A user with the `User: Edit` permission can assign or remove a user's profile memberships, and grant or revoke permissions directly on the user.
- [ ] Granting `Create`, `Edit`, or `Remove` on a resource always makes `View` on that resource effective, without a separate grant.
- [ ] When a user belongs to multiple profiles with conflicting permissions, the union of all granted permissions applies (most permissive wins).
- [ ] A direct grant on a user is always effective, even if none of the user's profiles grant that permission.
- [ ] A user with no profiles and no direct grants has no permissions.

## References

N/A

## Author & Date

| Person | Role | Date | Description |
| -------- | ---- | ---- | ----------- |
| Nicolas Filipe Cunha | Founder | 2026-08-16 | Created the Profile & Permission Management requirement. |
