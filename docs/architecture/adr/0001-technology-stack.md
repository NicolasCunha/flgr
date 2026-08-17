# ADR-0001: Technology Stack

## Summary

| Category | Status | Date | Author |
| -------- | ------ | ---- | ------ |
| Backend / Frontend / Database | Accepted | 2026-08-16 | Nicolas Filipe Cunha |

**Supersedes:** N/A

**Superseded by:** N/A

See [Status Values](README.md#status-values) and [Categories](README.md#categories) in the ADR index for the allowed values in each field.

## Context

flgr needs a technology stack to support three things: an administrative UI for managing feature flags, environments, users, and service keys; an API for consumer applications to evaluate and manage flags; and a data store to persist that information.

This ADR formalizes the stack that was already in use, documenting it as an ADR so future decisions can reference it and so it can be revisited deliberately if it needs to change.

### Related Documents

| Document | Type | Description | Link |
| -------- | ---- | ----------- | ---- |
| Architecture Overview | Architecture | High-level overview of flgr's client/server design. | [README.md](../README.md) |

## Decision

- **Frontend:** React.js for the user interface, with Redux for state management.
- **Backend:** Golang for the RESTful API, with Gin for routing and HTTP request handling.
- **Database:** SQLite for storing feature flags, user profiles, and service keys.

## Alternatives Considered

Not formally evaluated — this ADR documents a decision that was already made before the ADR process was adopted, rather than a decision made from a set of options.

## Consequences

- Go's single static binary and low operational footprint fit well with shipping flgr as one deployable container (see [ADR-0002](0002-single-docker-image-with-nginx.md)).
- SQLite requires no separate database server, which simplifies self-hosted deployment for early stages of the project.
- SQLite has limited write concurrency. If flgr later needs to scale to multiple instances writing concurrently, a migration to a client-server database (e.g., PostgreSQL) will need its own ADR.

## References

N/A

## Author & Date

| Person | Role | Date | Description |
| -------- | ---- | ---- | ----------- |
| Nicolas Filipe Cunha | Founder | 2026-08-16 | Documented the existing technology stack as an ADR. |
