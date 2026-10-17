/**
 * Make a relative URL resolve against the page origin, the way a browser
 * does — because this suite's runtime does not.
 *
 * #339 made the API client default to a RELATIVE base URL (same-origin,
 * proxied to the API by nginx and by the Vite dev server). In a browser
 * `new Request('/v1/facilities')` resolves against `document.baseURI` and is
 * perfectly ordinary. Under vitest it throws:
 *
 *     TypeError: Failed to parse URL from /v1/facilities
 *
 * because jsdom supplies no `fetch`, so the GLOBAL `Request` is Node's
 * (undici), and undici has no document to resolve against — it requires an
 * absolute URL. Ten tests in `src/api/__tests__/` broke on that alone, none
 * of them for a reason a browser would ever reproduce.
 *
 * The alternative was to give every one of those tests an explicit absolute
 * `baseUrl`, which would have deleted the coverage that matters: two of them
 * exist specifically to assert what the DEFAULT does. So the environment is
 * corrected instead of the assertions.
 *
 * This shim only touches inputs that are already relative; an absolute URL
 * is passed through untouched, so nothing else in the suite changes
 * behaviour.
 */
const NativeRequest = globalThis.Request

class OriginRelativeRequest extends NativeRequest {
  constructor(input: RequestInfo | URL, init?: RequestInit) {
    if (typeof input === 'string' && input.startsWith('/')) {
      super(new URL(input, globalThis.location?.origin ?? 'http://localhost').toString(), init)
      return
    }
    super(input, init)
  }
}

globalThis.Request = OriginRelativeRequest as unknown as typeof Request
