import type { FetchBaseQueryError } from '@reduxjs/toolkit/query/react'
import type { SerializedError } from '@reduxjs/toolkit'

// Matches the error envelope from
// docs/architecture/adr/0007-api-design-conventions.md:
// { "error": { "code": "...", "message": "...", "details": [...] } }
interface ApiErrorBody {
  error?: {
    code?: string
    message?: string
    details?: { field?: string; reason?: string }[]
  }
}

/**
 * Extracts a human-readable message from an RTK Query error, falling back
 * to a generic message. Accepts `unknown` (not just
 * `FetchBaseQueryError | SerializedError`) because that's what a
 * `try { await mutation(...).unwrap() } catch (error) { ... }` block
 * actually hands you under TypeScript's `useUnknownInCatchVariables`.
 */
export function getErrorMessage(error: unknown, fallback = 'Something went wrong. Please try again.'): string {
  if (!error || typeof error !== 'object') return fallback

  if ('status' in error) {
    const fetchError = error as FetchBaseQueryError
    const data = fetchError.data as ApiErrorBody | undefined
    if (data?.error?.message) return data.error.message
    if (typeof fetchError.status === 'number' && fetchError.status >= 500) {
      return 'The server encountered an error. Please try again.'
    }
    return fallback
  }

  return (error as SerializedError).message ?? fallback
}
