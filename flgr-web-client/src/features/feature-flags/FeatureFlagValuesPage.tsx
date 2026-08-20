import { useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { useGetEnvironmentsQuery } from '@/features/environments/api'
import { getErrorMessage } from '@/lib/errors'
import { useGetFeatureFlagQuery, useListFeatureFlagValuesQuery, useUpsertFeatureFlagValueMutation } from './api'
import type { FeatureFlagEnvironmentValue } from './types'

// A flag with no configured row for an environment evaluates as disabled
// there by default, per 0005 — there's no requirement to distinguish
// "explicitly disabled" from "never configured" in the UI, so both render
// as a plain off toggle. Resolves one environment's effective
// enabled/value, whether or not it has an explicit row yet.
function effectiveValue(
  environmentId: string,
  values: FeatureFlagEnvironmentValue[],
): { enabled: boolean; value: string } {
  const row = values.find((v) => v.environmentId === environmentId)
  return { enabled: row?.enabled ?? false, value: row?.value ?? '' }
}

interface ValueRowProps {
  environmentId: string
  environmentName: string
  flagId: string
  isBoolean: boolean
  saved: { enabled: boolean; value: string }
}

function ValueRow({ environmentId, environmentName, flagId, isBoolean, saved }: ValueRowProps) {
  const [upsert, { isLoading: isSaving }] = useUpsertFeatureFlagValueMutation()
  // Local editing state seeded from the last-saved server value, so typing
  // in the value field (or flipping the switch) doesn't fire a request per
  // keystroke — only the explicit actions below (toggle change, value
  // blur) do. The parent remounts this component (via a `key` derived from
  // `saved`, see below) whenever the server value actually changes, so this
  // never needs to resync from a prop change inside an effect.
  const [value, setValue] = useState(saved.value)

  async function save(next: { enabled: boolean; value: string }) {
    try {
      await upsert({
        flagId,
        environmentId,
        enabled: next.enabled,
        value: isBoolean ? undefined : next.value || null,
      }).unwrap()
    } catch (error) {
      toast.error(getErrorMessage(error, `Could not update "${environmentName}".`))
    }
  }

  return (
    <div className="flex items-center justify-between gap-4 border-b border-border py-3 last:border-b-0">
      <div className="flex items-center gap-3">
        <Switch
          checked={saved.enabled}
          disabled={isSaving}
          onCheckedChange={(checked) => void save({ enabled: checked, value })}
        />
        <span className="text-sm font-medium">{environmentName}</span>
      </div>
      {!isBoolean && (
        <Input
          className="max-w-xs"
          value={value}
          disabled={isSaving}
          placeholder="Value served when enabled"
          onChange={(event) => setValue(event.target.value)}
          onBlur={() => {
            if (value !== saved.value) void save({ enabled: saved.enabled, value })
          }}
        />
      )}
    </div>
  )
}

export function FeatureFlagValuesPage() {
  // `id` is only possibly undefined per useParams' generic signature — the
  // route this page is mounted on (`feature-flags/:id/values`, see
  // router.tsx) always supplies it, so the `?? ''` fallback can't actually
  // trigger. Same category of defensive-but-unreachable code as the
  // `if (!deleting) return` guard in EnvironmentsPage.tsx/FeatureFlagsPage.tsx.
  const { id } = useParams<{ id: string }>()
  const flagId = id ?? ''

  const { data: flag, isLoading: isFlagLoading, isError: isFlagError } = useGetFeatureFlagQuery(flagId, {
    skip: !flagId,
  })
  const { data: environmentsPage, isLoading: isEnvironmentsLoading } = useGetEnvironmentsQuery({
    page: 1,
    pageSize: 200,
  })
  const {
    data: values = [],
    isLoading: isValuesLoading,
    isError: isValuesError,
  } = useListFeatureFlagValuesQuery(flagId, { skip: !flagId })

  const environments = useMemo(() => environmentsPage?.data ?? [], [environmentsPage])

  if (isFlagLoading || isValuesLoading || isEnvironmentsLoading) {
    return <p className="text-sm text-muted-foreground">Loading...</p>
  }

  if (isFlagError || !flag) {
    return <p className="text-sm text-destructive">Could not load this feature flag.</p>
  }

  return (
    <div className="space-y-4">
      <div>
        <Link to="/feature-flags" className="text-sm text-muted-foreground hover:text-foreground">
          &larr; Feature Flags
        </Link>
        <div className="mt-1 flex items-center gap-2">
          <h1 className="text-2xl font-semibold">{flag.name}</h1>
          <code className="rounded bg-muted px-1.5 py-0.5 text-sm text-muted-foreground">{flag.key}</code>
        </div>
        <p className="mt-1 text-sm text-muted-foreground">
          Whether <code>{flag.key}</code> is enabled in each environment
          {flag.type !== 'Boolean' ? ', and the value it serves when it is' : ''}. An environment with
          no explicit value here is treated as disabled.
        </p>
      </div>

      {isValuesError ? (
        <p className="text-sm text-destructive">Could not load this flag's values.</p>
      ) : environments.length === 0 ? (
        <p className="text-sm text-muted-foreground">
          No environments exist yet — create one first to configure this flag's values.
        </p>
      ) : (
        <div className="rounded-md border border-border px-4">
          {environments.map((environment) => {
            const saved = effectiveValue(environment.id, values)
            return (
              <ValueRow
                // Remounts (resetting the row's local editing state) when
                // the server's *value* changes, e.g. after a successful
                // save — see the comment on `value` below. Deliberately
                // not keyed on `saved.enabled` too: that field only drives
                // the Switch's `checked` prop directly (no local state to
                // resync), and keying on it forced a fresh Switch DOM node
                // on every toggle, which skipped its CSS transition
                // instead of animating between states.
                key={`${environment.id}:${saved.value}`}
                environmentId={environment.id}
                environmentName={environment.name}
                flagId={flagId}
                isBoolean={flag.type === 'Boolean'}
                saved={saved}
              />
            )
          })}
        </div>
      )}

      <Button variant="outline" asChild>
        <Link to="/feature-flags">Done</Link>
      </Button>
    </div>
  )
}
