import { configureStore } from '@reduxjs/toolkit'
import { HttpResponse, http } from 'msw'
import { describe, expect, it } from 'vitest'
import { api } from '@/lib/api'
import { server } from '@/test/msw/server'
import { environmentsApi } from './api'

function createTestStore() {
  return configureStore({
    reducer: { [api.reducerPath]: api.reducer },
    middleware: (getDefaultMiddleware) => getDefaultMiddleware().concat(api.middleware),
  })
}

const environmentDto = {
  id: 'env-1',
  name: 'app-dev',
  description: 'Development environment',
  environment_category_id: 'cat-1',
  created_on: '2026-08-17T00:00:00Z',
  modified_on: '2026-08-17T00:00:00Z',
}

describe('environmentsApi', () => {
  it('getEnvironments sends page/page_size params and returns camelCase data', async () => {
    server.use(
      http.get('/api/v1/environments', ({ request }) => {
        const url = new URL(request.url)
        expect(url.searchParams.get('page')).toBe('2')
        expect(url.searchParams.get('page_size')).toBe('10')
        return HttpResponse.json({
          data: [environmentDto],
          pagination: { page: 2, page_size: 10, total: 11 },
        })
      }),
    )
    const store = createTestStore()
    const result = await store.dispatch(
      environmentsApi.endpoints.getEnvironments.initiate({ page: 2, pageSize: 10 }),
    )
    expect(result.data).toEqual({
      data: [
        {
          id: 'env-1',
          name: 'app-dev',
          description: 'Development environment',
          environmentCategoryId: 'cat-1',
          createdOn: '2026-08-17T00:00:00Z',
          modifiedOn: '2026-08-17T00:00:00Z',
        },
      ],
      pagination: { page: 2, pageSize: 10, total: 11 },
    })
  })

  // GET /environment-categories returns { data: [...] } (ADR-0007's shape
  // minus a pagination block, per flgr-server's
  // EnvironmentCategoryHandler.List) — not a bare array.
  it('getEnvironmentCategories unwraps the data envelope into a camelCase array', async () => {
    server.use(
      http.get('/api/v1/environment-categories', () =>
        HttpResponse.json({
          data: [
            { id: 'cat-1', name: 'Development', is_system: true },
            { id: 'cat-2', name: 'Staging', is_system: true },
          ],
        }),
      ),
    )
    const store = createTestStore()
    const result = await store.dispatch(environmentsApi.endpoints.getEnvironmentCategories.initiate())
    expect(result.data).toEqual([
      { id: 'cat-1', name: 'Development', isSystem: true },
      { id: 'cat-2', name: 'Staging', isSystem: true },
    ])
  })

  // There's no POST /environment-categories on flgr-server — creating an
  // environment with a new category_name resolves/creates the category
  // server-side in one call (see flgr-server's
  // EnvironmentService.resolveCategory), so category_name (a name, not an
  // id) is what the request body carries.
  it('createEnvironment sends a snake_case body with category_name and returns a camelCase environment', async () => {
    server.use(
      http.post('/api/v1/environments', async ({ request }) => {
        const body = (await request.json()) as Record<string, unknown>
        expect(body).toEqual({
          name: 'app-dev',
          description: 'Development environment',
          category_name: 'Development',
        })
        return HttpResponse.json(environmentDto, { status: 201 })
      }),
    )
    const store = createTestStore()
    const result = await store.dispatch(
      environmentsApi.endpoints.createEnvironment.initiate({
        name: 'app-dev',
        description: 'Development environment',
        categoryName: 'Development',
      }),
    )
    expect(result.data?.name).toBe('app-dev')
  })

  it('updateEnvironment PATCHes the given id with a snake_case body', async () => {
    server.use(
      http.patch('/api/v1/environments/env-1', async ({ request }) => {
        const body = (await request.json()) as Record<string, unknown>
        expect(body).toEqual({ name: 'app-dev-2' })
        return HttpResponse.json({ ...environmentDto, name: 'app-dev-2' })
      }),
    )
    const store = createTestStore()
    const result = await store.dispatch(
      environmentsApi.endpoints.updateEnvironment.initiate({ id: 'env-1', name: 'app-dev-2' }),
    )
    expect(result.data?.name).toBe('app-dev-2')
  })

  it('deleteEnvironment DELETEs the given id', async () => {
    server.use(
      http.delete('/api/v1/environments/env-1', () => new HttpResponse(null, { status: 204 })),
    )
    const store = createTestStore()
    const result = await store.dispatch(environmentsApi.endpoints.deleteEnvironment.initiate('env-1'))
    expect(result.error).toBeUndefined()
  })
})
