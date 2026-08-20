import { configureStore } from '@reduxjs/toolkit'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HttpResponse, http } from 'msw'
import { Provider } from 'react-redux'
import { toast } from 'sonner'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { api } from '@/lib/api'
import { server } from '@/test/msw/server'
import { ServiceKeysPage } from './ServiceKeysPage'

const environmentsHandler = http.get('/api/v1/environments', () =>
  HttpResponse.json({
    data: [
      {
        id: 'env-1',
        name: 'Development',
        description: null,
        environment_category_id: 'cat-1',
        created_on: '2026-08-17T00:00:00Z',
        modified_on: '2026-08-17T00:00:00Z',
      },
    ],
    pagination: { page: 1, page_size: 200, total: 1 },
  }),
)

interface ServiceKeyDto {
  id: string
  name: string
  status: 'Active' | 'Inactive'
  can_read: boolean
  can_write: boolean
  environment_ids: string[]
}

function serviceKeysHandler(serviceKeys: ServiceKeyDto[], total = serviceKeys.length) {
  return http.get('/api/v1/service-keys', ({ request }) => {
    const url = new URL(request.url)
    return HttpResponse.json({
      data: serviceKeys.map((serviceKey) => ({
        ...serviceKey,
        created_on: '2026-08-17T00:00:00Z',
        modified_on: '2026-08-17T00:00:00Z',
      })),
      pagination: {
        page: Number(url.searchParams.get('page') ?? '1'),
        page_size: Number(url.searchParams.get('page_size') ?? '20'),
        total,
      },
    })
  })
}

