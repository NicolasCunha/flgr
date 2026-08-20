import { configureStore } from '@reduxjs/toolkit'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HttpResponse, http } from 'msw'
import { Provider } from 'react-redux'
import { toast } from 'sonner'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { api } from '@/lib/api'
import { server } from '@/test/msw/server'
import { EnvironmentsPage } from './EnvironmentsPage'

const categoriesHandler = http.get('/api/v1/environment-categories', () =>
  HttpResponse.json({
    data: [
      { id: 'cat-1', name: 'Development', is_system: true },
      { id: 'cat-2', name: 'Staging', is_system: true },
    ],
  }),
)

function environmentsHandler(
  environments: {
    id: string
    name: string
    description: string | null
    environment_category_id: string
  }[],
  total = environments.length,
) {
  return http.get('/api/v1/environments', ({ request }) => {
    const url = new URL(request.url)
    return HttpResponse.json({
      data: environments,
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
      <EnvironmentsPage />
    </Provider>,
  )
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('EnvironmentsPage', () => {
  it('shows a loading state, then lists environments with their resolved category name', async () => {
    server.use(
      categoriesHandler,
      environmentsHandler([
        {
          id: 'env-1',
          name: 'app-dev',
          description: 'Development environment',
          environment_category_id: 'cat-1',
        },
      ]),
    )
    renderPage()

    expect(screen.getByText(/loading/i)).toBeInTheDocument()
    expect(await screen.findByText('app-dev')).toBeInTheDocument()
    expect(screen.getByText('Development')).toBeInTheDocument()
    expect(screen.getByText('Development environment')).toBeInTheDocument()
  })

  it('falls back to an em dash for an unresolved category id and an empty description', async () => {
    server.use(
      categoriesHandler,
      environmentsHandler([
        {
          id: 'env-1',
          name: 'app-orphan',
          description: null,
          environment_category_id: 'cat-does-not-exist',
        },
      ]),
    )
    renderPage()

    expect(await screen.findByText('app-orphan')).toBeInTheDocument()
    const row = screen.getByText('app-orphan').closest('tr')
    expect(row).not.toBeNull()
    // Two "—" cells: the unresolved category and the empty description.
    expect(within(row as HTMLElement).getAllByText('—')).toHaveLength(2)
  })

  it('shows an error state when the environments list fails to load', async () => {
    server.use(categoriesHandler, http.get('/api/v1/environments', () => HttpResponse.error()))
    renderPage()

    expect(await screen.findByText(/could not load environments/i)).toBeInTheDocument()
  })

  it('shows the empty-list message when there are no environments', async () => {
    server.use(categoriesHandler, environmentsHandler([], 0))
    renderPage()

    expect(await screen.findByText(/no environments yet/i)).toBeInTheDocument()
  })

  it('changes page when Next/Previous are clicked', async () => {
    const user = userEvent.setup()
    server.use(
      categoriesHandler,
      environmentsHandler(
        [
          {
            id: 'env-1',
            name: 'app-dev',
            description: '',
            environment_category_id: 'cat-1',
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

  it('opens the create form from "New environment" and the edit form from a row\'s Edit button', async () => {
    const user = userEvent.setup()
    server.use(
      categoriesHandler,
      environmentsHandler([
        {
          id: 'env-1',
          name: 'app-dev',
          description: 'Dev env',
          environment_category_id: 'cat-1',
        },
      ]),
    )
    renderPage()

    await user.click(screen.getByRole('button', { name: /new environment/i }))
    expect(await screen.findByRole('heading', { name: /new environment/i })).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: /cancel/i }))
    await waitFor(() =>
      expect(screen.queryByRole('heading', { name: /new environment/i })).not.toBeInTheDocument(),
    )

    await screen.findByText('app-dev')
    await user.click(screen.getByRole('button', { name: /edit/i }))
    expect(await screen.findByRole('heading', { name: /edit environment/i })).toBeInTheDocument()
    expect(screen.getByDisplayValue('app-dev')).toBeInTheDocument()
  })

  it('removes an environment after confirming, and shows a success toast', async () => {
    const successSpy = vi.spyOn(toast, 'success').mockImplementation(() => '')
    const user = userEvent.setup()
    server.use(
      categoriesHandler,
      environmentsHandler([
        {
          id: 'env-1',
          name: 'app-dev',
          description: '',
          environment_category_id: 'cat-1',
        },
      ]),
      http.delete('/api/v1/environments/env-1', () => new HttpResponse(null, { status: 200 })),
    )
    renderPage()

    await screen.findByText('app-dev')
    await user.click(screen.getByRole('button', { name: /remove/i }))
    expect(await screen.findByRole('heading', { name: /remove environment/i })).toBeInTheDocument()
    expect(screen.getByText(/remove "app-dev"\?/i)).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Remove' }))

    await waitFor(() =>
      expect(screen.queryByRole('heading', { name: /remove environment/i })).not.toBeInTheDocument(),
    )
    expect(successSpy).toHaveBeenCalledWith('Environment "app-dev" removed.')
  })

  it('cancels the remove confirmation without deleting', async () => {
    const user = userEvent.setup()
    server.use(
      categoriesHandler,
      environmentsHandler([
        {
          id: 'env-1',
          name: 'app-dev',
          description: '',
          environment_category_id: 'cat-1',
        },
      ]),
    )
    renderPage()

    await screen.findByText('app-dev')
    await user.click(screen.getByRole('button', { name: /remove/i }))
    expect(await screen.findByRole('heading', { name: /remove environment/i })).toBeInTheDocument()

    // The dialog's own "Cancel" button, not the row's "Remove" button.
    const dialog = screen.getByRole('dialog')
    await user.click(within(dialog).getByRole('button', { name: /cancel/i }))

    await waitFor(() =>
      expect(screen.queryByRole('heading', { name: /remove environment/i })).not.toBeInTheDocument(),
    )
  })

  it('shows an error toast when deletion fails', async () => {
    const errorSpy = vi.spyOn(toast, 'error').mockImplementation(() => '')
    const user = userEvent.setup()
    server.use(
      categoriesHandler,
      environmentsHandler([
        {
          id: 'env-1',
          name: 'app-dev',
          description: '',
          environment_category_id: 'cat-1',
        },
      ]),
      http.delete('/api/v1/environments/env-1', () =>
        HttpResponse.json(
          { error: { code: 'conflict', message: 'still referenced by a feature flag value' } },
          { status: 409 },
        ),
      ),
    )
    renderPage()

    await screen.findByText('app-dev')
    await user.click(screen.getByRole('button', { name: /remove/i }))
    await screen.findByRole('heading', { name: /remove environment/i })
    await user.click(screen.getByRole('button', { name: 'Remove' }))

    await waitFor(() =>
      expect(errorSpy).toHaveBeenCalledWith('still referenced by a feature flag value'),
    )
    // The confirm dialog stays open on failure, unlike the success path.
    expect(screen.getByRole('heading', { name: /remove environment/i })).toBeInTheDocument()
  })
})
