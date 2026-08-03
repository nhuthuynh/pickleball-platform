import { describe, it, expect, beforeEach } from 'vitest'
import { hasHostEvidence, recordHostEvidence, __resetHostEvidenceForTests } from '../roleEvidence'

describe('roleEvidence', () => {
  beforeEach(() => {
    __resetHostEvidenceForTests()
  })

  it('starts with no Host evidence', () => {
    expect(hasHostEvidence.value).toBe(false)
    expect(localStorage.getItem('pickleball.roleEvidence.host')).toBeNull()
  })

  it('recordHostEvidence sets the reactive ref and persists to localStorage', () => {
    recordHostEvidence()

    expect(hasHostEvidence.value).toBe(true)
    expect(localStorage.getItem('pickleball.roleEvidence.host')).toBe('true')
  })

  it('recordHostEvidence is idempotent across repeated calls', () => {
    recordHostEvidence()
    recordHostEvidence()

    expect(hasHostEvidence.value).toBe(true)
    expect(localStorage.getItem('pickleball.roleEvidence.host')).toBe('true')
  })

  it('__resetHostEvidenceForTests clears both the ref and localStorage', () => {
    recordHostEvidence()
    __resetHostEvidenceForTests()

    expect(hasHostEvidence.value).toBe(false)
    expect(localStorage.getItem('pickleball.roleEvidence.host')).toBeNull()
  })
})
