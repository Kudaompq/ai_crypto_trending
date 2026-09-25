import { describe, expect, it } from 'vitest'
import { api } from '../services/api'

describe('frontend API surface', () => {
  it('does not expose the retired trading opportunity endpoint', () => {
    expect(api).not.toHaveProperty('getOpportunities')
  })
})
