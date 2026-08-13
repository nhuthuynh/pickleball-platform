import { describe, it, expect, vi } from 'vitest'
import { nextTick } from 'vue'
import { mount, flushPromises, RouterLinkStub } from '@vue/test-utils'
import CompetitionLanding from '../CompetitionLanding.vue'
import type { CompetitionsClient } from '../../api/competitionsClient'
import type { IdentityClient } from '../../api/identityClient'
import type { FacilitiesClient } from '../../api/facilitiesClient'

const COMPETITION = {
  id: 'c1',
  hostId: 'host-1',
  name: 'Autumn Doubles Ladder',
  venueFacilityId: 'facility-1',
  sessions: [{ startsAt: '2026-09-01T09:00:00Z', endsAt: '2026-09-01T12:00:00Z', courtIds: ['court-1'] }],
  capacity: 16,
  guestAllowance: 2,
  paymentMethod: 'PAYMENT_METHOD_CASH',
  entryFee: { amountCents: '2500', currencyCode: 'USD' },
  format: 'COMPETITION_FORMAT_DOUBLES',
  status: 'COMPETITION_STATUS_SCHEDULED',
}

function fakeClient(handlers: { byToken?: () => unknown; list?: () => unknown } = {}): CompetitionsClient {
  const GET = vi.fn(async (path: string) => {
    if (path === '/v1/competitions/by-share-token/{shareToken}') {
      return (
        handlers.byToken?.() ?? {
          data: { competition: COMPETITION },
          error: undefined,
          response: { status: 200 },
        }
      )
    }
    if (path === '/v1/competitions') {
      return handlers.list?.() ?? { data: { competitions: [] }, error: undefined, response: { status: 200 } }
    }
    throw new Error(`unexpected GET ${path}`)
  })
  return { GET, POST: vi.fn() } as unknown as CompetitionsClient
}

/** T10.8 (closes #98): this landing is the audience the ticket's story
 * names directly — "a stranger arriving from a social link with no other
 * context" — so it forwards these to CompetitionDetailPanel too. Fakes
 * here keep the test suite from making a real network call. */
function fakeIdentityClient(): IdentityClient {
  return {
    GET: vi.fn(async () => ({ data: { user: { displayName: 'Ada Lovelace' } } })),
    POST: vi.fn(),
  } as unknown as IdentityClient
}

function fakeFacilitiesClient(): FacilitiesClient {
  return {
    GET: vi.fn(async () => ({ data: { facility: { name: 'Riverside Courts' } } })),
    POST: vi.fn(),
  } as unknown as FacilitiesClient
}

function mountLanding(client: CompetitionsClient, shareToken = 'tok-123') {
  return mount(CompetitionLanding, {
    props: { client, identityClient: fakeIdentityClient(), facilitiesClient: fakeFacilitiesClient(), shareToken },
    global: { stubs: { RouterLink: RouterLinkStub } },
  })
}

