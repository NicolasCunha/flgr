import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ThemeProvider } from 'next-themes'
import { afterEach, describe, expect, it } from 'vitest'
import { ThemeToggle } from './ThemeToggle'

function renderToggle() {
  return render(
    <ThemeProvider attribute="class" defaultTheme="light" enableSystem={false} storageKey="flgr-theme">
      <ThemeToggle />
    </ThemeProvider>,
  )
}

afterEach(() => {
  document.documentElement.classList.remove('dark')
  localStorage.removeItem('flgr-theme')
})

describe('ThemeToggle', () => {
  it('starts unchecked for the light theme and applies the dark class once toggled', async () => {
    const user = userEvent.setup()
    renderToggle()

    const toggle = await screen.findByRole('switch', { name: /toggle dark mode/i })
    expect(toggle).not.toBeChecked()
    expect(document.documentElement).not.toHaveClass('dark')

    await user.click(toggle)

    expect(toggle).toBeChecked()
    expect(document.documentElement).toHaveClass('dark')
  })

  it('toggles back to light and removes the dark class', async () => {
    const user = userEvent.setup()
    renderToggle()

    const toggle = await screen.findByRole('switch', { name: /toggle dark mode/i })
    await user.click(toggle)
    expect(toggle).toBeChecked()

    await user.click(toggle)

    expect(toggle).not.toBeChecked()
    expect(document.documentElement).not.toHaveClass('dark')
  })
})
