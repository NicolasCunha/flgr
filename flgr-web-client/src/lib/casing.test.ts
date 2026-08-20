import { describe, expect, it } from 'vitest'
import { keysToCamelCase, keysToSnakeCase } from './casing'

describe('keysToCamelCase', () => {
  it('converts snake_case object keys to camelCase', () => {
    expect(keysToCamelCase({ first_name: 'Ada', last_name: 'Lovelace' })).toEqual({
      firstName: 'Ada',
      lastName: 'Lovelace',
    })
  })

  it('converts keys recursively through nested objects and arrays', () => {
    expect(
      keysToCamelCase({
        data: [{ environment_category_id: '1', is_system: true }],
        pagination: { page_size: 50 },
      }),
    ).toEqual({
      data: [{ environmentCategoryId: '1', isSystem: true }],
      pagination: { pageSize: 50 },
    })
  })

  it('leaves primitive values untouched', () => {
    expect(keysToCamelCase('plain-string')).toBe('plain-string')
    expect(keysToCamelCase(42)).toBe(42)
    expect(keysToCamelCase(null)).toBeNull()
    expect(keysToCamelCase(undefined)).toBeUndefined()
    expect(keysToCamelCase(true)).toBe(true)
  })
})

describe('keysToSnakeCase', () => {
  it('converts camelCase object keys to snake_case', () => {
    expect(keysToSnakeCase({ firstName: 'Ada', lastName: 'Lovelace' })).toEqual({
      first_name: 'Ada',
      last_name: 'Lovelace',
    })
  })

  it('converts keys recursively through nested objects and arrays', () => {
    expect(
      keysToSnakeCase({
        environmentIds: ['1', '2'],
        options: [{ canRead: true }],
      }),
    ).toEqual({
      environment_ids: ['1', '2'],
      options: [{ can_read: true }],
    })
  })

  it('leaves primitive values untouched', () => {
    expect(keysToSnakeCase('plain-string')).toBe('plain-string')
    expect(keysToSnakeCase(7)).toBe(7)
    expect(keysToSnakeCase(null)).toBeNull()
  })
})
