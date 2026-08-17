# Agent Workflow

Steps an AI agent should follow when picking up a task in the flgr repository.

## 1. Ground the task in existing documentation

- Check [docs/business/requirements](../docs/business/requirements/README.md) for a business requirement that already covers the task.
- Check [docs/architecture/adr](../docs/architecture/adr/README.md) for ADRs governing the relevant area (see each ADR's `Category`).
- If the task isn't covered by either, see [documentation.md](documentation.md) for when a new document is warranted — and remember new documents always require the user's confirmation before being created.

## 2. Plan

- Break the task into steps.
- Identify which existing ADRs and requirements the implementation must respect.
- If the task seems to conflict with an existing `Accepted` ADR or requirement, do not silently override it — raise the conflict with the user before proceeding.

## 3. Implement

- Follow the decisions recorded in the applicable ADRs (e.g., stack and packaging decisions in [docs/architecture/adr](../docs/architecture/adr/README.md)).
- Coding conventions, folder structure, and testing strategy are not yet decided. Until they're recorded as ADRs, ask the user rather than assuming a convention.

## 4. Verify

- If the task is tied to a business requirement, check the implementation against that requirement's Acceptance Criteria.
- **Before marking any task as done, run the same checks CI gates on ([ADR-0011](../docs/architecture/adr/0011-ci-cd-pipeline.md)) locally, for whichever side of the repo the task touched, and confirm they pass clean — don't rely on inspection alone, and don't leave this for the CI run to catch:**
  - **Backend** (`flgr-server/`): `golangci-lint run --path-mode=abs` and `go test ./... -coverprofile=coverage.out -covermode=atomic`. Install/update `golangci-lint` to the exact version pinned in [`.github/workflows/ci.yml`](../.github/workflows/ci.yml) (`version:` under the `golangci-lint` step) before running it — a different local version can pass or fail differently than CI, which is what caused the CI-only lint failures on 2026-08-17 (see [roadmap.md](roadmap.md)). Install with: `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@<version from ci.yml>`.
  - **Frontend** (`flgr-web-client/`): `npm run lint`, `npx tsc -b`, and `npx vitest run --coverage`.
  - If any check fails or reports less than the coverage target from [ADR-0004](../docs/architecture/adr/0004-testing-and-coverage-standards.md), fix it before considering the task complete — a task isn't done just because CI hasn't run on it yet.

## 5. Document

- If new business requirements or ADRs were confirmed and created per [documentation.md](documentation.md), make sure the corresponding `README.md` index table was updated.
