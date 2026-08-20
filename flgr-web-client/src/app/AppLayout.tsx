import { LogOut } from 'lucide-react'
import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { ThemeToggle } from '@/components/ThemeToggle'
import { Button } from '@/components/ui/button'
import { useGetMeQuery, useLogoutMutation } from '@/features/auth/api'
import { api } from '@/lib/api'
import { cn } from '@/lib/utils'
import { useAppDispatch } from './hooks'

const navItems = [
  { to: '/', label: 'Home', end: true },
  { to: '/environments', label: 'Environments', end: false },
  { to: '/feature-flags', label: 'Feature Flags', end: false },
  { to: '/service-keys', label: 'Service Keys', end: false },
]

// Minimal authenticated shell: a top nav plus whatever screen is routed
// into the <Outlet />, so screens aren't floating standalone. Kept
// intentionally simple — this is scaffolding for future screens, not a
// full app-shell design pass.
export function AppLayout() {
  const navigate = useNavigate()
  const dispatch = useAppDispatch()
  const { data: me } = useGetMeQuery()
  const [logout, { isLoading: isLoggingOut }] = useLogoutMutation()

  async function handleLogout() {
    try {
      await logout().unwrap()
    } catch {
      // Still navigate to /login below even if the request itself failed
      // (e.g. a 500) — the user's intent was to log out, and resetting
      // local API state gets them there regardless of what the server did.
    } finally {
      dispatch(api.util.resetApiState())
      void navigate('/login', { replace: true })
    }
  }

  return (
    <div className="min-h-screen bg-background text-foreground">
      <header className="border-b border-border">
        <div className="mx-auto flex max-w-6xl items-center justify-between px-4 py-3">
          <div className="flex items-center gap-6">
            <span className="text-lg font-semibold">flgr</span>
            <nav className="flex items-center gap-4">
              {navItems.map((item) => (
                <NavLink
                  key={item.to}
                  to={item.to}
                  end={item.end}
                  className={({ isActive }) =>
                    cn(
                      'text-sm text-muted-foreground transition-colors hover:text-foreground',
                      isActive && 'font-medium text-foreground',
                    )
                  }
                >
                  {item.label}
                </NavLink>
              ))}
            </nav>
          </div>
          <div className="flex items-center gap-3">
            <ThemeToggle />
            {me && (
              <span className="text-sm text-muted-foreground">
                {me.firstName} {me.lastName}
              </span>
            )}
            <Button
              variant="outline"
              size="sm"
              onClick={() => void handleLogout()}
              disabled={isLoggingOut}
            >
              <LogOut className="size-4" />
              Log out
            </Button>
          </div>
        </div>
      </header>
      <main className="mx-auto max-w-6xl px-4 py-6">
        <Outlet />
      </main>
    </div>
  )
}
