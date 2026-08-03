import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import { createRouter, createMemoryHistory } from 'vue-router'
import AppNav from '../AppNav.vue'
import { routes } from '../../../router'

/**
 * Static `window.matchMedia` stand-in for a fixed viewport width — same
 * pattern as DiscoverFacilities.spec.ts's `matchMediaForWidth` (this
 * component only needs the breakpoint resolved once at mount for these
 * tests, not live resize updates — those are useBreakpoint's own unit
 * tests, see composables/__tests__/useBreakpoint.spec.ts).
 */
function matchMediaForWidth(width: number): Pick<Window, 'matchMedia'> {
  return {
    matchMedia: (query: string) => {
      const min = Number(query.match(/min-width:\s*(\d+)px/)?.[1] ?? -Infinity)
      const max = Number(query.match(/max-width:\s*(\d+)px/)?.[1] ?? Infinity)
      return {
        matches: width >= min && width <= max,
        media: query,
        addEventListener: () => {},
        removeEventListener: () => {},
      } as unknown as MediaQueryList
    },
  }
}

async function mountAtWidth(width: number) {
  const router = createRouter({ history: createMemoryHistory(), routes })
  router.push('/facilities')
  await router.isReady()
  return mount(AppNav, {
    props: { win: matchMediaForWidth(width) },
    global: { plugins: [router] },
  })
}

describe('AppNav', () => {
  it('renders the persistent sidebar on web (>=1280px)', async () => {
    const wrapper = await mountAtWidth(1440)
    expect(wrapper.get('.app-nav').attributes('data-nav-variant')).toBe('sidebar')
    expect(wrapper.find('.app-nav--sidebar').exists()).toBe(true)
    expect(wrapper.find('.app-nav--tabbar').exists()).toBe(false)
  })

  it('renders a bottom tab bar on iPhone (<600px)', async () => {
    const wrapper = await mountAtWidth(375)
    expect(wrapper.get('.app-nav').attributes('data-nav-variant')).toBe('tabbar')
    expect(wrapper.find('.app-nav--tabbar').exists()).toBe(true)
    expect(wrapper.find('.app-nav--sidebar').exists()).toBe(false)
  })

  it('renders a bottom tab bar on iPad (768-1180px)', async () => {
    const wrapper = await mountAtWidth(1024)
    expect(wrapper.get('.app-nav').attributes('data-nav-variant')).toBe('tabbar')
    expect(wrapper.find('.app-nav--tabbar').exists()).toBe(true)
    expect(wrapper.find('.app-nav--sidebar').exists()).toBe(false)
  })

  it('exposes exactly the Discover / Bookings / Games / Profile tabs, in that order', async () => {
    const wrapper = await mountAtWidth(1440)
    const labels = wrapper.findAll('.app-nav__item').map((item) => item.text())
    expect(labels).toEqual(['Discover', 'Bookings', 'Games', 'Profile'])
  })

  it("links Bookings and Profile to placeholder routes rather than omitting them", async () => {
    const wrapper = await mountAtWidth(1440)
    const links = wrapper.findAll('.app-nav__item')
    const byLabel = Object.fromEntries(links.map((l) => [l.text(), l.attributes('href')]))
    expect(byLabel['Discover']).toBe('/facilities')
    expect(byLabel['Bookings']).toBe('/bookings')
    expect(byLabel['Games']).toBe('/games')
    expect(byLabel['Profile']).toBe('/profile')
  })
})
