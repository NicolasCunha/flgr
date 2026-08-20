import { zodResolver } from '@hookform/resolvers/zod'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { Navigate, useNavigate } from 'react-router-dom'
import { z } from 'zod'
import { ThemeToggle } from '@/components/ThemeToggle'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useLoginMutation } from '@/features/auth/api'
import { useCompleteSetupMutation, useGetSetupStatusQuery } from '@/features/setup/api'
import { getApiErrorMessage, isFetchBaseQueryError } from '@/lib/api'

const setupSchema = z
  .object({
    firstName: z.string().trim().min(1, 'First name is required'),
    lastName: z.string().trim().min(1, 'Last name is required'),
    email: z.string().trim().min(1, 'Email is required').email('Enter a valid email address'),
    password: z.string().min(8, 'Password must be at least 8 characters'),
    confirmPassword: z.string(),
  })
  .refine((data) => data.password === data.confirmPassword, {
    message: "Passwords don't match",
    path: ['confirmPassword'],
  })

type SetupFormValues = z.infer<typeof setupSchema>

// The first-run setup wizard: creates the administrator account, per
// docs/business/requirements/0002-user-management.md. Reachable only
// while no User exists yet — RootGuard sends everyone else to "/", and
// this page redirects itself away once setup is done (including the
// backend racing us to a 409, if two tabs both submit the form).
export function SetupWizardPage() {
  const navigate = useNavigate()
  const { data: status, isLoading: isStatusLoading } = useGetSetupStatusQuery()
  const [completeSetup, { isLoading: isCompletingSetup, error: completeSetupError }] = useCompleteSetupMutation()
  const [login] = useLoginMutation()
  const [setupConflict, setSetupConflict] = useState(false)
  const [submitted, setSubmitted] = useState(false)

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<SetupFormValues>({ resolver: zodResolver(setupSchema) })

  async function onSubmit(values: SetupFormValues) {
    try {
      await completeSetup({
        firstName: values.firstName,
        lastName: values.lastName,
        email: values.email,
        password: values.password,
      }).unwrap()
    } catch (err) {
      if (isFetchBaseQueryError(err) && err.status === 409) {
        setSetupConflict(true)
      }
      return
    }

    // completeSetup invalidates the 'Setup' tag (so RootGuard picks up
    // needed:false once we navigate to "/"), but this component's own
    // status query below shares that same cache entry — without this
    // flag, its refetch landing needed:false mid-submit triggers the
    // status?.needed === false branch below immediately, navigating away
    // before the auto-login call underneath even starts, which briefly
    // sends an unauthenticated GET /me and stalls back at /login until
    // that login call finishes. Once *this* submission is the one that
    // completed setup, only the explicit navigate() at the end should act.
    setSubmitted(true)

    try {
      await login({ email: values.email, password: values.password }).unwrap()
    } catch {
      // Setup itself succeeded even if this follow-up auto-login didn't —
      // not fatal, the account still exists and can log in normally.
    }
    navigate('/', { replace: true })
  }

  if (isStatusLoading) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-background text-foreground">
        <p className="text-sm text-muted-foreground">Loading…</p>
      </main>
    )
  }

  if (setupConflict || (!submitted && status?.needed === false)) {
    return <Navigate to="/" replace />
  }

  return (
    <main className="flex min-h-screen items-center justify-center bg-background p-4 text-foreground">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>Welcome to flgr</CardTitle>
          <CardDescription>Create the administrator account to get started.</CardDescription>
          <CardAction>
            <ThemeToggle />
          </CardAction>
        </CardHeader>
        <CardContent>
          <form onSubmit={(event) => void handleSubmit(onSubmit)(event)} noValidate className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="firstName">First name</Label>
                <Input id="firstName" autoComplete="given-name" {...register('firstName')} />
                {errors.firstName && <p className="text-sm text-destructive">{errors.firstName.message}</p>}
              </div>
              <div className="space-y-2">
                <Label htmlFor="lastName">Last name</Label>
                <Input id="lastName" autoComplete="family-name" {...register('lastName')} />
                {errors.lastName && <p className="text-sm text-destructive">{errors.lastName.message}</p>}
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="email">Email</Label>
              <Input id="email" type="email" autoComplete="email" {...register('email')} />
              {errors.email && <p className="text-sm text-destructive">{errors.email.message}</p>}
            </div>

            <div className="space-y-2">
              <Label htmlFor="password">Password</Label>
              <Input id="password" type="password" autoComplete="new-password" {...register('password')} />
              {errors.password && <p className="text-sm text-destructive">{errors.password.message}</p>}
            </div>

            <div className="space-y-2">
              <Label htmlFor="confirmPassword">Confirm password</Label>
              <Input id="confirmPassword" type="password" autoComplete="new-password" {...register('confirmPassword')} />
              {errors.confirmPassword && <p className="text-sm text-destructive">{errors.confirmPassword.message}</p>}
            </div>

            {completeSetupError && (
              <p className="text-sm text-destructive">{getApiErrorMessage(completeSetupError)}</p>
            )}

            <Button type="submit" className="w-full" disabled={isCompletingSetup}>
              {isCompletingSetup ? 'Creating account…' : 'Create administrator account'}
            </Button>
          </form>
        </CardContent>
      </Card>
    </main>
  )
}
