# ADR-0011: CI/CD Pipeline

## Summary

| Category | Status | Date | Author |
| -------- | ------ | ---- | ------ |
| Infra | Accepted | 2026-08-16 | Nicolas Filipe Cunha |

**Supersedes:** N/A

**Superseded by:** N/A

See [Status Values](README.md#status-values) and [Categories](README.md#categories) in the ADR index for the allowed values in each field.

## Context

[ADR-0004](0004-testing-and-coverage-standards.md) set a 100% test coverage target but explicitly left CI enforcement as a follow-up. This ADR closes that gap and defines the full pipeline: branching model, what gates a merge, and how the single Docker image ([ADR-0002](0002-single-docker-image-with-nginx.md)) is built, tagged, and published.

### Related Documents

| Document | Type | Description | Link |
| -------- | ---- | ----------- | ---- |
| Testing & Coverage Standards | Architecture | The 100% coverage target this pipeline enforces mechanically. | [0004-testing-and-coverage-standards.md](0004-testing-and-coverage-standards.md) |
| Single Docker Image with Nginx Routing | Architecture | The image this pipeline builds and publishes. | [0002-single-docker-image-with-nginx.md](0002-single-docker-image-with-nginx.md) |

## Decision

### Branching model: Git Flow

- **`main`** — always reflects what's deployed to production; only updated by merging a `release/*` or `hotfix/*` branch.
- **`develop`** — integration branch for ongoing work.
- **`feature/*`** — one branch per unit of work, branched from `develop`, merged back via PR into `develop`.
- **`release/*`** — branched from `develop` when preparing a release; only receives bugfixes; merged into both `main` and `develop` when ready, tagged `vX.Y.Z` on the merge into `main`.
- **`hotfix/*`** — branched from `main` for urgent production fixes; merged into both `main` and `develop`.

`main` and `develop` are protected: no direct pushes, only merges via PR (or, for `main`, via a `release/*`/`hotfix/*` merge).

### CI platform & registry

**GitHub Actions**, publishing to the **GitHub Container Registry** (`ghcr.io`) — both are already native to where the repository lives, avoiding a separate registry account/credentials.

### What gates a merge

Every PR (into `develop`, `release/*`, or `main`) runs, and must pass before merge:

1. **Lint** — `golangci-lint` for `flgr-server`, ESLint + Prettier check for `flgr-web-client`.
2. **Tests + coverage gate** — `go test ./... -coverprofile` for the backend, `vitest --coverage` for the frontend; the build fails if either reports less than 100% line/branch coverage (per [ADR-0004](0004-testing-and-coverage-standards.md)), enforced via a coverage-checking step (a small script over `go tool cover -func` output for the backend; Vitest's own coverage `thresholds` config, set to 100, for the frontend).
3. **Build** — the Go binary compiles, the Vite production build succeeds, and the Docker image builds successfully (not pushed at this stage).

### Image build, tag, and publish

| Trigger | Action |
| ------- | ------ |
| Push to `develop` | Build and push `ghcr.io/.../flgr:sha-<short-sha>` and move the floating `ghcr.io/.../flgr:develop` tag |
| Push to `main` (via a `release/*` or `hotfix/*` merge) | Build and push `ghcr.io/.../flgr:sha-<short-sha>` |
| Git tag `vX.Y.Z` (created against `main`, typically as part of a release merge) | Build and push `ghcr.io/.../flgr:X.Y.Z` and move the floating `ghcr.io/.../flgr:latest` tag |

## Alternatives Considered

- **Trunk-based development** (`main` + feature branches + PRs, no `develop`/`release` branches) — rejected in favor of Git Flow's explicit separation between ongoing integration work and release preparation/hotfixes.
- **Docker Hub** — rejected in favor of `ghcr.io`, which reuses GitHub Actions' existing authentication instead of a separate registry account.
- **A different CI platform** (GitLab CI, Azure DevOps, CircleCI) — rejected; the repository already lives on GitHub, so GitHub Actions needs no additional integration.
- **Commit-SHA-only versioning, no formal releases** — rejected; paired with Git Flow's release branches, semantic version tags give the project real, identifiable releases from early on.

## Consequences

- Feature work always targets `develop`, never `main`, directly. `main` only moves via a `release/*` or `hotfix/*` merge.
- The `develop` floating tag provides a runnable "latest integration build," useful for the still-pending Local Development Environment ADR to build against.
- The 100% coverage target from [ADR-0004](0004-testing-and-coverage-standards.md) becomes a mechanically enforced merge gate, not just documented policy.
- Git Flow carries more process overhead than trunk-based development (release branches, keeping hotfixes synced back into `develop`) — accepted as a deliberate tradeoff for a more structured release cadence.

## References

N/A

## Author & Date

| Person | Role | Date | Description |
| -------- | ---- | ---- | ----------- |
| Nicolas Filipe Cunha | Founder | 2026-08-16 | Documented the CI/CD pipeline decision. |
