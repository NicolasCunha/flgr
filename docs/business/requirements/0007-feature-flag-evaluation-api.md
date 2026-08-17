# 0007 - Feature Flag Evaluation API

## Summary

| Type | Status | Date | Author |
| ---- | ------ | ---- | ------ |
| Functional | Accepted | 2026-08-16 | Nicolas Filipe Cunha |

## Overview

This requirement defines how a consumer application reads feature flag values from flgr, authenticating with a Service Key ([0004-service-key-management.md](0004-service-key-management.md)) rather than a User. It covers the read (pull) path only — push notifications for emergency changes are covered by [0006-feature-flag-killswitch.md](0006-feature-flag-killswitch.md). A key with `can_write` enabled may also use its credentials against the write actions described in [0005-feature-flag-management.md](0005-feature-flag-management.md) and [0006-feature-flag-killswitch.md](0006-feature-flag-killswitch.md); this requirement does not duplicate those rules.

### Related Documents

| Document | Type | Description | Link |
| -------- | ---- | ----------- | ---- |
| Service Key Management | Business | Defines the credential and environment scope used to authenticate these requests. | [0004-service-key-management.md](0004-service-key-management.md) |
| Feature Flag Management | Business | Defines the flags and per-environment values being read. | [0005-feature-flag-management.md](0005-feature-flag-management.md) |
| Feature Flag Audit Trail | Business | Records write operations made through this API; read/evaluation calls are not part of that audit trail (see Content below). | [0008-feature-flag-audit-trail.md](0008-feature-flag-audit-trail.md) |

## Content

A consumer application authenticates by presenting a service key's secret. For a request scoped to a given environment, flgr:

1. Rejects the request if the key is `Inactive`, has `can_read = false`, or is not linked to the requested environment (see [0004-service-key-management.md](0004-service-key-management.md)).
2. Otherwise, returns the requested flag(s) as configured for that environment: whether each is `enabled`, and — for non-`Boolean` types — its `value`. A flag with no configured value for the environment is returned as disabled, consistent with [0005-feature-flag-management.md](0005-feature-flag-management.md).

Evaluation (read) calls are a high-volume, latency-sensitive path and are not written to the Feature Flag Audit Trail, which is reserved for changes (see [0008-feature-flag-audit-trail.md](0008-feature-flag-audit-trail.md)).

## Data Model

N/A — this requirement defines read behavior over the schema already introduced in [0004-service-key-management.md](0004-service-key-management.md) and [0005-feature-flag-management.md](0005-feature-flag-management.md). No new tables are introduced.

## Acceptance Criteria

- [ ] A request authenticated with an `Inactive` service key is rejected.
- [ ] A request authenticated with a key that has `can_read = false` is rejected.
- [ ] A request for an environment the key is not linked to is rejected.
- [ ] A successful request returns, for each requested flag, at least its key, `enabled` state, and (for non-`Boolean` types) its `value` for the requested environment.
- [ ] A flag with no configured value for the requested environment is returned as disabled.
- [ ] Evaluation calls are not written to the Feature Flag Audit Trail.

## References

N/A

## Author & Date

| Person | Role | Date | Description |
| -------- | ---- | ---- | ----------- |
| Nicolas Filipe Cunha | Founder | 2026-08-16 | Created the Feature Flag Evaluation API requirement. |
