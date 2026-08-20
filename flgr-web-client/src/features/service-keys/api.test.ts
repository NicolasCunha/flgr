import { configureStore } from '@reduxjs/toolkit'
import { HttpResponse, http } from 'msw'
import { describe, expect, it } from 'vitest'
import { api } from '@/lib/api'
import { server } from '@/test/msw/server'
import { serviceKeysApi } from './api'

function createTestStore() {
  return configureStore({
    reducer: { [api.reducerPath]: api.reducer },
    middleware: (getDefaultMiddleware) => getDefaultMiddleware().concat(api.middleware),
  })
}

const serviceKeyDto = {
  id: 'key-1',
  name: 'checkout-service prod key',
  status: 'Active',
  can_read: true,
  can_write: false,
  environment_ids: ['env-1'],
  created_on: '2026-08-17T00:00:00Z',
  modified_on: '2026-08-17T00:00:00Z',
}

const serviceKeyCamel = {
  id: 'key-1',
  name: 'checkout-service prod key',
  status: 'Active',
  canRead: true,
  canWrite: false,
  environmentIds: ['env-1'],
  createdOn: '2026-08-17T00:00:00Z',
  modifiedOn: '2026-08-17T00:00:00Z',
}

describe('serviceKeysApi', () => {
  it('getServiceKeys sends page/page_size params and returns camelCase data', async () => {
    server.use(
      http.get('/api/v1/service-keys', ({ request }) => {
        const url = new URL(request.url)
        expect(url.searchParams.get('page')).toBe('2')
        expect(url.searchParams.get('page_size')).toBe('10')
        return HttpResponse.json({
          data: [serviceKeyDto],
          pagination: { page: 2, page_size: 10, total: 11 },
        })
      }),
    )
    const store = createTestStore()
    const result = await store.dispatch(
      serviceKeysApi.endpoints.getServiceKeys.initiate({ page: 2, pageSize: 10 }),
    )
    expect(result.data).toEqual({
      data: [serviceKeyCamel],
      pagination: { page: 2, pageSize: 10, total: 11 },
    })
  })

  it('getServiceKey returns a camelCase service key', async () => {
    server.use(http.get('/api/v1/service-keys/key-1', () => HttpResponse.json(serviceKeyDto)))
    const store = createTestStore()
    const result = await store.dispatch(serviceKeysApi.endpoints.getServiceKey.initiate('key-1'))
    expect(result.data).toEqual(serviceKeyCamel)
  })

  // The only endpoint whose response ever carries the plaintext secret —
  // see CreateServiceKeyResponse's comment in ./types.ts.
  it('createServiceKey sends a snake_case body and returns a camelCase key plus the plaintext secret', async () => {
    server.use(
      http.post('/api/v1/service-keys', async ({ request }) => {
        const body = (await request.json()) as Record<string, unknown>
        expect(body).toEqual({
          name: 'checkout-service prod key',
          can_read: true,
          can_write: false,
          environment_ids: ['env-1'],
        })
        return HttpResponse.json({ ...serviceKeyDto, secret: 'sk_live_abc123' }, { status: 201 })
      }),
    )
    const store = createTestStore()
    const result = await store.dispatch(
      serviceKeysApi.endpoints.createServiceKey.initiate({
        name: 'checkout-service prod key',
        canRead: true,
        canWrite: false,
        environmentIds: ['env-1'],
      }),
    )
    expect(result.data).toEqual({ ...serviceKeyCamel, secret: 'sk_live_abc123' })
  })

  it('updateServiceKey PATCHes the given id with a snake_case body, and never receives a secret back', async () => {
    server.use(
      http.patch('/api/v1/service-keys/key-1', async ({ request }) => {
        const body = (await request.json()) as Record<string, unknown>
        expect(body).toEqual({ name: 'renamed key', status: 'Inactive' })
        return HttpResponse.json({ ...serviceKeyDto, name: 'renamed key', status: 'Inactive' })
      }),
    )
    const store = createTestStore()
    const result = await store.dispatch(
      serviceKeysApi.endpoints.updateServiceKey.initiate({
        id: 'key-1',
        name: 'renamed key',
        status: 'Inactive',
      }),
    )
    expect(result.data).toEqual({ ...serviceKeyCamel, name: 'renamed key', status: 'Inactive' })
  })

  // Unlike deleteEnvironment (204/no body), DELETE /service-keys/:id is a
  // deactivation that returns the full updated resource — see api.ts's
  // comment on deactivateServiceKey.
  it('deactivateServiceKey DELETEs the given id and returns the updated (now Inactive) key', async () => {
    server.use(
      http.delete('/api/v1/service-keys/key-1', () =>
        HttpResponse.json({ ...serviceKeyDto, status: 'Inactive' }),
      ),
    )
    const store = createTestStore()
    const result = await store.dispatch(
      serviceKeysApi.endpoints.deactivateServiceKey.initiate('key-1'),
    )
    expect(result.data).toEqual({ ...serviceKeyCamel, status: 'Inactive' })
  })
})
