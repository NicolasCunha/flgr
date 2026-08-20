import { createBrowserRouter } from 'react-router-dom'
import { AppLayout } from '@/app/AppLayout'
import { HomePage } from '@/app/HomePage'
import { RequireAuth } from '@/app/RequireAuth'
import { RootGuard } from '@/app/RootGuard'
import { LoginPage } from '@/features/auth/LoginPage'
import { EnvironmentsPage } from '@/features/environments/EnvironmentsPage'
import { FeatureFlagsPage } from '@/features/feature-flags/FeatureFlagsPage'
import { FeatureFlagValuesPage } from '@/features/feature-flags/FeatureFlagValuesPage'
import { ServiceKeysPage } from '@/features/service-keys/ServiceKeysPage'
import { SetupWizardPage } from '@/features/setup/SetupWizardPage'

// RootGuard redirects to /setup on a fresh install (no User exists yet);
// /setup redirects back once it isn't needed anymore. /login self-guards
// the same way (see LoginPage.tsx) rather than being nested under
// RootGuard here — nesting it made RootGuard persist across the
// RequireAuth → /login redirect instead of remounting, which broke
// App.test.tsx's assumption that leaving "/" fully unmounts it. Below
// RootGuard, RequireAuth (per docs/architecture/adr/0008-frontend-routing-and-ui-component-library.md)
// redirects to /login when there's no valid session, and AppLayout is the
// authenticated shell (nav + logout) every protected screen renders
// inside of.
export const router = createBrowserRouter([
  {
    path: '/',
    element: <RootGuard />,
    children: [
      {
        element: <RequireAuth />,
        children: [
          {
            element: <AppLayout />,
            children: [
              { index: true, element: <HomePage /> },
              { path: 'environments', element: <EnvironmentsPage /> },
              { path: 'feature-flags', element: <FeatureFlagsPage /> },
              { path: 'feature-flags/:id/values', element: <FeatureFlagValuesPage /> },
              { path: 'service-keys', element: <ServiceKeysPage /> },
            ],
          },
        ],
      },
    ],
  },
  {
    path: '/login',
    element: <LoginPage />,
  },
  {
    path: '/setup',
    element: <SetupWizardPage />,
  },
])
