import { describe, it, expect } from 'vitest'
import { mapToUserProfile, type RawUser } from '../identity'

const rawUser: RawUser = {
  id: 'user-1',
  displayName: 'Alex Rivera',
  roles: ['ROLE_PLAYER'],
  selfReportedStartingLevel: 3,
}

describe('mapToUserProfile', () => {
  it('maps id/displayName/selfReportedStartingLevel', () => {
    expect(mapToUserProfile(rawUser)).toEqual({
      id: 'user-1',
      displayName: 'Alex Rivera',
      selfReportedStartingLevel: 3,
    })
  })

  it('maps every valid level in the domain\'s 1..5 range', () => {
    for (const level of [1, 2, 3, 4, 5]) {
      expect(mapToUserProfile({ ...rawUser, selfReportedStartingLevel: level }).selfReportedStartingLevel).toBe(
        level,
      )
    }
  })

  // T10.5 instructions #4: no fabricated fields. The wire's zero value (a
  // proto int32's default when the field is omitted/never set) must map to
  // `null` — an honest "not set" — never to a displayed "0" or, worse, a
  // silently-substituted "1" that would look like a real player choice.
  it('maps a zero (wire default / unset) level to null, not a fabricated default', () => {
    expect(mapToUserProfile({ ...rawUser, selfReportedStartingLevel: 0 }).selfReportedStartingLevel).toBeNull()
  })

  it('maps a missing level field to null', () => {
    const { selfReportedStartingLevel, ...rest } = rawUser
    expect(mapToUserProfile(rest).selfReportedStartingLevel).toBeNull()
  })

  // Defense in depth (see this function's own doc comment): domain.NewUser
  // never actually persists a value outside 1..5, but a malformed/partial
  // response should still never be displayed as if it were real.
  it('maps an out-of-range level (e.g. a malformed response) to null rather than displaying it as real', () => {
    expect(mapToUserProfile({ ...rawUser, selfReportedStartingLevel: 6 }).selfReportedStartingLevel).toBeNull()
    expect(mapToUserProfile({ ...rawUser, selfReportedStartingLevel: -1 }).selfReportedStartingLevel).toBeNull()
  })

  it('defaults missing id/displayName to empty strings', () => {
    expect(mapToUserProfile({})).toEqual({ id: '', displayName: '', selfReportedStartingLevel: null })
  })
})
