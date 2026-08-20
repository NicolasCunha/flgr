import { cleanup, render, screen } from '@testing-library/react'
import { HttpResponse, http } from 'msw'
import { afterEach, describe, expect, it } from 'vitest'
import { store } from '@/app/store'
import { router } from '@/app/router'
import { api } from '@/lib/api'
import { server } from '@/test/msw/server'
import { App } from './App'

// App.tsx renders against the app's real singletons — `store` and
// `router` (from src/app/store.ts and src/app/router.tsx) — rather than
// a fresh instance per test like every other test file in this project.
// Both leak state across tests here:
//  - `router` was built once via createBrowserRouter, which owns actual
//    browser history; the first test's redirect permanently navigates it
//    away from "/", so a second render with the same router never
//    revisits RootGuard at all.
//  - `store`'s RTK Query cache persists across renders too: even after
//    navigating back to "/", a fresh RootGuard mount within the default
//    60s keepUnusedDataFor window would be served the first test's
//    (now wrong) cached GET /setup result instead of refetching.
// An explicit cleanup() (rather than relying on RTL's own auto-registered
// afterEach, whose relative ordering against this hook isn't guaranteed)
// is required here too: LoginPage now runs its own GET /setup check
// (self-guarded the same way SetupWizardPage is — see LoginPage.tsx), so
// if its tree is still mounted when resetApiState() fires, it reacts to
// the wiped cache by refetching and redirecting on its own, racing the
// deliberate router.navigate('/') below and leaving the router back on
// /login instead of "/" for the next test.
afterEach(async () => {
  cleanup()
  store.dispatch(api.util.resetApiState())
  await router.navigate('/')
})

describe('App', () => {
  it('redirects to /login when no session exists (default MSW handlers)', async () => {
    render(<App />)
    // CardTitle (src/components/ui/card.tsx) renders a plain <div>, not a
    // heading element, so this is a text query rather than a role query.
    expect(await screen.findByText(/sign in to flgr/i)).toBeInTheDocument()
  })

  it('redirects to /setup when no administrator account exists yet', async () => {
    server.use(http.get('/api/v1/setup', () => HttpResponse.json({ needed: true })))
    render(<App />)
    expect(await screen.findByText(/welcome to flgr/i)).toBeInTheDocument()
  })
})
