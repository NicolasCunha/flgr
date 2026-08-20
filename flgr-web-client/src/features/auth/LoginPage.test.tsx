import { configureStore } from '@reduxjs/toolkit'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HttpResponse, http } from 'msw'
import { Provider } from 'react-redux'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { api } from '@/lib/api'
import { server } from '@/test/msw/server'
import { LoginPage } from './LoginPage'

function renderPage(initialEntries: { pathname: string; state?: unknown }[] = [{ pathname: '/login' }]) {
  const store = configureStore({
    reducer: { [api.reducerPath]: api.reducer },
    middleware: (getDefaultMiddleware) => getDefaultMiddleware().concat(api.middleware),
  })
  return render(
    <Provider store={store}>
      <MemoryRouter initialEntries={initialEntries}>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/" element={<div>Home page</div>} />
          <Route path="/environments" element={<div>Environments page</div>} />
          <Route path="/setup" element={<div>Setup wizard page</div>} />
        </Routes>
      </MemoryRouter>
    </Provider>,
  )
}

describe('LoginPage', () => {
  it('redirects to /setup when no administrator account exists yet', async () => {
    server.use(http.get('/api/v1/setup', () => HttpResponse.json({ needed: true })))
    renderPage()

    expect(await screen.findByText('Setup wizard page')).toBeInTheDocument()
  })

  it('shows validation errors when submitted empty', async () => {
    const user = userEvent.setup()
    renderPage()

    await user.click(await screen.findByRole('button', { name: /sign in/i }))

    expect(await screen.findByText(/enter a valid email address/i)).toBeInTheDocument()
    expect(screen.getByText(/password is required/i)).toBeInTheDocument()
  })

  it('logs in and redirects to / on success', async () => {
    server.use(
      http.post('/api/v1/login', () =>
        HttpResponse.json({
          id: 'user-1',
          first_name: 'Ada',
          last_name: 'Lovelace',
          email: 'ada@example.com',
          status: 'Active',
        }),
      ),
    )
    const user = userEvent.setup()
    renderPage()

    await user.type(await screen.findByLabelText(/email/i), 'ada@example.com')
    await user.type(screen.getByLabelText(/password/i), 'hunter2')
    await user.click(screen.getByRole('button', { name: /sign in/i }))

    expect(await screen.findByText('Home page')).toBeInTheDocument()
  })

  it('redirects to the originally requested page after login', async () => {
    server.use(
      http.post('/api/v1/login', () =>
        HttpResponse.json({
          id: 'user-1',
          first_name: 'Ada',
          last_name: 'Lovelace',
          email: 'ada@example.com',
          status: 'Active',
        }),
      ),
    )
    const user = userEvent.setup()
    renderPage([{ pathname: '/login', state: { from: '/environments' } }])

    await user.type(await screen.findByLabelText(/email/i), 'ada@example.com')
    await user.type(screen.getByLabelText(/password/i), 'hunter2')
    await user.click(screen.getByRole('button', { name: /sign in/i }))

    expect(await screen.findByText('Environments page')).toBeInTheDocument()
  })

  it('shows an error message and stays on the page on invalid credentials', async () => {
    server.use(
      http.post('/api/v1/login', () =>
        HttpResponse.json(
          { error: { code: 'unauthorized', message: 'invalid email or password' } },
          { status: 401 },
        ),
      ),
    )
    const user = userEvent.setup()
    renderPage()

    await user.type(await screen.findByLabelText(/email/i), 'ada@example.com')
    await user.type(screen.getByLabelText(/password/i), 'wrong')
    await user.click(screen.getByRole('button', { name: /sign in/i }))

    expect(await screen.findByRole('alert')).toHaveTextContent(/invalid email or password/i)
  })

  it('disables the submit button while the request is in flight', async () => {
    let resolveRequest: () => void = () => {}
    const pending = new Promise<void>((resolve) => {
      resolveRequest = resolve
    })
    server.use(
      http.post('/api/v1/login', async () => {
        await pending
        return HttpResponse.json({
          id: 'user-1',
          first_name: 'Ada',
          last_name: 'Lovelace',
          email: 'ada@example.com',
          status: 'Active',
        })
      }),
    )
    const user = userEvent.setup()
    renderPage()

    await user.type(await screen.findByLabelText(/email/i), 'ada@example.com')
    await user.type(screen.getByLabelText(/password/i), 'hunter2')
    await user.click(screen.getByRole('button', { name: /sign in/i }))

    expect(await screen.findByRole('button', { name: /signing in/i })).toBeDisabled()
    resolveRequest()
    await screen.findByText('Home page')
  })
})
