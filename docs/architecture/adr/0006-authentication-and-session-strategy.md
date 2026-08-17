# ADR-0006: Authentication & Session Strategy

## Summary

| Category | Status | Date | Author |
| -------- | ------ | ---- | ------ |
| Backend / Frontend | Accepted | 2026-08-16 | Nicolas Filipe Cunha |

**Supersedes:** N/A

**Superseded by:** N/A

See [Status Values](README.md#status-values) and [Categories](README.md#categories) in the ADR index for the allowed values in each field.

## Context

flgr authenticates two different kinds of callers: Users logging into the web UI ([0002](../../business/requirements/0002-user-management.md)), and Service Keys used by consumer applications ([0004](../../business/requirements/0004-service-key-management.md), [0007](../../business/requirements/0007-feature-flag-evaluation-api.md)). Revocation needs to take effect immediately — deactivating a User or changing their Profile/permissions ([0003](../../business/requirements/0003-profile-and-permission-management.md)) must not remain bypassable through a stale credential.

### Related Documents

| Document | Type | Description | Link |
| -------- | ---- | ----------- | ---- |
| Single Docker Image with Nginx Routing | Architecture | Frontend and backend share an origin behind Nginx, which is what makes a cookie-based session workable without CORS. | [0002-single-docker-image-with-nginx.md](0002-single-docker-image-with-nginx.md) |
| User Management | Business | Defines the `users` table and password hashing this ADR builds on. | [0002-user-management.md](../../business/requirements/0002-user-management.md) |
| Profile & Permission Management | Business | Permissions must be re-evaluated from current state on every request, not cached in a session or token. | [0003-profile-and-permission-management.md](../../business/requirements/0003-profile-and-permission-management.md) |
| Service Key Management | Business | Defines the secret this ADR specifies the hashing algorithm for. | [0004-service-key-management.md](../../business/requirements/0004-service-key-management.md) |

## Decision

### User authentication (UI)

Server-side session, not a JWT:

1. `POST /login` with email and password; the backend verifies the password against the stored bcrypt hash ([0002](../../business/requirements/0002-user-management.md)).
2. On success, a row is created in a `sessions` table (`token_hash`, `user_id`, `issued_at`, `expires_at`, `last_seen_at`). The token itself is a high-entropy random value; only its hash is stored, mirroring how passwords and service key secrets are handled.
3. The response sets an `HttpOnly`, `Secure`, `SameSite=Lax` cookie containing the opaque token.
4. Every authenticated request looks up the session by token hash, checks it hasn't expired, and checks the associated user's `status` is still `Active`. A valid request slides the expiration forward.
5. Session lifetime is a **sliding 30-minute inactivity window** — `expires_at` is extended by 30 minutes on every valid request, rather than being a fixed lifetime from login.
6. `POST /logout` deletes the session row and clears the cookie.

Deactivating a user, or a permission change, takes effect on the user's very next request, since both are read fresh from the database rather than trusted from the session or a token payload.

### Service Key authentication (API)

Sent as `Authorization: Bearer <secret>` on every request covered by [0007](../../business/requirements/0007-feature-flag-evaluation-api.md) and the write actions under [0005](../../business/requirements/0005-feature-flag-management.md)/[0006](../../business/requirements/0006-feature-flag-killswitch.md). The secret is hashed with **SHA-256**, not bcrypt — it's a high-entropy, randomly generated value rather than a human-chosen password, so a fast hash is sufficient and avoids adding bcrypt's deliberate cost to what is expected to be a high-volume evaluation path.

### Login attempt protection

After 5 consecutive failed login attempts for the same email within a 15-minute window, further login attempts for that email are rejected for 15 minutes, regardless of source IP. This applies only to User login (`/login`), not to Service Key requests. Tracked in a `login_attempts` table (`email`, `failed_count`, `first_failed_at`, `locked_until`).

## Alternatives Considered

- **JWT (stateless)** for User sessions — rejected. Immediate revocation on deactivation or a permission change would require a blocklist anyway, which erases JWT's main advantage while adding complexity, for an app running as a single instance with no cross-service token-sharing need.
- **bcrypt for Service Key secrets** (same as passwords) — rejected; the secret is already high-entropy and randomly generated, so bcrypt's deliberate slowness only adds latency to the Evaluation API's high-volume path without a corresponding security benefit.
- **No login attempt protection** — rejected for this phase; basic brute-force protection was explicitly wanted from the start rather than deferred.

## Consequences

- A `sessions` table and a `login_attempts` table are added. Neither uses the standard audit columns from [ADR-0005](0005-audit-columns-and-soft-delete-convention.md): a session's own `issued_at`/`expires_at`/`last_seen_at` already cover it (its `created_by` would trivially always be its own `user_id`), and login attempts are tracked by email, not by an authenticated actor.
- Every authenticated request costs a database read to validate the session or service key. Acceptable at flgr's expected scale (SQLite, single instance) — worth revisiting if the Evaluation API's volume ever demands a cache in front of it.
- The frontend and backend must remain same-origin behind Nginx ([ADR-0002](0002-single-docker-image-with-nginx.md)) for the cookie-based session to work without CORS configuration.
- Expired sessions and resolved login-attempt records will need periodic cleanup (e.g., a scheduled sweep); the exact mechanism is an implementation detail, not part of this decision.

## References

N/A

## Author & Date

| Person | Role | Date | Description |
| -------- | ---- | ---- | ----------- |
| Nicolas Filipe Cunha | Founder | 2026-08-16 | Documented the authentication and session strategy decision. |
