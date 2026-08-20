import { configureStore } from '@reduxjs/toolkit'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HttpResponse, http } from 'msw'
import { Provider } from 'react-redux'
import { describe, expect, it, vi } from 'vitest'
import { api } from '@/lib/api'
import { server } from '@/test/msw/server'
import { environmentsApi } from './api'
import { EnvironmentForm } from './EnvironmentForm'
import type { Environment } from './types'

// GET /environment-categories returns { data: [...] } (see api.ts), never a
// bare array.
const categoriesHandler = http.get('/api/v1/environment-categories', () =>
  HttpResponse.json({
    data: [
      { id: 'cat-1', name: 'Development', is_system: true },
      { id: 'cat-2', name: 'Staging', is_system: true },
    ],
  }),
)

async function renderForm(props: Partial<React.ComponentProps<typeof EnvironmentForm>> = {}) {
  const store = configureStore({
    reducer: { [api.reducerPath]: api.reducer },
    middleware: (getDefaultMiddleware) => getDefaultMiddleware().concat(api.middleware),
  })
  // Pre-resolve the categories query in the store before the form ever
  // mounts, purely so every other test here can assert against a
  // fully-loaded category list without an extra findBy/waitFor round
  // trip. (This used to be load-bearing rather than a convenience — see
  // the regression test at the bottom of this file for the infinite-loop
  // bug that made mounting open with a still-pending query hang the JS
  // thread outright, fixed in EnvironmentForm.tsx via the module-level
  // EMPTY_CATEGORIES constant.)
  await store.dispatch(environmentsApi.endpoints.getEnvironmentCategories.initiate())
  const onOpenChange = vi.fn()
  const utils = render(
    <Provider store={store}>
      <EnvironmentForm open onOpenChange={onOpenChange} {...props} />
    </Provider>,
  )
  return { ...utils, onOpenChange }
}

async function pickCategory(user: ReturnType<typeof userEvent.setup>, name: string) {
  await user.click(screen.getByRole('combobox', { name: /category/i }))
  await user.click(await screen.findByRole('option', { name }))
  // Radix's Select schedules its close/position cleanup asynchronously;
  // waiting for the listbox to actually unmount keeps every state update
  // it triggers inside this helper's `act()` scope instead of leaking
  // into whatever assertion runs next.
  await waitFor(() => expect(screen.queryByRole('listbox')).not.toBeInTheDocument())
}

