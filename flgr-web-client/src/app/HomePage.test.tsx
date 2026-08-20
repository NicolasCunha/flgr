import { configureStore } from '@reduxjs/toolkit'
import { render, screen } from '@testing-library/react'
import { HttpResponse, http } from 'msw'
import { Provider } from 'react-redux'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { api } from '@/lib/api'
import { server } from '@/test/msw/server'
import { HomePage } from './HomePage'

function renderWithProviders() {
  const store = configureStore({
    reducer: { [api.reducerPath]: api.reducer },
    middleware: (getDefaultMiddleware) => getDefaultMiddleware().concat(api.middleware),
  })
  return render(
    <Provider store={store}>
      <MemoryRouter>
        <HomePage />
      </MemoryRouter>
    </Provider>,
  )
}

describe('HomePage', () => {
  it('shows a generic welcome before the current user loads, then the personalized greeting', async () => {
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
    renderWithProviders()

    expect(screen.getByRole('heading', { name: 'Welcome' })).toBeInTheDocument()
    expect(await screen.findByRole('heading', { name: 'Welcome, Ada' })).toBeInTheDocument()
  })

  it('links to the environments and service keys screens', async () => {
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
    renderWithProviders()

    expect(await screen.findByRole('link', { name: /go to environments/i })).toHaveAttribute(
      'href',
      '/environments',
    )
    expect(screen.getByRole('link', { name: /go to service keys/i })).toHaveAttribute(
      'href',
      '/service-keys',
    )
  })
})
