import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { DataTable } from './DataTable'

interface Row {
  id: string
  name: string
}

const columns = [
  { accessorKey: 'name', header: 'Name' },
] satisfies { accessorKey: keyof Row; header: string }[]

describe('DataTable', () => {
  it('renders rows and column headers', () => {
    render(
      <DataTable<Row>
        columns={columns}
        data={[
          { id: '1', name: 'app-dev' },
          { id: '2', name: 'app-prod' },
        ]}
        pagination={{ page: 1, pageSize: 50, total: 2 }}
        onPageChange={() => {}}
      />,
    )

    expect(screen.getByText('Name')).toBeInTheDocument()
    expect(screen.getByText('app-dev')).toBeInTheDocument()
    expect(screen.getByText('app-prod')).toBeInTheDocument()
    expect(screen.getByText('Page 1 of 1 — 2 results')).toBeInTheDocument()
  })

  it('shows a loading state', () => {
    render(
      <DataTable<Row>
        columns={columns}
        data={[]}
        isLoading
        pagination={{ page: 1, pageSize: 50, total: 0 }}
        onPageChange={() => {}}
      />,
    )
    expect(screen.getByText('Loading...')).toBeInTheDocument()
  })

  it('shows an error state', () => {
    render(
      <DataTable<Row>
        columns={columns}
        data={[]}
        isError
        errorMessage="Could not load environments."
        pagination={{ page: 1, pageSize: 50, total: 0 }}
        onPageChange={() => {}}
      />,
    )
    expect(screen.getByText('Could not load environments.')).toBeInTheDocument()
  })

  it('shows an empty state with 0 results', () => {
    render(
      <DataTable<Row>
        columns={columns}
        data={[]}
        emptyMessage="No environments yet."
        pagination={{ page: 1, pageSize: 50, total: 0 }}
        onPageChange={() => {}}
      />,
    )
    expect(screen.getByText('No environments yet.')).toBeInTheDocument()
    expect(screen.getByText('0 results')).toBeInTheDocument()
  })

  it('disables Previous on the first page and Next on the last page', () => {
    render(
      <DataTable<Row>
        columns={columns}
        data={[{ id: '1', name: 'app-dev' }]}
        pagination={{ page: 1, pageSize: 1, total: 1 }}
        onPageChange={() => {}}
      />,
    )
    expect(screen.getByRole('button', { name: 'Previous' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Next' })).toBeDisabled()
  })

  it('calls onPageChange when Previous/Next are clicked', async () => {
    const onPageChange = vi.fn()
    const user = userEvent.setup()
    render(
      <DataTable<Row>
        columns={columns}
        data={[{ id: '1', name: 'app-dev' }]}
        pagination={{ page: 2, pageSize: 1, total: 3 }}
        onPageChange={onPageChange}
      />,
    )
    await user.click(screen.getByRole('button', { name: 'Previous' }))
    await user.click(screen.getByRole('button', { name: 'Next' }))
    expect(onPageChange).toHaveBeenCalledWith(1)
    expect(onPageChange).toHaveBeenCalledWith(3)
  })
})
