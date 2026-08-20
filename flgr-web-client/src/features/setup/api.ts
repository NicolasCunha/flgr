import { api } from '@/lib/api'
import { keysToCamelCase, keysToSnakeCase } from '@/lib/casing'
import type { User } from '@/types/user'

export interface SetupStatus {
  needed: boolean
}

export interface CompleteSetupRequest {
  firstName: string
  lastName: string
  email: string
  password: string
}

const setupApi = api.injectEndpoints({
  endpoints: (builder) => ({
    getSetupStatus: builder.query<SetupStatus, void>({
      query: () => 'setup',
      providesTags: ['Setup'],
    }),
    completeSetup: builder.mutation<User, CompleteSetupRequest>({
      query: (body) => ({
        url: 'setup',
        method: 'POST',
        body: keysToSnakeCase(body as never),
      }),
      transformResponse: (response) => keysToCamelCase<User>(response as never),
      // Without this, RootGuard's own getSetupStatus subscription (it
      // mounts the same query with the same args) keeps serving the
      // pre-setup needed:true from cache after navigate('/'), bouncing
      // straight back to /setup instead of showing the dashboard.
      invalidatesTags: ['Setup'],
    }),
  }),
})

export const { useGetSetupStatusQuery, useCompleteSetupMutation } = setupApi
