import { api } from '@/lib/api'
import { keysToCamelCase, keysToSnakeCase } from '@/lib/casing'
import type { User } from '@/types/user'
import type { LoginRequest } from './types'

// Session auth per docs/architecture/adr/0006-authentication-and-session-strategy.md:
// POST /login sets an HttpOnly session cookie, POST /logout clears it, and
// GET /me is the lightweight check protected routes use to confirm the
// session is still valid (the cookie itself isn't readable by JS).
export const authApi = api.injectEndpoints({
  endpoints: (builder) => ({
    login: builder.mutation<User, LoginRequest>({
      // email/password are already single-word keys, but keysToSnakeCase
      // is still correct (a no-op) and keeps every request body going
      // through the same conversion path.
      query: (body) => ({
        url: 'login',
        method: 'POST',
        body: keysToSnakeCase(body as never),
      }),
      transformResponse: (response) => keysToCamelCase<User>(response as never),
      invalidatesTags: ['Me'],
    }),
    logout: builder.mutation<void, void>({
      query: () => ({ url: 'logout', method: 'POST' }),
      invalidatesTags: ['Me'],
    }),
    getMe: builder.query<User, void>({
      query: () => 'me',
      transformResponse: (response) => keysToCamelCase<User>(response as never),
      providesTags: ['Me'],
    }),
  }),
})

export const { useLoginMutation, useLogoutMutation, useGetMeQuery } = authApi
