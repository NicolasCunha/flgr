import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { toast } from 'sonner'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ServiceKeySecretDialog } from './ServiceKeySecretDialog'

afterEach(() => {
  vi.restoreAllMocks()
})

// jsdom's navigator.clipboard is a getter-only accessor, so overriding it
// needs defineProperty rather than a plain assignment. Crucially, this must
// run AFTER userEvent.setup() — setup() itself calls
// @testing-library/user-event's own attachClipboardStubToView(), which
// redefines navigator.clipboard with its own no-op stub; calling this
// before setup() would just have that stub silently clobber the mock
// again, leaving `writeText` never actually invoked.
function stubClipboard(writeText: ReturnType<typeof vi.fn>) {
  Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true })
}

describe('ServiceKeySecretDialog', () => {
  it('renders nothing when secret is null', () => {
    render(<ServiceKeySecretDialog secret={null} onClose={() => {}} />)
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('shows the secret and copies it to the clipboard with a success toast', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    const successSpy = vi.spyOn(toast, 'success').mockImplementation(() => '')
    const user = userEvent.setup()
    stubClipboard(writeText)
    render(<ServiceKeySecretDialog secret="sk_live_abc123" onClose={() => {}} />)

    expect(screen.getByDisplayValue('sk_live_abc123')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: /copy/i }))

    expect(writeText).toHaveBeenCalledWith('sk_live_abc123')
    expect(successSpy).toHaveBeenCalledWith('Secret copied to clipboard.')
  })

  it('shows an error toast when copying to the clipboard fails', async () => {
    const writeText = vi.fn().mockRejectedValue(new Error('denied'))
    const errorSpy = vi.spyOn(toast, 'error').mockImplementation(() => '')
    const user = userEvent.setup()
    stubClipboard(writeText)
    render(<ServiceKeySecretDialog secret="sk_live_abc123" onClose={() => {}} />)

    await user.click(screen.getByRole('button', { name: /copy/i }))

    expect(errorSpy).toHaveBeenCalledWith('Could not copy the secret. Please copy it manually.')
  })

  it('calls onClose when Done is clicked', async () => {
    const onClose = vi.fn()
    const user = userEvent.setup()
    render(<ServiceKeySecretDialog secret="sk_live_abc123" onClose={onClose} />)

    await user.click(screen.getByRole('button', { name: /done/i }))
    expect(onClose).toHaveBeenCalled()
  })

  // Exercises the Dialog's own onOpenChange handler (distinct from the
  // "Done" button's direct onClose call above) — triggered by Radix's
  // built-in close (X) button, which calls onOpenChange(false) rather than
  // onClose directly.
  it('calls onClose when the dialog is dismissed via its built-in close control', async () => {
    const onClose = vi.fn()
    const user = userEvent.setup()
    render(<ServiceKeySecretDialog secret="sk_live_abc123" onClose={onClose} />)

    await user.click(screen.getByRole('button', { name: 'Close' }))
    expect(onClose).toHaveBeenCalled()
  })
})
