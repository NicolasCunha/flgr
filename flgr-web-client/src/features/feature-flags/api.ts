import { api } from '@/lib/api'
import { keysToCamelCase, keysToSnakeCase } from '@/lib/casing'
import type {
  FeatureFlag,
  FeatureFlagEnvironmentValue,
  FeatureFlagInput,
  FeatureFlagUpdateInput,
  ListFeatureFlagsParams,
  ListResponse,
  UpsertValueInput,
} from './types'

// Feature flags per docs/business/requirements/0005-feature-flag-management.md.
// Flag definitions (this file's first half) and their per-environment
// values (the second half, listValues/upsertValue) are gated by separate
// permission surfaces on the backend (FeatureFlag: * vs. FeatureFlagValue:
// Read/Write — see flgr-server's feature_flag_value.go), but both live
// here since they're the same UI feature slice.
export const featureFlagsApi = api.injectEndpoints({
  endpoints: (builder) => ({
    getFeatureFlags: builder.query<ListResponse<FeatureFlag>, ListFeatureFlagsParams>({
      query: ({ page, pageSize }) => ({
        url: 'feature-flags',
        params: { page, page_size: pageSize },
      }),
      transformResponse: (response) => keysToCamelCase<ListResponse<FeatureFlag>>(response as never),
      providesTags: (result) =>
        result
          ? [
              ...result.data.map(({ id }) => ({ type: 'FeatureFlags' as const, id })),
              { type: 'FeatureFlags' as const, id: 'LIST' },
            ]
          : [{ type: 'FeatureFlags' as const, id: 'LIST' }],
    }),
    getFeatureFlag: builder.query<FeatureFlag, string>({
      query: (id) => `feature-flags/${id}`,
      transformResponse: (response) => keysToCamelCase<FeatureFlag>(response as never),
      providesTags: (_result, _error, id) => [{ type: 'FeatureFlags', id }],
    }),
    createFeatureFlag: builder.mutation<FeatureFlag, FeatureFlagInput>({
      query: (body) => ({
        url: 'feature-flags',
        method: 'POST',
        body: keysToSnakeCase(body as never),
      }),
      transformResponse: (response) => keysToCamelCase<FeatureFlag>(response as never),
      invalidatesTags: [{ type: 'FeatureFlags', id: 'LIST' }],
    }),
    // key and type are deliberately not accepted here — see
    // FeatureFlagUpdateInput's doc comment in ./types.ts.
    updateFeatureFlag: builder.mutation<FeatureFlag, { id: string } & FeatureFlagUpdateInput>({
      query: ({ id, ...body }) => ({
        url: `feature-flags/${id}`,
        method: 'PATCH',
        body: keysToSnakeCase(body as never),
      }),
      transformResponse: (response) => keysToCamelCase<FeatureFlag>(response as never),
      invalidatesTags: (_result, _error, { id }) => [
        { type: 'FeatureFlags', id },
        { type: 'FeatureFlags', id: 'LIST' },
      ],
    }),
    deleteFeatureFlag: builder.mutation<void, string>({
      query: (id) => ({ url: `feature-flags/${id}`, method: 'DELETE' }),
      // A flag delete cascades its per-environment values away too (see
      // flgr-server's FeatureFlagHandler.Delete), so drop any cached
      // FeatureFlagValues for it as well.
      invalidatesTags: (_result, _error, id) => [
        { type: 'FeatureFlags', id },
        { type: 'FeatureFlags', id: 'LIST' },
        { type: 'FeatureFlagValues', id },
      ],
    }),
    listFeatureFlagValues: builder.query<FeatureFlagEnvironmentValue[], string>({
      query: (flagId) => `feature-flags/${flagId}/values`,
      transformResponse: (response) =>
        keysToCamelCase<FeatureFlagEnvironmentValue[]>((response as { data: unknown[] }).data as never),
      providesTags: (_result, _error, flagId) => [{ type: 'FeatureFlagValues', id: flagId }],
    }),
    upsertFeatureFlagValue: builder.mutation<FeatureFlagEnvironmentValue, UpsertValueInput>({
      query: ({ flagId, environmentId, enabled, value }) => ({
        url: `feature-flags/${flagId}/values/${environmentId}`,
        method: 'PATCH',
        body: keysToSnakeCase({ enabled, value } as never),
      }),
      transformResponse: (response) => keysToCamelCase<FeatureFlagEnvironmentValue>(response as never),
      invalidatesTags: (_result, _error, { flagId }) => [{ type: 'FeatureFlagValues', id: flagId }],
    }),
  }),
})

export const {
  useGetFeatureFlagsQuery,
  useGetFeatureFlagQuery,
  useCreateFeatureFlagMutation,
  useUpdateFeatureFlagMutation,
  useDeleteFeatureFlagMutation,
  useListFeatureFlagValuesQuery,
  useUpsertFeatureFlagValueMutation,
} = featureFlagsApi
