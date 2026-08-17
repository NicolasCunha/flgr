import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react'

// Base API slice. Requests/responses are snake_case on the wire (see
// docs/architecture/adr/0007-api-design-conventions.md) — a shared
// transformResponse/transformRequest for camelCase conversion belongs
// here once an endpoint actually has multi-word fields to convert.
export const api = createApi({
  reducerPath: 'api',
  // An absolute URL, rather than relying on the browser to resolve a
  // relative one — Node's fetch (used under jsdom in tests) has no page
  // origin to resolve against, unlike a real browser.
  baseQuery: fetchBaseQuery({ baseUrl: `${window.location.origin}/api/v1` }),
  endpoints: (builder) => ({
    getHealth: builder.query<{ status: string }, void>({
      query: () => 'health',
    }),
  }),
})

export const { useGetHealthQuery } = api
