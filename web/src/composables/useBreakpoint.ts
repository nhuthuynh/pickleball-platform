import { ref, computed, onScopeDispose, getCurrentScope, type Ref, type ComputedRef } from 'vue'

/**
 * The three responsive breakpoints, verbatim from the external design
 * handoff's Platform Notes (docs/design/handoff-2026-08/README.md),
 * adopted by docs/design/v1-external-reference-reconciliation.md and
 * restated as the T7 sprint goal's own acceptance criterion. Single source
 * of truth also mirrored in src/styles/breakpoints.css's comment block —
 * change both together if these ever move.
 */
export const BREAKPOINTS = {
  iphoneMax: 599,
  ipadMin: 768,
  ipadMax: 1180,
  webMin: 1280,
} as const

export type BreakpointName = 'iphone' | 'ipad' | 'web' | 'between'

const QUERIES = {
  iphone: `(max-width: ${BREAKPOINTS.iphoneMax}px)`,
  ipad: `(min-width: ${BREAKPOINTS.ipadMin}px) and (max-width: ${BREAKPOINTS.ipadMax}px)`,
  web: `(min-width: ${BREAKPOINTS.webMin}px)`,
} as const

export interface UseBreakpointResult {
  /** Which of the three reviewed breakpoints currently matches. `'between'`
   * covers the two real gaps in the reviewed system (600-767px and
   * 1181-1279px) — not a bug in this composable, see README. */
  breakpoint: Ref<BreakpointName>
  isIphone: ComputedRef<boolean>
  isIpad: ComputedRef<boolean>
  isWeb: ComputedRef<boolean>
}

/**
 * Tracks which of the three reviewed breakpoints the current viewport
 * matches, via `matchMedia` change listeners (not a resize-event width
 * poll, which would fire far more often than needed). Accepts an injectable
 * `win` (defaults to `window`) so it's testable by mocking `matchMedia`
 * without a real browser viewport — see
 * src/composables/__tests__/useBreakpoint.spec.ts.
 */
export function useBreakpoint(win: Pick<Window, 'matchMedia'> = window): UseBreakpointResult {
  const mediaQueryLists = {
    iphone: win.matchMedia(QUERIES.iphone),
    ipad: win.matchMedia(QUERIES.ipad),
    web: win.matchMedia(QUERIES.web),
  }

  function resolve(): BreakpointName {
    if (mediaQueryLists.iphone.matches) return 'iphone'
    if (mediaQueryLists.ipad.matches) return 'ipad'
    if (mediaQueryLists.web.matches) return 'web'
    return 'between'
  }

  const breakpoint = ref<BreakpointName>(resolve())

  function update() {
    breakpoint.value = resolve()
  }

  const lists = Object.values(mediaQueryLists)
  for (const mql of lists) {
    mql.addEventListener('change', update)
  }

  // Only register cleanup when called inside an active effect scope (a
  // component's setup(), or a test wrapped in its own effectScope()) — a
  // bare call (e.g. outside any component, as some tests below do) would
  // otherwise just emit a harmless-but-noisy dev warning from Vue.
  if (getCurrentScope()) {
    onScopeDispose(() => {
      for (const mql of lists) {
        mql.removeEventListener('change', update)
      }
    })
  }

  return {
    breakpoint,
    isIphone: computed(() => breakpoint.value === 'iphone'),
    isIpad: computed(() => breakpoint.value === 'ipad'),
    isWeb: computed(() => breakpoint.value === 'web'),
  }
}
