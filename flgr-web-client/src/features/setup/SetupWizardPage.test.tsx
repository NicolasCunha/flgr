import { configureStore } from '@reduxjs/toolkit'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HttpResponse, http } from 'msw'
import { Provider } from 'react-redux'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { api } from '@/lib/api'
import { server } from '@/test/msw/server'
import { SetupWizardPage } from './SetupWizardPage'

function renderWizard() {
  const store = configureStore({
    reducer: { [api.reducerPath]: api.reducer },
    middleware: (getDefaultMiddleware) => getDefaultMiddleware().concat(api.middleware),
  })
  return render(
    <Provider store={store}>
      <MemoryRouter initialEntries={['/setup']}>
        <Routes>
          <Route path="/" element={<div>home page</div>} />
          <Route path="/setup" element={<SetupWizardPage />} />
        </Routes>
      </MemoryRouter>
    </Provider>,
  )
}

async function fillValidForm(user: ReturnType<typeof userEvent.setup>) {
  await user.type(await screen.findByLabelText(/first name/i), 'Ada')
  await user.type(screen.getByLabelText(/last name/i), 'Lovelace')
  await user.type(screen.getByLabelText(/email/i), 'ada@example.com')
  await user.type(screen.getByLabelText(/^password$/i), 'supersecret')
  await user.type(screen.getByLabelText(/confirm password/i), 'supersecret')
}

