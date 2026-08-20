# Agent Workflow

Steps an AI agent should follow when picking up a task in the flgr repository.

**Since 2026-08-18, phases from Phase 6 onward (and any new Phase 5 frontend work) use the multi-agent pipeline described in [Multi-Agent Orchestration](#multi-agent-orchestration) below instead of one agent doing the whole phase.** Steps 1–5 in this file still apply — they're just now followed by five specialized agents, each inside their own scope, instead of one generalist agent doing everything for a task.

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

## Multi-Agent Orchestration

Adopted 2026-08-18 to stop building the entire backend before starting any frontend work, which was correct but slow. The user-facing Claude session (whoever is talking to the human) acts as **orchestrator**: it never writes feature code itself for a phase under this model — it plans, spawns/resumes the five specialized agents below, relays contracts and blockers between them, and is the only one who talks to the user.

### The five agents

Each is a custom subagent defined in `.claude/agents/*.md` — read those files for their exact scope, tools, and conventions; this section only covers how the orchestrator uses them together.

| Agent | Owns | Writes tests? |
| --- | --- | --- |
| `database-agent` | `flgr-server/migrations/`, `internal/model/`, `internal/repository/` | No |
| `backend-agent` | `internal/service/`, `internal/api/handler/`, `internal/api/middleware/`, `router.go` | No |
| `backend-test-agent` | every `_test.go` under `flgr-server/internal/`, coverage gate, lint, real-server smoke run | — (is the test writer) |
| `frontend-agent` | `flgr-web-client/src/features/*`, `src/app/*`, `src/components/*`, `src/lib/*` | No |
| `frontend-test-agent` | every `*.test.ts(x)`, `src/test/msw/handlers.ts`, coverage gate, lint | — (is the test writer) |

Testing is deliberately **not** co-located with implementation in this model: `database-agent`/`backend-agent`/`frontend-agent` hand off compiling-but-untested code; `backend-test-agent`/`frontend-test-agent` own the entire test suite for their side. If a test agent finds a real bug, it reports it back rather than silently patching the implementer's files — the orchestrator routes the fix back to the right implementer.

### Pipeline: action-level, not phase-level

Within a phase, work is broken into individual actions/endpoints (e.g. Create, List, Get, Update, Delete for a resource) and pipelined like an assembly line, backend track running one action ahead of frontend track:

```
database-agent (schema for the phase, usually up front)
      │
      ▼
backend-agent: action 1 ──► backend-test-agent: action 1 ──► (orchestrator relays contract) ──► frontend-agent: action 1 ──► frontend-test-agent: action 1
      │
      ▼ (while frontend track works action 1 above)
backend-agent: action 2 ──► backend-test-agent: action 2 ──► ...
```

The orchestrator does not wait for a whole phase's backend to finish before frontend starts — it starts frontend on action 1 as soon as `backend-test-agent` confirms action 1 is solid, while `backend-agent` is already on action 2. Exception: if a phase's backend already fully exists (e.g. Phase 5 at the time this model was adopted), there's no backend lag to pipeline against — `frontend-agent`/`frontend-test-agent` can just proceed through the phase's actions back-to-back.

### Worktrees: two per phase, not five

Git worktree isolation happens **per track**, not per agent — `database-agent`, `backend-agent`, and `backend-test-agent` are sequentially dependent within the same phase and share one **backend-track worktree**; `frontend-agent` and `frontend-test-agent` share one **frontend-track worktree**. The orchestrator creates each worktree once (first agent call in that track, `isolation: "worktree"`), then passes that worktree's absolute path explicitly to every subsequent agent invocation in the same track so they operate on the same checkout and branch — the `Agent` tool's `isolation` param does not attach to an existing worktree, so this has to be done by instruction, not by re-requesting isolation.

`frontend-agent` does not need the backend track's code merged in to build — it works against the contract the orchestrator relays (or the documented requirement, contract-first) and `frontend-test-agent` mocks that contract with MSW. Real integration (both tracks merged, a running server, actual requests) happens once at the end of the phase, verified by `backend-test-agent`'s end-to-end smoke pass plus a final orchestrator check — not continuously.

### Communication protocol

Agents never talk to each other — everything routes through the orchestrator, who:

- Relays completed contracts (exact method/path/request/response shape) from `backend-agent` to `frontend-agent`.
- Relays bugs/mismatches found by either test agent back to the implementer that owns the affected file.
- Posts a short status ping to the user every time an agent reports something, so the pipeline stays visible instead of only showing up as one big summary at the end — e.g.:

  ```
  🧱 database-agent → schema da Fase 6 pronta
  🔧 backend-agent → POST /api/v1/killswitch pronto
  🧪 backend-test-agent → cobertura 99.1%, lint limpo ✅
  🎨 frontend-agent → botão "ativar killswitch" implementado
  💥 frontend-test-agent → falhou: mock desalinhado com o contrato
  ```

- Never commits or merges to `main` without the user's explicit go-ahead, same standing rule as before — the pipeline changes who writes the code, not the commit policy.
