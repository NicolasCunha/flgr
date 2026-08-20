import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { useGetEnvironmentsQuery } from '@/features/environments/api'
import type { Environment } from '@/features/environments/types'
import { getErrorMessage } from '@/lib/errors'
import { useCreateServiceKeyMutation, useUpdateServiceKeyMutation } from './api'
import type { ServiceKey } from './types'

// Stable reference for the "no environments yet" fallback — see
// EnvironmentForm.tsx's EMPTY_CATEGORIES comment for why this matters
// (avoids an infinite reset loop from a fresh [] allocated every render).
const EMPTY_ENVIRONMENTS: Environment[] = []

const serviceKeySchema = z
  .object({
    name: z.string().trim().min(1, 'Name is required'),
    canRead: z.boolean(),
    canWrite: z.boolean(),
    environmentIds: z.array(z.string()).min(1, 'Select at least one environment'),
    // Only surfaced in the UI when editing — a new key is always created
    // Active server-side (see 0004's Data Model, status defaults to
    // Active), so there's no "initial status" choice at creation time.
    active: z.boolean(),
  })
  .refine((data) => data.canRead || data.canWrite, {
    message: 'Enable at least one of Can read / Can write',
    path: ['canWrite'],
  })

type ServiceKeyFormValues = z.infer<typeof serviceKeySchema>

interface ServiceKeyFormProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  serviceKey?: ServiceKey
  // Called once, right after a successful create, with the plaintext
  // secret — the parent (ServiceKeysPage) is responsible for showing it
  // via ServiceKeySecretDialog, since it's only ever returned this once.
  onCreated?: (secret: string) => void
}

export function ServiceKeyForm({ open, onOpenChange, serviceKey, onCreated }: ServiceKeyFormProps) {
  const isEditing = !!serviceKey
  const { data: environmentsPage, refetch: refetchEnvironments } = useGetEnvironmentsQuery({
    page: 1,
    pageSize: 200,
  })
  const environments = environmentsPage?.data ?? EMPTY_ENVIRONMENTS
  const [createServiceKey, { isLoading: isCreating, error: createError }] = useCreateServiceKeyMutation()
  const [updateServiceKey, { isLoading: isUpdating, error: updateError }] = useUpdateServiceKeyMutation()

  const form = useForm<ServiceKeyFormValues>({
    resolver: zodResolver(serviceKeySchema),
    defaultValues: { name: '', canRead: true, canWrite: false, environmentIds: [], active: true },
  })

  useEffect(() => {
    if (!open) return
    // Same reasoning as EnvironmentForm.tsx's refetchCategories effect —
    // force a fresh fetch rather than trusting whatever's already cached.
    void refetchEnvironments()
  }, [open, refetchEnvironments])

  useEffect(() => {
    if (!open) return
    form.reset({
      name: serviceKey?.name ?? '',
      canRead: serviceKey?.canRead ?? true,
      canWrite: serviceKey?.canWrite ?? false,
      environmentIds: serviceKey?.environmentIds ?? [],
      active: serviceKey ? serviceKey.status === 'Active' : true,
    })
  }, [open, serviceKey, form])

  const isSubmitting = isCreating || isUpdating
  const error = createError ?? updateError

  async function onSubmit(values: ServiceKeyFormValues) {
    if (isEditing && serviceKey) {
      await updateServiceKey({
        id: serviceKey.id,
        name: values.name,
        status: values.active ? 'Active' : 'Inactive',
        canRead: values.canRead,
        canWrite: values.canWrite,
        environmentIds: values.environmentIds,
      }).unwrap()
      onOpenChange(false)
      return
    }

    const created = await createServiceKey({
      name: values.name,
      canRead: values.canRead,
      canWrite: values.canWrite,
      environmentIds: values.environmentIds,
    }).unwrap()
    onOpenChange(false)
    onCreated?.(created.secret)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{isEditing ? 'Edit service key' : 'New service key'}</DialogTitle>
          <DialogDescription>
            {isEditing
              ? 'Update this key’s name, access, status, or environment scope.'
              : 'Consumer applications authenticate against flgr with a service key, scoped to the environments and access it’s granted here.'}
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form
            onSubmit={(event) =>
              void form.handleSubmit(async (values) => {
                try {
                  await onSubmit(values)
                } catch {
                  // Surfaced below via the `error` returned from the mutation hooks.
                }
              })(event)
            }
            className="space-y-4"
            noValidate
          >
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel htmlFor="service-key-name">Name</FormLabel>
                  <FormControl>
                    <Input id="service-key-name" placeholder="checkout-service prod key" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            {isEditing && (
              <FormField
                control={form.control}
                name="active"
                render={({ field }) => (
                  <FormItem className="flex flex-row items-center justify-between rounded-md border border-border px-3 py-2">
                    <FormLabel htmlFor="service-key-active" className="mb-0">
                      {field.value ? 'Active' : 'Inactive'}
                    </FormLabel>
                    <FormControl>
                      <Switch
                        id="service-key-active"
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    </FormControl>
                  </FormItem>
                )}
              />
            )}

            <FormField
              control={form.control}
              name="canRead"
              render={({ field }) => (
                <FormItem className="flex flex-row items-center justify-between rounded-md border border-border px-3 py-2">
                  <FormLabel htmlFor="service-key-can-read" className="mb-0">
                    Can read
                  </FormLabel>
                  <FormControl>
                    <Switch
                      id="service-key-can-read"
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                  </FormControl>
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="canWrite"
              render={({ field }) => (
                <FormItem>
                  <div className="flex flex-row items-center justify-between rounded-md border border-border px-3 py-2">
                    <FormLabel htmlFor="service-key-can-write" className="mb-0">
                      Can write
                    </FormLabel>
                    <FormControl>
                      <Switch
                        id="service-key-can-write"
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    </FormControl>
                  </div>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="environmentIds"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Environments</FormLabel>
                  <div className="max-h-48 space-y-2 overflow-y-auto rounded-md border border-border p-3">
                    {environments.length === 0 ? (
                      <p className="text-sm text-muted-foreground">
                        No environments exist yet — create one first.
                      </p>
                    ) : (
                      environments.map((environment) => {
                        const checked = field.value.includes(environment.id)
                        return (
                          <label
                            key={environment.id}
                            htmlFor={`service-key-env-${environment.id}`}
                            className="flex items-center gap-2 text-sm"
                          >
                            <Checkbox
                              id={`service-key-env-${environment.id}`}
                              checked={checked}
                              onCheckedChange={(next) => {
                                field.onChange(
                                  next
                                    ? [...field.value, environment.id]
                                    : field.value.filter((id: string) => id !== environment.id),
                                )
                              }}
                            />
                            {environment.name}
                          </label>
                        )
                      })
                    )}
                  </div>
                  <FormMessage />
                </FormItem>
              )}
            />

            {error && (
              <p role="alert" className="text-sm text-destructive">
                {getErrorMessage(error)}
              </p>
            )}
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={isSubmitting}>
                {isSubmitting ? 'Saving...' : isEditing ? 'Save changes' : 'Create service key'}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
