import { configureStore } from '@reduxjs/toolkit'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HttpResponse, http } from 'msw'
import { Provider } from 'react-redux'
import { describe, expect, it, vi } from 'vitest'
import { api } from '@/lib/api'
import { server } from '@/test/msw/server'
import { ServiceKeyForm } from './ServiceKeyForm'
import type { ServiceKey } from './types'

// GET /environments returns the ADR-0007 { data, pagination } envelope (see
// environments/api.ts) — ServiceKeyForm fetches page 1 / pageSize 200 to
// populate the environment checkbox list.
function environmentsHandler(
  environments: { id: string; name: string }[] = [
    { id: 'env-1', name: 'Development' },
    { id: 'env-2', name: 'Production' },
  ],
) {
  return http.get('/api/v1/environments', () =>
    HttpResponse.json({
      data: environments.map((environment) => ({
        ...environment,
        description: null,
        environment_category_id: 'cat-1',
        created_on: '2026-08-17T00:00:00Z',
        modified_on: '2026-08-17T00:00:00Z',
      })),
      pagination: { page: 1, page_size: 200, total: environments.length },
    }),
  )
}

async function renderForm(props: Partial<React.ComponentProps<typeof ServiceKeyForm>> = {}) {
  const store = configureStore({
    reducer: { [api.reducerPath]: api.reducer },
    middleware: (getDefaultMiddleware) => getDefaultMiddleware().concat(api.middleware),
  })
  const onOpenChange = vi.fn()
  const onCreated = vi.fn()
  const utils = render(
    <Provider store={store}>
      <ServiceKeyForm open onOpenChange={onOpenChange} onCreated={onCreated} {...props} />
    </Provider>,
  )
  return { ...utils, onOpenChange, onCreated }
}

const existingKey: ServiceKey = {
  id: 'key-1',
  name: 'checkout-service prod key',
  status: 'Active',
  canRead: true,
  canWrite: false,
  environmentIds: ['env-1'],
  createdOn: '2026-08-17T00:00:00Z',
  modifiedOn: '2026-08-17T00:00:00Z',
}

