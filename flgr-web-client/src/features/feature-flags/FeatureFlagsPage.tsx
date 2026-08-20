import { useState } from 'react'
import { toast } from 'sonner'
import { Link } from 'react-router-dom'
import { ConfirmDialog } from '@/components/ConfirmDialog'
import { DataTable } from '@/components/DataTable'
import { Button } from '@/components/ui/button'
import { getErrorMessage } from '@/lib/errors'
import { useDeleteFeatureFlagMutation, useGetFeatureFlagsQuery } from './api'
import { FeatureFlagForm } from './FeatureFlagForm'
import type { FeatureFlag } from './types'

const PAGE_SIZE = 20

export function FeatureFlagsPage() {
  const [page, setPage] = useState(1)
  const { data, isLoading, isError } = useGetFeatureFlagsQuery({
    page,
    pageSize: PAGE_SIZE,
  })
  const [deleteFeatureFlag, { isLoading: isDeleting }] = useDeleteFeatureFlagMutation()

  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<FeatureFlag | undefined>(undefined)
  const [deleting, setDeleting] = useState<FeatureFlag | undefined>(undefined)

  function openCreateForm() {
    setEditing(undefined)
    setFormOpen(true)
  }

  function openEditForm(featureFlag: FeatureFlag) {
    setEditing(featureFlag)
    setFormOpen(true)
  }

  async function confirmDelete() {
    // See EnvironmentsPage's identical guard comment — unreachable through
    // ConfirmDialog's own open={!!deleting} gating, kept as a defensive
    // guard rather than a coverage gap to chase.
    if (!deleting) return
    try {
      await deleteFeatureFlag(deleting.id).unwrap()
      toast.success(`Feature flag "${deleting.name}" removed.`)
      setDeleting(undefined)
    } catch (error) {
      toast.error(getErrorMessage(error, 'Could not remove this feature flag.'))
    }
  }

  const columns = [
    {
      id: 'key',
      header: 'Key',
      cell: ({ row }: { row: { original: FeatureFlag } }) => (
        <code className="text-sm">{row.original.key}</code>
      ),
    },
    { accessorKey: 'name', header: 'Name' },
    { accessorKey: 'type', header: 'Type' },
    {
      id: 'description',
      header: 'Description',
      cell: ({ row }: { row: { original: FeatureFlag } }) => (
        <span className="block max-w-xs truncate" title={row.original.description ?? undefined}>
          {row.original.description || '—'}
        </span>
      ),
    },
    {
      id: 'actions',
      header: '',
      cell: ({ row }: { row: { original: FeatureFlag } }) => (
        <div className="flex justify-end gap-2">
          <Button variant="ghost" size="sm" asChild>
            <Link to={`/feature-flags/${row.original.id}/values`}>Values</Link>
          </Button>
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
          <h1 className="text-2xl font-semibold">Feature Flags</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Definitions consumer applications reference by key. Configure their per-environment
            values from the Values screen.
          </p>
        </div>
        <Button onClick={openCreateForm}>New flag</Button>
      </div>

      <DataTable<FeatureFlag>
        columns={columns}
        data={data?.data ?? []}
        isLoading={isLoading}
        isError={isError}
        emptyMessage="No feature flags yet. Create the first one to get started."
        errorMessage="Could not load feature flags."
        pagination={{ page, pageSize: PAGE_SIZE, total: data?.pagination.total ?? 0 }}
        onPageChange={setPage}
      />

      <FeatureFlagForm open={formOpen} onOpenChange={setFormOpen} featureFlag={editing} />

      <ConfirmDialog
        open={!!deleting}
        onOpenChange={(open) => !open && setDeleting(undefined)}
        title="Remove feature flag"
        description={
          deleting
            ? `Remove "${deleting.name}"? This can't be undone, and will also delete its values in every environment.`
            : ''
        }
        confirmLabel="Remove"
        isConfirming={isDeleting}
        onConfirm={() => void confirmDelete()}
      />
    </div>
  )
}
