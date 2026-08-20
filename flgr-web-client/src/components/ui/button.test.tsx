import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { Button } from './button'

describe('Button', () => {
  it('renders a native button by default', () => {
    render(<Button>Click me</Button>)
    const button = screen.getByRole('button', { name: 'Click me' })
    expect(button).toHaveAttribute('data-variant', 'default')
    expect(button).toHaveAttribute('data-size', 'default')
  })

  it('renders the given variant and size', () => {
    render(
      <Button variant="destructive" size="sm">
        Delete
      </Button>,
    )
    const button = screen.getByRole('button', { name: 'Delete' })
    expect(button).toHaveAttribute('data-variant', 'destructive')
    expect(button).toHaveAttribute('data-size', 'sm')
  })

  it('renders its child element directly when asChild is set', () => {
    render(
      <Button asChild>
        <a href="/setup">Go to setup</a>
      </Button>,
    )
    const link = screen.getByRole('link', { name: 'Go to setup' })
    expect(link).toHaveAttribute('data-slot', 'button')
    expect(link).toHaveAttribute('href', '/setup')
  })
})