describe('CompetitionLanding — the happy path', () => {
  it('resolves the token and renders the competition with the entry action', async () => {
    const wrapper = mountLanding(
      fakeClient({
        list: () => ({
          data: { competitions: [{ competition: COMPETITION, spotsLeft: 4 }] },
          error: undefined,
          response: { status: 200 },
        }),
      }),
    )
    await flushPromises()

    expect(wrapper.text()).toContain('Autumn Doubles Ladder')
    expect(wrapper.text()).toContain('4 spots left')
    expect(wrapper.find('.competition-entry__form').exists()).toBe(true)

    // T10.8 (closes #98): a stranger arriving from this link sees a real
    // Host and venue name, not raw ids — the whole point of this ticket.
    expect(wrapper.text()).toContain('Ada Lovelace')
    expect(wrapper.text()).toContain('Riverside Courts')
    expect(wrapper.text()).not.toContain('host-1')
    expect(wrapper.text()).not.toContain('facility-1')
  })

  it('shows a designed loading state first, never a blank screen', async () => {
    const wrapper = mountLanding(fakeClient())
    await nextTick()
    expect(wrapper.get('[role="status"]').text()).toContain('Loading this competition')
  })

  // T11.7 fix: this view used to root itself in a <main>, which — mounted
  // standalone here — looks harmless, but this screen ALWAYS renders inside
  // App.vue's own <main class="app-shell__main">, making it a second, nested
  // <main> landmark on the real page (invalid: WCAG 1.3.1/landmark
  // structure — axe-core's landmark-main-is-top-level/
  // landmark-no-duplicate-main/landmark-unique rules all fired on this,
  // caught by src/__tests__/accessibility.spec.ts's full-App route sweep,
  // which this component-level test alone could never catch since it has no
  // outer <main> to nest inside). Asserted here too, at the component level,
  // so a regression is caught right at its source even before the full-page
  // sweep would also catch it.
  it('roots itself in a <section>, not a <main> — this view always nests inside the app shell\'s own <main>', async () => {
    const wrapper = mountLanding(fakeClient())
    await flushPromises()
    expect(wrapper.element.tagName).toBe('SECTION')
    expect(wrapper.find('main').exists()).toBe(false)
  })

  // WCAG 2.4.7 / 2.1.1: this is the one screen a brand-new visitor may hit
  // first, with no prior navigation state to inherit focus from.
  it('moves focus to its own heading on arrival so keyboard users start somewhere real', async () => {
    // `attachTo` is required for a focus assertion: an element that is not
    // in the document cannot become `document.activeElement`, in jsdom or in
    // a real browser.
    const wrapper = mount(CompetitionLanding, {
      props: {
        client: fakeClient(),
        identityClient: fakeIdentityClient(),
        facilitiesClient: fakeFacilitiesClient(),
        shareToken: 'tok-123',
      },
      global: { stubs: { RouterLink: RouterLinkStub } },
      attachTo: document.body,
    })
    await flushPromises()

    const heading = wrapper.get('.competition-landing__heading')
    expect(heading.attributes('tabindex')).toBe('-1')
    expect(document.activeElement).toBe(heading.element)
    wrapper.unmount()
  })
})

describe('CompetitionLanding — the unknown-token state', () => {
  it('says the link is invalid or expired and offers browsing instead', async () => {
    const wrapper = mountLanding(
      fakeClient({
        byToken: () => ({ data: undefined, error: { message: 'not found' }, response: { status: 404 } }),
      }),
    )
    await flushPromises()

    const alert = wrapper.get('[role="alert"]').text()
    expect(alert).toMatch(/invalid or (has )?expired/i)
    expect(wrapper.find('.competition-entry__form').exists()).toBe(false)

    const browse = wrapper.getComponent(RouterLinkStub)
    expect(browse.props('to')).toBe('/competitions')
  })
})

describe('CompetitionLanding — the cancelled-competition state', () => {
  it('renders an honest cancelled state with a route onward, not a 404 and not a dead button', async () => {
    const wrapper = mountLanding(
      fakeClient({
        byToken: () => ({
          data: { competition: { ...COMPETITION, status: 'COMPETITION_STATUS_CANCELLED' } },
          error: undefined,
          response: { status: 200 },
        }),
      }),
    )
    await flushPromises()

    // Honest: it names the competition and says it was cancelled…
    expect(wrapper.text()).toContain('Autumn Doubles Ladder')
    expect(wrapper.get('.competition-landing__cancelled').text()).toMatch(/cancelled/i)
    // …offers a real way onward…
    expect(wrapper.getComponent(RouterLinkStub).props('to')).toBe('/competitions')
    // …and does not present an entry form at all (rather than a disabled one).
    expect(wrapper.find('.competition-entry__form').exists()).toBe(false)
    expect(wrapper.find('button[disabled]').exists()).toBe(false)
  })
})

describe('CompetitionLanding — the error state', () => {
  it('offers a retry when the server fails, and distinguishes it from a bad link', async () => {
    const wrapper = mountLanding(
      fakeClient({
        byToken: () => ({ data: undefined, error: { message: 'boom' }, response: { status: 500 } }),
      }),
    )
    await flushPromises()

    expect(wrapper.get('[role="alert"]').text()).toContain('Could not load')
    expect(wrapper.get('[role="alert"]').text()).not.toMatch(/invalid or expired/i)
    expect(wrapper.find('.competition-landing__retry').exists()).toBe(true)
  })
})

describe('CompetitionLanding — a missing token in the URL', () => {
  it('treats an empty token as an invalid link rather than calling the API', async () => {
    const client = fakeClient()
    const wrapper = mountLanding(client, '')
    await flushPromises()

    expect(client.GET).not.toHaveBeenCalled()
    expect(wrapper.get('[role="alert"]').text()).toMatch(/invalid or (has )?expired/i)
  })
})
