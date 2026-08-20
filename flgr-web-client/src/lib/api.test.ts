import { configureStore } from '@reduxjs/toolkit'
import type { SerializedError } from '@reduxjs/toolkit'
import type { FetchBaseQueryError } from '@reduxjs/toolkit/query/react'
import { describe, expect, it } from 'vitest'
import { api, getApiErrorMessage, isFetchBaseQueryError } from './api'

describe('getHealth', () => {
  it('fetches GET /health', async () => {
    const store = configureStore({
      reducer: { [api.reducerPath]: api.reducer },
      middleware: (getDefaultMiddleware) => getDefaultMiddleware().concat(api.middleware),
    })
    const result = await store.dispatch(api.endpoints.getHealth.initiate())
    expect(result.data).toEqual({ status: 'ok' })
  })
})

describe('isFetchBaseQueryError', () => {
  it('is true for an object with a status field', () => {
    expect(isFetchBaseQueryError({ status: 409, data: {} })).toBe(true)
  })

  it('is false for a SerializedError', () => {
    expect(isFetchBaseQueryError({ message: 'boom' })).toBe(false)
  })

  it('is false for null/undefined', () => {
    expect(isFetchBaseQueryError(null)).toBe(false)
    expect(isFetchBaseQueryError(undefined)).toBe(false)
  })
})

describe('getApiErrorMessage', () => {
  it('returns undefined for undefined input', () => {
    expect(getApiErrorMessage(undefined)).toBeUndefined()
  })

  it('extracts the ADR-0007 error envelope message', () => {
    const error: FetchBaseQueryError = {
      status: 409,
      data: { error: { code: 'conflict', message: 'email is already in use' } },
    }
    expect(getApiErrorMessage(error)).toBe('email is already in use')
  })

  it('falls back to a generic message for a numeric status with no envelope', () => {
    const error: FetchBaseQueryError = { status: 500, data: 'oops' }
    expect(getApiErrorMessage(error)).toBe('Request failed (status 500)')
  })

  it('falls back to a generic message for a non-numeric status', () => {
    const error: FetchBaseQueryError = { status: 'FETCH_ERROR', error: 'network down' }
    expect(getApiErrorMessage(error)).toBe('Request failed')
  })

  it('uses the message from a SerializedError', () => {
    const error: SerializedError = { message: 'something broke' }
    expect(getApiErrorMessage(error)).toBe('something broke')
  })
})
