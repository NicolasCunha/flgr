# ADRs - Architecture Decision Records

## Overview

Architecture Decision Records (ADRs) are a way to document the architectural decisions made during the development of a software project. They provide a historical record of the decisions, the context in which they were made, and the consequences of those decisions.

## Template

Check [0000-adr-template.md](0000-adr-template.md) for a template to create new ADRs.

## Status Values

| Status | Meaning |
| ------ | ------- |
| Proposed | Under discussion, not yet approved. |
| Accepted | Approved and currently in effect. |
| Rejected | Considered but not approved; kept for reference so the option isn't re-evaluated without new information. |
| Deprecated | No longer recommended, but nothing has formally replaced it yet. |
| Superseded | Replaced by a newer ADR. See that ADR's `Supersedes` field, and this one's `Superseded by` field. |

## Categories

Each ADR is tagged with the area of the system it affects: `Backend`, `Frontend`, `Database`, `Infra`, or `API`. An ADR may span more than one category if needed.

## ADRs

The following ADRs have been created for the flgr application:

| ADR Number | Title | Category | Status | Date |
|------------|-------|----------|--------|------|
| 0000 | ADR Template | N/A | Accepted | 2026-08-16 |
| 0001 | [Technology Stack](0001-technology-stack.md) | Backend / Frontend / Database | Accepted | 2026-08-16 |
| 0002 | [Single Docker Image with Nginx Routing](0002-single-docker-image-with-nginx.md) | Infra | Accepted | 2026-08-16 |
| 0003 | [Frontend Tooling & State Management](0003-frontend-tooling-and-state-management.md) | Frontend | Accepted | 2026-08-16 |
| 0004 | [Testing & Coverage Standards](0004-testing-and-coverage-standards.md) | Backend / Frontend | Accepted | 2026-08-16 |
| 0005 | [Audit Columns & Soft-Delete Convention](0005-audit-columns-and-soft-delete-convention.md) | Database | Accepted | 2026-08-16 |
| 0006 | [Authentication & Session Strategy](0006-authentication-and-session-strategy.md) | Backend / Frontend | Accepted | 2026-08-16 |
| 0007 | [API Design Conventions](0007-api-design-conventions.md) | Backend / API | Accepted | 2026-08-16 |
| 0008 | [Frontend Routing & UI Component Library](0008-frontend-routing-and-ui-component-library.md) | Frontend | Accepted | 2026-08-16 |
| 0009 | [Kafka & HTTP Webhook Notification Delivery](0009-kafka-and-webhook-notification-delivery.md) | Backend / Infra | Accepted | 2026-08-16 |
| 0010 | [Application Configuration & Secrets](0010-application-configuration-and-secrets.md) | Backend | Accepted | 2026-08-16 |
| 0011 | [CI/CD Pipeline](0011-ci-cd-pipeline.md) | Infra | Accepted | 2026-08-16 |
| 0012 | [Local Development Environment](0012-local-development-environment.md) | Infra | Accepted | 2026-08-16 |