import { api } from '@/lib/api'
import { keysToCamelCase, keysToSnakeCase } from '@/lib/casing'
import type {
  CreateServiceKeyResponse,
  ListServiceKeysParams,
  ListResponse,
  ServiceKey,
  ServiceKeyInput,
  ServiceKeyUpdateInput,
} from './types'

// Service keys per docs/business/requirements/0004-service-key-management.md.
// The 'ServiceKeys' tag is declared once in src/lib/api.ts's shared
// tagTypes array (alongside Environments, FeatureFlags, ...) rather than
// here.
export const serviceKeysApi = api.injectEndpoints({
  endpoints: (builder) => ({
    getServiceKeys: builder.query<ListResponse<ServiceKey>, ListServiceKeysParams>({
      query: ({ page, pageSize }) => ({
        url: 'service-keys',
        params: { page, page_size: pageSize },
      }),
      transformResponse: (response) => keysToCamelCase<ListResponse<ServiceKey>>(response as never),
      providesTags: (result) =>
        result
          ? [
              ...result.data.map(({ id }) => ({ type: 'ServiceKeys' as const, id })),
              { type: 'ServiceKeys' as const, id: 'LIST' },
            ]
          : [{ type: 'ServiceKeys' as const, id: 'LIST' }],
    }),
    getServiceKey: builder.query<ServiceKey, string>({
      query: (id) => `service-keys/${id}`,
      transformResponse: (response) => keysToCamelCase<ServiceKey>(response as never),
      providesTags: (_result, _error, id) => [{ type: 'ServiceKeys', id }],
    }),
    // The only endpoint whose response ever carries the plaintext secret —
    // see CreateServiceKeyResponse's comment in ./types.ts.
    createServiceKey: builder.mutation<CreateServiceKeyResponse, ServiceKeyInput>({
      query: (body) => ({
        url: 'service-keys',
        method: 'POST',
        body: keysToSnakeCase(body as never),
      }),
      transformResponse: (response) => keysToCamelCase<CreateServiceKeyResponse>(response as never),
      invalidatesTags: [{ type: 'ServiceKeys', id: 'LIST' }],
    }),
    updateServiceKey: builder.mutation<ServiceKey, { id: string } & ServiceKeyUpdateInput>({
      query: ({ id, ...body }) => ({
        url: `service-keys/${id}`,
        method: 'PATCH',
        body: keysToSnakeCase(body as never),
      }),
      transformResponse: (response) => keysToCamelCase<ServiceKey>(response as never),
      invalidatesTags: (_result, _error, { id }) => [
        { type: 'ServiceKeys', id },
        { type: 'ServiceKeys', id: 'LIST' },
      ],
    }),
    // DELETE /service-keys/:id is a deactivation (status -> Inactive), not
    // a real delete — see ServiceKeyHandler.Deactivate. It returns the
    // updated serviceKeyResponse (200), unlike environments' DELETE which
    // returns nothing, so this mutation's result type reflects that rather
    // than following deleteEnvironment's `void` shape.
    deactivateServiceKey: builder.mutation<ServiceKey, string>({
      query: (id) => ({ url: `service-keys/${id}`, method: 'DELETE' }),
      transformResponse: (response) => keysToCamelCase<ServiceKey>(response as never),
      invalidatesTags: (_result, _error, id) => [
        { type: 'ServiceKeys', id },
        { type: 'ServiceKeys', id: 'LIST' },
      ],
    }),
  }),
})

export const {
  useGetServiceKeysQuery,
  useGetServiceKeyQuery,
  useCreateServiceKeyMutation,
  useUpdateServiceKeyMutation,
  useDeactivateServiceKeyMutation,
} = serviceKeysApi
