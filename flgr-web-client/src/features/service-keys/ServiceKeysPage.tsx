import { useState } from 'react'
import { toast } from 'sonner'
import { ConfirmDialog } from '@/components/ConfirmDialog'
import { DataTable } from '@/components/DataTable'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { useGetEnvironmentsQuery } from '@/features/environments/api'
import { getErrorMessage } from '@/lib/errors'
import {
  useDeactivateServiceKeyMutation,
  useGetServiceKeysQuery,
  useUpdateServiceKeyMutation,
} from './api'
import { ServiceKeyForm } from './ServiceKeyForm'
import { ServiceKeySecretDialog } from './ServiceKeySecretDialog'
import type { ServiceKey } from './types'

const PAGE_SIZE = 20

export function ServiceKeysPage() {
  const [page, setPage] = useState(1)
  const { data, isLoading, isError } = useGetServiceKeysQuery({ page, pageSize: PAGE_SIZE })
  const { data: environmentsPage } = useGetEnvironmentsQuery({ page: 1, pageSize: 200 })
  const environments = environmentsPage?.data ?? []
  const [deactivateServiceKey, { isLoading: isDeactivating }] = useDeactivateServiceKeyMutation()
  const [updateServiceKey, { isLoading: isReactivating }] = useUpdateServiceKeyMutation()

  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<ServiceKey | undefined>(undefined)
  const [deactivating, setDeactivating] = useState<ServiceKey | undefined>(undefined)
  const [newSecret, setNewSecret] = useState<string | null>(null)

  const environmentNames = (ids: string[]) =>
    ids.map((id) => environments.find((e) => e.id === id)?.name ?? '—').join(', ') || '—'

  function openCreateForm() {
    setEditing(undefined)
    setFormOpen(true)
  }

  function openEditForm(serviceKey: ServiceKey) {
    setEditing(serviceKey)
    setFormOpen(true)
  }

  async function confirmDeactivate() {
    // Guards `deactivating` being unset — see EnvironmentsPage.tsx's
    // confirmDelete comment for why this is defensive-but-unreachable
    // through ConfirmDialog's own open={!!deactivating} gating.
    if (!deactivating) return
    try {
      await deactivateServiceKey(deactivating.id).unwrap()
      toast.success(`Service key "${deactivating.name}" deactivated.`)
      setDeactivating(undefined)
    } catch (error) {
      toast.error(getErrorMessage(error, 'Could not deactivate this service key.'))
    }
  }

  async function reactivate(serviceKey: ServiceKey) {
    try {
      await updateServiceKey({ id: serviceKey.id, status: 'Active' }).unwrap()
      toast.success(`Service key "${serviceKey.name}" reactivated.`)
    } catch (error) {
      toast.error(getErrorMessage(error, 'Could not reactivate this service key.'))
    }
  }

  const columns = [
    { accessorKey: 'name', header: 'Name' },
    {
      id: 'status',
      header: 'Status',
      cell: ({ row }: { row: { original: ServiceKey } }) => (
        <Badge variant={row.original.status === 'Active' ? 'default' : 'outline'}>
          {row.original.status}
        </Badge>
      ),
    },
    {
      id: 'access',
      header: 'Access',
      cell: ({ row }: { row: { original: ServiceKey } }) => (
        <div className="flex gap-1">
          {row.original.canRead && <Badge variant="secondary">Read</Badge>}
          {row.original.canWrite && <Badge variant="secondary">Write</Badge>}
        </div>
      ),
    },
    {
      id: 'environments',
      header: 'Environments',
      cell: ({ row }: { row: { original: ServiceKey } }) => environmentNames(row.original.environmentIds),
    },
    {
      id: 'actions',
      header: '',
      cell: ({ row }: { row: { original: ServiceKey } }) => (
        <div className="flex justify-end gap-2">
          <Button variant="ghost" size="sm" onClick={() => openEditForm(row.original)}>
            Edit
          </Button>
          {row.original.status === 'Active' ? (
            <Button variant="ghost" size="sm" onClick={() => setDeactivating(row.original)}>
              Deactivate
            </Button>
          ) : (
            <Button
              variant="ghost"
              size="sm"
              disabled={isReactivating}
              onClick={() => void reactivate(row.original)}
            >
              Reactivate
            </Button>
          )}
        </div>
      ),
    },
  ]

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold">Service Keys</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Credentials consumer applications use to read or write flag values.
          </p>
        </div>
        <Button onClick={openCreateForm}>New service key</Button>
      </div>

      <DataTable<ServiceKey>
        columns={columns}
        data={data?.data ?? []}
        isLoading={isLoading}
        isError={isError}
        emptyMessage="No service keys yet. Create the first one to get started."
        errorMessage="Could not load service keys."
        pagination={{ page, pageSize: PAGE_SIZE, total: data?.pagination.total ?? 0 }}
        onPageChange={setPage}
      />

      <ServiceKeyForm
        open={formOpen}
        onOpenChange={setFormOpen}
        serviceKey={editing}
        onCreated={setNewSecret}
      />

      <ServiceKeySecretDialog secret={newSecret} onClose={() => setNewSecret(null)} />

      <ConfirmDialog
        open={!!deactivating}
        onOpenChange={(open) => !open && setDeactivating(undefined)}
        title="Deactivate service key"
        description={
          deactivating
            ? `Deactivate "${deactivating.name}"? Any consumer application using it will immediately lose access. It can be reactivated later.`
            : ''
        }
        confirmLabel="Deactivate"
        isConfirming={isDeactivating}
        onConfirm={() => void confirmDeactivate()}
      />
    </div>
  )
}
