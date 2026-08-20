import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'

interface ServiceKeySecretDialogProps {
  // null when there's no secret to show; the dialog is closed in that case.
  secret: string | null
  onClose: () => void
}

// Shown immediately after a successful create — per 0004, the plaintext
// secret is only ever returned once (POST /service-keys' response), never
// retrievable again afterward. There's no cancel/back here: the key
// already exists server-side by the time this renders, "Done" is the only
// way out.
export function ServiceKeySecretDialog({ secret, onClose }: ServiceKeySecretDialogProps) {
  async function copySecret() {
    // Guards `secret` being null, but the Copy button is only reachable in
    // the DOM while the Dialog is open, and it's open exactly when
    // open={!!secret} is true — so this can't actually fire with a null
    // secret. Not a coverage gap to chase; see EnvironmentsPage.tsx's
    // confirmDelete for the same defensive-guard convention.
    if (!secret) return
    try {
      await navigator.clipboard.writeText(secret)
      toast.success('Secret copied to clipboard.')
    } catch {
      toast.error('Could not copy the secret. Please copy it manually.')
    }
  }

  return (
    <Dialog open={!!secret} onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Service key created</DialogTitle>
          <DialogDescription>
            Copy this secret now — it won&apos;t be shown again. If you lose it, you&apos;ll need to
            create a new service key.
          </DialogDescription>
        </DialogHeader>
        <div className="flex items-center gap-2">
          <Input readOnly value={secret ?? ''} className="font-mono text-sm" onFocus={(e) => e.target.select()} />
          <Button type="button" variant="outline" onClick={() => void copySecret()}>
            Copy
          </Button>
        </div>
        <DialogFooter>
          <Button type="button" onClick={onClose}>
            Done
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
