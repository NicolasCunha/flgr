import { configureStore } from '@reduxjs/toolkit'
import { HttpResponse, http } from 'msw'
import { describe, expect, it } from 'vitest'
import { api } from '@/lib/api'
import { server } from '@/test/msw/server'
import { featureFlagsApi } from './api'

function createTestStore() {
  return configureStore({
    reducer: { [api.reducerPath]: api.reducer },
    middleware: (getDefaultMiddleware) => getDefaultMiddleware().concat(api.middleware),
  })
}

const featureFlagDto = {
  id: 'flag-1',
  key: 'new-checkout-flow',
  name: 'New checkout flow',
  description: 'Enables the redesigned checkout',
  type: 'Boolean',
  created_on: '2026-08-17T00:00:00Z',
  modified_on: '2026-08-17T00:00:00Z',
}

const featureFlagCamel = {
  id: 'flag-1',
  key: 'new-checkout-flow',
  name: 'New checkout flow',
  description: 'Enables the redesigned checkout',
  type: 'Boolean',
  createdOn: '2026-08-17T00:00:00Z',
  modifiedOn: '2026-08-17T00:00:00Z',
}

describe('featureFlagsApi', () => {
  it('getFeatureFlags sends page/page_size params and returns camelCase data', async () => {
    server.use(
      http.get('/api/v1/feature-flags', ({ request }) => {
        const url = new URL(request.url)
        expect(url.searchParams.get('page')).toBe('2')
        expect(url.searchParams.get('page_size')).toBe('10')
        return HttpResponse.json({
          data: [featureFlagDto],
          pagination: { page: 2, page_size: 10, total: 11 },
        })
      }),
    )
    const store = createTestStore()
    const result = await store.dispatch(
      featureFlagsApi.endpoints.getFeatureFlags.initiate({ page: 2, pageSize: 10 }),
    )
    expect(result.data).toEqual({
      data: [featureFlagCamel],
      pagination: { page: 2, pageSize: 10, total: 11 },
    })
  })

  it('getFeatureFlag returns a camelCase flag', async () => {
    server.use(http.get('/api/v1/feature-flags/flag-1', () => HttpResponse.json(featureFlagDto)))
    const store = createTestStore()
    const result = await store.dispatch(featureFlagsApi.endpoints.getFeatureFlag.initiate('flag-1'))
    expect(result.data).toEqual(featureFlagCamel)
  })

  it('createFeatureFlag sends a snake_case body with key/name/description/type and returns a camelCase flag', async () => {
    server.use(
      http.post('/api/v1/feature-flags', async ({ request }) => {
        const body = (await request.json()) as Record<string, unknown>
        expect(body).toEqual({
          key: 'new-checkout-flow',
          name: 'New checkout flow',
          description: 'Enables the redesigned checkout',
          type: 'Boolean',
        })
        return HttpResponse.json(featureFlagDto, { status: 201 })
      }),
    )
    const store = createTestStore()
    const result = await store.dispatch(
      featureFlagsApi.endpoints.createFeatureFlag.initiate({
        key: 'new-checkout-flow',
        name: 'New checkout flow',
        description: 'Enables the redesigned checkout',
        type: 'Boolean',
      }),
    )
    expect(result.data?.id).toBe('flag-1')
  })

  // key and type are deliberately absent from the request — see
  // FeatureFlagUpdateInput's doc comment in ./types.ts and
  // updateFeatureFlag's comment in ./api.ts.
  it('updateFeatureFlag PATCHes the given id with only a name/description body', async () => {
    server.use(
      http.patch('/api/v1/feature-flags/flag-1', async ({ request }) => {
        const body = (await request.json()) as Record<string, unknown>
        expect(body).toEqual({ name: 'Renamed flow', description: 'Updated description' })
        return HttpResponse.json({
          ...featureFlagDto,
          name: 'Renamed flow',
          description: 'Updated description',
        })
      }),
    )
    const store = createTestStore()
    const result = await store.dispatch(
      featureFlagsApi.endpoints.updateFeatureFlag.initiate({
        id: 'flag-1',
        name: 'Renamed flow',
        description: 'Updated description',
      }),
    )
    expect(result.data?.name).toBe('Renamed flow')
  })

  it('deleteFeatureFlag DELETEs the given id', async () => {
    server.use(
      http.delete('/api/v1/feature-flags/flag-1', () => new HttpResponse(null, { status: 204 })),
    )
    const store = createTestStore()
    const result = await store.dispatch(featureFlagsApi.endpoints.deleteFeatureFlag.initiate('flag-1'))
    expect(result.error).toBeUndefined()
  })

  // GET .../values returns { data: [...] } with only explicitly-configured
  // rows (see 0005's Content section) — never a bare array.
  it('listFeatureFlagValues unwraps the data envelope into a camelCase array', async () => {
    server.use(
      http.get('/api/v1/feature-flags/flag-1/values', () =>
        HttpResponse.json({
          data: [
            {
              id: 'val-1',
              feature_flag_id: 'flag-1',
              environment_id: 'env-1',
              enabled: true,
              value: null,
              created_on: '2026-08-17T00:00:00Z',
              modified_on: '2026-08-17T00:00:00Z',
            },
          ],
        }),
      ),
    )
    const store = createTestStore()
    const result = await store.dispatch(
      featureFlagsApi.endpoints.listFeatureFlagValues.initiate('flag-1'),
    )
    expect(result.data).toEqual([
      {
        id: 'val-1',
        featureFlagId: 'flag-1',
        environmentId: 'env-1',
        enabled: true,
        value: null,
        createdOn: '2026-08-17T00:00:00Z',
        modifiedOn: '2026-08-17T00:00:00Z',
      },
    ])
  })

  it('upsertFeatureFlagValue PATCHes flagId/environmentId with an enabled/value body', async () => {
    server.use(
      http.patch('/api/v1/feature-flags/flag-1/values/env-1', async ({ request }) => {
        const body = (await request.json()) as Record<string, unknown>
        expect(body).toEqual({ enabled: true, value: 'on' })
        return HttpResponse.json({
          id: 'val-1',
          feature_flag_id: 'flag-1',
          environment_id: 'env-1',
          enabled: true,
          value: 'on',
          created_on: '2026-08-17T00:00:00Z',
          modified_on: '2026-08-17T00:00:00Z',
        })
      }),
    )
    const store = createTestStore()
    const result = await store.dispatch(
      featureFlagsApi.endpoints.upsertFeatureFlagValue.initiate({
        flagId: 'flag-1',
        environmentId: 'env-1',
        enabled: true,
        value: 'on',
      }),
    )
    expect(result.data?.value).toBe('on')
  })

  // Boolean flags omit `value` entirely (see UpsertValueInput's doc
  // comment) — keysToSnakeCase keeps the `value: undefined` entry, but
  // JSON.stringify (used by fetchBaseQuery to serialize the body) drops
  // any key whose value is undefined, so the wire body never carries it.
  it('upsertFeatureFlagValue omits value from the body when undefined (Boolean flags)', async () => {
    server.use(
      http.patch('/api/v1/feature-flags/flag-1/values/env-1', async ({ request }) => {
        const body = (await request.json()) as Record<string, unknown>
        expect(body).toEqual({ enabled: false })
        expect('value' in body).toBe(false)
        return HttpResponse.json({
          id: 'val-1',
          feature_flag_id: 'flag-1',
          environment_id: 'env-1',
          enabled: false,
          value: null,
          created_on: '2026-08-17T00:00:00Z',
          modified_on: '2026-08-17T00:00:00Z',
        })
      }),
    )
    const store = createTestStore()
    const result = await store.dispatch(
      featureFlagsApi.endpoints.upsertFeatureFlagValue.initiate({
        flagId: 'flag-1',
        environmentId: 'env-1',
        enabled: false,
      }),
    )
    expect(result.error).toBeUndefined()
  })
})