describe('ServiceKeyForm', () => {
  it('creates a new service key and surfaces the returned secret via onCreated', async () => {
    server.use(environmentsHandler())
    server.use(
      http.post('/api/v1/service-keys', async ({ request }) => {
        const body = (await request.json()) as Record<string, unknown>
        expect(body).toEqual({
          name: 'checkout-service prod key',
          can_read: true,
          can_write: false,
          environment_ids: ['env-1'],
        })
        return HttpResponse.json(
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
        )
      }),
    )
    const user = userEvent.setup()
    const { onOpenChange, onCreated } = await renderForm()

    await user.type(screen.getByLabelText(/^name$/i), 'checkout-service prod key')
    await user.click(await screen.findByLabelText('Development'))
    await user.click(screen.getByRole('button', { name: /create service key/i }))

    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false))
    expect(onCreated).toHaveBeenCalledWith('sk_live_abc123')
  })

  it('edits an existing service key, sending the full form state (including status) and does not call onCreated', async () => {
    server.use(environmentsHandler())
    server.use(
      http.patch('/api/v1/service-keys/key-1', async ({ request }) => {
        const body = (await request.json()) as Record<string, unknown>
        expect(body).toEqual({
          name: 'checkout-service prod key 2',
          status: 'Inactive',
          can_read: true,
          can_write: false,
          environment_ids: ['env-1'],
        })
        return HttpResponse.json({
          id: 'key-1',
          name: 'checkout-service prod key 2',
          status: 'Inactive',
          can_read: true,
          can_write: false,
          environment_ids: ['env-1'],
          created_on: '2026-08-17T00:00:00Z',
          modified_on: '2026-08-17T00:00:00Z',
        })
      }),
    )
    const user = userEvent.setup()
    const { onOpenChange, onCreated } = await renderForm({ serviceKey: existingKey })

    expect(await screen.findByDisplayValue('checkout-service prod key')).toBeInTheDocument()
    const nameInput = screen.getByLabelText(/^name$/i)
    await user.clear(nameInput)
    await user.type(nameInput, 'checkout-service prod key 2')
    // Flip the status switch from Active to Inactive.
    await user.click(screen.getByRole('switch', { name: /active/i }))
    await user.click(screen.getByRole('button', { name: /save changes/i }))

    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false))
    expect(onCreated).not.toHaveBeenCalled()
  })

  it('edits an existing service key without touching the status switch, sending status Active unchanged', async () => {
    server.use(environmentsHandler())
    server.use(
      http.patch('/api/v1/service-keys/key-1', async ({ request }) => {
        const body = (await request.json()) as Record<string, unknown>
        expect(body).toMatchObject({ status: 'Active' })
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
    const user = userEvent.setup()
    const { onOpenChange } = await renderForm({ serviceKey: existingKey })

    expect(await screen.findByDisplayValue('checkout-service prod key')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: /save changes/i }))

    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false))
  })

  it('hides the status switch when creating', async () => {
    server.use(environmentsHandler())
    await renderForm()
    await screen.findByLabelText('Development')
    expect(screen.queryByRole('switch', { name: /active|inactive/i })).not.toBeInTheDocument()
  })

  it('shows the status switch when editing', async () => {
    server.use(environmentsHandler())
    await renderForm({ serviceKey: existingKey })
    expect(await screen.findByRole('switch', { name: /active/i })).toBeInTheDocument()
  })

  it('shows validation errors for a missing name, no environment selected, and neither access flag enabled', async () => {
    server.use(environmentsHandler())
    const user = userEvent.setup()
    await renderForm()
    await screen.findByLabelText('Development')

    // Turn off "Can read" (default true) so both access switches are off,
    // then submit with nothing else filled in.
    await user.click(screen.getByRole('switch', { name: /can read/i }))
    await user.click(screen.getByRole('button', { name: /create service key/i }))

    expect(await screen.findByText(/name is required/i)).toBeInTheDocument()
    expect(screen.getByText(/select at least one environment/i)).toBeInTheDocument()
    expect(screen.getByText(/enable at least one of can read \/ can write/i)).toBeInTheDocument()
  })

  it('toggles an environment checkbox on and back off', async () => {
    server.use(environmentsHandler())
    const user = userEvent.setup()
    await renderForm()

    const devCheckbox = await screen.findByLabelText('Development')
    expect(devCheckbox).not.toBeChecked()
    await user.click(devCheckbox)
    expect(devCheckbox).toBeChecked()
    await user.click(devCheckbox)
    expect(devCheckbox).not.toBeChecked()
  })

  it('toggles the canWrite switch on', async () => {
    server.use(environmentsHandler())
    server.use(
      http.post('/api/v1/service-keys', async ({ request }) => {
        const body = (await request.json()) as Record<string, unknown>
        expect(body).toMatchObject({ can_read: true, can_write: true })
        return HttpResponse.json(
          {
            id: 'key-1',
            name: 'my key',
            status: 'Active',
            can_read: true,
            can_write: true,
            environment_ids: ['env-1'],
            created_on: '2026-08-17T00:00:00Z',
            modified_on: '2026-08-17T00:00:00Z',
            secret: 'sk_live_xyz',
          },
          { status: 201 },
        )
      }),
    )
    const user = userEvent.setup()
    const { onCreated } = await renderForm()

    await user.type(screen.getByLabelText(/^name$/i), 'my key')
    await user.click(await screen.findByLabelText('Development'))
    await user.click(screen.getByRole('switch', { name: /can write/i }))
    await user.click(screen.getByRole('button', { name: /create service key/i }))

    await waitFor(() => expect(onCreated).toHaveBeenCalledWith('sk_live_xyz'))
  })

  it('shows a fallback message when there are no environments to select', async () => {
    server.use(environmentsHandler([]))
    await renderForm()
    expect(await screen.findByText(/no environments exist yet/i)).toBeInTheDocument()
  })

  it('shows "Saving..." on the submit button while the create request is in flight', async () => {
    let resolveRequest: () => void = () => {}
    const pending = new Promise<void>((resolve) => {
      resolveRequest = resolve
    })
    server.use(environmentsHandler())
    server.use(
      http.post('/api/v1/service-keys', async () => {
        await pending
        return HttpResponse.json(
          {
            id: 'key-1',
            name: 'my key',
            status: 'Active',
            can_read: true,
            can_write: false,
            environment_ids: ['env-1'],
            created_on: '2026-08-17T00:00:00Z',
            modified_on: '2026-08-17T00:00:00Z',
            secret: 'sk_live_xyz',
          },
          { status: 201 },
        )
      }),
    )
    const user = userEvent.setup()
    await renderForm()

    await user.type(screen.getByLabelText(/^name$/i), 'my key')
    await user.click(await screen.findByLabelText('Development'))
    await user.click(screen.getByRole('button', { name: /create service key/i }))

    expect(await screen.findByRole('button', { name: /saving/i })).toBeDisabled()
    resolveRequest()
    await waitFor(() => expect(screen.queryByRole('button', { name: /saving/i })).not.toBeInTheDocument())
  })

  it('shows a server error message when creation fails', async () => {
    server.use(environmentsHandler())
    server.use(
      http.post('/api/v1/service-keys', () =>
        HttpResponse.json(
          { error: { code: 'conflict', message: 'name already in use' } },
          { status: 409 },
        ),
      ),
    )
    const user = userEvent.setup()
    await renderForm()

    await user.type(screen.getByLabelText(/^name$/i), 'checkout-service prod key')
    await user.click(await screen.findByLabelText('Development'))
    await user.click(screen.getByRole('button', { name: /create service key/i }))

    expect(await screen.findByRole('alert')).toHaveTextContent(/name already in use/i)
  })

  it('closes the dialog when Cancel is clicked', async () => {
    server.use(environmentsHandler())
    const user = userEvent.setup()
    const { onOpenChange } = await renderForm()

    await user.click(screen.getByRole('button', { name: /cancel/i }))
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })

  it('renders nothing interactive when closed', async () => {
    server.use(environmentsHandler())
    await renderForm({ open: false })
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })
})
