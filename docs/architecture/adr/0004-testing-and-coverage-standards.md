# ADR-0004: Testing & Coverage Standards

## Summary

| Category | Status | Date | Author |
| -------- | ------ | ---- | ------ |
| Backend / Frontend | Accepted | 2026-08-16 | Nicolas Filipe Cunha |

**Supersedes:** N/A

**Superseded by:** N/A

See [Status Values](README.md#status-values) and [Categories](README.md#categories) in the ADR index for the allowed values in each field.

## Context

flgr needs a consistent testing approach and coverage bar across both `flgr-server` (Go) and `flgr-web-client` (React), so quality expectations are explicit and uniform rather than decided ad hoc per task or contributor.

### Related Documents

| Document | Type | Description | Link |
| -------- | ---- | ----------- | ---- |
| Technology Stack | Architecture | Establishes Go/Gin and React/Redux, the stacks this testing strategy applies to. | [0001-technology-stack.md](0001-technology-stack.md) |
| Frontend Tooling & State Management | Architecture | Establishes Vite/RTK Query, which determine the frontend test tooling below. | [0003-frontend-tooling-and-state-management.md](0003-frontend-tooling-and-state-management.md) |
| Backend Guidelines | AI Agent Docs | Day-to-day testing conventions for `flgr-server`. | [backend.md](../../../.ai/backend.md) |
| Frontend Guidelines | AI Agent Docs | Day-to-day testing conventions for `flgr-web-client`. | [frontend.md](../../../.ai/frontend.md) |

## Decision

- **Minimum coverage:** 100%, measured on both line and branch coverage. Every conditional path — success and error/edge cases alike — must be exercised by a test, not just the common path.
- **Backend (Go):** the standard `testing` package plus `testify` (`assert`/`require`, `mock`). The repository layer is tested against a real SQLite instance; the service layer is tested with mocked repositories; the handler layer is tested via `net/http/httptest`.
- **Frontend (React):** Vitest as the test runner, React Testing Library for component tests, and MSW to mock the HTTP calls made through RTK Query.
- Coverage is checked with `go test ./... -coverprofile` (backend) and `npm run test -- --coverage` (frontend); both must report 100% before a task is considered complete.

## Alternatives Considered

- **Lower coverage targets (70–90%)** — rejected; 100% including conditional branches was chosen deliberately to keep the quality bar unambiguous project-wide, accepting the extra upfront cost while the codebase is still small.
- **gomock** instead of `testify/mock` for Go — rejected in favor of testify's simpler, less code-generation-dependent API, consistent with already using testify for assertions.
- **Jest** instead of Vitest for the frontend — rejected; Vitest is the native pairing for a Vite-based project ([ADR-0003](0003-frontend-tooling-and-state-management.md)) and avoids configuring a second, redundant toolchain.
- **Cypress** instead of MSW/RTL for API interaction tests — rejected for this scope; Cypress is an end-to-end tool, out of scope for unit/component-level coverage. An E2E testing strategy is left for a future ADR if/when it's needed.

## Consequences

- Code without a test that exercises every branch is considered incomplete, not merely "untested."
- A genuinely untestable line (e.g., defensive unreachable code) requires an inline comment justifying the exclusion, rather than being silently skipped — this keeps the 100% target meaningful rather than gamed.
- CI enforcement of this coverage gate is not yet set up; a CI/CD pipeline is a follow-up decision that will need its own ADR.
- Detailed testing conventions are maintained in [`.ai/backend.md`](../../../.ai/backend.md) and [`.ai/frontend.md`](../../../.ai/frontend.md), which must stay in sync with this ADR.

## References

N/A

## Author & Date

| Person | Role | Date | Description |
| -------- | ---- | ---- | ----------- |
| Nicolas Filipe Cunha | Founder | 2026-08-16 | Documented the testing and coverage standards decision. |
