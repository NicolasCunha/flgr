# ADR-0010: Application Configuration & Secrets

## Summary

| Category | Status | Date | Author |
| -------- | ------ | ---- | ------ |
| Backend | Accepted | 2026-08-16 | Nicolas Filipe Cunha |

**Supersedes:** N/A

**Superseded by:** N/A

See [Status Values](README.md#status-values) and [Categories](README.md#categories) in the ADR index for the allowed values in each field.

## Context

`flgr-server` needs a consistent way to be configured at startup (HTTP port, database path, Kafka brokers, etc.) and a way to protect secrets that — unlike passwords or service key secrets, which are only ever hashed and compared — must be retrievable in plaintext by the application. The clearest example is the HTTP webhook `signing_secret` from [ADR-0009](0009-kafka-and-webhook-notification-delivery.md), which the admin needs to read back to configure the receiving system.

### Related Documents

| Document | Type | Description | Link |
| -------- | ---- | ----------- | ---- |
| Single Docker Image with Nginx Routing | Architecture | flgr ships as a single container, which favors environment-variable configuration over a mounted config file. | [0002-single-docker-image-with-nginx.md](0002-single-docker-image-with-nginx.md) |
| Kafka & HTTP Webhook Notification Delivery | Architecture | Defines the `signing_secret` this ADR specifies the at-rest protection for. | [0009-kafka-and-webhook-notification-delivery.md](0009-kafka-and-webhook-notification-delivery.md) |

## Decision

- **Configuration source:** environment variables only, no config file. Every variable is prefixed `FLGR_` to keep it unambiguous and to avoid colliding with the platform's own `Environment` business entity ([0001](../../business/requirements/0001-environment-management.md)), which is an unrelated concept.
- **Loading:** parsed manually with the standard library (`os.Getenv`) into a single `Config` struct, in `flgr-server/internal/config`, consistent with [`.ai/backend.md`](../../../.ai/backend.md)'s preference for the standard library over a new dependency. Validation happens once, at startup.
- **Fail-fast startup:** a missing or invalid required variable logs a clear error and exits the process (`os.Exit(1)`) before the server starts listening — this is one of the few places a startup-time `panic`/exit is acceptable, per [`.ai/backend.md`](../../../.ai/backend.md).
- **Baseline variables:**

  | Variable | Required | Default | Purpose |
  | -------- | -------- | ------- | ------- |
  | `FLGR_HTTP_PORT` | No | `8080` | Port the Go server listens on |
  | `FLGR_DB_PATH` | Yes | — | Path to the SQLite database file |
  | `FLGR_SESSION_COOKIE_SECURE` | No | `true` | Whether the session cookie ([ADR-0006](0006-authentication-and-session-strategy.md)) requires HTTPS; only set `false` for local HTTP development |
  | `FLGR_KAFKA_BROKERS` | No | empty | Comma-separated broker addresses; empty disables Kafka delivery (HTTP channels still work) |
  | `FLGR_LOG_LEVEL` | No | `info` | Logging verbosity |
  | `FLGR_ENCRYPTION_KEY` | Yes | — | Base64-encoded 32-byte AES-256 key used to encrypt retrievable secrets at rest |

  This list is expected to grow as implementation proceeds; new variables follow the same `FLGR_` prefix and are validated in the same place.

- **Secrets at rest:** values the application must be able to retrieve in plaintext (currently, the HTTP webhook `signing_secret`) are encrypted with AES-256-GCM using `FLGR_ENCRYPTION_KEY`, which is never itself stored in the database. Values that only ever need to be verified, not retrieved — user passwords (bcrypt, [0002](../../business/requirements/0002-user-management.md)) and service key secrets (SHA-256, [ADR-0006](0006-authentication-and-session-strategy.md)) — continue to use one-way hashing and are unaffected by this decision.

## Alternatives Considered

- **Config file + environment variable overrides** — rejected; adds an artifact to manage and mount inside the single Docker image ([ADR-0002](0002-single-docker-image-with-nginx.md)) for no real benefit at flgr's current scale. Environment variables are natively supported by Docker and any orchestrator without extra plumbing.
- **A config-loading library** (`caarlos0/env`, `spf13/viper`) — rejected; consistent with preferring the standard library when it already covers the need ([`.ai/backend.md`](../../../.ai/backend.md)). The configuration surface is small enough that manual `os.Getenv` parsing isn't a real burden, and viper in particular would bring unused file/remote-config capability.
- **Plaintext secrets at rest** — rejected; storing a webhook signing secret unencrypted in the SQLite file means anyone with read access to that file (backup, disk access, etc.) obtains it directly.

## Consequences

- The `internal/config` package owns parsing and validating every `FLGR_*` variable into one `Config` struct passed into `main.go` — a single, testable place, not scattered `os.Getenv` calls.
- Operators must generate and securely provide `FLGR_ENCRYPTION_KEY` before first run (e.g., `openssl rand -base64 32`). Losing this key makes previously encrypted secrets unrecoverable — they would need to be regenerated.
- Local development needs its own documented set of `FLGR_*` values; this is addressed by the (still pending) Local Development Environment ADR.

## References

N/A

## Author & Date

| Person | Role | Date | Description |
| -------- | ---- | ---- | ----------- |
| Nicolas Filipe Cunha | Founder | 2026-08-16 | Documented the application configuration and secrets decision. |
