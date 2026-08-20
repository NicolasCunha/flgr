// Wire-format boundary helpers: the backend uses snake_case for every JSON
// key (see docs/architecture/adr/0007-api-design-conventions.md); the rest
// of the frontend works with idiomatic camelCase. These helpers are the
// single conversion point, meant to be called from each feature's
// `api.ts` (transformResponse for responses, before-send for request
// bodies) rather than scattered across components. Recursive — several
// real response shapes nest objects/arrays that need every level
// converted (e.g. the {data, pagination} list envelope from ADR-0007, or
// environment-categories' array-of-objects response).

type JsonValue =
  | string
  | number
  | boolean
  | null
  | undefined
  | JsonValue[]
  | { [key: string]: JsonValue }

function snakeToCamel(key: string): string {
  return key.replace(/_([a-z0-9])/g, (_match, char: string) => char.toUpperCase())
}

function camelToSnake(key: string): string {
  return key.replace(/[A-Z]/g, (char) => `_${char.toLowerCase()}`)
}

function isPlainObject(value: unknown): value is Record<string, JsonValue> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

/** Recursively converts every object key from snake_case to camelCase. */
export function keysToCamelCase<T = unknown>(value: JsonValue): T {
  if (Array.isArray(value)) {
    return value.map((item) => keysToCamelCase(item)) as T
  }
  if (isPlainObject(value)) {
    return Object.fromEntries(
      Object.entries(value).map(([key, val]) => [snakeToCamel(key), keysToCamelCase(val)]),
    ) as T
  }
  return value as T
}

/** Recursively converts every object key from camelCase to snake_case. */
export function keysToSnakeCase<T = unknown>(value: JsonValue): T {
  if (Array.isArray(value)) {
    return value.map((item) => keysToSnakeCase(item)) as T
  }
  if (isPlainObject(value)) {
    return Object.fromEntries(
      Object.entries(value).map(([key, val]) => [camelToSnake(key), keysToSnakeCase(val)]),
    ) as T
  }
  return value as T
}
