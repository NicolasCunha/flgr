import { configureStore } from '@reduxjs/toolkit'
import { render, screen } from '@testing-library/react'
import { HttpResponse, http } from 'msw'
import { Provider } from 'react-redux'
import { describe, expect, it } from 'vitest'
import { api } from '@/lib/api'
import { server } from '@/test/msw/server'
import { HomePage } from './HomePage'

function renderWithStore() {
  const store = configureStore({
    reducer: { [api.reducerPath]: api.reducer },
    middleware: (getDefaultMiddleware) => getDefaultMiddleware().concat(api.middleware),
  })
  return render(
    <Provider store={store}>
      <HomePage />
    </Provider>,
  )
}

describe('HomePage', () => {
  it('shows a loading state, then the backend status', async () => {
    renderWithStore()
    expect(screen.getByText(/checking backend/i)).toBeInTheDocument()
    expect(await screen.findByText(/backend status: ok/i)).toBeInTheDocument()
  })

  it('shows an error state when the backend is unreachable', async () => {
    server.use(http.get('/api/v1/health', () => HttpResponse.error()))
    renderWithStore()
    expect(await screen.findByText(/backend unreachable/i)).toBeInTheDocument()
  })
})
