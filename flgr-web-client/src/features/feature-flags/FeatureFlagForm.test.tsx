import { configureStore } from '@reduxjs/toolkit'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HttpResponse, http } from 'msw'
import { Provider } from 'react-redux'
import { describe, expect, it, vi } from 'vitest'
import { api } from '@/lib/api'
import { server } from '@/test/msw/server'
import { FeatureFlagForm } from './FeatureFlagForm'
import type { FeatureFlag } from './types'

async function renderForm(props: Partial<React.ComponentProps<typeof FeatureFlagForm>> = {}) {
  const store = configureStore({
    reducer: { [api.reducerPath]: api.reducer },
    middleware: (getDefaultMiddleware) => getDefaultMiddleware().concat(api.middleware),
  })
  const onOpenChange = vi.fn()
  const utils = render(
    <Provider store={store}>
      <FeatureFlagForm open onOpenChange={onOpenChange} {...props} />
    </Provider>,
  )
  return { ...utils, onOpenChange }
}

async function pickType(user: ReturnType<typeof userEvent.setup>, name: string) {
  await user.click(screen.getByRole('combobox', { name: /type/i }))
  await user.click(await screen.findByRole('option', { name }))
  // Radix's Select schedules its close/position cleanup asynchronously —
  // see EnvironmentForm.test.tsx's identical pickCategory helper.
  await waitFor(() => expect(screen.queryByRole('listbox')).not.toBeInTheDocument())
}

