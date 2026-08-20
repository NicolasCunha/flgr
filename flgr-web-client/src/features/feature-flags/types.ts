// Per docs/business/requirements/0005-feature-flag-management.md.
export const FEATURE_FLAG_TYPES = ['Boolean', 'String', 'Number', 'JSON'] as const

export type FeatureFlagType = (typeof FEATURE_FLAG_TYPES)[number]

export interface FeatureFlag {
  id: string
  key: string
  name: string
  description: string | null
  type: FeatureFlagType
  createdOn: string
  modifiedOn: string
}

export interface ListFeatureFlagsParams {
  page: number
  pageSize: number
}

export interface ListResponse<T> {
  data: T[]
  pagination: {
    page: number
    pageSize: number
    total: number
  }
}

/** POST /feature-flags body. `key` and `type` are fixed at creation time. */
export interface FeatureFlagInput {
  key: string
  name: string
  description: string
  type: FeatureFlagType
}

/**
 * PATCH /feature-flags/:id body — `key` and `type` are deliberately absent:
 * both are immutable after creation per 0005's Data Model and Acceptance
 * Criteria, and flgr-server's updateFeatureFlagRequest doesn't bind them.
 */
export interface FeatureFlagUpdateInput {
  name: string
  description: string
}

// A flag's per-environment value. Only environments with an explicit,
// previously-upserted row are ever returned by GET .../values — an
// environment absent from that list is implicitly disabled there, per
// 0005's Content section ("A flag with no configured value for a given
// environment is treated as disabled by default"). The UI cross-references
// this against the full Environments list to render every environment's
// effective state.
export interface FeatureFlagEnvironmentValue {
  id: string
  featureFlagId: string
  environmentId: string
  enabled: boolean
  value: string | null
  createdOn: string
  modifiedOn: string
}

/**
 * PATCH /feature-flags/:id/values/:environmentId body — a true upsert, per
 * flgr-server's FeatureFlagValueHandler.Upsert doc comment: creates the row
 * on first write for that environment, updates it thereafter. `value` only
 * matters for non-Boolean flag types.
 */
export interface UpsertValueInput {
  flagId: string
  environmentId: string
  enabled: boolean
  value?: string | null
}
