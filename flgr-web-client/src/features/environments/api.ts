import { api } from '@/lib/api'
import { keysToCamelCase, keysToSnakeCase } from '@/lib/casing'
import type {
  Environment,
  EnvironmentCategory,
  EnvironmentInput,
  ListEnvironmentsParams,
  ListResponse,
} from './types'

// Environments per docs/business/requirements/0001-environment-management.md.
// Categories are a small, dropdown-sized reference list (the seeded
// Development/Staging/Production plus any custom ones created implicitly
// via an environment's category_name — see EnvironmentInput in
// ./types.ts) rather than a paginated business resource of their own, so
// GET /environment-categories's { data: [...] } is unwrapped straight to
// an array here rather than exposing the envelope. There's no create/
// update/delete endpoint for categories — flgr-server only ever creates
// one as a side effect of creating/editing an environment with a new
// category_name.
export const environmentsApi = api.injectEndpoints({
  endpoints: (builder) => ({
    getEnvironments: builder.query<ListResponse<Environment>, ListEnvironmentsParams>({
      query: ({ page, pageSize }) => ({
        url: 'environments',
        params: { page, page_size: pageSize },
      }),
      transformResponse: (response) => keysToCamelCase<ListResponse<Environment>>(response as never),
      providesTags: (result) =>
        result
          ? [
              ...result.data.map(({ id }) => ({ type: 'Environments' as const, id })),
              { type: 'Environments' as const, id: 'LIST' },
            ]
          : [{ type: 'Environments' as const, id: 'LIST' }],
    }),
    getEnvironmentCategories: builder.query<EnvironmentCategory[], void>({
      query: () => 'environment-categories',
      transformResponse: (response) =>
        keysToCamelCase<EnvironmentCategory[]>((response as { data: unknown[] }).data as never),
      providesTags: [{ type: 'EnvironmentCategories', id: 'LIST' }],
    }),
    createEnvironment: builder.mutation<Environment, EnvironmentInput>({
      query: (body) => ({
        url: 'environments',
        method: 'POST',
        body: keysToSnakeCase(body as never),
      }),
      transformResponse: (response) => keysToCamelCase<Environment>(response as never),
      // A new environment may also implicitly create a new category.
      invalidatesTags: [
        { type: 'Environments', id: 'LIST' },
        { type: 'EnvironmentCategories', id: 'LIST' },
      ],
    }),
    updateEnvironment: builder.mutation<Environment, { id: string } & Partial<EnvironmentInput>>({
      query: ({ id, ...body }) => ({
        url: `environments/${id}`,
        method: 'PATCH',
        body: keysToSnakeCase(body as never),
      }),
      transformResponse: (response) => keysToCamelCase<Environment>(response as never),
      invalidatesTags: (_result, _error, { id }) => [
        { type: 'Environments', id },
        { type: 'Environments', id: 'LIST' },
        { type: 'EnvironmentCategories', id: 'LIST' },
      ],
    }),
    deleteEnvironment: builder.mutation<void, string>({
      query: (id) => ({ url: `environments/${id}`, method: 'DELETE' }),
      invalidatesTags: (_result, _error, id) => [
        { type: 'Environments', id },
        { type: 'Environments', id: 'LIST' },
      ],
    }),
  }),
})

export const {
  useGetEnvironmentsQuery,
  useGetEnvironmentCategoriesQuery,
  useCreateEnvironmentMutation,
  useUpdateEnvironmentMutation,
  useDeleteEnvironmentMutation,
} = environmentsApi
