import { configureStore } from '@reduxjs/toolkit'
import { HttpResponse, http } from 'msw'
import { describe, expect, it } from 'vitest'
import { api } from '@/lib/api'
import { server } from '@/test/msw/server'
import { authApi } from './api'

function createTestStore() {
  return configureStore({
    reducer: { [api.reducerPath]: api.reducer },
    middleware: (getDefaultMiddleware) => getDefaultMiddleware().concat(api.middleware),
  })
}

describe('authApi', () => {
  it('login sends a snake_case body and returns a camelCase user', async () => {
    server.use(
      http.post('/api/v1/login', async ({ request }) => {
        const body = (await request.json()) as Record<string, unknown>
        expect(body).toEqual({ email: 'ada@example.com', password: 'hunter2' })
        return HttpResponse.json({
          id: 'user-1',
          first_name: 'Ada',
          last_name: 'Lovelace',
          email: 'ada@example.com',
          status: 'Active',
        })
      }),
    )
    const store = createTestStore()
    const result = await store.dispatch(
      authApi.endpoints.login.initiate({ email: 'ada@example.com', password: 'hunter2' }),
    )
    expect(result.data).toEqual({
      id: 'user-1',
      firstName: 'Ada',
      lastName: 'Lovelace',
      email: 'ada@example.com',
      status: 'Active',
    })
  })

  it('logout calls POST /logout', async () => {
    server.use(http.post('/api/v1/logout', () => new HttpResponse(null, { status: 204 })))
    const store = createTestStore()
    const result = await store.dispatch(authApi.endpoints.logout.initiate())
    expect(result.error).toBeUndefined()
  })

  it('getMe returns a camelCase user', async () => {
    server.use(
      http.get('/api/v1/me', () =>
        HttpResponse.json({
          id: 'user-1',
          first_name: 'Ada',
          last_name: 'Lovelace',
          email: 'ada@example.com',
          status: 'Active',
        }),
      ),
    )
    const store = createTestStore()
    const result = await store.dispatch(authApi.endpoints.getMe.initiate())
    expect(result.data?.firstName).toBe('Ada')
  })

  it('getMe surfaces a 401 error when unauthenticated', async () => {
    server.use(
      http.get('/api/v1/me', () =>
        HttpResponse.json({ error: { code: 'unauthorized', message: 'not logged in' } }, { status: 401 }),
      ),
    )
    const store = createTestStore()
    const result = await store.dispatch(authApi.endpoints.getMe.initiate())
    expect(result.error).toMatchObject({ status: 401 })
  })
})