describe('FeatureFlagForm', () => {
  it('creates a new flag, sending key/name/description/type', async () => {
    server.use(
      http.post('/api/v1/feature-flags', async ({ request }) => {
        const body = (await request.json()) as Record<string, unknown>
        expect(body).toEqual({
          key: 'new-checkout-flow',
          name: 'New checkout flow',
          description: 'Enables the redesign',
          type: 'String',
        })
        return HttpResponse.json(
          {
            id: 'flag-1',
            key: 'new-checkout-flow',
            name: 'New checkout flow',
            description: 'Enables the redesign',
            type: 'String',
            created_on: '2026-08-17T00:00:00Z',
            modified_on: '2026-08-17T00:00:00Z',
          },
          { status: 201 },
        )
      }),
    )
    const user = userEvent.setup()
    const { onOpenChange } = await renderForm()

    await user.type(screen.getByLabelText(/^key$/i), 'new-checkout-flow')
    await user.type(screen.getByLabelText(/^name$/i), 'New checkout flow')
    await user.type(screen.getByLabelText(/description/i), 'Enables the redesign')
    await pickType(user, 'String')
    await user.click(screen.getByRole('button', { name: /create flag/i }))

    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false))
  })

  it('shows validation errors for missing key and name', async () => {
    const user = userEvent.setup()
    await renderForm()

    await user.click(screen.getByRole('button', { name: /create flag/i }))
    expect(await screen.findByText(/key is required/i)).toBeInTheDocument()
    expect(screen.getByText(/name is required/i)).toBeInTheDocument()
  })

  it('rejects a key containing characters outside letters, numbers, hyphens, and underscores', async () => {
    const user = userEvent.setup()
    await renderForm()

    await user.type(screen.getByLabelText(/^key$/i), 'bad key!')
    await user.type(screen.getByLabelText(/^name$/i), 'Bad key flag')
    await user.click(screen.getByRole('button', { name: /create flag/i }))

    expect(
      await screen.findByText(/only letters, numbers, hyphens, and underscores are allowed/i),
    ).toBeInTheDocument()
  })

  it('accepts a key made only of letters, numbers, hyphens, and underscores', async () => {
    server.use(
      http.post('/api/v1/feature-flags', () =>
        HttpResponse.json(
          {
            id: 'flag-1',
            key: 'Good_Key-123',
            name: 'Good key flag',
            description: '',
            type: 'Boolean',
            created_on: '2026-08-17T00:00:00Z',
            modified_on: '2026-08-17T00:00:00Z',
          },
          { status: 201 },
        ),
      ),
    )
    const user = userEvent.setup()
    const { onOpenChange } = await renderForm()

    await user.type(screen.getByLabelText(/^key$/i), 'Good_Key-123')
    await user.type(screen.getByLabelText(/^name$/i), 'Good key flag')
    await user.click(screen.getByRole('button', { name: /create flag/i }))

    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false))
  })

  it('shows the key and type as read-only in edit mode and submits only name/description', async () => {
    const featureFlag: FeatureFlag = {
      id: 'flag-1',
      key: 'new-checkout-flow',
      name: 'New checkout flow',
      description: 'Old description',
      type: 'String',
      createdOn: '2026-08-17T00:00:00Z',
      modifiedOn: '2026-08-17T00:00:00Z',
    }
    server.use(
      http.patch('/api/v1/feature-flags/flag-1', async ({ request }) => {
        const body = (await request.json()) as Record<string, unknown>
        expect(body).toEqual({ name: 'Renamed flow', description: 'Old description' })
        return HttpResponse.json({
          id: 'flag-1',
          key: 'new-checkout-flow',
          name: 'Renamed flow',
          description: 'Old description',
          type: 'String',
          created_on: '2026-08-17T00:00:00Z',
          modified_on: '2026-08-17T00:00:00Z',
        })
      }),
    )
    const user = userEvent.setup()
    const { onOpenChange } = await renderForm({ featureFlag })

    // Key and type are plain read-only text, not form controls, in edit mode.
    expect(screen.queryByLabelText(/^key$/i)).not.toBeInTheDocument()
    expect(screen.queryByRole('combobox', { name: /type/i })).not.toBeInTheDocument()
    expect(screen.getByText('new-checkout-flow')).toBeInTheDocument()
    expect(screen.getByText('String')).toBeInTheDocument()

    const nameInput = await screen.findByDisplayValue('New checkout flow')
    await user.clear(nameInput)
    await user.type(nameInput, 'Renamed flow')
    await user.click(screen.getByRole('button', { name: /save changes/i }))

    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false))
  })

  it('shows "Saving..." on the submit button while the create request is in flight', async () => {
    let resolveRequest: () => void = () => {}
    const pending = new Promise<void>((resolve) => {
      resolveRequest = resolve
    })
    server.use(
      http.post('/api/v1/feature-flags', async () => {
        await pending
        return HttpResponse.json(
          {
            id: 'flag-1',
            key: 'new-flag',
            name: 'New flag',
            description: '',
            type: 'Boolean',
            created_on: '2026-08-17T00:00:00Z',
            modified_on: '2026-08-17T00:00:00Z',
          },
          { status: 201 },
        )
      }),
    )
    const user = userEvent.setup()
    await renderForm()

    await user.type(screen.getByLabelText(/^key$/i), 'new-flag')
    await user.type(screen.getByLabelText(/^name$/i), 'New flag')
    await user.click(screen.getByRole('button', { name: /create flag/i }))

    expect(await screen.findByRole('button', { name: /saving/i })).toBeDisabled()
    resolveRequest()
    await waitFor(() => expect(screen.queryByRole('button', { name: /saving/i })).not.toBeInTheDocument())
  })

  it('shows a server error message when creation fails', async () => {
    server.use(
      http.post('/api/v1/feature-flags', () =>
        HttpResponse.json(
          { error: { code: 'conflict', message: 'key already in use' } },
          { status: 409 },
        ),
      ),
    )
    const user = userEvent.setup()
    await renderForm()

    await user.type(screen.getByLabelText(/^key$/i), 'new-flag')
    await user.type(screen.getByLabelText(/^name$/i), 'New flag')
    await user.click(screen.getByRole('button', { name: /create flag/i }))

    expect(await screen.findByRole('alert')).toHaveTextContent(/key already in use/i)
  })

  it('closes the dialog when Cancel is clicked', async () => {
    const user = userEvent.setup()
    const { onOpenChange } = await renderForm()

    await user.click(screen.getByRole('button', { name: /cancel/i }))
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })

  it('renders nothing interactive when closed', async () => {
    await renderForm({ open: false })
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })
})