function renderPage() {
  const store = configureStore({
    reducer: { [api.reducerPath]: api.reducer },
    middleware: (getDefaultMiddleware) => getDefaultMiddleware().concat(api.middleware),
  })
  return render(
    <Provider store={store}>
      <ServiceKeysPage />
    </Provider>,
  )
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('ServiceKeysPage', () => {
  it('shows a loading state, then lists service keys with name, status, access, and environments', async () => {
    server.use(
      environmentsHandler,
      serviceKeysHandler([
        {
          id: 'key-1',
          name: 'checkout-service prod key',
          status: 'Active',
          can_read: true,
          can_write: true,
          environment_ids: ['env-1'],
        },
      ]),
    )
    renderPage()

    expect(screen.getByText(/loading/i)).toBeInTheDocument()
    expect(await screen.findByText('checkout-service prod key')).toBeInTheDocument()
    const row = screen.getByText('checkout-service prod key').closest('tr') as HTMLElement
    expect(within(row).getByText('Active')).toBeInTheDocument()
    expect(within(row).getByText('Read')).toBeInTheDocument()
    expect(within(row).getByText('Write')).toBeInTheDocument()
    expect(within(row).getByText('Development')).toBeInTheDocument()
  })

  it('shows only the enabled access badges, and an em dash for an unresolved or empty environment scope', async () => {
    server.use(
      environmentsHandler,
      serviceKeysHandler([
        {
          id: 'key-1',
          name: 'read-only key',
          status: 'Inactive',
          can_read: true,
          can_write: false,
          environment_ids: [],
        },
        {
          id: 'key-2',
          name: 'orphan-scoped key',
          status: 'Active',
          can_read: false,
          can_write: true,
          environment_ids: ['env-does-not-exist'],
        },
      ]),
    )
    renderPage()

    const readOnlyRow = (await screen.findByText('read-only key')).closest('tr') as HTMLElement
    expect(within(readOnlyRow).getByText('Read')).toBeInTheDocument()
    expect(within(readOnlyRow).queryByText('Write')).not.toBeInTheDocument()
    expect(within(readOnlyRow).getByText('Inactive')).toBeInTheDocument()
    expect(within(readOnlyRow).getByText('—')).toBeInTheDocument()

    const orphanRow = screen.getByText('orphan-scoped key').closest('tr') as HTMLElement
    expect(within(orphanRow).queryByText('Read')).not.toBeInTheDocument()
    expect(within(orphanRow).getByText('Write')).toBeInTheDocument()
    expect(within(orphanRow).getByText('—')).toBeInTheDocument()
  })

  it('shows an error state when the service keys list fails to load', async () => {
    server.use(environmentsHandler, http.get('/api/v1/service-keys', () => HttpResponse.error()))
    renderPage()

    expect(await screen.findByText(/could not load service keys/i)).toBeInTheDocument()
  })

  it('shows the empty-list message when there are no service keys', async () => {
    server.use(environmentsHandler, serviceKeysHandler([], 0))
    renderPage()

    expect(await screen.findByText(/no service keys yet/i)).toBeInTheDocument()
  })

  it('changes page when Next/Previous are clicked', async () => {
    const user = userEvent.setup()
    server.use(
      environmentsHandler,
      serviceKeysHandler(
        [
          {
            id: 'key-1',
            name: 'checkout-service prod key',
            status: 'Active',
            can_read: true,
            can_write: false,
            environment_ids: ['env-1'],
          },
        ],
        40,
      ),
    )
    renderPage()

    expect(await screen.findByText(/page 1 of 2/i)).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Next' }))
    expect(await screen.findByText(/page 2 of 2/i)).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Previous' }))
    expect(await screen.findByText(/page 1 of 2/i)).toBeInTheDocument()
  })

  it('opens the create form from "New service key" and the edit form from a row\'s Edit button', async () => {
    const user = userEvent.setup()
    server.use(
      environmentsHandler,
      serviceKeysHandler([
        {
          id: 'key-1',
          name: 'checkout-service prod key',
          status: 'Active',
          can_read: true,
          can_write: false,
          environment_ids: ['env-1'],
        },
      ]),
    )
    renderPage()

    await user.click(screen.getByRole('button', { name: /new service key/i }))
    expect(await screen.findByRole('heading', { name: /new service key/i })).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: /cancel/i }))
    await waitFor(() =>
      expect(screen.queryByRole('heading', { name: /new service key/i })).not.toBeInTheDocument(),
    )

    await screen.findByText('checkout-service prod key')
    await user.click(screen.getByRole('button', { name: /edit/i }))
    expect(await screen.findByRole('heading', { name: /edit service key/i })).toBeInTheDocument()
    expect(screen.getByDisplayValue('checkout-service prod key')).toBeInTheDocument()
  })

  it('creates a service key end to end: the form closes and the secret dialog opens immediately with the returned secret', async () => {
    const user = userEvent.setup()
    server.use(
      environmentsHandler,
      serviceKeysHandler([]),
      http.post('/api/v1/service-keys', () =>
        HttpResponse.json(
          {
            id: 'key-1',
            name: 'checkout-service prod key',
            status: 'Active',
            can_read: true,
            can_write: false,
            environment_ids: ['env-1'],
            created_on: '2026-08-17T00:00:00Z',
            modified_on: '2026-08-17T00:00:00Z',
            secret: 'sk_live_abc123',
          },
          { status: 201 },
        ),
      ),
    )
    renderPage()

    await user.click(screen.getByRole('button', { name: /new service key/i }))
    expect(await screen.findByRole('heading', { name: /new service key/i })).toBeInTheDocument()

    await user.type(screen.getByLabelText(/^name$/i), 'checkout-service prod key')
    await user.click(await screen.findByLabelText('Development'))
    await user.click(screen.getByRole('button', { name: /create service key/i }))

    await waitFor(() =>
      expect(screen.queryByRole('heading', { name: /new service key/i })).not.toBeInTheDocument(),
    )
    expect(await screen.findByRole('heading', { name: /service key created/i })).toBeInTheDocument()
    expect(screen.getByDisplayValue('sk_live_abc123')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /done/i }))
    await waitFor(() =>
      expect(screen.queryByRole('heading', { name: /service key created/i })).not.toBeInTheDocument(),
    )
  })

  it('deactivates a service key after confirming, and shows a success toast', async () => {
    const successSpy = vi.spyOn(toast, 'success').mockImplementation(() => '')
    const user = userEvent.setup()
    server.use(
      environmentsHandler,
      serviceKeysHandler([
        {
          id: 'key-1',
          name: 'checkout-service prod key',
          status: 'Active',
          can_read: true,
          can_write: false,
          environment_ids: ['env-1'],
        },
      ]),
      http.delete('/api/v1/service-keys/key-1', () =>
        HttpResponse.json({
          id: 'key-1',
          name: 'checkout-service prod key',
          status: 'Inactive',
          can_read: true,
          can_write: false,
          environment_ids: ['env-1'],
          created_on: '2026-08-17T00:00:00Z',
          modified_on: '2026-08-17T00:00:00Z',
        }),
      ),
    )
    renderPage()

    await screen.findByText('checkout-service prod key')
    await user.click(screen.getByRole('button', { name: /deactivate/i }))
    expect(await screen.findByRole('heading', { name: /deactivate service key/i })).toBeInTheDocument()
    expect(screen.getByText(/deactivate "checkout-service prod key"\?/i)).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Deactivate' }))

    await waitFor(() =>
      expect(screen.queryByRole('heading', { name: /deactivate service key/i })).not.toBeInTheDocument(),
    )
    expect(successSpy).toHaveBeenCalledWith('Service key "checkout-service prod key" deactivated.')
  })

  it('cancels the deactivate confirmation without deactivating', async () => {
    const user = userEvent.setup()
    server.use(
      environmentsHandler,
      serviceKeysHandler([
        {
          id: 'key-1',
          name: 'checkout-service prod key',
          status: 'Active',
          can_read: true,
          can_write: false,
          environment_ids: ['env-1'],
        },
      ]),
    )
    renderPage()

    await screen.findByText('checkout-service prod key')
    await user.click(screen.getByRole('button', { name: /deactivate/i }))
    expect(await screen.findByRole('heading', { name: /deactivate service key/i })).toBeInTheDocument()

    const dialog = screen.getByRole('dialog')
    await user.click(within(dialog).getByRole('button', { name: /cancel/i }))

    await waitFor(() =>
      expect(screen.queryByRole('heading', { name: /deactivate service key/i })).not.toBeInTheDocument(),
    )
  })

  it('shows an error toast when deactivation fails, and keeps the confirm dialog open', async () => {
    const errorSpy = vi.spyOn(toast, 'error').mockImplementation(() => '')
    const user = userEvent.setup()
    server.use(
      environmentsHandler,
      serviceKeysHandler([
        {
          id: 'key-1',
          name: 'checkout-service prod key',
          status: 'Active',
          can_read: true,
          can_write: false,
          environment_ids: ['env-1'],
        },
      ]),
      http.delete('/api/v1/service-keys/key-1', () =>
        HttpResponse.json(
          { error: { code: 'conflict', message: 'still in use by a consumer application' } },
          { status: 409 },
        ),
      ),
    )
    renderPage()

    await screen.findByText('checkout-service prod key')
    await user.click(screen.getByRole('button', { name: /deactivate/i }))
    await screen.findByRole('heading', { name: /deactivate service key/i })
    await user.click(screen.getByRole('button', { name: 'Deactivate' }))

    await waitFor(() => expect(errorSpy).toHaveBeenCalledWith('still in use by a consumer application'))
    expect(screen.getByRole('heading', { name: /deactivate service key/i })).toBeInTheDocument()
  })

  it('reactivates an inactive service key directly, with no confirm dialog', async () => {
    const successSpy = vi.spyOn(toast, 'success').mockImplementation(() => '')
    const user = userEvent.setup()
    server.use(
      environmentsHandler,
      serviceKeysHandler([
        {
          id: 'key-1',
          name: 'checkout-service prod key',
          status: 'Inactive',
          can_read: true,
          can_write: false,
          environment_ids: ['env-1'],
        },
      ]),
      http.patch('/api/v1/service-keys/key-1', async ({ request }) => {
        const body = (await request.json()) as Record<string, unknown>
        expect(body).toEqual({ status: 'Active' })
        return HttpResponse.json({
          id: 'key-1',
          name: 'checkout-service prod key',
          status: 'Active',
          can_read: true,
          can_write: false,
          environment_ids: ['env-1'],
          created_on: '2026-08-17T00:00:00Z',
          modified_on: '2026-08-17T00:00:00Z',
        })
      }),
    )
    renderPage()

    await screen.findByText('checkout-service prod key')
    expect(screen.queryByRole('button', { name: /deactivate/i })).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: /reactivate/i }))

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    await waitFor(() =>
      expect(successSpy).toHaveBeenCalledWith('Service key "checkout-service prod key" reactivated.'),
    )
  })

  it('shows an error toast when reactivation fails', async () => {
    const errorSpy = vi.spyOn(toast, 'error').mockImplementation(() => '')
    const user = userEvent.setup()
    server.use(
      environmentsHandler,
      serviceKeysHandler([
        {
          id: 'key-1',
          name: 'checkout-service prod key',
          status: 'Inactive',
          can_read: true,
          can_write: false,
          environment_ids: ['env-1'],
        },
      ]),
      http.patch('/api/v1/service-keys/key-1', () =>
        HttpResponse.json({ error: { code: 'conflict', message: 'cannot reactivate' } }, { status: 409 }),
      ),
    )
    renderPage()

    await screen.findByText('checkout-service prod key')
    await user.click(screen.getByRole('button', { name: /reactivate/i }))

    await waitFor(() => expect(errorSpy).toHaveBeenCalledWith('cannot reactivate'))
  })
})
