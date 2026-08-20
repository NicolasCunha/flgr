import { describe, expect, it } from 'vitest'
import { getErrorMessage } from './errors'

describe('getErrorMessage', () => {
  it('returns the fallback when no error is given', () => {
    expect(getErrorMessage(undefined)).toBe('Something went wrong. Please try again.')
  })

  it('returns a custom fallback when provided', () => {
    expect(getErrorMessage(undefined, 'custom fallback')).toBe('custom fallback')
  })

  it('extracts the message from the API error envelope', () => {
    expect(
      getErrorMessage({
        status: 400,
        data: { error: { code: 'validation_error', message: 'email is already in use' } },
      }),
    ).toBe('email is already in use')
  })

  it('falls back to a generic server error message on 5xx responses with no envelope', () => {
    expect(getErrorMessage({ status: 500, data: undefined })).toBe(
      'The server encountered an error. Please try again.',
    )
  })

  it('falls back to the default message on a 4xx response with no envelope', () => {
    expect(getErrorMessage({ status: 404, data: undefined })).toBe(
      'Something went wrong. Please try again.',
    )
  })

  it('falls back on a fetch-error status with no data', () => {
    expect(getErrorMessage({ status: 'FETCH_ERROR', error: 'Failed to fetch' })).toBe(
      'Something went wrong. Please try again.',
    )
  })

  it('extracts the message from a SerializedError', () => {
    expect(getErrorMessage({ message: 'network down' })).toBe('network down')
  })

  it('falls back when a SerializedError has no message', () => {
    expect(getErrorMessage({})).toBe('Something went wrong. Please try again.')
  })
})
