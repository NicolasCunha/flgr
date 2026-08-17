# Backend Guidelines (Go)

Stack recap (see [ADR-0001](../docs/architecture/adr/0001-technology-stack.md)): Go + [Gin](https://github.com/gin-gonic/gin), SQLite as the database.

## Project Structure

The backend lives under `flgr-server/` at the repository root and follows the standard Go application layout, using `internal/` so nothing is importable outside this module — flgr's backend is an application, not a library.

```
flgr-server/
  cmd/
    server/
      main.go            # entrypoint: wires config, DB, router, and starts the HTTP server
  internal/
    api/
      handler/            # HTTP handlers (Gin), one file per resource (e.g. flag_handler.go)
      middleware/          # auth, logging, recovery, etc.
      router.go            # route registration
    service/               # business logic, orchestrates repositories
    repository/            # data access layer (SQL against SQLite)
    model/                 # domain types/entities shared across layers
    config/                # configuration loading (env vars, flags)
  migrations/               # SQL migration files (see Migrations below)
  go.mod
  go.sum
```

Layering rule: `handler` depends on `service`, `service` depends on `repository` interfaces (not concrete SQL), `repository` implements those interfaces. This is what makes the service layer testable without a real database — see [Testing](#testing).

## Dependency Management

Go modules (`go.mod` / `go.sum`), both committed to version control.

- Add a dependency: `go get <module>@<version>` (pin an explicit version, avoid `@latest` in commits).
- Remove unused dependencies and sync `go.sum`: `go mod tidy` — run this before every commit that touches imports.
- Prefer the standard library over a new dependency when the stdlib already covers the need.
- Before adding a new dependency, check it's actively maintained and widely used — a new third-party dependency is an architectural decision; if it's a significant one (e.g., a new database driver, a new HTTP framework), it needs an ADR (see [documentation.md](documentation.md)).

## Code Style

- Format with `gofmt` and `goimports` — no exceptions, no manual formatting debates.
- Lint with [`golangci-lint`](https://golangci-lint.run/) before committing.
- Package names: short, lowercase, no underscores (`service`, not `Service` or `svc_layer`).
- File names: lowercase with underscores when needed (`flag_handler.go`, `flag_handler_test.go`).
- Errors: wrap with context using `fmt.Errorf("doing X: %w", err)`; never discard an error silently. Handlers translate errors to HTTP status codes in one place (e.g., a shared error-mapping helper), not ad hoc per handler.
- Avoid `panic` outside of unrecoverable startup failures (e.g., invalid config at boot). Application-level errors are returned, not panicked.
- Prefer accepting interfaces and returning concrete structs (idiomatic Go), particularly at `service`/`repository` boundaries, so dependencies can be faked in tests.

## Migrations

SQL migrations live in `flgr-server/migrations/`, as paired `<version>_<name>.up.sql` / `<version>_<name>.down.sql` files (e.g., `000001_initial_schema.up.sql`), applied by a small runner in `internal/database` (`Migrate`) — not `golang-migrate`. That library's SQLite driver depends on `mattn/go-sqlite3` (cgo), which needs a C toolchain to build; the database itself uses the pure-Go [`modernc.org/sqlite`](https://gitlab.com/cznic/sqlite) driver instead, to keep the build cgo-free (consistent with the reasoning in [ADR-0009](../docs/architecture/adr/0009-kafka-and-webhook-notification-delivery.md) for the Kafka client), and `golang-migrate`'s driver isn't compatible with it. The runner tracks applied versions in a `schema_migrations` table and is idempotent — safe to call on every startup.

Each schema change (new table, column, index, FK — see the `Data Model` section of the relevant [business requirement](../docs/business/requirements/README.md)) ships as a new migration file, never as an edit to an already-applied one.

## Testing

- Framework: the standard library `testing` package plus [`testify`](https://github.com/stretchr/testify) (`assert`/`require`) for readable assertions, and `testify/mock` for mocking `repository` interfaces in `service` tests.
- Test files are co-located with the code they test (`flag_service.go` + `flag_service_test.go`), per Go convention — not in a separate test tree.
- Use table-driven tests for anything with more than one input/branch combination.
- `service` layer tests mock the `repository` interfaces — no real database involved, so every conditional branch (including error paths returned by the repository) can be exercised directly.
- `repository` layer tests run against a real SQLite database (a temp file or `:memory:`), to verify the actual SQL. This is the one layer that's closer to an integration test than a pure unit test.
- `handler` layer tests use `net/http/httptest` with Gin's test mode, asserting on status code and response body for both success and error paths.

## Test Coverage

Target: **100% coverage, including conditional branches**, not just line coverage. Every `if`/`else`, error path, and edge case needs a test that exercises it — not only the common path.

Run tests with coverage:

```
go test ./... -coverprofile=coverage.out -covermode=atomic
go tool cover -func=coverage.out   # per-function % in the terminal
go tool cover -html=coverage.out   # visual HTML report
```

If a specific line is genuinely untestable (e.g., a defensive `panic` that should never be reachable), it must be called out with a comment explaining why, rather than silently left uncovered — this is an exception to justify, not a default.

## Running Tests

```
go test ./...              # run all tests
go test ./... -v           # verbose output
go test ./internal/service/...  # run tests for a single package
```

## Running the App Locally

The primary way to run the full stack (including Kafka) locally is `docker compose up` from the repository root, per [ADR-0012](../docs/architecture/adr/0012-local-development-environment.md) — it runs `flgr-server` under `air` for hot-reload, alongside Kafka and Kafka UI.

For standalone backend work (no Kafka, no containers — e.g., quick debugging under an IDE), the binary can still be run natively:

```
go run ./flgr-server/cmd/server
```

A local `.env` file (see [ADR-0012](../docs/architecture/adr/0012-local-development-environment.md)) is auto-loaded in this mode via `godotenv`; production never has a `.env` file, so this has no effect there (see [ADR-0010](../docs/architecture/adr/0010-application-configuration-and-secrets.md)).

Packaging into the single production Docker image (Nginx + Go binary) is defined in [ADR-0002](../docs/architecture/adr/0002-single-docker-image-with-nginx.md) — this is a different artifact from the dev-only Compose setup above.

## When in Doubt

If a task calls for a new dependency, a new pattern, or deviates from something above, don't guess — see [documentation.md](documentation.md) for when that needs an ADR, and remember: propose it and get explicit confirmation before creating or changing any document.
