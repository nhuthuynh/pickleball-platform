import { describe, it, expect } from 'vitest'
import { effectScope } from 'vue'
import { useBreakpoint, BREAKPOINTS } from '../useBreakpoint'

/**
 * A minimal `window.matchMedia` mock: evaluates `min-width`/`max-width`
 * against an in-memory viewport width and supports `addEventListener`/
 * `removeEventListener('change', ...)` so useBreakpoint's live-update path
 * (a real resize or orientation change) is exercised too, not just the
 * initial resolve.
 */
function createMatchMediaMock(initialWidth: number) {
  let width = initialWidth
  const listenersByQuery = new Map<string, Set<(event: { matches: boolean; media: string }) => void>>()
  const mqlsByQuery = new Map<string, MediaQueryList>()

  function matchesQuery(query: string): boolean {
    const min = Number(query.match(/min-width:\s*(\d+)px/)?.[1] ?? -Infinity)
    const max = Number(query.match(/max-width:\s*(\d+)px/)?.[1] ?? Infinity)
    return width >= min && width <= max
  }

  function matchMedia(query: string): MediaQueryList {
    let mql = mqlsByQuery.get(query)
    if (!mql) {
      const listeners = new Set<(event: { matches: boolean; media: string }) => void>()
      listenersByQuery.set(query, listeners)
      mql = {
        get matches() {
          return matchesQuery(query)
        },
        media: query,
        addEventListener: (_type: string, cb: EventListenerOrEventListenerObject) =>
          listeners.add(cb as unknown as (event: { matches: boolean; media: string }) => void),
        removeEventListener: (_type: string, cb: EventListenerOrEventListenerObject) =>
          listeners.delete(cb as unknown as (event: { matches: boolean; media: string }) => void),
      } as unknown as MediaQueryList
      mqlsByQuery.set(query, mql)
    }
    return mql
  }

  function setWidth(newWidth: number) {
    width = newWidth
    for (const [query, listeners] of listenersByQuery) {
      for (const cb of listeners) cb({ matches: matchesQuery(query), media: query })
    }
  }

  return { matchMedia, setWidth }
}

describe('useBreakpoint', () => {
  it('exposes the exact breakpoint values from the external design handoff Platform Notes', () => {
    expect(BREAKPOINTS).toEqual({ iphoneMax: 599, ipadMin: 768, ipadMax: 1180, webMin: 1280 })
  })

  it('resolves "iphone" below 600px', () => {
    const { matchMedia } = createMatchMediaMock(375)
    const { breakpoint, isIphone, isIpad, isWeb } = useBreakpoint({ matchMedia })
    expect(breakpoint.value).toBe('iphone')
    expect(isIphone.value).toBe(true)
    expect(isIpad.value).toBe(false)
    expect(isWeb.value).toBe(false)
  })

  it('resolves "ipad" within 768-1180px inclusive', () => {
    expect(useBreakpoint({ matchMedia: createMatchMediaMock(768).matchMedia }).breakpoint.value).toBe('ipad')
    expect(useBreakpoint({ matchMedia: createMatchMediaMock(1180).matchMedia }).breakpoint.value).toBe('ipad')
  })

  it('resolves "web" at 1280px and above', () => {
    const { matchMedia } = createMatchMediaMock(1440)
    const { breakpoint, isWeb } = useBreakpoint({ matchMedia })
    expect(breakpoint.value).toBe('web')
    expect(isWeb.value).toBe(true)
  })

  it('resolves "between" in the two gaps the reviewed system leaves open (600-767px, 1181-1279px)', () => {
    expect(useBreakpoint({ matchMedia: createMatchMediaMock(680).matchMedia }).breakpoint.value).toBe('between')
    expect(useBreakpoint({ matchMedia: createMatchMediaMock(1200).matchMedia }).breakpoint.value).toBe('between')
  })

  it('reacts live to a matchMedia change event (resize/orientation change)', () => {
    const { matchMedia, setWidth } = createMatchMediaMock(375)
    const { breakpoint } = useBreakpoint({ matchMedia })
    expect(breakpoint.value).toBe('iphone')

    setWidth(1440)
    expect(breakpoint.value).toBe('web')
  })

  it('removes its matchMedia listeners on scope dispose (no leak across component unmounts)', () => {
    const { matchMedia, setWidth } = createMatchMediaMock(375)
    const scope = effectScope()
    let breakpoint!: ReturnType<typeof useBreakpoint>['breakpoint']

    scope.run(() => {
      breakpoint = useBreakpoint({ matchMedia }).breakpoint
    })
    expect(breakpoint.value).toBe('iphone')

    scope.stop()
    setWidth(1440)
    // Listener was removed on scope stop, so the ref must not have updated.
    expect(breakpoint.value).toBe('iphone')
  })
})
