# ADR-0003: Frontend Tooling & State Management

## Summary

| Category | Status | Date | Author |
| -------- | ------ | ---- | ------ |
| Frontend | Accepted | 2026-08-16 | Nicolas Filipe Cunha |

**Supersedes:** N/A

**Superseded by:** N/A

See [Status Values](README.md#status-values) and [Categories](README.md#categories) in the ADR index for the allowed values in each field.

## Context

[ADR-0001](0001-technology-stack.md) established React with Redux for the frontend, but left the concrete implementation tooling open: language, build tool, package manager, styling approach, and how components fetch data from the backend. These need to be fixed so frontend code is written consistently across `flgr-web-client`.

### Related Documents

| Document | Type | Description | Link |
| -------- | ---- | ----------- | ---- |
| Technology Stack | Architecture | Establishes React + Redux as the frontend stack this ADR builds on. | [0001-technology-stack.md](0001-technology-stack.md) |
| Frontend Guidelines | AI Agent Docs | Day-to-day conventions (project structure, code style) implementing this decision. | [frontend.md](../../../.ai/frontend.md) |

## Decision

- **Language:** TypeScript, in strict mode.
- **Build tool:** Vite.
- **Package manager:** npm.
- **State management:** Redux Toolkit (RTK), as the concrete implementation of the Redux decision in ADR-0001.
- **Data fetching:** RTK Query for all backend API calls.
- **Styling:** Tailwind CSS.

## Alternatives Considered

- **JavaScript** instead of TypeScript — rejected; static typing reduces bugs and improves maintainability for an admin panel with forms and API integration.
- **Next.js** — rejected; its SSR/routing framework adds complexity not needed for an admin panel that ships as a static build served behind Nginx ([ADR-0002](0002-single-docker-image-with-nginx.md)).
- **Create React App (CRA)** — rejected; discontinued and unmaintained since 2023.
- **pnpm / yarn** instead of npm — rejected in favor of npm's zero-setup availability (ships with Node) and broad familiarity.
- **Manual thunks + axios/fetch** for data fetching — rejected in favor of RTK Query's typed hooks, caching, and loading/error state handling, since Redux Toolkit was already the chosen state management library.
- **CSS Modules / styled-components** for styling — rejected in favor of Tailwind's utility-first approach, which fits well with admin-dashboard UI and its component ecosystem.

## Consequences

- All frontend code is written in TypeScript strict mode; any use of `any` needs justification.
- Every backend interaction goes through an RTK Query API slice — no ad hoc `fetch`/`axios` calls inside components.
- Styling is done with Tailwind utility classes; configuration lives in `flgr-web-client/tailwind.config.ts`.
- Detailed conventions (project structure, code style) are maintained in [`.ai/frontend.md`](../../../.ai/frontend.md), which must stay in sync with this ADR.

## References

N/A

## Author & Date

| Person | Role | Date | Description |
| -------- | ---- | ---- | ----------- |
| Nicolas Filipe Cunha | Founder | 2026-08-16 | Documented the frontend tooling and state management decision. |
