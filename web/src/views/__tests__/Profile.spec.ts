import { describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { nextTick } from 'vue'
import Profile from '../Profile.vue'
import { MOCK_PLAYER_ID } from '../../composables/useProfile'
import type { IdentityClient } from '../../api/identityClient'
import { findGenderControls } from '../../test-support/genderControlAssertions'

function rawUser(level: number, displayName = 'Alex Rivera') {
  return { id: MOCK_PLAYER_ID, displayName, roles: ['ROLE_PLAYER'], selfReportedStartingLevel: level }
}

function makeClient(overrides: {
  get?: () => unknown
  post?: (body: unknown) => unknown
} = {}): IdentityClient {
  return {
    GET: vi.fn(
      async () =>
        overrides.get?.() ?? { data: { user: rawUser(3) }, error: undefined, response: { status: 200 } },
    ),
    POST: vi.fn(async (_path: string, options: { body: unknown }) => {
      return (
        overrides.post?.(options.body) ?? {
          data: { user: rawUser(4) },
          error: undefined,
          response: { status: 200 },
        }
      )
    }),
  } as unknown as IdentityClient
}

function mountProfile(client?: IdentityClient) {
  return mount(Profile, { props: { client } })
}

describe('Profile — loading a real profile', () => {
  it('shows a loading state on mount', async () => {
    const wrapper = mountProfile(makeClient({ get: () => new Promise(() => {}) }))
    await nextTick()
    expect(wrapper.get('[role="status"]').text()).toContain('Loading')
  })

  it('shows the read-only display name once loaded', async () => {
    const wrapper = mountProfile(makeClient())
    await flushPromises()
    expect(wrapper.get('[data-testid="display-name"]').text()).toBe('Alex Rivera')
  })

  it('renders the current self-reported level when it is set', async () => {
    const wrapper = mountProfile(makeClient({ get: () => ({ data: { user: rawUser(4) }, error: undefined, response: { status: 200 } }) }))
    await flushPromises()

    expect(wrapper.get('[data-testid="level-value"]').text()).toContain('4')
    expect(wrapper.find('[data-testid="level-empty-state"]').exists()).toBe(false)
  })

  // T10.5 instructions #4: honest empty state, never a fabricated default.
  it('renders an honest "Not set" state, never a default value, when the level is unset', async () => {
    const wrapper = mountProfile(
      makeClient({ get: () => ({ data: { user: rawUser(0) }, error: undefined, response: { status: 200 } }) }),
    )
    await flushPromises()

    expect(wrapper.get('[data-testid="level-empty-state"]').text()).toContain('Not set')
    expect(wrapper.find('[data-testid="level-value"]').exists()).toBe(false)
    // Never a fabricated "Level 1" (or any level) standing in for the unset value.
    expect(wrapper.text()).not.toMatch(/Level 1\b/)
  })

  it('renders an honest "no profile yet" state on a 404, not fabricated data', async () => {
    const wrapper = mountProfile(
      makeClient({ get: () => ({ data: undefined, error: { message: 'not found' }, response: { status: 404 } }) }),
    )
    await flushPromises()

    expect(wrapper.get('[data-testid="profile-not-found"]').text().toLowerCase()).toContain("don't have a profile")
    expect(wrapper.find('[data-testid="display-name"]').exists()).toBe(false)
  })

  it('shows a retry action on a non-404 failure', async () => {
    const wrapper = mountProfile(
      makeClient({ get: () => ({ data: undefined, error: { message: 'boom' }, response: { status: 500 } }) }),
    )
    await flushPromises()

    expect(wrapper.find('[role="alert"]').exists()).toBe(true)
    expect(wrapper.get('.profile__retry').text()).toContain('Try again')
  })
})

describe('Profile — editing the self-reported starting level', () => {
  it('disables Save until a level is chosen', async () => {
    const wrapper = mountProfile(
      makeClient({ get: () => ({ data: { user: rawUser(0) }, error: undefined, response: { status: 200 } }) }),
    )
    await flushPromises()

    expect(wrapper.get('[data-testid="save-level"]').attributes('disabled')).toBeDefined()

    await wrapper.get('#self-reported-level').setValue('3')
    expect(wrapper.get('[data-testid="save-level"]').attributes('disabled')).toBeUndefined()
  })

  it('calls UpdateSelfReportedLevel with the chosen level and the MOCK_PLAYER_ID as both user and actor, and announces success', async () => {
    const post = vi.fn((body: unknown) => ({ data: { user: rawUser(5) }, error: undefined, response: { status: 200 } }))
    const wrapper = mountProfile(makeClient({ get: () => ({ data: { user: rawUser(2) }, error: undefined, response: { status: 200 } }), post }))
    await flushPromises()

    await wrapper.get('#self-reported-level').setValue('5')
    await wrapper.get('.profile__level-form').trigger('submit')
    await flushPromises()

    expect(post).toHaveBeenCalledWith({ actorUserId: MOCK_PLAYER_ID, selfReportedStartingLevel: 5 })
    expect(wrapper.get('[role="status"]').text()).toContain('saved')
    expect(wrapper.get('[data-testid="level-value"]').text()).toContain('5')
  })

  it('surfaces a save error and leaves the previously saved level displayed', async () => {
    const wrapper = mountProfile(
      makeClient({
        get: () => ({ data: { user: rawUser(2) }, error: undefined, response: { status: 200 } }),
        post: () => ({ data: undefined, error: { message: 'not your profile' }, response: { status: 403 } }),
      }),
    )
    await flushPromises()

    await wrapper.get('#self-reported-level').setValue('4')
    await wrapper.get('.profile__level-form').trigger('submit')
    await flushPromises()

    expect(wrapper.find('[role="alert"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="level-value"]').text()).toContain('2')
  })
})

// T10.5 instructions #3: explicitly no matching UI control of any kind.
// Mirrors GameCreation.spec.ts's/CompetitionCreation.spec.ts's/
// CompetitionManage.spec.ts's own absence-assertion discipline — assert
// absence in a test, not just by omission.
describe('Profile — no matching control of any kind', () => {
  it('has no range/slider input anywhere (no level-range control)', async () => {
    const wrapper = mountProfile(makeClient())
    await flushPromises()

    expect(wrapper.find('input[type="range"]').exists()).toBe(false)
  })

  it('has no gender field or control of any kind', async () => {
    const wrapper = mountProfile(makeClient())
    await flushPromises()

    // "gender-mix" legitimately appears in the disclosure prose (naming
    // ADR-0012's Q2) — that is not a control, so it is not banned here;
    // what's asserted (via the shared helper, checking native id/name,
    // aria-label, label association, and ARIA radiogroup/radio — see that
    // file's header for why an id/name-only check missed real shapes) is
    // that no *element* is gender-related.
    expect(findGenderControls(wrapper)).toHaveLength(0)
  })

  it('has no auto-match toggle, "find a match" button, or seeding/bracket control', async () => {
    const wrapper = mountProfile(makeClient())
    await flushPromises()

    const text = wrapper.text().toLowerCase()
    expect(text).not.toContain('auto-match')
    expect(text).not.toContain('automatch')
    expect(text).not.toContain('find a match')
    expect(text).not.toContain('seeding')
    expect(text).not.toContain('bracket')
    expect(wrapper.findAll('button').map((b) => b.text().toLowerCase())).not.toContain('find a match')
  })

  // The one <select> on this screen is the self-reported level itself
  // (explicitly in scope, T10.5 instructions #1) — proves it is a plain
  // 1..5 choice, not disguised matching UI.
  it('has exactly one <select>, and it is the 1..5 self-reported level, not a matching control', async () => {
    const wrapper = mountProfile(makeClient())
    await flushPromises()

    const selects = wrapper.findAll('select')
    expect(selects).toHaveLength(1)
    expect(selects[0]!.attributes('id')).toBe('self-reported-level')
    const optionTexts = selects[0]!.findAll('option').map((o) => o.text())
    expect(optionTexts).toEqual(['Choose a level', 'Level 1', 'Level 2', 'Level 3', 'Level 4', 'Level 5'])
  })
})

describe('Profile — the ADR-0012-precise disclosure', () => {
  it('names that Identity now exists and the two specific escalated decisions, not generic "coming soon" copy', async () => {
    const wrapper = mountProfile(makeClient())
    await flushPromises()

    const text = wrapper.text()
    expect(text).toContain('Identity now exists')
    expect(text).toContain('Player Level formula is weighted')
    expect(text).toContain('gender-mix matching is in scope')
    expect(text.toLowerCase()).not.toContain('coming soon')
    expect(text.toLowerCase()).not.toContain('next sprint')
  })
})
