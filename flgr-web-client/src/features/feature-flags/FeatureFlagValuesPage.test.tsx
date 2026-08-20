import { configureStore } from '@reduxjs/toolkit'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HttpResponse, http } from 'msw'
import { Provider } from 'react-redux'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { toast } from 'sonner'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { api } from '@/lib/api'
import { server } from '@/test/msw/server'
import { FeatureFlagValuesPage } from './FeatureFlagValuesPage'

function flagHandler(flag: { id: string; key: string; name: string; description: string | null; type: string }) {
  return http.get(`/api/v1/feature-flags/${flag.id}`, () =>
    HttpResponse.json({
      ...flag,
      created_on: '2026-08-17T00:00:00Z',
      modified_on: '2026-08-17T00:00:00Z',
    }),
  )
}

function environmentsHandler(environments: { id: string; name: string }[]) {
  return http.get('/api/v1/environments', () =>
    HttpResponse.json({
      data: environments.map((e) => ({
        id: e.id,
        name: e.name,
        description: '',
        environment_category_id: 'cat-1',
        created_on: '2026-08-17T00:00:00Z',
        modified_on: '2026-08-17T00:00:00Z',
      })),
      pagination: { page: 1, page_size: 200, total: environments.length },
    }),
  )
}

function valuesHandler(
  flagId: string,
  values: { id: string; environmentId: string; enabled: boolean; value: string | null }[],
) {
  return http.get(`/api/v1/feature-flags/${flagId}/values`, () =>
    HttpResponse.json({
      data: values.map((v) => ({
        id: v.id,
        feature_flag_id: flagId,
        environment_id: v.environmentId,
        enabled: v.enabled,
        value: v.value,
        created_on: '2026-08-17T00:00:00Z',
        modified_on: '2026-08-17T00:00:00Z',
      })),
    }),
  )
}

function renderPage(flagId = 'flag-1') {
  const store = configureStore({
    reducer: { [api.reducerPath]: api.reducer },
    middleware: (getDefaultMiddleware) => getDefaultMiddleware().concat(api.middleware),
  })
  return render(
    <Provider store={store}>
      <MemoryRouter initialEntries={[`/feature-flags/${flagId}/values`]}>
        <Routes>
          <Route path="/feature-flags/:id/values" element={<FeatureFlagValuesPage />} />
          <Route path="/feature-flags" element={<div>Feature Flags list</div>} />
        </Routes>
      </MemoryRouter>
    </Provider>,
  )
}

const booleanFlag = { id: 'flag-1', key: 'new-checkout-flow', name: 'New checkout flow', description: '', type: 'Boolean' }
const stringFlag = { id: 'flag-1', key: 'welcome-message', name: 'Welcome message', description: '', type: 'String' }

afterEach(() => {
  vi.restoreAllMocks()
})

