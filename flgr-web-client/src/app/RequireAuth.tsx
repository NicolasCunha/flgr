import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { LoadingScreen } from '@/components/LoadingScreen'
import { useGetMeQuery } from '@/features/auth/api'

// Protects a route subtree behind an authenticated session, per
// docs/architecture/adr/0008-frontend-routing-and-ui-component-library.md:
// checks GET /me (the session cookie is HttpOnly, so this is the only way
// to know from JS whether the session is still valid) and redirects to
// /login, remembering where the user was headed, on failure.
export function RequireAuth() {
  const location = useLocation()
  const { isLoading, isError } = useGetMeQuery()

  if (isLoading) {
    return <LoadingScreen />
  }

  if (isError) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />
  }

  return <Outlet />
}
