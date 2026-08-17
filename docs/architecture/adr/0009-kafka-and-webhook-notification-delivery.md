# ADR-0009: Kafka & HTTP Webhook Notification Delivery

## Summary

| Category | Status | Date | Author |
| -------- | ------ | ---- | ------ |
| Backend / Infra | Accepted | 2026-08-16 | Nicolas Filipe Cunha |

**Supersedes:** N/A

**Superseded by:** N/A

See [Status Values](README.md#status-values) and [Categories](README.md#categories) in the ADR index for the allowed values in each field.

## Context

[0006-feature-flag-killswitch.md](../../business/requirements/0006-feature-flag-killswitch.md) requires that triggering the Killswitch publish an event to every enabled notification channel (Kafka and/or HTTP) configured on the flag. This ADR decides how that delivery is actually implemented: client library, message format, and whether delivery blocks the killswitch's response.

### Related Documents

| Document | Type | Description | Link |
| -------- | ---- | ----------- | ---- |
| Feature Flag Killswitch | Business | Defines the trigger and the `feature_flag_notification_channels` table this ADR delivers events for. | [0006-feature-flag-killswitch.md](../../business/requirements/0006-feature-flag-killswitch.md) |
| Single Docker Image with Nginx Routing | Architecture | The chosen Kafka client must not require cgo/librdkafka, to keep the image build simple. | [0002-single-docker-image-with-nginx.md](0002-single-docker-image-with-nginx.md) |
| API Design Conventions | Architecture | JSON as the payload format, consistent with the rest of the API. | [0007-api-design-conventions.md](0007-api-design-conventions.md) |

## Decision

- **Kafka client:** [`segmentio/kafka-go`](https://github.com/segmentio/kafka-go) — pure Go, no cgo/librdkafka dependency, keeping the single Docker image ([ADR-0002](0002-single-docker-image-with-nginx.md)) simple to build.
- **Message format:** JSON, for both Kafka and HTTP, consistent with the rest of the API ([ADR-0007](0007-api-design-conventions.md)). No schema registry.
- **Delivery mechanism: outbox pattern, asynchronous.** Triggering the Killswitch writes the value change and one `notification_outbox` row per enabled channel in the same database transaction, then returns immediately. A background worker polls for pending rows and delivers them, independent of the request/response cycle — the Killswitch's own latency and availability never depend on Kafka or an external HTTP endpoint being reachable.
- **Retry policy:** exponential backoff — attempts at 30s, 1m, 2m, 4m, 8m after the previous failure, up to 5 attempts. After the 5th failed attempt, the row is marked `Failed` and surfaced wherever notification delivery status is shown; it is not retried further automatically.
- **HTTP webhook signing:** each HTTP channel has its own `signing_secret` (generated when the channel is created, and viewable afterward — unlike a Service Key secret, the receiving system needs to know it to verify signatures, so it isn't a one-time reveal). Every HTTP delivery includes an `X-Flgr-Signature` header containing an HMAC-SHA256 of the raw request body, computed with that channel's secret.
- **Event payload** (shared shape for Kafka and HTTP):

  ```json
  {
    "event": "feature_flag.killed",
    "feature_flag_key": "new-checkout-flow",
    "environment": "app-prod",
    "enabled": false,
    "occurred_at": "2026-08-16T14:32:00Z"
  }
  ```

- **Kafka partition key:** `<feature_flag_key>:<environment>`, so all events for the same flag/environment pair stay ordered relative to each other.

## Alternatives Considered

- **`IBM/sarama`** — rejected in favor of `segmentio/kafka-go`'s simpler API; both are pure Go, but sarama's configuration surface is larger than needed here.
- **Avro + Schema Registry** — rejected; would require standing up and operating a schema registry, infrastructure not otherwise needed for flgr at this stage. JSON is sufficient and matches the rest of the API.
- **Synchronous delivery** (blocking the Killswitch response until Kafka/webhook delivery is confirmed) — rejected; would couple the Killswitch's own reliability and latency to external systems that may be slow or temporarily unreachable, undermining its "emergency action" purpose.
- **No HTTP webhook signing for this phase** — rejected; unsigned webhooks let anyone who discovers the URL forge a killswitch notification, which is a meaningful risk for a security/incident-response feature.

## Consequences

- A `notification_outbox` table is added: `id`, `feature_flag_notification_channel_id` (FK), `payload` (JSON), `status` (`Pending`/`Delivered`/`Failed`), `attempt_count`, `next_attempt_at`, `last_error`, `created_on`, `delivered_on`. It does not use the standard [ADR-0005](0005-audit-columns-and-soft-delete-convention.md) audit columns — it's a system-generated delivery record, not a user-editable entity.
- A background worker (goroutine(s) inside the same Go process — no separate worker service, consistent with flgr shipping as a single instance) is needed to poll and deliver pending outbox rows.
- Consumers integrating an HTTP channel must implement HMAC verification using the channel's `signing_secret` to trust incoming requests.
- A flag can appear "killed" in flgr before its consumers have actually received the notification (asynchronous delivery); the audit trail ([0008](../../business/requirements/0008-feature-flag-audit-trail.md)) reflects the state change immediately, while outbox delivery status is tracked separately.

## References

N/A

## Author & Date

| Person | Role | Date | Description |
| -------- | ---- | ---- | ----------- |
| Nicolas Filipe Cunha | Founder | 2026-08-16 | Documented the Kafka and HTTP webhook notification delivery decision. |
