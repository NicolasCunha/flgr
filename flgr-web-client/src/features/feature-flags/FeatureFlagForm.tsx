import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'
import { Button } from '@/components/ui/button'
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { getErrorMessage } from '@/lib/errors'
import { useCreateFeatureFlagMutation, useUpdateFeatureFlagMutation } from './api'
import { FEATURE_FLAG_TYPES, type FeatureFlag } from './types'

// Restricted to letters, numbers, hyphens, and underscores, per 0005's
// Content section. This is a fast client-side check only — the real
// rejection (including uniqueness, which the client can't check) is
// server-side and surfaced via the `error` block below.
const KEY_PATTERN = /^[A-Za-z0-9_-]+$/

const featureFlagSchema = z.object({
  key: z
    .string()
    .trim()
    .min(1, 'Key is required')
    .regex(KEY_PATTERN, 'Only letters, numbers, hyphens, and underscores are allowed'),
  name: z.string().trim().min(1, 'Name is required'),
  description: z.string(),
  type: z.enum(FEATURE_FLAG_TYPES, { message: 'Type is required' }),
})

type FeatureFlagFormValues = z.infer<typeof featureFlagSchema>

interface FeatureFlagFormProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  featureFlag?: FeatureFlag
}

export function FeatureFlagForm({ open, onOpenChange, featureFlag }: FeatureFlagFormProps) {
  const isEditing = !!featureFlag
  const [createFeatureFlag, { isLoading: isCreating, error: createError }] =
    useCreateFeatureFlagMutation()
  const [updateFeatureFlag, { isLoading: isUpdating, error: updateError }] =
    useUpdateFeatureFlagMutation()

  const form = useForm<FeatureFlagFormValues>({
    resolver: zodResolver(featureFlagSchema),
    defaultValues: { key: '', name: '', description: '', type: 'Boolean' },
  })

  useEffect(() => {
    if (!open) return
    form.reset({
      key: featureFlag?.key ?? '',
      name: featureFlag?.name ?? '',
      description: featureFlag?.description ?? '',
      type: featureFlag?.type ?? 'Boolean',
    })
  }, [open, featureFlag, form])

  const isSubmitting = isCreating || isUpdating
  const error = createError ?? updateError

  async function onSubmit(values: FeatureFlagFormValues) {
    if (isEditing && featureFlag) {
      await updateFeatureFlag({
        id: featureFlag.id,
        name: values.name,
        description: values.description,
      }).unwrap()
    } else {
      await createFeatureFlag({
        key: values.key,
        name: values.name,
        description: values.description,
        type: values.type,
      }).unwrap()
    }
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{isEditing ? 'Edit feature flag' : 'New feature flag'}</DialogTitle>
          <DialogDescription>
            {isEditing
              ? 'Update this flag’s name or description. Its key and type can’t be changed.'
              : 'A flag’s key and type are fixed once created, so consumer applications can rely on a stable shape.'}
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
            {isEditing ? (
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1">
                  <p className="text-sm font-medium">Key</p>
                  <p className="text-sm text-muted-foreground">{featureFlag.key}</p>
                </div>
                <div className="space-y-1">
                  <p className="text-sm font-medium">Type</p>
                  <p className="text-sm text-muted-foreground">{featureFlag.type}</p>
                </div>
              </div>
            ) : (
              <>
                <FormField
                  control={form.control}
                  name="key"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel htmlFor="flag-key">Key</FormLabel>
                      <FormControl>
                        <Input id="flag-key" placeholder="new-checkout-flow" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="type"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel htmlFor="flag-type">Type</FormLabel>
                      <Select value={field.value} onValueChange={field.onChange}>
                        <FormControl>
                          <SelectTrigger id="flag-type" className="w-full">
                            <SelectValue placeholder="Select a type" />
                          </SelectTrigger>
                        </FormControl>
                        <SelectContent>
                          {FEATURE_FLAG_TYPES.map((type) => (
                            <SelectItem key={type} value={type}>
                              {type}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </>
            )}
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel htmlFor="flag-name">Name</FormLabel>
                  <FormControl>
                    <Input id="flag-name" placeholder="New checkout flow" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="description"
              render={({ field }) => (
                <FormItem>
                  <FormLabel htmlFor="flag-description">Description</FormLabel>
                  <FormControl>
                    <Textarea id="flag-description" {...field} />
                  </FormControl>
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
                {isSubmitting ? 'Saving...' : isEditing ? 'Save changes' : 'Create flag'}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