describe('SetupWizardPage', () => {
  it('shows a loading state before the setup status resolves', async () => {
    server.use(http.get('/api/v1/setup', async () => HttpResponse.json({ needed: true })))
    renderWizard()
    expect(screen.getByText(/loading/i)).toBeInTheDocument()
    await screen.findByText(/welcome to flgr/i)
  })

  it('redirects to / when setup is not needed', async () => {
    server.use(http.get('/api/v1/setup', () => HttpResponse.json({ needed: false })))
    renderWizard()
    expect(await screen.findByText('home page')).toBeInTheDocument()
  })

  it('renders the form when setup is needed', async () => {
    server.use(http.get('/api/v1/setup', () => HttpResponse.json({ needed: true })))
    renderWizard()
    expect(await screen.findByRole('button', { name: /create administrator account/i })).toBeInTheDocument()
  })

  it('shows validation errors for empty required fields', async () => {
    server.use(http.get('/api/v1/setup', () => HttpResponse.json({ needed: true })))
    const user = userEvent.setup()
    renderWizard()

    await user.click(await screen.findByRole('button', { name: /create administrator account/i }))

    expect(await screen.findByText(/first name is required/i)).toBeInTheDocument()
    expect(screen.getByText(/last name is required/i)).toBeInTheDocument()
    expect(screen.getByText(/email is required/i)).toBeInTheDocument()
    expect(screen.getByText(/password must be at least 8 characters/i)).toBeInTheDocument()
  })

  it('shows an error when the email is malformed', async () => {
    server.use(http.get('/api/v1/setup', () => HttpResponse.json({ needed: true })))
    const user = userEvent.setup()
    renderWizard()

    await user.type(await screen.findByLabelText(/email/i), 'not-an-email')
    await user.type(screen.getByLabelText(/^password$/i), 'supersecret')
    await user.type(screen.getByLabelText(/confirm password/i), 'supersecret')
    await user.click(screen.getByRole('button', { name: /create administrator account/i }))

    expect(await screen.findByText(/enter a valid email address/i)).toBeInTheDocument()
  })

  it('shows an error when the passwords do not match', async () => {
    server.use(http.get('/api/v1/setup', () => HttpResponse.json({ needed: true })))
    const user = userEvent.setup()
    renderWizard()

    await user.type(await screen.findByLabelText(/first name/i), 'Ada')
    await user.type(screen.getByLabelText(/last name/i), 'Lovelace')
    await user.type(screen.getByLabelText(/email/i), 'ada@example.com')
    await user.type(screen.getByLabelText(/^password$/i), 'supersecret')
    await user.type(screen.getByLabelText(/confirm password/i), 'different')
    await user.click(screen.getByRole('button', { name: /create administrator account/i }))

    expect(await screen.findByText(/passwords don't match/i)).toBeInTheDocument()
  })

  it('creates the account, logs in, and navigates home on success', async () => {
    server.use(
      http.get('/api/v1/setup', () => HttpResponse.json({ needed: true })),
      http.post('/api/v1/setup', async ({ request }) => {
        const body = (await request.json()) as Record<string, string>
        expect(body).toEqual({
          first_name: 'Ada',
          last_name: 'Lovelace',
          email: 'ada@example.com',
          password: 'supersecret',
        })
        return HttpResponse.json(
          {
            id: '1',
            first_name: 'Ada',
            last_name: 'Lovelace',
            email: 'ada@example.com',
            status: 'Active',
            created_on: '2026-08-17T00:00:00Z',
            modified_on: '2026-08-17T00:00:00Z',
          },
          { status: 201 },
        )
      }),
      http.post('/api/v1/login', () =>
        HttpResponse.json({
          id: '1',
          first_name: 'Ada',
          last_name: 'Lovelace',
          email: 'ada@example.com',
          status: 'Active',
          created_on: '2026-08-17T00:00:00Z',
          modified_on: '2026-08-17T00:00:00Z',
        }),
      ),
    )
    const user = userEvent.setup()
    renderWizard()
    await fillValidForm(user)
    await user.click(await screen.findByRole('button', { name: /create administrator account/i }))

    expect(await screen.findByText('home page')).toBeInTheDocument()
  })

  it('still navigates home if the auto-login after setup fails', async () => {
    server.use(
      http.get('/api/v1/setup', () => HttpResponse.json({ needed: true })),
      http.post('/api/v1/setup', () =>
        HttpResponse.json(
          {
            id: '1',
            first_name: 'Ada',
            last_name: 'Lovelace',
            email: 'ada@example.com',
            status: 'Active',
            created_on: '2026-08-17T00:00:00Z',
            modified_on: '2026-08-17T00:00:00Z',
          },
          { status: 201 },
        ),
      ),
      http.post('/api/v1/login', () =>
        HttpResponse.json({ error: { code: 'invalid_credentials', message: 'invalid email or password' } }, { status: 401 }),
      ),
    )
    const user = userEvent.setup()
    renderWizard()
    await fillValidForm(user)
    await user.click(await screen.findByRole('button', { name: /create administrator account/i }))

    expect(await screen.findByText('home page')).toBeInTheDocument()
  })

  it('redirects home when the backend reports setup already completed (409)', async () => {
    server.use(
      http.get('/api/v1/setup', () => HttpResponse.json({ needed: true })),
      http.post('/api/v1/setup', () =>
        HttpResponse.json({ error: { code: 'conflict', message: 'initial setup has already been completed' } }, { status: 409 }),
      ),
    )
    const user = userEvent.setup()
    renderWizard()
    await fillValidForm(user)
    await user.click(await screen.findByRole('button', { name: /create administrator account/i }))

    expect(await screen.findByText('home page')).toBeInTheDocument()
  })

  it('shows the server error message for a non-conflict failure', async () => {
    server.use(
      http.get('/api/v1/setup', () => HttpResponse.json({ needed: true })),
      http.post('/api/v1/setup', () =>
        HttpResponse.json({ error: { code: 'validation_error', message: 'email is not well-formed' } }, { status: 400 }),
      ),
    )
    const user = userEvent.setup()
    renderWizard()
    await fillValidForm(user)
    await user.click(await screen.findByRole('button', { name: /create administrator account/i }))

    expect(await screen.findByText('email is not well-formed')).toBeInTheDocument()
  })

  it('disables the submit button while the request is in flight', async () => {
    server.use(
      http.get('/api/v1/setup', () => HttpResponse.json({ needed: true })),
      http.post('/api/v1/setup', async () => {
        await new Promise((resolve) => setTimeout(resolve, 20))
        return HttpResponse.json(
          {
            id: '1',
            first_name: 'Ada',
            last_name: 'Lovelace',
            email: 'ada@example.com',
            status: 'Active',
            created_on: '2026-08-17T00:00:00Z',
            modified_on: '2026-08-17T00:00:00Z',
          },
          { status: 201 },
        )
      }),
      http.post('/api/v1/login', () =>
        HttpResponse.json({
          id: '1',
          first_name: 'Ada',
          last_name: 'Lovelace',
          email: 'ada@example.com',
          status: 'Active',
          created_on: '2026-08-17T00:00:00Z',
          modified_on: '2026-08-17T00:00:00Z',
        }),
      ),
    )
    const user = userEvent.setup()
    renderWizard()
    await fillValidForm(user)
    const submitButton = screen.getByRole('button', { name: /create administrator account/i })
    await user.click(submitButton)

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /creating account/i })).toBeDisabled()
    })
    await screen.findByText('home page')
  })
})
