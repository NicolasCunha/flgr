import type { SerializedError } from '@reduxjs/toolkit'
import { createApi, fetchBaseQuery, type FetchBaseQueryError } from '@reduxjs/toolkit/query/react'

// Base API slice. Requests/responses are snake_case on the wire (see
// docs/architecture/adr/0007-api-design-conventions.md) — feature api.ts
// files convert to/from camelCase at this boundary (see
// src/lib/casing.ts) for any endpoint with multi-word fields.
export const api = createApi({
  reducerPath: 'api',
  baseQuery: fetchBaseQuery({
    // An absolute URL, rather than relying on the browser to resolve a
    // relative one — Node's fetch (used under jsdom in tests) has no page
    // origin to resolve against, unlike a real browser.
    baseUrl: `${window.location.origin}/api/v1`,
    // Cookie-based sessions (see ADR-0006) need the HttpOnly session
    // cookie sent on every request; same-origin already implies this in
    // a real browser, but being explicit keeps behavior consistent under
    // jsdom in tests.
    credentials: 'same-origin',
  }),
  // Shared across every feature's injectEndpoints — declared once here so
  // a mutation in one feature (e.g. auth's login/logout) can invalidate a
  // query defined in another (e.g. AppLayout re-fetching /me after logout).
  tagTypes: [
    'Me',
    'Setup',
    'Environments',
    'EnvironmentCategories',
    'ServiceKeys',
    'FeatureFlags',
    'FeatureFlagValues',
  ],
  endpoints: (builder) => ({
    getHealth: builder.query<{ status: string }, void>({
      query: () => 'health',
    }),
  }),
})

export const { useGetHealthQuery } = api

/** The shape of the ADR-0007 error envelope's body. */
interface ApiErrorBody {
  error?: { code?: string; message?: string }
}

export function isFetchBaseQueryError(error: unknown): error is FetchBaseQueryError {
  return typeof error === 'object' && error != null && 'status' in error
}

// NOTE: getApiErrorMessage below overlaps with src/lib/errors.ts's
// getErrorMessage (added alongside the auth/environments features) — both
// extract a message from the ADR-0007 error envelope. Kept separate
// rather than merged: this one is depended on (with its exact fallback
// strings) by the already-shipped Setup wizard and its tests, and
// getErrorMessage has different fallback text/behavior for new call
// sites. New code should prefer '@/lib/errors' getErrorMessage;
// this one exists for the setup feature's existing call sites.
/** Extracts a user-facing message from an RTK Query mutation/query error. */
export function getApiErrorMessage(error: FetchBaseQueryError | SerializedError | undefined): string | undefined {
  if (!error) return undefined

  if (isFetchBaseQueryError(error)) {
    const body = error.data as ApiErrorBody | undefined
    if (body?.error?.message) return body.error.message
    return typeof error.status === 'number' ? `Request failed (status ${error.status})` : 'Request failed'
  }

  return error.message
}
