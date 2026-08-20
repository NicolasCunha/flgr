import { configureStore } from '@reduxjs/toolkit'
import { render, screen } from '@testing-library/react'
import { HttpResponse, http } from 'msw'
import { Provider } from 'react-redux'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { api } from '@/lib/api'
import { server } from '@/test/msw/server'
import { RootGuard } from './RootGuard'

function renderRootGuard() {
  const store = configureStore({
    reducer: { [api.reducerPath]: api.reducer },
    middleware: (getDefaultMiddleware) => getDefaultMiddleware().concat(api.middleware),
  })
  return render(
    <Provider store={store}>
      <MemoryRouter initialEntries={['/']}>
        <Routes>
          <Route path="/" element={<RootGuard />}>
            <Route index element={<div>protected content</div>} />
          </Route>
          <Route path="/setup" element={<div>setup wizard</div>} />
        </Routes>
      </MemoryRouter>
    </Provider>,
  )
}

describe('RootGuard', () => {
  it('shows a loading state, then the protected content when setup is not needed', async () => {
    renderRootGuard()
    expect(screen.getByText(/loading/i)).toBeInTheDocument()
    expect(await screen.findByText('protected content')).toBeInTheDocument()
  })

  it('redirects to /setup when setup is needed', async () => {
    server.use(http.get('/api/v1/setup', () => HttpResponse.json({ needed: true })))
    renderRootGuard()
    expect(await screen.findByText('setup wizard')).toBeInTheDocument()
  })

  it('shows an error state when the backend is unreachable', async () => {
    server.use(http.get('/api/v1/setup', () => HttpResponse.error()))
    renderRootGuard()
    expect(await screen.findByText(/backend unreachable/i)).toBeInTheDocument()
  })
})
