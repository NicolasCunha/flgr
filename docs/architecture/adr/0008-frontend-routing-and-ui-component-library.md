# ADR-0008: Frontend Routing & UI Component Library

## Summary

| Category | Status | Date | Author |
| -------- | ------ | ---- | ------ |
| Frontend | Accepted | 2026-08-16 | Nicolas Filipe Cunha |

**Supersedes:** N/A

**Superseded by:** N/A

See [Status Values](README.md#status-values) and [Categories](README.md#categories) in the ADR index for the allowed values in each field.

## Context

`flgr-web-client` needs routing (including redirecting unauthenticated users to `/login`, per [ADR-0006](0006-authentication-and-session-strategy.md)), a base set of UI components (modals, dropdowns, tables, forms) on top of the Tailwind CSS decision in [ADR-0003](0003-frontend-tooling-and-state-management.md), a forms approach for the roughly eight CRUD screens implied by the business requirements ([0001](../../business/requirements/0001-environment-management.md)–[0004](../../business/requirements/0004-service-key-management.md), [0005](../../business/requirements/0005-feature-flag-management.md)), and a data table approach for the equally numerous listing screens, all of which share the same pagination/filter/sort contract defined in [ADR-0007](0007-api-design-conventions.md).

### Related Documents

| Document | Type | Description | Link |
| -------- | ---- | ----------- | ---- |
| Frontend Tooling & State Management | Architecture | Establishes TypeScript, Vite, and Tailwind CSS, which this ADR builds on. | [0003-frontend-tooling-and-state-management.md](0003-frontend-tooling-and-state-management.md) |
| API Design Conventions | Architecture | Defines the pagination/filter/sort envelope the data table component consumes. | [0007-api-design-conventions.md](0007-api-design-conventions.md) |
| Authentication & Session Strategy | Architecture | Defines the session cookie that protected routes must check for. | [0006-authentication-and-session-strategy.md](0006-authentication-and-session-strategy.md) |
| Frontend Guidelines | AI Agent Docs | Day-to-day conventions implementing this decision. | [frontend.md](../../../.ai/frontend.md) |

## Decision

- **Routing:** [React Router](https://reactrouter.com/). Protected routes are wrapped in a guard component that checks the current session (via a lightweight `GET /api/v1/me` call, since the session cookie is `HttpOnly` and unreadable by JS) and redirects to `/login` if unauthenticated.
- **UI components:** [shadcn/ui](https://ui.shadcn.com/) (Radix primitives styled with Tailwind). Components are generated directly into `flgr-web-client/src/components/ui/` and become part of the codebase — owned and editable, not an opaque npm dependency — reviewed like any other source file.
- **Forms:** [React Hook Form](https://react-hook-form.com/) for form state, with [Zod](https://zod.dev/) schemas for validation. Zod schemas mirror the backend's validation rules (the error envelope from [ADR-0007](0007-api-design-conventions.md)), so client-side and server-side validation stay consistent.
- **Data tables:** [TanStack Table](https://tanstack.com/table) (headless), used to build one reusable `DataTable` component that consumes the pagination/filter/sort contract from [ADR-0007](0007-api-design-conventions.md). Every listing screen (Environments, Users, Service Keys, Profiles, Feature Flags, Audit Log) uses this same component rather than a bespoke table each.
- **Icons:** [lucide-react](https://lucide.dev/), the standard icon set paired with shadcn/ui.

## Alternatives Considered

- **TanStack Router** — rejected for now in favor of React Router's larger ecosystem and community, despite TanStack Router's stronger built-in type safety.
- **Radix UI primitives with no pre-built styling** — rejected; would mean building every component's visual layer from scratch, too slow given the breadth of CRUD screens needed.
- **A full component library (Mantine, Chakra, MUI, ...)** — rejected; each brings its own styling system, which would conflict with or duplicate the Tailwind approach already chosen in [ADR-0003](0003-frontend-tooling-and-state-management.md).
- **Native `useState`-based forms, no library** — rejected; would repeat validation and state-handling boilerplate across roughly eight separate CRUD forms.
- **A bespoke HTML table per listing screen** — rejected; would duplicate pagination/filter/sort logic across every listing screen instead of sharing it once.

## Consequences

- shadcn/ui's generated components live in-repo and are subject to the same code review, lint, and test coverage expectations ([ADR-0004](0004-testing-and-coverage-standards.md)) as any other frontend code.
- A single `DataTable` component, built once against [ADR-0007](0007-api-design-conventions.md)'s envelope, is reused across every listing screen — a change to the pagination/filter/sort UX is made in one place.
- Protected routes depend on a `GET /api/v1/me` (or equivalent) endpoint existing on the backend to check session validity on load; this endpoint needs to be implemented alongside [ADR-0006](0006-authentication-and-session-strategy.md).

## References

N/A

## Author & Date

| Person | Role | Date | Description |
| -------- | ---- | ---- | ----------- |
| Nicolas Filipe Cunha | Founder | 2026-08-16 | Documented the frontend routing and UI component library decision. |
