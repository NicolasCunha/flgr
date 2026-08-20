import { useState } from 'react'
import { toast } from 'sonner'
import { ConfirmDialog } from '@/components/ConfirmDialog'
import { DataTable } from '@/components/DataTable'
import { Button } from '@/components/ui/button'
import { getErrorMessage } from '@/lib/errors'
import {
  useDeleteEnvironmentMutation,
  useGetEnvironmentCategoriesQuery,
  useGetEnvironmentsQuery,
} from './api'
import { EnvironmentForm } from './EnvironmentForm'
import type { Environment } from './types'

const PAGE_SIZE = 20

export function EnvironmentsPage() {
  const [page, setPage] = useState(1)
  const { data, isLoading, isError } = useGetEnvironmentsQuery({
    page,
    pageSize: PAGE_SIZE,
  })
  const { data: categories = [] } = useGetEnvironmentCategoriesQuery()
  const [deleteEnvironment, { isLoading: isDeleting }] = useDeleteEnvironmentMutation()

  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<Environment | undefined>(undefined)
  const [deleting, setDeleting] = useState<Environment | undefined>(undefined)

  const categoryName = (id: string) => categories.find((c) => c.id === id)?.name ?? '—'

  function openCreateForm() {
    setEditing(undefined)
    setFormOpen(true)
  }

  function openEditForm(environment: Environment) {
    setEditing(environment)
    setFormOpen(true)
  }

  async function confirmDelete() {
    // Guards `deleting` being unset, but the only way to invoke this is
    // ConfirmDialog's onConfirm — and ConfirmDialog is rendered with
    // open={!!deleting}, so its confirm button (and thus this function)
    // isn't reachable in the DOM unless `deleting` is already set. Not a
    // coverage gap to chase; kept as a defensive guard against a future
    // caller that doesn't have that same invariant.
    if (!deleting) return
    try {
      await deleteEnvironment(deleting.id).unwrap()
      toast.success(`Environment "${deleting.name}" removed.`)
      setDeleting(undefined)
    } catch (error) {
      toast.error(getErrorMessage(error, 'Could not remove this environment.'))
    }
  }

  const columns = [
    { accessorKey: 'name', header: 'Name' },
    {
      id: 'category',
      header: 'Category',
      cell: ({ row }: { row: { original: Environment } }) =>
        categoryName(row.original.environmentCategoryId),
    },
    {
      id: 'description',
      header: 'Description',
      cell: ({ row }: { row: { original: Environment } }) => row.original.description || '—',
    },
    {
      id: 'actions',
      header: '',
      cell: ({ row }: { row: { original: Environment } }) => (
        <div className="flex justify-end gap-2">
          <Button variant="ghost" size="sm" onClick={() => openEditForm(row.original)}>
            Edit
          </Button>
          <Button variant="ghost" size="sm" onClick={() => setDeleting(row.original)}>
            Remove
          </Button>
        </div>
      ),
    },
  ]

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold">Environments</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Stages a feature flag can hold an independent value in.
          </p>
        </div>
        <Button onClick={openCreateForm}>New environment</Button>
      </div>

      <DataTable<Environment>
        columns={columns}
        data={data?.data ?? []}
        isLoading={isLoading}
        isError={isError}
        emptyMessage="No environments yet. Create the first one to get started."
        errorMessage="Could not load environments."
        pagination={{ page, pageSize: PAGE_SIZE, total: data?.pagination.total ?? 0 }}
        onPageChange={setPage}
      />

      <EnvironmentForm open={formOpen} onOpenChange={setFormOpen} environment={editing} />

      <ConfirmDialog
        open={!!deleting}
        onOpenChange={(open) => !open && setDeleting(undefined)}
        title="Remove environment"
        description={
          deleting
            ? `Remove "${deleting.name}"? This can't be undone, and will fail if it's still referenced elsewhere (e.g. by a feature flag value or service key).`
            : ''
        }
        confirmLabel="Remove"
        isConfirming={isDeleting}
        onConfirm={() => void confirmDelete()}
      />
    </div>
  )
}
