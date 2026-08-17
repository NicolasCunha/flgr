# 0002 - User Management

## Summary

| Type | Status | Date | Author |
| ---- | ------ | ---- | ------ |
| Functional | Accepted | 2026-08-16 | Nicolas Filipe Cunha |

## Overview

A User is a person who administers the flgr platform through the web UI (as opposed to a consumer application, which accesses flgr through a Service Key — see [0004-service-key-management.md](0004-service-key-management.md)). Users are created by other administrators, not through public self-registration, and authenticate with an email and password. What a user can do once authenticated is governed by [0003-profile-and-permission-management.md](0003-profile-and-permission-management.md), which this requirement does not cover.

### Related Documents

| Document | Type | Description | Link |
| -------- | ---- | ----------- | ---- |
| Audit Columns & Soft-Delete Convention | Architecture | Defines the audit columns referenced in the Data Model below, and the soft-delete/deactivation principle applied to `status` here. | [0005-audit-columns-and-soft-delete-convention.md](../../architecture/adr/0005-audit-columns-and-soft-delete-convention.md) |
| Profile & Permission Management | Business | Defines the `User` permission that gates this requirement's actions, and how users are assigned profiles. | [0003-profile-and-permission-management.md](0003-profile-and-permission-management.md) |

## Content

Administrators with the `User: Create` permission can create a user account. A user has:

- **First Name** and **Last Name** (text, required).
- **Email Address** (required, unique) — used to log in.
- **Password** — hashed with bcrypt (or an equivalent adaptive hashing algorithm); the plaintext password is never stored or logged.
- **Status** — `Active` or `Inactive`. A deactivated user cannot log in, but their historical `created_by`/`modified_by` references remain intact (see [ADR-0005](../../architecture/adr/0005-audit-columns-and-soft-delete-convention.md) — users are never hard-deleted, since they may be referenced as the author of other records).

A user can always view and update their own name, email, and password, regardless of their assigned permissions. Managing other users' accounts requires the corresponding `User` permission.

## Data Model

In addition to the columns below, every table also includes the audit columns defined in [ADR-0005](../../architecture/adr/0005-audit-columns-and-soft-delete-convention.md).

### Table: `users`

| Column | Type | Nullable | Description |
| ------ | ---- | -------- | ----------- |
| id | UUID | No | Unique identifier |
| first_name | VARCHAR(100) | No | User's first name |
| last_name | VARCHAR(100) | No | User's last name |
| email | VARCHAR(255) | No | Login identifier, unique |
| password_hash | VARCHAR(255) | No | bcrypt hash of the user's password |
| status | VARCHAR(20) | No | `Active` or `Inactive`, default `Active` |

**Primary Key:** `id`

**Foreign Keys:** N/A

**Indexes:**

| Name | Columns | Type | Description |
| ---- | ------- | ---- | ----------- |
| idx_users_email | email | UNIQUE | Enforces email uniqueness and speeds up login lookups |

## Acceptance Criteria

- [ ] A user with the `User: Create` permission can create a user with first name, last name, email, and an initial password.
- [ ] Email addresses are unique and validated for a well-formed format; creation or update to an email already in use is rejected.
- [ ] Passwords are stored only as a bcrypt hash; the plaintext value is never persisted or logged.
- [ ] A user can only authenticate (log in) if their `status` is `Active`.
- [ ] A user can view and update their own first name, last name, email, and password without needing the `User: Edit` permission.
- [ ] A user with the `User: Edit` permission can update another user's first name, last name, email, status, or reset their password.
- [ ] A user with the `User: Remove` permission deactivates (`status = Inactive`) a user rather than deleting the record, per [ADR-0005](../../architecture/adr/0005-audit-columns-and-soft-delete-convention.md).
- [ ] A user with the `User: View` permission (or `Create`/`Edit`/`Remove`, which always imply `View`) can list and view users.

## References

N/A

## Author & Date

| Person | Role | Date | Description |
| -------- | ---- | ---- | ----------- |
| Nicolas Filipe Cunha | Founder | 2026-08-16 | Created the User Management requirement. |
