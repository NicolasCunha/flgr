import { configureStore } from '@reduxjs/toolkit'
import { render, screen } from '@testing-library/react'
import { HttpResponse, http } from 'msw'
import { Provider } from 'react-redux'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { api } from '@/lib/api'
import { server } from '@/test/msw/server'
import { RequireAuth } from './RequireAuth'

function renderAt(path: string) {
  const store = configureStore({
    reducer: { [api.reducerPath]: api.reducer },
    middleware: (getDefaultMiddleware) => getDefaultMiddleware().concat(api.middleware),
  })
  return render(
    <Provider store={store}>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/login" element={<div>Login page</div>} />
          <Route element={<RequireAuth />}>
            <Route path="/" element={<div>Protected home</div>} />
          </Route>
        </Routes>
      </MemoryRouter>
    </Provider>,
  )
}

describe('RequireAuth', () => {
  it('renders the protected route when authenticated', async () => {
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
    renderAt('/')

    expect(await screen.findByText('Protected home')).toBeInTheDocument()
  })

  it('redirects to /login when the session is invalid', async () => {
    server.use(
      http.get('/api/v1/me', () =>
        HttpResponse.json({ error: { code: 'unauthorized', message: 'not logged in' } }, { status: 401 }),
      ),
    )
    renderAt('/')

    expect(await screen.findByText('Login page')).toBeInTheDocument()
  })
})
