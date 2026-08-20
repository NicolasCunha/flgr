import { Loader2 } from 'lucide-react'

export function LoadingScreen() {
  return (
    <main className="flex min-h-screen items-center justify-center bg-background text-foreground">
      <Loader2 className="size-6 animate-spin text-muted-foreground" aria-hidden="true" />
      <span className="sr-only">Loading...</span>
    </main>
  )
}
