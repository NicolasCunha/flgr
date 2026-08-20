import { Moon, Sun } from 'lucide-react'
import { useTheme } from 'next-themes'
import { Switch } from '@/components/ui/switch'

// A single light/dark switch, not a light/dark/system picker. "system" is
// still the default appearance (see ThemeProvider's defaultTheme in
// App.tsx) until someone interacts with this control, at which point
// they've chosen an explicit theme — same behavior as most apps' quick
// toggle, just without a separate "system" option to switch back to.
export function ThemeToggle() {
  const { resolvedTheme, setTheme } = useTheme()
  const isDark = resolvedTheme === 'dark'

  return (
    <div className="flex items-center gap-2">
      <Sun className="size-4 text-muted-foreground" aria-hidden="true" />
      <Switch
        checked={isDark}
        onCheckedChange={(checked) => setTheme(checked ? 'dark' : 'light')}
        aria-label="Toggle dark mode"
      />
      <Moon className="size-4 text-muted-foreground" aria-hidden="true" />
    </div>
  )
}
