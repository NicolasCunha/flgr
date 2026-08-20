// Per docs/business/requirements/0004-service-key-management.md.
export const SERVICE_KEY_STATUSES = ['Active', 'Inactive'] as const

export type ServiceKeyStatus = (typeof SERVICE_KEY_STATUSES)[number]

export interface ServiceKey {
  id: string
  name: string
  status: ServiceKeyStatus
  canRead: boolean
  canWrite: boolean
  environmentIds: string[]
  createdOn: string
  modifiedOn: string
}

// POST /service-keys' response is a ServiceKey plus the plaintext secret —
// present only in this one response, per 0004 ("shown in full only once,
// at creation time"). Get/List/Update never return it.
export interface CreateServiceKeyResponse extends ServiceKey {
  secret: string
}

export interface ListServiceKeysParams {
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

// POST /service-keys body. canRead/canWrite are optional on the wire
// (defaulting server-side to true/false), but the create form always sends
// both explicitly since that's the meaningful choice at creation time.
export interface ServiceKeyInput {
  name: string
  canRead: boolean
  canWrite: boolean
  environmentIds: string[]
}

// PATCH /service-keys/:id body — every field optional (partial update).
// Unlike Deactivate (DELETE), status here can move either direction:
// Active -> Inactive or Inactive -> Active. Omitting environmentIds leaves
// the existing scope unchanged; if included, the backend rejects an empty
// array (at least one environment is always required).
export interface ServiceKeyUpdateInput {
  name?: string
  status?: ServiceKeyStatus
  canRead?: boolean
  canWrite?: boolean
  environmentIds?: string[]
}
