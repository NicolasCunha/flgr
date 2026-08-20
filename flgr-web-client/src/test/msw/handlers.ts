import { http, HttpResponse } from 'msw'

// Defaults: backend healthy, setup already done (so tests that don't care
// about the wizard land on the normal app, not a redirect to /setup), and
// no active session (so tests that don't care about auth land on
// /login rather than an authenticated screen). Tests exercising the
// wizard or a logged-in state override these with server.use(...).
//
// The Feature Flags GET defaults below (empty list / 404 / no values) exist
// for the same reason: several test files (api.test.ts,
// FeatureFlagsPage.test.tsx, FeatureFlagForm.test.tsx,
// FeatureFlagValuesPage.test.tsx) all mount against these endpoints, and
// src/test/setup.ts's onUnhandledRequest: 'error' means any of them that
// forgets to override a GET it doesn't care about fails loudly rather than
// silently using stale data. Mutations (POST/PATCH/DELETE) are deliberately
// NOT defaulted here — every test that exercises one needs to assert on its
// exact request body and craft its own response, so a generic default would
// only hide bugs.
export const handlers = [
  http.get('/api/v1/health', () => HttpResponse.json({ status: 'ok' })),
  http.get('/api/v1/setup', () => HttpResponse.json({ needed: false })),
  http.get('/api/v1/me', () =>
    HttpResponse.json({ error: { code: 'unauthenticated', message: 'not logged in' } }, { status: 401 }),
  ),
  http.get('/api/v1/feature-flags', () =>
    HttpResponse.json({ data: [], pagination: { page: 1, page_size: 20, total: 0 } }),
  ),
  http.get('/api/v1/feature-flags/:id', () =>
    HttpResponse.json(
      { error: { code: 'not_found', message: 'feature flag not found' } },
      { status: 404 },
    ),
  ),
  // GET .../values only ever returns explicitly-configured rows — per
  // 0005's Content section, an environment absent from this array is
  // implicitly disabled, not a zero-value row. Default: no flag has any
  // configured values.
  http.get('/api/v1/feature-flags/:id/values', () => HttpResponse.json({ data: [] })),
]
