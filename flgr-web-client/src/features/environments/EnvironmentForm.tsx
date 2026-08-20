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
import {
  useCreateEnvironmentMutation,
  useGetEnvironmentCategoriesQuery,
  useUpdateEnvironmentMutation,
} from './api'
import type { Environment, EnvironmentCategory } from './types'

// Stable reference for the "no categories yet" fallback below — `data ?? []`
// would allocate a brand-new array every render while the query is
// pending (data is undefined), which re-triggers the effect that depends
// on `categories`, which calls form.reset(), which re-renders, which
// allocates a new [] again: an infinite loop. A module-level constant
// keeps the fallback reference-equal across renders so the effect's
// dependency array only changes once the real data arrives.
const EMPTY_CATEGORIES: EnvironmentCategory[] = []

// Sentinel for the "Other" option in the category select — per
// docs/business/requirements/0001-environment-management.md, picking it
// reveals a free-text field. There's no separate "create category"
// endpoint (only GET /environment-categories exists): the typed name is
// sent straight through as the environment's category_name, and
// flgr-server resolves it to an existing category or creates a new one,
// which is then a selectable category for future environments.
const OTHER_CATEGORY = '__other__'

const environmentSchema = z
  .object({
    name: z.string().trim().min(1, 'Name is required'),
    description: z.string(),
    categoryName: z.string().min(1, 'Category is required'),
    newCategoryName: z.string().trim(),
  })
  .refine((data) => data.categoryName !== OTHER_CATEGORY || data.newCategoryName.length > 0, {
    message: 'Enter a name for the new category',
    path: ['newCategoryName'],
  })

type EnvironmentFormValues = z.infer<typeof environmentSchema>

interface EnvironmentFormProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  environment?: Environment
}

export function EnvironmentForm({ open, onOpenChange, environment }: EnvironmentFormProps) {
  const isEditing = !!environment
  const { data: categories = EMPTY_CATEGORIES, refetch: refetchCategories } =
    useGetEnvironmentCategoriesQuery()
  const [createEnvironment, { isLoading: isCreating, error: createError }] =
    useCreateEnvironmentMutation()
  const [updateEnvironment, { isLoading: isUpdating, error: updateError }] =
    useUpdateEnvironmentMutation()

  const form = useForm<EnvironmentFormValues>({
    resolver: zodResolver(environmentSchema),
    defaultValues: { name: '', description: '', categoryName: '', newCategoryName: '' },
  })

  useEffect(() => {
    if (!open) return
    // Force a fresh fetch every time the dialog opens rather than trusting
    // whatever's already cached — this component and EnvironmentsPage both
    // subscribe to the same query, and if the very first fetch (e.g. right
    // after login) ever came back empty or failed, RTK Query's default
    // caching would keep serving that stale result to every later mount
    // without a network call, leaving the Category select permanently
    // empty until something else happened to evict the cache.
    void refetchCategories()
  }, [open, refetchCategories])

  useEffect(() => {
    if (!open) return
    // Editing shows the environment's current category, resolved from its
    // environmentCategoryId (the only thing the environment itself
    // carries) via the categories list — the select's options are
    // category names, not ids, since that's what the write side takes.
    const currentCategoryName = environment
      ? (categories.find((category) => category.id === environment.environmentCategoryId)?.name ?? '')
      : ''
    form.reset({
      name: environment?.name ?? '',
      description: environment?.description ?? '',
      categoryName: currentCategoryName,
      newCategoryName: '',
    })
  }, [open, environment, categories, form])

  const categoryName = form.watch('categoryName')
  const isSubmitting = isCreating || isUpdating
  const error = createError ?? updateError

  async function onSubmit(values: EnvironmentFormValues) {
    const resolvedCategoryName =
      values.categoryName === OTHER_CATEGORY ? values.newCategoryName : values.categoryName

    if (isEditing && environment) {
      await updateEnvironment({
        id: environment.id,
        name: values.name,
        description: values.description,
        categoryName: resolvedCategoryName,
      }).unwrap()
    } else {
      await createEnvironment({
        name: values.name,
        description: values.description,
        categoryName: resolvedCategoryName,
      }).unwrap()
    }
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{isEditing ? 'Edit environment' : 'New environment'}</DialogTitle>
          <DialogDescription>
            {isEditing
              ? 'Update this environment’s name, description, or category.'
              : 'Environments scope where a feature flag can hold an independent value.'}
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
                  <FormLabel htmlFor="env-name">Name</FormLabel>
                  <FormControl>
                    <Input id="env-name" placeholder="app-dev" {...field} />
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
                  <FormLabel htmlFor="env-description">Description</FormLabel>
                  <FormControl>
                    <Textarea id="env-description" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="categoryName"
              render={({ field }) => (
                <FormItem>
                  <FormLabel htmlFor="env-category">Category</FormLabel>
                  <Select value={field.value} onValueChange={field.onChange}>
                    <FormControl>
                      <SelectTrigger id="env-category" className="w-full">
                        <SelectValue placeholder="Select a category" />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      {categories.map((category) => (
                        <SelectItem key={category.id} value={category.name}>
                          {category.name}
                        </SelectItem>
                      ))}
                      <SelectItem value={OTHER_CATEGORY}>Other...</SelectItem>
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />
            {categoryName === OTHER_CATEGORY && (
              <FormField
                control={form.control}
                name="newCategoryName"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel htmlFor="env-new-category">New category name</FormLabel>
                    <FormControl>
                      <Input id="env-new-category" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            )}
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
                {isSubmitting ? 'Saving...' : isEditing ? 'Save changes' : 'Create environment'}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
