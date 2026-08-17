import { createBrowserRouter } from 'react-router-dom'
import { HomePage } from '@/app/HomePage'

// Protected routes (redirecting to /login when unauthenticated, per
// docs/architecture/adr/0008-frontend-routing-and-ui-component-library.md)
// are added here once authentication is implemented.
export const router = createBrowserRouter([
  {
    path: '/',
    element: <HomePage />,
  },
])
