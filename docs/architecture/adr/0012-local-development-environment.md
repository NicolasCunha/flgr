# ADR-0012: Local Development Environment

## Summary

| Category | Status | Date | Author |
| -------- | ------ | ---- | ------ |
| Infra | Accepted | 2026-08-16 | Nicolas Filipe Cunha |

**Supersedes:** N/A

**Superseded by:** N/A

See [Status Values](README.md#status-values) and [Categories](README.md#categories) in the ADR index for the allowed values in each field.

## Context

[ADR-0009](0009-kafka-and-webhook-notification-delivery.md) made Kafka a real dependency, which isn't practical to install manually on every developer's machine. This ADR defines how the full stack — `flgr-server`, `flgr-web-client`, and Kafka — runs locally, mirroring production's containerized shape ([ADR-0002](0002-single-docker-image-with-nginx.md)) while still supporting a fast development loop.

### Related Documents

| Document | Type | Description | Link |
| -------- | ---- | ----------- | ---- |
| Single Docker Image with Nginx Routing | Architecture | The production image this local setup mirrors, without being identical to it (see Decision). | [0002-single-docker-image-with-nginx.md](0002-single-docker-image-with-nginx.md) |
| Kafka & HTTP Webhook Notification Delivery | Architecture | The reason Kafka is now a required local dependency. | [0009-kafka-and-webhook-notification-delivery.md](0009-kafka-and-webhook-notification-delivery.md) |
| Application Configuration & Secrets | Architecture | The `FLGR_*` environment variables this local setup must supply. | [0010-application-configuration-and-secrets.md](0010-application-configuration-and-secrets.md) |
| Authentication & Session Strategy | Architecture | The same-origin assumption behind the session cookie, which the Vite dev proxy (see Decision) preserves locally. | [0006-authentication-and-session-strategy.md](0006-authentication-and-session-strategy.md) |

## Decision

The whole stack runs via `docker compose up`, as separate services on one Compose network — this is **not** the same artifact as the production image from [ADR-0002](0002-single-docker-image-with-nginx.md), which is a single, already-built, static-file-serving image with no hot-reload. Local dev uses its own dev-only Dockerfiles/stages instead:

- **`kafka`** — a single container running Kafka in **KRaft mode** (no ZooKeeper — the mode Kafka itself now recommends for new setups).
- **`kafka-ui`** — a lightweight web UI (e.g., `provectuslabs/kafka-ui`) for inspecting topics and messages, so a killswitch notification ([ADR-0009](0009-kafka-and-webhook-notification-delivery.md)) can be visually confirmed during development.
- **`flgr-server`** — built from a dev-only Dockerfile/stage that runs the Go app under [`air`](https://github.com/air-verse/air) for hot-reload, with the source bind-mounted so file changes trigger an automatic rebuild/restart. Not the production binary from [ADR-0002](0002-single-docker-image-with-nginx.md).
- **`flgr-web-client`** — built from a dev-only Dockerfile/stage running `npm run dev -- --host 0.0.0.0` (required to expose Vite's dev server outside the container), with the source bind-mounted for HMR.

**Preserving same-origin for the session cookie:** since `flgr-web-client` and `flgr-server` run as separate containers/ports locally (unlike production, where Nginx unifies them behind one origin), Vite's dev server is configured to **proxy `/api/*` requests to the `flgr-server` container** (`server.proxy` in `vite.config.ts`). From the browser's point of view, the app is still same-origin, so the `HttpOnly` session cookie ([ADR-0006](0006-authentication-and-session-strategy.md)) works locally exactly as it does in production, without needing CORS.

**Environment variables:** a `.env.example` file is committed as a template for the `FLGR_*` variables ([ADR-0010](0010-application-configuration-and-secrets.md)); each developer copies it to a gitignored `.env`. Docker Compose's `env_file` reads it directly into the `flgr-server` container. In addition, `flgr-server` uses [`joho/godotenv`](https://github.com/joho/godotenv) to auto-load a local `.env` file (if present) at startup — this covers running the binary natively (e.g., `go run`, or attaching a debugger from an IDE) outside Compose entirely. In production, no `.env` file exists, so `godotenv` is a silent no-op and [ADR-0010](0010-application-configuration-and-secrets.md)'s environment-variables-only decision is unaffected.

SQLite ([ADR-0001](0001-technology-stack.md)) needs no service of its own — the database file lives on a bind-mounted path so it survives `docker compose down`/`up` cycles.

## Alternatives Considered

- **Kafka + ZooKeeper (classic mode)** — rejected in favor of KRaft mode, which Kafka itself is moving new deployments toward, avoiding a second container for no benefit here.
- **No local Kafka UI** — rejected; a lightweight UI is cheap to run and materially speeds up debugging notification delivery ([ADR-0009](0009-kafka-and-webhook-notification-delivery.md)) versus a command-line consumer.
- **Only a Makefile + shell-exported `.env`, no `godotenv`** — reconsidered in favor of also adding `godotenv` to `flgr-server`, so running the Go binary natively (outside Compose, e.g., under an IDE debugger) still picks up local configuration automatically, without requiring the developer to `docker compose up` just to debug the backend alone.
- **Running `flgr-server`/`flgr-web-client` natively (`go run` / `npm run dev`), with Compose only for Kafka** — reconsidered in favor of running everything in Compose, for closer parity with how the pieces are networked together in production, at the cost of needing dev-specific Dockerfiles (`air`, bind mounts) to keep the hot-reload loop fast despite containerization. The native commands documented in [`.ai/backend.md`](../../../.ai/backend.md) and [`.ai/frontend.md`](../../../.ai/frontend.md) remain valid and useful for quick, container-free debugging.

## Consequences

- `flgr-server` gains a dev-only dependency on `air` and `godotenv`; neither ships in the production image built per [ADR-0002](0002-single-docker-image-with-nginx.md)/[ADR-0011](0011-ci-cd-pipeline.md).
- A `docker-compose.yml` (plus dev Dockerfiles for `flgr-server` and `flgr-web-client`) is added at the repository root, alongside a committed `.env.example`.
- `vite.config.ts` needs a `server.proxy` entry for `/api`, kept in sync with the path prefix defined in [ADR-0007](0007-api-design-conventions.md).
- [`.ai/backend.md`](../../../.ai/backend.md) and [`.ai/frontend.md`](../../../.ai/frontend.md) need updating to document `docker compose up` as the primary way to run the full stack locally, alongside the existing native commands for standalone use.

## References

N/A

## Author & Date

| Person | Role | Date | Description |
| -------- | ---- | ---- | ----------- |
| Nicolas Filipe Cunha | Founder | 2026-08-16 | Documented the local development environment decision. |
