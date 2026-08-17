# Frontend Guidelines (React)

Stack recap: React with Redux for state management (see [ADR-0001](../docs/architecture/adr/0001-technology-stack.md)), implemented with TypeScript, Vite, npm, Redux Toolkit + RTK Query, and Tailwind CSS (see [ADR-0003](../docs/architecture/adr/0003-frontend-tooling-and-state-management.md)).

## Project Structure

The frontend lives under `flgr-web-client/` at the repository root, structured by feature rather than by file type — each domain area of flgr (flags, environments, users, service keys, ...) owns its own folder with everything it needs.

```
flgr-web-client/
  src/
    app/
      store.ts             # Redux store setup
      router.tsx            # route definitions
      App.tsx
    features/
      flags/
        api.ts               # RTK Query endpoints for this feature
        slice.ts              # Redux Toolkit slice (only if local UI state is needed beyond RTK Query cache)
        components/
        FlagsPage.tsx
        types.ts
      environments/
        ...same shape as above...
    components/               # shared, feature-agnostic UI components (Button, Modal, Table, ...)
    hooks/                     # shared hooks
    lib/                       # shared utilities (formatting, constants, etc.)
    types/                     # shared/global TypeScript types
  index.html
  package.json
  tsconfig.json
  vite.config.ts
  tailwind.config.ts
```

Tests are co-located with the code they test (`FlagsPage.tsx` + `FlagsPage.test.tsx`), not in a separate test tree — see [Testing](#testing).

## Dependency Management

npm (`package.json` / `package-lock.json`), both committed to version control.

- Add a dependency: `npm install <package>@<version>` (runtime) or `npm install -D <package>@<version>` (dev-only, e.g. test tooling). Pin to an explicit version.
- Remove a dependency: `npm uninstall <package>` — don't just delete the line from `package.json` and leave the lockfile stale.
- Before adding a new dependency, check it's actively maintained and check its bundle size impact for anything shipped to the client. A new dependency that shapes how a whole category of code is written (a new state/data-fetching library, a new UI kit) is an architectural decision and needs an ADR (see [documentation.md](documentation.md)).

## Code Style

- TypeScript in strict mode (`strict: true` in `tsconfig.json`) — no implicit `any`.
- Format with [Prettier](https://prettier.io/), lint with [ESLint](https://eslint.org/) (`eslint-plugin-react`, `eslint-plugin-react-hooks`, `@typescript-eslint`) — run both before committing.
- Functional components with hooks only — no class components.
- One component per file; file name matches the component name (`FlagsTable.tsx` exports `FlagsTable`).
- Component props are typed with a TypeScript `interface` (not `type`, for consistency), named `<Component>Props`.
- Styling with Tailwind utility classes directly in JSX; avoid inline `style={}` except for genuinely dynamic values that can't be expressed as a class.

## State Management & Data Fetching

- Global state: [Redux Toolkit](https://redux-toolkit.js.org/) — one slice per feature, only for state that doesn't belong to server-cache (e.g., UI state like "which row is expanded").
- Server data (anything from the backend API): [RTK Query](https://redux-toolkit.js.org/rtk-query/overview) — one API slice per feature (`features/<feature>/api.ts`) defining typed endpoints. Components consume the generated hooks (e.g., `useGetFlagsQuery`, `useCreateFlagMutation`) directly; don't hand-write thunks or `fetch` calls for backend data.

## Testing

- Test runner: [Vitest](https://vitest.dev/) (native pairing with Vite).
- Component tests: [React Testing Library](https://testing-library.com/react) with `@testing-library/user-event` for interactions — test components through what the user sees/does, not through internal implementation details.
- API mocking: [MSW](https://mswjs.io/) (Mock Service Worker) to intercept RTK Query's network calls in tests, so components are tested against realistic HTTP responses (including error responses) without hitting a real backend.
- Test files are co-located with the code they test (`Component.tsx` + `Component.test.tsx`).

## Test Coverage

Target: **100% coverage, including conditional branches** — every conditional render, error state, and edge case needs its own test, not just the happy path.

Run tests with coverage:

```
npm run test -- --coverage
```

Vitest's coverage (via `@vitest/coverage-v8`) reports line, branch, function, and statement coverage; all four must reach 100%. If a specific branch is genuinely untestable, it must be called out with a comment explaining why, rather than silently left uncovered — this is an exception to justify, not a default.

## Running Tests

```
npm run test              # run once
npm run test -- --watch   # watch mode
npm run test -- --coverage  # with coverage report
```

## Running the App Locally

The primary way to run the full stack locally is `docker compose up` from the repository root, per [ADR-0012](../docs/architecture/adr/0012-local-development-environment.md) — it runs Vite's dev server (with HMR via a bind-mounted volume) alongside `flgr-server` and Kafka, with `/api` proxied to the backend container so the session cookie stays same-origin (see [ADR-0006](../docs/architecture/adr/0006-authentication-and-session-strategy.md)).

For standalone frontend work, the dev server can still be run natively:

```
npm run dev      # Vite dev server
npm run build    # production build (this is what ships inside the Docker image, see ADR-0002)
```

## When in Doubt

If a task calls for a new dependency, a new pattern, or deviates from something above, don't guess — see [documentation.md](documentation.md) for when that needs an ADR, and remember: propose it and get explicit confirmation before creating or changing any document.
