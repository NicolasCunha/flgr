import { Navigate, Outlet } from 'react-router-dom'
import { useGetSetupStatusQuery } from '@/features/setup/api'

// Redirects to the first-run setup wizard when no User exists yet, per
// docs/business/requirements/0002-user-management.md — there's no public
// signup, so a fresh install needs this one bootstrap path.
export function RootGuard() {
  const { data, isLoading, isError } = useGetSetupStatusQuery()

  if (isLoading) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-background text-foreground">
        <p className="text-sm text-muted-foreground">Loading…</p>
      </main>
    )
  }

  if (isError) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-background text-foreground">
        <p className="text-sm text-destructive">Backend unreachable</p>
      </main>
    )
  }

  if (data?.needed) {
    return <Navigate to="/setup" replace />
  }

  return <Outlet />
}
