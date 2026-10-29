// #344: the signed-in player can recover their own user id.
import { describe, it, expect, beforeEach, vi } from 'vitest'

import { ensureCurrentUserId } from '../currentUser'
import {
  authUserId,
  profileMissing,
  rememberUserId,
  clearProfileMissing,
  signOut,
} from '../../state/authSession'

/** A client whose GET returns whatever the test hands it. */
function stubClient(result: { data?: unknown; response?: { status: number } }) {
  return { GET: vi.fn().mockResolvedValue(result) } as never
}

describe('ensureCurrentUserId', () => {
  beforeEach(() => {
    // signOut clears the stored token AND the stored user id, which is the
    // "fresh browser" state every case here starts from.
    signOut()
    clearProfileMissing()
  })

  it('makes no request when the id is already known', async () => {
    rememberUserId('known-id')
    const client = stubClient({ data: { user: { id: 'from-server' } } })

    expect(await ensureCurrentUserId(client)).toBe('already-known')
    expect((client as unknown as { GET: ReturnType<typeof vi.fn> }).GET).not.toHaveBeenCalled()
    expect(authUserId.value).toBe('known-id')
  })

  it('resolves the id from the server and stores it', async () => {
    const client = stubClient({
      data: { user: { id: 'user-42' } },
      response: { status: 200 },
    })

    expect(await ensureCurrentUserId(client)).toBe('resolved')
    expect(authUserId.value).toBe('user-42')
    expect(profileMissing.value).toBe(false)
  })

  // The case this whole module exists for: signed in, account exists
  // server-side, but this browser has no id. It must come back resolved
  // rather than sending the player to sign up again.
  it('recovers after site data is cleared', async () => {
    expect(authUserId.value).toBeNull()
    const client = stubClient({ data: { user: { id: 'user-42' } }, response: { status: 200 } })

    await ensureCurrentUserId(client)

    expect(authUserId.value).toBe('user-42')
  })

  it('raises the profile banner on 404, and only on 404', async () => {
    const client = stubClient({ data: undefined, response: { status: 404 } })

    expect(await ensureCurrentUserId(client)).toBe('no-profile')
    expect(profileMissing.value).toBe(true)
    expect(authUserId.value).toBeNull()
  })

  // A 401 or a 5xx must not tell the player to finish signing up: that
  // would send them to a screen which cannot help, for a problem that is
  // not theirs.
  it.each([401, 500, 503])('does not raise the profile banner on %i', async (status) => {
    const client = stubClient({ data: undefined, response: { status } })

    expect(await ensureCurrentUserId(client)).toBe('unavailable')
    expect(profileMissing.value).toBe(false)
    expect(authUserId.value).toBeNull()
  })

  it('does not raise the profile banner when the call throws no response at all', async () => {
    const client = stubClient({ data: undefined })

    expect(await ensureCurrentUserId(client)).toBe('unavailable')
    expect(profileMissing.value).toBe(false)
  })
})
