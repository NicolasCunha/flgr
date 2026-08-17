# 0006 - Feature Flag Killswitch

## Summary

| Type | Status | Date | Author |
| ---- | ------ | ---- | ------ |
| Functional | Accepted | 2026-08-16 | Nicolas Filipe Cunha |

## Overview

The Killswitch is an emergency action for a single (flag, environment) pair: it turns the flag off in that environment and immediately notifies consumer applications through whatever channels are configured on the flag. It is not a separate kind of resource and not a locked or irreversible state — it changes the same `enabled` value described in [0005-feature-flag-management.md](0005-feature-flag-management.md), and can be undone the same way any other value change is made (which, unlike the Killswitch, does not itself notify anyone).

### Related Documents

| Document | Type | Description | Link |
| -------- | ---- | ----------- | ---- |
| Audit Columns & Soft-Delete Convention | Architecture | Defines the audit columns referenced in the Data Model below. | [0005-audit-columns-and-soft-delete-convention.md](../../architecture/adr/0005-audit-columns-and-soft-delete-convention.md) |
| Feature Flag Management | Business | Defines the flag and per-environment value this action operates on. | [0005-feature-flag-management.md](0005-feature-flag-management.md) |
| Feature Flag Audit Trail | Business | Every killswitch trigger is recorded here, like any other flag operation. | [0008-feature-flag-audit-trail.md](0008-feature-flag-audit-trail.md) |

## Content

Each flag can have zero or more notification channels configured, independently of any particular environment. A channel is either:

- **Kafka** — a topic name that a change event is published to, or
- **HTTP** — a URL that receives a request when a change event fires.

A flag may have both a Kafka channel and one or more HTTP channels enabled at the same time; they all fire together when the flag is killed.

Triggering the Killswitch for a specific (flag, environment) pair:

1. Sets `enabled = false` on that flag's value for that environment only — other environments are unaffected.
2. Publishes a change event to every enabled notification channel configured on the flag, containing at least: the flag's key, the environment, the new `enabled` state, and the timestamp of the change.

Re-enabling a flag after a killswitch is done through the normal edit action in [0005-feature-flag-management.md](0005-feature-flag-management.md), and — consistent with that requirement — does not itself trigger a notification.

Using the Killswitch requires the same `Feature Flag Value: Write` permission as any other value change; configuring a flag's notification channels requires `Feature Flag: Edit`.

## Data Model

In addition to the columns below, every table also includes the audit columns defined in [ADR-0005](../../architecture/adr/0005-audit-columns-and-soft-delete-convention.md).

### Table: `feature_flag_notification_channels`

| Column | Type | Nullable | Description |
| ------ | ---- | -------- | ----------- |
| id | UUID | No | Unique identifier |
| feature_flag_id | UUID | No | The flag this channel is configured on |
| channel_type | VARCHAR(20) | No | `Kafka` or `HTTP` |
| destination | VARCHAR(500) | No | Kafka topic name, or HTTP URL, depending on `channel_type` |
| enabled | BOOLEAN | No | Whether this channel currently fires on a change, default `true` |

**Primary Key:** `id`

**Foreign Keys:**

| Column | References | On Delete |
| ------ | ---------- | --------- |
| feature_flag_id | feature_flags(id) | CASCADE |

**Indexes:**

| Name | Columns | Type | Description |
| ---- | ------- | ---- | ----------- |
| idx_feature_flag_notification_channels_flag | feature_flag_id | INDEX | Speeds up looking up a flag's channels when a change fires |

## Acceptance Criteria

- [ ] A user with the `Feature Flag: Edit` permission can add, update, disable, or remove a Kafka or HTTP notification channel on a flag.
- [ ] A flag can have a Kafka channel and one or more HTTP channels enabled simultaneously.
- [ ] A user with the `Feature Flag Value: Write` permission can trigger the Killswitch for a specific (flag, environment) pair.
- [ ] Triggering the Killswitch sets `enabled = false` only for the targeted environment; other environments' values for the same flag are unchanged.
- [ ] Triggering the Killswitch publishes an event to every enabled channel configured on the flag, including at minimum the flag key, environment, new state, and timestamp.
- [ ] A disabled (`enabled = false`) notification channel does not fire.
- [ ] Re-enabling a flag after a killswitch (via [0005-feature-flag-management.md](0005-feature-flag-management.md)) does not trigger a notification.
- [ ] Every Killswitch trigger is recorded in the Feature Flag Audit Trail ([0008-feature-flag-audit-trail.md](0008-feature-flag-audit-trail.md)).

## References

N/A

## Author & Date

| Person | Role | Date | Description |
| -------- | ---- | ---- | ----------- |
| Nicolas Filipe Cunha | Founder | 2026-08-16 | Created the Feature Flag Killswitch requirement. |
