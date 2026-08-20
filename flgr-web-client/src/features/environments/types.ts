export interface EnvironmentCategory {
  id: string
  name: string
  isSystem: boolean
}

export interface Environment {
  id: string
  name: string
  description: string | null
  environmentCategoryId: string
  createdOn: string
  modifiedOn: string
}

export interface ListEnvironmentsParams {
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

// categoryName is always a name — an existing category (e.g.
// "Development") or a brand-new one the user typed after picking
// "Other" — never an id. The backend resolves it to a category id
// itself, creating a new non-system category on the fly if the name
// doesn't exist yet (see flgr-server's EnvironmentService.resolveCategory
// and docs/business/requirements/0001-environment-management.md). There
// is no separate "create category" endpoint — only GET
// /environment-categories exists — so the frontend never creates one
// directly.
export interface EnvironmentInput {
  name: string
  description: string
  categoryName: string
}
