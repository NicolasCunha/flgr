/// <reference types="node" />
// tsconfig.app.json scopes `types` to just `vite/client` (this is a
// browser app; Node globals aren't ambiently available to app code by
// design). This test needs the real Node `process` global to spy on
// 'unhandledRejection' — the same mechanism Vitest itself surfaces these
// through — so it opts in with a file-scoped reference to @types/node
// (already a devDependency, used elsewhere for vite.config.ts) instead of
// widening every file under src/ to see Node's globals.
import { configureStore } from '@reduxjs/toolkit'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HttpResponse, http } from 'msw'
import { Provider } from 'react-redux'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'
import { api } from '@/lib/api'
import { server } from '@/test/msw/server'
import { AppLayout } from './AppLayout'

function renderLayout() {
  const store = configureStore({
    reducer: { [api.reducerPath]: api.reducer },
    middleware: (getDefaultMiddleware) => getDefaultMiddleware().concat(api.middleware),
  })
  return render(
    <Provider store={store}>
      <MemoryRouter initialEntries={['/']}>
        <Routes>
          <Route path="/login" element={<div>Login page</div>} />
          <Route element={<AppLayout />}>
            <Route path="/" element={<div>Home content</div>} />
          </Route>
        </Routes>
      </MemoryRouter>
    </Provider>,
  )
}

describe('AppLayout', () => {
  it('renders nav links, the current user, and the outlet content', async () => {
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
    renderLayout()

    expect(await screen.findByText('Ada Lovelace')).toBeInTheDocument()
    expect(screen.getByText('Home content')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Environments' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Service Keys' })).toBeInTheDocument()
  })

  it('logs out and navigates to /login', async () => {
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
      http.post('/api/v1/logout', () => new HttpResponse(null, { status: 204 })),
    )
    const user = userEvent.setup()
    renderLayout()

    await screen.findByText('Ada Lovelace')
    await user.click(screen.getByRole('button', { name: /log out/i }))

    expect(await screen.findByText('Login page')).toBeInTheDocument()
  })

  it('still navigates to /login if the logout request fails, without an unhandled rejection', async () => {
    // Regression test: handleLogout() used to be `try { await
    // logout().unwrap() } finally { ...; navigate(...) }` with no catch,
    // so a failed logout still navigated correctly (the finally ran) but
    // then the exception kept propagating out of handleLogout() — and
    // since the click handler is `() => void handleLogout()`, `void` only
    // discards the returned promise, not its eventual rejection, leaving a
    // genuine unhandled promise rejection every time logout failed. Fixed
    // in AppLayout.tsx with a `catch` block. Listening on process-level
    // 'unhandledRejection' here (the same mechanism Vitest itself surfaces
    // these through) catches a regression even though the visible
    // navigation behavior alone wouldn't.
    const onUnhandledRejection = vi.fn()
    process.on('unhandledRejection', onUnhandledRejection)

    try {
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
        http.post('/api/v1/logout', () => HttpResponse.json({ error: { message: 'boom' } }, { status: 500 })),
      )
      const user = userEvent.setup()
      renderLayout()

      await screen.findByText('Ada Lovelace')
      await user.click(screen.getByRole('button', { name: /log out/i }))

      expect(await screen.findByText('Login page')).toBeInTheDocument()

      // Give any unhandled rejection a chance to surface before asserting.
      await new Promise((resolve) => setTimeout(resolve, 0))
      expect(onUnhandledRejection).not.toHaveBeenCalled()
    } finally {
      // Only remove the listener this test added, not Vitest's own
      // process-level 'unhandledRejection' hook (which is what surfaces
      // this class of bug for the rest of the suite).
      process.off('unhandledRejection', onUnhandledRejection)
    }
  })
})
