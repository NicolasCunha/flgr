import { Link } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { useGetMeQuery } from '@/features/auth/api'

const links = [
  {
    to: '/environments',
    title: 'Environments',
    description: 'Create and manage the environments feature flags are scoped to.',
  },
  {
    to: '/service-keys',
    title: 'Service Keys',
    description: 'Issue and manage credentials consumer applications use to read flags.',
  },
]

export function HomePage() {
  const { data: me } = useGetMeQuery()

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">{me ? `Welcome, ${me.firstName}` : 'Welcome'}</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Pick an area to manage from the links below.
        </p>
      </div>
      <div className="grid gap-4 sm:grid-cols-2">
        {links.map((link) => (
          <Card key={link.to}>
            <CardHeader>
              <CardTitle>{link.title}</CardTitle>
              <CardDescription>{link.description}</CardDescription>
            </CardHeader>
            <CardContent>
              <Button asChild variant="outline" size="sm">
                <Link to={link.to}>Go to {link.title.toLowerCase()}</Link>
              </Button>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  )
}
