import { useGetHealthQuery } from '@/lib/api'

export function HomePage() {
  const { data, isLoading, isError } = useGetHealthQuery()

  return (
    <main className="flex min-h-screen items-center justify-center bg-background text-foreground">
      <div className="text-center">
        <h1 className="text-2xl font-semibold">flgr</h1>
        <p className="mt-2 text-sm text-muted-foreground">
          {isLoading && 'Checking backend...'}
          {isError && 'Backend unreachable'}
          {data && `Backend status: ${data.status}`}
        </p>
      </div>
    </main>
  )
}
