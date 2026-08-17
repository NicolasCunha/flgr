import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { App } from './App'

describe('App', () => {
  it('renders the home page at /', async () => {
    render(<App />)
    expect(await screen.findByText('flgr')).toBeInTheDocument()
  })
})
