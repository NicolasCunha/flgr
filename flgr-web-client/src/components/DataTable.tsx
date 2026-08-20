import { flexRender } from '@tanstack/react-table'
// TanStack Table v9 replaced v8's `useReactTable`/`getCoreRowModel()` API
// with a tree-shakeable "feature slot" architecture (`tableFeatures`,
// `useTable`, TanStack Store atoms) built for opting into sorting,
// filtering, grouping, etc. This table only ever needs core row/column
// rendering (pagination and sorting are handled by the backend, driven by
// plain props — see the module comment below), so the officially-shipped
// v8-compatible `useLegacyTable` is used instead of adopting that bigger
// API surface for no behavioral benefit. Revisit if a future screen needs
// a v9-only feature (e.g. client-side column filtering).
import { getCoreRowModel, useLegacyTable, type LegacyColumnDef } from '@tanstack/react-table/legacy'
import type { RowData } from '@tanstack/react-table'
import { Button } from '@/components/ui/button'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'

export interface DataTablePagination {
  page: number
  pageSize: number
  total: number
}

// useLegacyTable (and TanStack Table generally) constrains row data to
// RowData = Record<string, any> | Array<any> — every row shape this app
// passes in (Environment, etc.) is a plain object, so this is satisfied
// automatically; it just needs to be stated so TData can flow through to
// useLegacyTable below.
interface DataTableProps<TData extends RowData> {
  // Column value types vary per column; matches TanStack's own ColumnDef<TData, any>.
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  columns: LegacyColumnDef<TData, any>[]
  data: TData[]
  isLoading?: boolean
  isError?: boolean
  emptyMessage?: string
  errorMessage?: string
  pagination: DataTablePagination
  onPageChange: (page: number) => void
}

// Shared list-screen table per
// docs/architecture/adr/0008-frontend-routing-and-ui-component-library.md:
// every listing screen (Environments here, Service Keys next, more later)
// consumes the same page/page_size/total envelope from
// docs/architecture/adr/0007-api-design-conventions.md through this one
// component instead of a bespoke table each. Sorting/filtering controls
// aren't wired in yet — nothing built on top of this so far has a
// documented need for them; extend here, not per-screen, when one does.
export function DataTable<TData extends RowData>({
  columns,
  data,
  isLoading = false,
  isError = false,
  emptyMessage = 'No results.',
  errorMessage = 'Something went wrong loading this list.',
  pagination,
  onPageChange,
}: DataTableProps<TData>) {
  const table = useLegacyTable({
    data,
    columns,
    getCoreRowModel: getCoreRowModel(),
  })

  const totalPages = Math.max(1, Math.ceil(pagination.total / pagination.pageSize))

  return (
    <div className="space-y-3">
      <div className="rounded-md border border-border">
        <Table>
          <TableHeader>
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id}>
                {headerGroup.headers.map((header) => (
                  <TableHead key={header.id}>
                    {/* isPlaceholder is only ever true for a header cell spanned by a
                        grouped/multi-row column header (TanStack's column-grouping
                        feature). DataTablePagination's `columns` prop only accepts flat,
                        single-row LegacyColumnDef entries — no `columns: [...]` nesting —
                        so no column definition passed into this component can ever
                        produce a placeholder header. The `true` branch is unreachable
                        through this component's actual usage; not a coverage gap to
                        chase. */}
                    {header.isPlaceholder
                      ? null
                      : flexRender(header.column.columnDef.header, header.getContext())}
                  </TableHead>
                ))}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {isError ? (
              <TableRow>
                <TableCell colSpan={columns.length} className="h-24 text-center text-destructive">
                  {errorMessage}
                </TableCell>
              </TableRow>
            ) : isLoading ? (
              <TableRow>
                <TableCell colSpan={columns.length} className="h-24 text-center text-muted-foreground">
                  Loading...
                </TableCell>
              </TableRow>
            ) : table.getRowModel().rows.length === 0 ? (
              <TableRow>
                <TableCell colSpan={columns.length} className="h-24 text-center text-muted-foreground">
                  {emptyMessage}
                </TableCell>
              </TableRow>
            ) : (
              table.getRowModel().rows.map((row) => (
                <TableRow key={row.id}>
                  {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id}>
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">
          {pagination.total === 0
            ? '0 results'
            : `Page ${pagination.page} of ${totalPages} — ${pagination.total} result${pagination.total === 1 ? '' : 's'}`}
        </p>
        <div className="flex gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => onPageChange(pagination.page - 1)}
            disabled={pagination.page <= 1}
          >
            Previous
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => onPageChange(pagination.page + 1)}
            disabled={pagination.page >= totalPages}
          >
            Next
          </Button>
        </div>
      </div>
    </div>
  )
}
