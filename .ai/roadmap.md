# Roadmap & Session Handoff

## Purpose

This file exists so a fresh agent session (or a human) can pick up work on flgr without re-deriving context from scratch. It is **not** a substitute for `git log` (authoritative for what changed) or the docs under `docs/business/requirements` and `docs/architecture/adr` (authoritative for what was decided and why). It captures the one thing those sources don't: *where the project stands right now, in what order the remaining work is meant to happen, and what's mid-discussion but not yet decided.*

**Update this file at the end of a work session** if the Current State, Build Order, or Open Decisions changed. Keep entries short — this is a map, not a diary.

## How to use this file

1. Read [Current State](#current-state) to know what's actually implemented vs. only documented.
2. Read [Open Decisions](#open-decisions--in-flight-discussions) before touching an area listed there — there may be an unresolved design conversation you'd otherwise redo or contradict.
3. Use [Suggested Build Order](#suggested-build-order) as a default sequence, not a hard rule — deviate if the user asks for something else.
4. Follow [workflow.md](workflow.md) for the actual step-by-step process on a task, and [documentation.md](documentation.md) for when docs need updating and when that needs the user's confirmation first.

## Current State

_Last updated: 2026-08-17_

**Documentation** — all 12 ADRs ([index](../docs/architecture/adr/README.md)) and 8 business requirements ([index](../docs/business/requirements/README.md)) are `Accepted`. The system is fully specified on paper; implementation has barely started.

**Backend (`flgr-server`)** — scaffolding only:
- `internal/config` — env-based config loading.
- `internal/database` — SQLite connection (`modernc.org/sqlite`) + custom migration runner (`Migrate`, not `golang-migrate` — see [backend.md](backend.md#migrations)).
- `internal/api` — Gin router with a single `/health` handler; no domain routes yet.
- `migrations/000001_initial_schema` + `000002_generic_audit_log` — the **entire** schema for all 8 accepted requirements already exists (users, profiles/permissions, environments, service keys, feature flags + per-environment values, `notification_outbox`, etc.). `000002` replaced `000001`'s flag-only `feature_flag_audit_logs` with the generic `audit_log` + per-entity link tables pattern ([ADR-0013](../docs/architecture/adr/0013-generic-audit-log-pattern.md)) — only `audit_log_feature_flag` has write logic planned; the other link tables (`audit_log_user`, `audit_log_environment`, `audit_log_service_key`, `audit_log_profile`) are schema-only until each entity gets its own audit requirement. Schema is ahead of code.
- No `service`/`repository`/`model` code for any domain yet — none of the 8 business requirements has a working vertical slice.

**Frontend (`flgr-web-client`)** — scaffolding only: Vite + React shell, Redux store skeleton, a placeholder `HomePage`, no `features/*` slices yet.

**CI/CD** — GitHub Actions pipeline per [ADR-0011](../docs/architecture/adr/0011-ci-cd-pipeline.md) (lint/test/coverage/build gates + Docker build validation) is wired up and green as of 2026-08-17 (see [Session Notes](#session-notes) for the golangci-lint fix).

## Suggested Build Order

Dependency-driven phases — later phases lean on actors/entities created in earlier ones (audit `actor_*` columns need real users, flags need environments to have values in, etc.).

| Phase | Scope | Requirement(s) |
| ----- | ----- | -------------- |
| 0 | Schema + scaffolding | — (done) |
| 1 | Users & Auth (sessions, login) | [0002](../docs/business/requirements/0002-user-management.md) business req + [ADR-0006](../docs/architecture/adr/0006-authentication-and-session-strategy.md) |
| 2 | Profiles & Permissions (RBAC) | [0003](../docs/business/requirements/0003-profile-and-permission-management.md) |
| 3 | Environments | [0001](../docs/business/requirements/0001-environment-management.md) |
| 4 | Service Keys | [0004](../docs/business/requirements/0004-service-key-management.md) |
| 5 | Feature Flags (definitions + per-environment values) | [0005](../docs/business/requirements/0005-feature-flag-management.md) |
| 6 | Killswitch | [0006](../docs/business/requirements/0006-feature-flag-killswitch.md) (business req — not to be confused with ADR-0006, auth) |
| 7 | Evaluation API (external read-only) | [0007](../docs/business/requirements/0007-feature-flag-evaluation-api.md) |
| 8 | Audit Trail | [0008](../docs/business/requirements/0008-feature-flag-audit-trail.md) — **design under revision, see below** |
| 9 | Notifications (Kafka/webhook outbox) | [ADR-0009](../docs/architecture/adr/0009-kafka-and-webhook-notification-delivery.md) |
| 10 | Frontend | one `features/*` slice per phase above, roughly in step with its backend phase |

Phases 1–4 (actors, RBAC, environments, service keys) are prerequisites almost everything else touches — start there unless there's a specific reason to jump ahead.

## Open Decisions / In-Flight Discussions

_(none currently open)_

## Session Notes

Short, reverse-chronological. Only for things not obvious from `git log` — decisions discussed, questions raised, blockers hit.

- **2026-08-17 (2):** CI's newly-fixed golangci-lint run (see below) surfaced `errcheck` failures on unchecked `db.Close()`/`tx.Rollback()` in `internal/database` and `cmd/server/main.go` — fixed all of them (not just the ones CI printed; `max-same-issues` defaults to 3, so CI's error list undercounts identical-message issues). Installed `golangci-lint` locally (matching `ci.yml`'s pinned version) and confirmed a clean `0 issues` run plus `go build`/`go test` passing before calling this done. Added a mandatory "run CI's own checks locally before calling a task done" step to [workflow.md](workflow.md#4-verify), specifically to stop backend lint failures from only being caught by CI going forward.
- **2026-08-17:** Created this roadmap file. Fixed CI: `golangci-lint-action` was resolving to a v1 golangci-lint binary (1.64.8, built with Go 1.24) too old to parse a `go.mod` targeting `go 1.26.4` — bumped to `golangci-lint-action@v8` with `version: v2.12` pinned explicitly. Generalized the feature flag audit log (requirement 0008) into a system-wide pattern: new [ADR-0013](../docs/architecture/adr/0013-generic-audit-log-pattern.md), 0008's Data Model updated, migration `000002_generic_audit_log` adds `audit_log` + one link table per entity (only `audit_log_feature_flag` has write logic planned so far — see [Current State](#current-state)).
