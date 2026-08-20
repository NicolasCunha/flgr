import { configureStore } from '@reduxjs/toolkit'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HttpResponse, http } from 'msw'
import { Provider } from 'react-redux'
import { MemoryRouter } from 'react-router-dom'
import { toast } from 'sonner'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { api } from '@/lib/api'
import { server } from '@/test/msw/server'
import { FeatureFlagsPage } from './FeatureFlagsPage'

function featureFlagsHandler(
  flags: {
    id: string
    key: string
    name: string
    description: string | null
    type: string
  }[],
  total = flags.length,
) {
  return http.get('/api/v1/feature-flags', ({ request }) => {
    const url = new URL(request.url)
    return HttpResponse.json({
      data: flags,
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
      <MemoryRouter>
        <FeatureFlagsPage />
      </MemoryRouter>
    </Provider>,
  )
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('FeatureFlagsPage', () => {
  it('shows a loading state, then lists feature flags', async () => {
    server.use(
      featureFlagsHandler([
        {
          id: 'flag-1',
          key: 'new-checkout-flow',
          name: 'New checkout flow',
          description: 'Enables the redesigned checkout',
          type: 'Boolean',
        },
      ]),
    )
    renderPage()

    expect(screen.getByText(/loading/i)).toBeInTheDocument()
    expect(await screen.findByText('new-checkout-flow')).toBeInTheDocument()
    expect(screen.getByText('New checkout flow')).toBeInTheDocument()
    expect(screen.getByText('Boolean')).toBeInTheDocument()
    expect(screen.getByText('Enables the redesigned checkout')).toBeInTheDocument()
  })

  it('falls back to an em dash for an empty description', async () => {
    server.use(
      featureFlagsHandler([
        { id: 'flag-1', key: 'no-description', name: 'No description', description: null, type: 'String' },
      ]),
    )
    renderPage()

    expect(await screen.findByText('no-description')).toBeInTheDocument()
    const row = screen.getByText('no-description').closest('tr')
    expect(row).not.toBeNull()
    expect(within(row as HTMLElement).getByText('—')).toBeInTheDocument()
  })

  it('shows an error state when the list fails to load', async () => {
    server.use(http.get('/api/v1/feature-flags', () => HttpResponse.error()))
    renderPage()

    expect(await screen.findByText(/could not load feature flags/i)).toBeInTheDocument()
  })

  it('shows the empty-list message when there are no feature flags', async () => {
    server.use(featureFlagsHandler([], 0))
    renderPage()

    expect(await screen.findByText(/no feature flags yet/i)).toBeInTheDocument()
  })

  it('links each row\'s "Values" action to that flag\'s values screen', async () => {
    server.use(
      featureFlagsHandler([
        {
          id: 'flag-1',
          key: 'new-checkout-flow',
          name: 'New checkout flow',
          description: '',
          type: 'Boolean',
        },
      ]),
    )
    renderPage()

    await screen.findByText('new-checkout-flow')
    expect(screen.getByRole('link', { name: 'Values' })).toHaveAttribute(
      'href',
      '/feature-flags/flag-1/values',
    )
  })

  it('opens the create form from "New flag" and the edit form from a row\'s Edit button', async () => {
    const user = userEvent.setup()
    server.use(
      featureFlagsHandler([
        {
          id: 'flag-1',
          key: 'new-checkout-flow',
          name: 'New checkout flow',
          description: 'Dev flag',
          type: 'Boolean',
        },
      ]),
    )
    renderPage()

    await user.click(screen.getByRole('button', { name: /new flag/i }))
    expect(await screen.findByRole('heading', { name: /new feature flag/i })).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: /cancel/i }))
    await waitFor(() =>
      expect(screen.queryByRole('heading', { name: /new feature flag/i })).not.toBeInTheDocument(),
    )

    await screen.findByText('new-checkout-flow')
    await user.click(screen.getByRole('button', { name: /edit/i }))
    expect(await screen.findByRole('heading', { name: /edit feature flag/i })).toBeInTheDocument()
    expect(screen.getByDisplayValue('New checkout flow')).toBeInTheDocument()
    // Key/type render read-only in edit mode — this asserts the page wires
    // the row's flag into the form (full field-level read-only behavior is
    // covered by FeatureFlagForm.test.tsx). Scoped to the dialog since the
    // table row behind it also renders the same key text.
    expect(within(screen.getByRole('dialog')).getByText('new-checkout-flow')).toBeInTheDocument()
  })

  it('removes a feature flag after confirming, and shows a success toast', async () => {
    const successSpy = vi.spyOn(toast, 'success').mockImplementation(() => '')
    const user = userEvent.setup()
    server.use(
      featureFlagsHandler([
        {
          id: 'flag-1',
          key: 'new-checkout-flow',
          name: 'New checkout flow',
          description: '',
          type: 'Boolean',
        },
      ]),
      http.delete('/api/v1/feature-flags/flag-1', () => new HttpResponse(null, { status: 200 })),
    )
    renderPage()

    await screen.findByText('new-checkout-flow')
    await user.click(screen.getByRole('button', { name: /remove/i }))
    expect(await screen.findByRole('heading', { name: /remove feature flag/i })).toBeInTheDocument()
    expect(screen.getByText(/remove "new checkout flow"\?/i)).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Remove' }))

    await waitFor(() =>
      expect(screen.queryByRole('heading', { name: /remove feature flag/i })).not.toBeInTheDocument(),
    )
    expect(successSpy).toHaveBeenCalledWith('Feature flag "New checkout flow" removed.')
  })

  it('cancels the remove confirmation without deleting', async () => {
    const user = userEvent.setup()
    server.use(
      featureFlagsHandler([
        {
          id: 'flag-1',
          key: 'new-checkout-flow',
          name: 'New checkout flow',
          description: '',
          type: 'Boolean',
        },
      ]),
    )
    renderPage()

    await screen.findByText('new-checkout-flow')
    await user.click(screen.getByRole('button', { name: /remove/i }))
    expect(await screen.findByRole('heading', { name: /remove feature flag/i })).toBeInTheDocument()

    const dialog = screen.getByRole('dialog')
    await user.click(within(dialog).getByRole('button', { name: /cancel/i }))

    await waitFor(() =>
      expect(screen.queryByRole('heading', { name: /remove feature flag/i })).not.toBeInTheDocument(),
    )
  })

  it('shows an error toast when deletion fails', async () => {
    const errorSpy = vi.spyOn(toast, 'error').mockImplementation(() => '')
    const user = userEvent.setup()
    server.use(
      featureFlagsHandler([
        {
          id: 'flag-1',
          key: 'new-checkout-flow',
          name: 'New checkout flow',
          description: '',
          type: 'Boolean',
        },
      ]),
      http.delete('/api/v1/feature-flags/flag-1', () =>
        HttpResponse.json(
          { error: { code: 'conflict', message: 'still referenced elsewhere' } },
          { status: 409 },
        ),
      ),
    )
    renderPage()

    await screen.findByText('new-checkout-flow')
    await user.click(screen.getByRole('button', { name: /remove/i }))
    await screen.findByRole('heading', { name: /remove feature flag/i })
    await user.click(screen.getByRole('button', { name: 'Remove' }))

    await waitFor(() => expect(errorSpy).toHaveBeenCalledWith('still referenced elsewhere'))
    expect(screen.getByRole('heading', { name: /remove feature flag/i })).toBeInTheDocument()
  })
})
