# ADR-0002: Single Docker Image with Nginx Routing

## Summary

| Category | Status | Date | Author |
| -------- | ------ | ---- | ------ |
| Infra | Accepted | 2026-08-16 | Nicolas Filipe Cunha |

**Supersedes:** N/A

**Superseded by:** N/A

See [Status Values](README.md#status-values) and [Categories](README.md#categories) in the ADR index for the allowed values in each field.

## Context

flgr needs a packaging and deployment strategy. To keep operations simple — particularly for self-hosted deployments — the goal is a single artifact to build, version, and run, rather than coordinating separate frontend and backend containers (e.g., via docker-compose).

### Related Documents

| Document | Type | Description | Link |
| -------- | ---- | ----------- | ---- |
| Technology Stack | Architecture | Establishes React for the frontend and Go/Gin for the backend. | [0001-technology-stack.md](0001-technology-stack.md) |

## Decision

flgr ships as a single Docker image containing:

- The compiled Go backend binary (the API server).
- The React production build (static assets).
- Nginx, configured to serve the React static build directly and reverse-proxy API requests to the Go backend process running inside the same container.

Nginx and the Go binary both run inside the same container, which requires a process supervisor (e.g., a lightweight init or supervisord) to start and monitor both. The exact supervisor and nginx routing configuration are implementation details to be resolved separately (in the Dockerfile/deployment config, or a follow-up ADR if the choice turns out to be non-trivial).

## Alternatives Considered

- **Separate frontend and backend containers** (e.g., docker-compose) — rejected because it works against the goal of a single, simple artifact to distribute and run.
- **Go embed (`go:embed`) + Gin static route, no Nginx** — considered, since it would keep the container to a single process. Not chosen because Nginx provides more mature static file serving and reverse proxy configuration out of the box.

## Consequences

- Single image to build, version, and deploy; self-hosted users can run it with a single `docker run`.
- Requires a process supervisor inside the container to run both Nginx and the Go binary, which deviates from the "one process per container" convention.
- The image is slightly larger than a Go-only image, since it also bundles Nginx.
- Follow-up needed: pick the process supervisor and define the Nginx routing rules (static assets vs. API proxy path, e.g. `/api/*`).

## References

N/A

## Author & Date

| Person | Role | Date | Description |
| -------- | ---- | ---- | ----------- |
| Nicolas Filipe Cunha | Founder | 2026-08-16 | Documented the single-image packaging decision. |