describe('FeatureFlagValuesPage', () => {
  it('shows a loading state before the flag and its values load', () => {
    server.use(
      flagHandler(booleanFlag),
      environmentsHandler([{ id: 'env-1', name: 'Development' }]),
      valuesHandler('flag-1', []),
    )
    renderPage()

    expect(screen.getByText(/loading/i)).toBeInTheDocument()
  })

  it('shows an error state when the flag itself fails to load', async () => {
    server.use(
      http.get('/api/v1/feature-flags/flag-1', () => HttpResponse.error()),
      environmentsHandler([{ id: 'env-1', name: 'Development' }]),
      valuesHandler('flag-1', []),
    )
    renderPage()

    expect(await screen.findByText(/could not load this feature flag/i)).toBeInTheDocument()
  })

  it('shows an error message when the values list fails to load', async () => {
    server.use(
      flagHandler(booleanFlag),
      environmentsHandler([{ id: 'env-1', name: 'Development' }]),
      http.get('/api/v1/feature-flags/flag-1/values', () => HttpResponse.error()),
    )
    renderPage()

    expect(await screen.findByText(/new checkout flow/i)).toBeInTheDocument()
    expect(screen.getByText(/could not load this flag's values/i)).toBeInTheDocument()
  })

  it('shows a message when no environments exist yet', async () => {
    server.use(flagHandler(booleanFlag), environmentsHandler([]), valuesHandler('flag-1', []))
    renderPage()

    expect(await screen.findByText(/no environments exist yet/i)).toBeInTheDocument()
  })

  it('renders a configured environment as an "on" switch and an unconfigured one as "off", with no value input for a Boolean flag', async () => {
    server.use(
      flagHandler(booleanFlag),
      environmentsHandler([
        { id: 'env-1', name: 'Development' },
        { id: 'env-2', name: 'Production' },
      ]),
      valuesHandler('flag-1', [{ id: 'val-1', environmentId: 'env-1', enabled: true, value: null }]),
    )
    renderPage()

    await screen.findByText('Development')
    const switches = screen.getAllByRole('switch')
    expect(switches).toHaveLength(2)
    expect(switches[0]).toHaveAttribute('aria-checked', 'true')
    expect(switches[1]).toHaveAttribute('aria-checked', 'false')
    // No value input at all for a Boolean flag.
    expect(screen.queryByRole('textbox')).not.toBeInTheDocument()
  })

  it('shows a value input for a non-Boolean flag, pre-filled from the configured value or empty when unconfigured', async () => {
    server.use(
      flagHandler(stringFlag),
      environmentsHandler([
        { id: 'env-1', name: 'Development' },
        { id: 'env-2', name: 'Production' },
      ]),
      valuesHandler('flag-1', [{ id: 'val-1', environmentId: 'env-1', enabled: true, value: 'Hello' }]),
    )
    renderPage()

    await screen.findByText('Development')
    const inputs = screen.getAllByPlaceholderText(/value served when enabled/i)
    expect(inputs).toHaveLength(2)
    expect(inputs[0]).toHaveValue('Hello')
    expect(inputs[1]).toHaveValue('')
  })

  it('toggling a switch fires the upsert immediately', async () => {
    let patchCalls = 0
    let lastBody: Record<string, unknown> | undefined
    server.use(
      flagHandler(booleanFlag),
      environmentsHandler([{ id: 'env-1', name: 'Development' }]),
      valuesHandler('flag-1', []),
      http.patch('/api/v1/feature-flags/flag-1/values/env-1', async ({ request }) => {
        patchCalls += 1
        lastBody = (await request.json()) as Record<string, unknown>
        return HttpResponse.json({
          id: 'val-1',
          feature_flag_id: 'flag-1',
          environment_id: 'env-1',
          enabled: true,
          value: null,
          created_on: '2026-08-17T00:00:00Z',
          modified_on: '2026-08-17T00:00:00Z',
        })
      }),
    )
    const user = userEvent.setup()
    renderPage()

    await screen.findByText('Development')
    await user.click(screen.getByRole('switch'))

    await waitFor(() => expect(patchCalls).toBe(1))
    // Boolean flags omit `value` — see api.test.ts's identical assertion for upsertFeatureFlagValue.
    expect(lastBody).toEqual({ enabled: true })
  })

  it('shows an error toast when an upsert fails', async () => {
    const errorSpy = vi.spyOn(toast, 'error').mockImplementation(() => '')
    server.use(
      flagHandler(booleanFlag),
      environmentsHandler([{ id: 'env-1', name: 'Development' }]),
      valuesHandler('flag-1', []),
      http.patch('/api/v1/feature-flags/flag-1/values/env-1', () =>
        HttpResponse.json({ error: { code: 'forbidden', message: 'not allowed' } }, { status: 403 }),
      ),
    )
    const user = userEvent.setup()
    renderPage()

    await screen.findByText('Development')
    await user.click(screen.getByRole('switch'))

    await waitFor(() => expect(errorSpy).toHaveBeenCalledWith('not allowed'))
  })

  it('does not save on every keystroke, and does not save on blur when the value has not changed', async () => {
    let patchCalls = 0
    server.use(
      flagHandler(stringFlag),
      environmentsHandler([{ id: 'env-1', name: 'Development' }]),
      valuesHandler('flag-1', [{ id: 'val-1', environmentId: 'env-1', enabled: true, value: 'Hello' }]),
      http.patch('/api/v1/feature-flags/flag-1/values/env-1', async () => {
        patchCalls += 1
        return HttpResponse.json({
          id: 'val-1',
          feature_flag_id: 'flag-1',
          environment_id: 'env-1',
          enabled: true,
          value: 'Hello',
          created_on: '2026-08-17T00:00:00Z',
          modified_on: '2026-08-17T00:00:00Z',
        })
      }),
    )
    const user = userEvent.setup()
    renderPage()

    const input = await screen.findByDisplayValue('Hello')
    // Click into the field and click away without typing anything: no
    // change, so no save on blur.
    await user.click(input)
    await user.tab()
    expect(patchCalls).toBe(0)

    // Typing alone (before blur) must not fire a request per keystroke.
    await user.click(input)
    await user.type(input, ' world')
    expect(patchCalls).toBe(0)
  })

  it('saves the value input on blur once it has actually changed', async () => {
    let patchCalls = 0
    let lastBody: Record<string, unknown> | undefined
    server.use(
      flagHandler(stringFlag),
      environmentsHandler([{ id: 'env-1', name: 'Development' }]),
      valuesHandler('flag-1', [{ id: 'val-1', environmentId: 'env-1', enabled: true, value: 'Hello' }]),
      http.patch('/api/v1/feature-flags/flag-1/values/env-1', async ({ request }) => {
        patchCalls += 1
        lastBody = (await request.json()) as Record<string, unknown>
        return HttpResponse.json({
          id: 'val-1',
          feature_flag_id: 'flag-1',
          environment_id: 'env-1',
          enabled: true,
          value: 'Hello world',
          created_on: '2026-08-17T00:00:00Z',
          modified_on: '2026-08-17T00:00:00Z',
        })
      }),
    )
    const user = userEvent.setup()
    renderPage()

    const input = await screen.findByDisplayValue('Hello')
    await user.click(input)
    await user.type(input, ' world')
    await user.tab()

    await waitFor(() => expect(patchCalls).toBe(1))
    expect(lastBody).toEqual({ enabled: true, value: 'Hello world' })
  })

  // ValueRow's save() sends `next.value || null` for non-Boolean flags — an
  // empty string must go over the wire as null, not as "".
  it('saves an empty value as null when the value is cleared', async () => {
    let lastBody: Record<string, unknown> | undefined
    server.use(
      flagHandler(stringFlag),
      environmentsHandler([{ id: 'env-1', name: 'Development' }]),
      valuesHandler('flag-1', [{ id: 'val-1', environmentId: 'env-1', enabled: true, value: 'Hello' }]),
      http.patch('/api/v1/feature-flags/flag-1/values/env-1', async ({ request }) => {
        lastBody = (await request.json()) as Record<string, unknown>
        return HttpResponse.json({
          id: 'val-1',
          feature_flag_id: 'flag-1',
          environment_id: 'env-1',
          enabled: true,
          value: null,
          created_on: '2026-08-17T00:00:00Z',
          modified_on: '2026-08-17T00:00:00Z',
        })
      }),
    )
    const user = userEvent.setup()
    renderPage()

    const input = await screen.findByDisplayValue('Hello')
    await user.clear(input)
    await user.tab()

    await waitFor(() => expect(lastBody).toEqual({ enabled: true, value: null }))
  })

  // Regression coverage for the remount trick described in the
  // frontend-agent handoff: ValueRow is keyed on
  // `${environmentId}:${enabled}:${value}`, so after a save actually
  // changes the server value, the values-list refetch (invalidated by the
  // upsert) produces a new key and remounts the row, resetting its local
  // editing state from the freshly-fetched server value.
  it('reflects the newly-saved value after the row remounts on a real server-value change', async () => {
    let currentValue = 'initial'
    server.use(
      flagHandler(stringFlag),
      environmentsHandler([{ id: 'env-1', name: 'Development' }]),
      http.get('/api/v1/feature-flags/flag-1/values', () =>
        HttpResponse.json({
          data: [
            {
              id: 'val-1',
              feature_flag_id: 'flag-1',
              environment_id: 'env-1',
              enabled: true,
              value: currentValue,
              created_on: '2026-08-17T00:00:00Z',
              modified_on: '2026-08-17T00:00:00Z',
            },
          ],
        }),
      ),
      http.patch('/api/v1/feature-flags/flag-1/values/env-1', async ({ request }) => {
        const body = (await request.json()) as { enabled: boolean; value: string }
        currentValue = body.value
        return HttpResponse.json({
          id: 'val-1',
          feature_flag_id: 'flag-1',
          environment_id: 'env-1',
          enabled: body.enabled,
          value: currentValue,
          created_on: '2026-08-17T00:00:00Z',
          modified_on: '2026-08-17T00:00:00Z',
        })
      }),
    )
    const user = userEvent.setup()
    renderPage()

    const input = await screen.findByDisplayValue('initial')
    await user.clear(input)
    await user.type(input, 'updated')
    await user.tab()

    await waitFor(() => expect(screen.getByDisplayValue('updated')).toBeInTheDocument())
  })

  it('links back to the Feature Flags list from the header link and the Done button', async () => {
    server.use(
      flagHandler(booleanFlag),
      environmentsHandler([{ id: 'env-1', name: 'Development' }]),
      valuesHandler('flag-1', []),
    )
    renderPage()

    await screen.findByText('Development')
    expect(screen.getByRole('link', { name: /feature flags/i })).toHaveAttribute('href', '/feature-flags')
    expect(screen.getByRole('link', { name: /^done$/i })).toHaveAttribute('href', '/feature-flags')
  })

  it("shows the flag's non-Boolean type copy mentioning the served value", async () => {
    server.use(
      flagHandler(stringFlag),
      environmentsHandler([{ id: 'env-1', name: 'Development' }]),
      valuesHandler('flag-1', []),
    )
    renderPage()

    expect(await screen.findByText(/and the value it serves when it is/i)).toBeInTheDocument()
  })
})