describe('EnvironmentForm', () => {
  it('creates a new environment, sending an existing category’s name as category_name', async () => {
    server.use(categoriesHandler)
    server.use(
      http.post('/api/v1/environments', async ({ request }) => {
        const body = (await request.json()) as Record<string, unknown>
        expect(body).toEqual({
          name: 'app-dev',
          description: 'Dev env',
          category_name: 'Development',
        })
        return HttpResponse.json(
          {
            id: 'env-1',
            name: 'app-dev',
            description: 'Dev env',
            environment_category_id: 'cat-1',
            created_on: '2026-08-17T00:00:00Z',
            modified_on: '2026-08-17T00:00:00Z',
          },
          { status: 201 },
        )
      }),
    )
    const user = userEvent.setup()
    const { onOpenChange } = await renderForm()

    await user.type(screen.getByLabelText(/^name$/i), 'app-dev')
    await user.type(screen.getByLabelText(/description/i), 'Dev env')
    await pickCategory(user, 'Development')
    await user.click(screen.getByRole('button', { name: /create environment/i }))

    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false))
  })

  it('sends a typed "Other" category name directly as category_name, with no separate category-creation request', async () => {
    server.use(categoriesHandler)
    server.use(
      http.post('/api/v1/environments', async ({ request }) => {
        const body = (await request.json()) as Record<string, unknown>
        expect(body).toEqual({
          name: 'app-qa',
          description: '',
          category_name: 'QA',
        })
        return HttpResponse.json(
          {
            id: 'env-2',
            name: 'app-qa',
            description: '',
            environment_category_id: 'cat-3',
            created_on: '2026-08-17T00:00:00Z',
            modified_on: '2026-08-17T00:00:00Z',
          },
          { status: 201 },
        )
      }),
    )
    // Deliberately NO handler for POST /environment-categories: flgr-server
    // only ever exposes GET /environment-categories (see
    // flgr-server/internal/api/handler/environment_category.go) — a new
    // category is resolved/created server-side as a side effect of the
    // environment write. src/test/setup.ts's onUnhandledRequest: 'error'
    // means this test fails loudly if the form ever tries to call a
    // nonexistent create-category endpoint.
    const user = userEvent.setup()
    const { onOpenChange } = await renderForm()

    await user.type(screen.getByLabelText(/^name$/i), 'app-qa')
    await pickCategory(user, 'Other...')
    await user.type(screen.getByLabelText(/new category name/i), 'QA')
    await user.click(screen.getByRole('button', { name: /create environment/i }))

    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false))
  })

  it('shows validation errors, including a missing new category name', async () => {
    server.use(categoriesHandler)
    const user = userEvent.setup()
    await renderForm()

    await user.click(screen.getByRole('button', { name: /create environment/i }))
    expect(await screen.findByText(/name is required/i)).toBeInTheDocument()
    expect(screen.getByText(/category is required/i)).toBeInTheDocument()

    await pickCategory(user, 'Other...')
    await user.click(screen.getByRole('button', { name: /create environment/i }))
    expect(await screen.findByText(/enter a name for the new category/i)).toBeInTheDocument()
  })

  it('pre-fills the category from the environment’s environmentCategoryId and submits an update with its name', async () => {
    const environment: Environment = {
      id: 'env-1',
      name: 'app-dev',
      description: 'Dev env',
      environmentCategoryId: 'cat-1',
      createdOn: '2026-08-17T00:00:00Z',
      modifiedOn: '2026-08-17T00:00:00Z',
    }
    server.use(categoriesHandler)
    server.use(
      http.patch('/api/v1/environments/env-1', async ({ request }) => {
        const body = (await request.json()) as Record<string, unknown>
        // cat-1 resolves to "Development" via the categories list — proves
        // the pre-fill round-trips through the id -> name lookup and back
        // out as a name on submit, without the user touching the category
        // field at all.
        expect(body).toEqual({
          name: 'app-dev-renamed',
          description: 'Dev env',
          category_name: 'Development',
        })
        return HttpResponse.json({
          id: 'env-1',
          name: 'app-dev-renamed',
          description: 'Dev env',
          environment_category_id: 'cat-1',
          created_on: '2026-08-17T00:00:00Z',
          modified_on: '2026-08-17T00:00:00Z',
        })
      }),
    )
    const user = userEvent.setup()
    const { onOpenChange } = await renderForm({ environment })

    expect(await screen.findByDisplayValue('app-dev')).toBeInTheDocument()
    const nameInput = screen.getByLabelText(/^name$/i)
    await user.clear(nameInput)
    await user.type(nameInput, 'app-dev-renamed')
    await user.click(screen.getByRole('button', { name: /save changes/i }))

    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false))
  })

  it('leaves the category blank when editing an environment whose category id matches none of the known categories', async () => {
    const environment: Environment = {
      id: 'env-1',
      name: 'app-orphan',
      description: '',
      environmentCategoryId: 'cat-does-not-exist',
      createdOn: '2026-08-17T00:00:00Z',
      modifiedOn: '2026-08-17T00:00:00Z',
    }
    server.use(categoriesHandler)
    await renderForm({ environment })

    expect(await screen.findByDisplayValue('app-orphan')).toBeInTheDocument()
    expect(screen.getByRole('combobox', { name: /category/i })).toHaveTextContent(
      'Select a category',
    )
  })

  it('shows "Saving..." on the submit button while the create request is in flight', async () => {
    let resolveRequest: () => void = () => {}
    const pending = new Promise<void>((resolve) => {
      resolveRequest = resolve
    })
    server.use(categoriesHandler)
    server.use(
      http.post('/api/v1/environments', async () => {
        await pending
        return HttpResponse.json(
          {
            id: 'env-1',
            name: 'app-dev',
            description: '',
            environment_category_id: 'cat-1',
            created_on: '2026-08-17T00:00:00Z',
            modified_on: '2026-08-17T00:00:00Z',
          },
          { status: 201 },
        )
      }),
    )
    const user = userEvent.setup()
    await renderForm()

    await user.type(screen.getByLabelText(/^name$/i), 'app-dev')
    await pickCategory(user, 'Development')
    await user.click(screen.getByRole('button', { name: /create environment/i }))

    expect(await screen.findByRole('button', { name: /saving/i })).toBeDisabled()
    resolveRequest()
    await waitFor(() => expect(screen.queryByRole('button', { name: /saving/i })).not.toBeInTheDocument())
  })

  it('shows a server error message when creation fails', async () => {
    server.use(categoriesHandler)
    server.use(
      http.post('/api/v1/environments', () =>
        HttpResponse.json(
          { error: { code: 'conflict', message: 'name already in use' } },
          { status: 409 },
        ),
      ),
    )
    const user = userEvent.setup()
    await renderForm()

    await user.type(screen.getByLabelText(/^name$/i), 'app-dev')
    await pickCategory(user, 'Development')
    await user.click(screen.getByRole('button', { name: /create environment/i }))

    expect(await screen.findByRole('alert')).toHaveTextContent(/name already in use/i)
  })

  it('closes the dialog when Cancel is clicked', async () => {
    server.use(categoriesHandler)
    const user = userEvent.setup()
    const { onOpenChange } = await renderForm()

    await user.click(screen.getByRole('button', { name: /cancel/i }))
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })

  it('renders nothing interactive when closed', async () => {
    server.use(categoriesHandler)
    await renderForm({ open: false })
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  // Regression test for a real infinite-loop bug that used to live here:
  // EnvironmentForm.tsx had `const { data: categories = [] } =
  // useGetEnvironmentCategoriesQuery()` feeding a useEffect that depended
  // on `categories` and unconditionally called `form.reset(...)`. While
  // the query was pending, `data` was `undefined`, so the `?? []` fallback
  // allocated a *new* array literal every render; that new reference
  // re-triggered the effect -> form.reset -> re-render -> new `[]` ->
  // forever, synchronously, inside React's `act()` flush — never yielding
  // back to the event loop, so neither React's "Maximum update depth"
  // guard nor Vitest's test timeout could save it. Confirmed hanging the
  // test process indefinitely (150s wall-clock kill) before the fix.
  // Fixed in EnvironmentForm.tsx with a stable module-level
  // EMPTY_CATEGORIES constant, so this deliberately does NOT use
  // renderForm()'s pre-resolved-cache convenience above — the whole point
  // is to mount the dialog open while the categories query is still in
  // flight, which used to hang and must now resolve normally.
  it('mounts open and resolves normally while the categories query is still pending', async () => {
    server.use(
      http.get('/api/v1/environment-categories', async () => {
        await new Promise((resolve) => setTimeout(resolve, 10))
        return HttpResponse.json({
          data: [{ id: 'cat-1', name: 'Development', is_system: true }],
        })
      }),
    )
    const store = configureStore({
      reducer: { [api.reducerPath]: api.reducer },
      middleware: (getDefaultMiddleware) => getDefaultMiddleware().concat(api.middleware),
    })
    const user = userEvent.setup()
    render(
      <Provider store={store}>
        <EnvironmentForm open onOpenChange={() => {}} />
      </Provider>,
    )

    expect(screen.getByRole('dialog')).toBeInTheDocument()
    await user.click(screen.getByRole('combobox', { name: /category/i }))
    expect(await screen.findByRole('option', { name: 'Development' })).toBeInTheDocument()
  })
})
