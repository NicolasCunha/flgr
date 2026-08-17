# 0000 - Business Requirements Template

## Summary

| Type | Status | Date | Author |
| ---- | ------ | ---- | ------ |
| Functional / Non-functional | Proposed | 2026-08-16 | Nicolas Filipe Cunha |

## Overview

A short paragraph describing what the requirement is about and why it matters, from the perspective of the business or the user.

### Related Documents

Links to related requirements, ADRs, or other documents that provide useful context for this requirement.

| Document | Type | Description | Link |
| -------- | ---- | ----------- | ---- |
| ADR Template | Architecture | Documents the architectural decisions made for the flgr application. | [0000-adr-template.md](../../architecture/adr/0000-adr-template.md) |

## Content

A business requirement can be **functional** (a specific behavior or feature the application must provide) or **non-functional** (a quality attribute, performance characteristic, or constraint the application must satisfy). Use the `Type` field in the summary table above to indicate which kind of requirement this is.

Regardless of type, the content should be direct, measurable, and testable, so it's possible to verify whether the application meets it. If the requirement introduces or changes persisted data, describe the affected fields here and detail the underlying tables in the [Data Model](#data-model) section below.

**Example — functional requirement:**

> User Registration: The application must allow users to register for an account using their email address and a password. The registration process must include email verification to confirm the validity of the provided email address.

Fields of the User Registration form:
- First Name (text, required)
- Last Name (text, required)
- Email Address (email, required, unique)
- Password (password, required, minimum 8 characters)

**Example — non-functional requirement:**

> Performance: The application must respond to user requests within 2 seconds under normal load conditions.
>
> Security: The application must implement secure authentication and authorization mechanisms, hashing passwords with bcrypt (or an equivalent adaptive hashing algorithm).
>
> Scalability: The application must support a 100% increase in user traffic without degradation in performance.

## Data Model

_Optional — include this section only if the requirement introduces or changes persisted data. Otherwise, remove it._

Describe the tables involved, including columns, primary keys, foreign keys, and indexes needed to support this requirement. Repeat the block below for each table affected.

### Table: `users`

| Column | Type | Nullable | Description |
| ------ | ---- | -------- | ----------- |
| id | UUID | No | Unique identifier |
| first_name | VARCHAR(100) | No | User's first name |
| last_name | VARCHAR(100) | No | User's last name |
| email | VARCHAR(255) | No | User's email address |
| password_hash | VARCHAR(255) | No | bcrypt hash of the user's password |
| created_at | TIMESTAMP | No | Record creation timestamp |

**Primary Key:** `id`

**Foreign Keys:**

| Column | References | On Delete |
| ------ | ---------- | --------- |
| N/A | N/A | N/A |

**Indexes:**

| Name | Columns | Type | Description |
| ---- | ------- | ---- | ----------- |
| idx_users_email | email | UNIQUE | Enforces email uniqueness and speeds up login lookups |

## Acceptance Criteria

A list of conditions that must be true for this requirement to be considered met, written so they can be verified through testing or review.

- [ ] Criterion 1
- [ ] Criterion 2

## References

References to any relevant documentation, research, or other sources used when defining this requirement.

## Author & Date

Involves the author of the requirement and the date it was created. This section should include the name of the author, their role in the project, and the date the requirement was created.

| Person | Role | Date | Description |
| -------- | ---- | ---- | ----------- |
| Nicolas Filipe Cunha | Founder | 2026-08-16 | Created the business requirements template. |
