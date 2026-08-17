# ADR-0007: API Design Conventions

## Summary

| Category | Status | Date | Author |
| -------- | ------ | ---- | ------ |
| Backend / API | Accepted | 2026-08-16 | Nicolas Filipe Cunha |

**Supersedes:** N/A

**Superseded by:** N/A

See [Status Values](README.md#status-values) and [Categories](README.md#categories) in the ADR index for the allowed values in each field.

## Context

flgr's backend exposes a REST API consumed by two very different clients: the `flgr-web-client` admin panel, and consumer applications reading/writing flags via a Service Key ([0007](../../business/requirements/0007-feature-flag-evaluation-api.md)). Several requirements (e.g., the Audit Log in [0008](../../business/requirements/0008-feature-flag-audit-trail.md)) need list endpoints with pagination, filtering, and sorting, so a single, consistent convention is needed rather than deciding this ad hoc per endpoint.

### Related Documents

| Document | Type | Description | Link |
| -------- | ---- | ----------- | ---- |
| Single Docker Image with Nginx Routing | Architecture | Nginx routes an `/api/*` path prefix to the Go backend. | [0002-single-docker-image-with-nginx.md](0002-single-docker-image-with-nginx.md) |
| Audit Columns & Soft-Delete Convention | Architecture | "Remove" actions are soft-deletes/deactivations under the hood, which this ADR maps onto `DELETE`. | [0005-audit-columns-and-soft-delete-convention.md](0005-audit-columns-and-soft-delete-convention.md) |
| Authentication & Session Strategy | Architecture | Defines how a request authenticates (cookie for Users, `Authorization: Bearer` for Service Keys); this ADR does not redefine that. | [0006-authentication-and-session-strategy.md](0006-authentication-and-session-strategy.md) |
| Feature Flag Audit Trail | Business | The clearest consumer of the pagination/filter/sort conventions defined here. | [0008-feature-flag-audit-trail.md](../../business/requirements/0008-feature-flag-audit-trail.md) |

## Decision

- **Base path & versioning:** every route is prefixed `/api/v1/...` from the start (e.g., `/api/v1/feature-flags`).
- **Resource naming:** plural nouns, kebab-case for multi-word resources (`/service-keys`, `/feature-flags`, `/environment-categories`).
- **JSON casing:** `snake_case` for all request and response body keys, matching Go and the database's own naming. The frontend converts to/from `camelCase` at a single boundary (the RTK Query API slice's `transformResponse`/`transformRequest`), so the rest of the frontend code works with idiomatic `camelCase` TypeScript.
- **Write verbs:** `POST` to create, `PATCH` for partial updates (matching that most edits touch a subset of fields), `DELETE` to remove. `DELETE` is used even where the underlying action is a deactivation rather than a physical delete (per [ADR-0005](0005-audit-columns-and-soft-delete-convention.md)) — the caller doesn't need to know the difference; the API stays predictable and RESTful.
- **List endpoint envelope:**

  ```
  GET /api/v1/feature-flags/audit-log?page=1&page_size=50&sort=-occurred_on&environment_id=...

  {
    "data": [ ... ],
    "pagination": { "page": 1, "page_size": 50, "total": 812 }
  }
  ```

  - Pagination is page/page_size based (not cursor-based) — simpler to implement and sufficient at flgr's expected scale, and it supports jumping to an arbitrary page, which a cursor can't.
  - Sorting is a `sort` query parameter naming a field, with a `-` prefix for descending (e.g., `sort=-occurred_on`).
  - Filtering is one query parameter per filterable field, exact-match (e.g., `environment_id=...`, `action=ValueChanged`), as defined per endpoint by its business requirement (e.g., the filters listed in [0008](../../business/requirements/0008-feature-flag-audit-trail.md)).

- **Error envelope:**

  ```
  {
    "error": {
      "code": "validation_error",
      "message": "email is already in use",
      "details": [ { "field": "email", "reason": "duplicate" } ]
    }
  }
  ```

  `details` is optional and only present for multi-field validation errors.

- **HTTP status codes:** `200` (success), `201` (created), `400` (validation error), `401` (not authenticated), `403` (authenticated but lacking the required permission — see [0003](../../business/requirements/0003-profile-and-permission-management.md)), `404` (not found), `409` (conflict, e.g., a uniqueness violation), `500` (unexpected server error).

## Alternatives Considered

- **camelCase JSON** — rejected in favor of `snake_case`, to stay consistent with Go and the database's own naming, at the cost of a small conversion layer on the frontend (kept to one place, not scattered).
- **Cursor-based pagination** — rejected for now; page/page_size is simpler and sufficient at the expected scale. Can be revisited for a specific high-volume endpoint (most likely the audit log) later, without needing to change every other endpoint.
- **No API versioning yet** — rejected; a version prefix costs almost nothing to add now and avoids a painful URL migration later, which matters more here since external consumers depend on the Evaluation API.
- **Dedicated action endpoints for deactivation** (e.g., `POST /users/{id}/deactivate`) — rejected in favor of overloading `DELETE`, to keep the API's verb usage predictable; the soft-delete behavior underneath is an implementation detail, not part of the contract.

## Consequences

- Every list endpoint (Environments, Users, Service Keys, Profiles, Feature Flags, Audit Log) shares the same envelope, pagination, sorting, and filtering shape, so the frontend can build one reusable data-table hook instead of one per screen.
- The frontend needs a `snake_case` ↔ `camelCase` conversion at the RTK Query boundary; this is centralized, not duplicated per feature.
- All new endpoints are added under `/api/v1`; a breaking change gets its own `/api/v2` rather than modifying `v1` in place.

## References

N/A

## Author & Date

| Person | Role | Date | Description |
| -------- | ---- | ---- | ----------- |
| Nicolas Filipe Cunha | Founder | 2026-08-16 | Documented the API design conventions decision. |
