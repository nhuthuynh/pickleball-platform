import createFetchClient, { type Client, type Middleware } from 'openapi-fetch'

import { authToken, recordAuthRejected } from '../state/authSession'

/**
 * Base URL for the gRPC-gateway REST API. Overridable via the Vite env var
 * `VITE_API_BASE_URL` (see web/README.md); defaults to the local
 * `cmd/server` dev address.
 */
export const DEFAULT_API_BASE_URL: string =
  (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? 'http://localhost:8080'

export interface CreateTypedClientOptions {
  baseUrl?: string
  /** Injectable for tests; defaults to the ambient `fetch`. */
  fetch?: (input: Request) => Promise<Response>
  /**
   * Opt out of the auth middleware. Only for tests that assert the unauthed
   * wire shape — production code must not set it, which is why it is not
   * merely `middleware?: Middleware[]`.
   */
  withoutAuth?: boolean
}

/**
 * Attaches the session bearer token, and notices a rejected one.
 *
 * **T65.1.** Before this, the client could not authenticate at all and every
 * one of the server's 31 authenticated RPCs answered `Unauthenticated` — see
 * `state/authSession.ts` for why that was invisible to 717 green tests.
 *
 * Two deliberate choices:
 *
 *   - **The header is attached here, once**, not per composable. Twelve call
 *     sites write; a per-call-site header is twelve places to forget one.
 *   - **A 401 clears the token and raises one signal.** A stale token that
 *     stays in `localStorage` makes every subsequent call fail the same way
 *     with no explanation, which is the shape of bug a user cannot report
 *     usefully.
 *
 * It does **not** retry, refresh or redirect: there is no refresh token and no
 * identity provider (#145, #137). A 401 is surfaced, not papered over.
 */
export const authMiddleware: Middleware = {
  onRequest({ request }) {
    const token = authToken.value
    if (token !== null) {
      request.headers.set('Authorization', `Bearer ${token}`)
    }
    return request
  },
  onResponse({ response }) {
    if (response.status === 401) {
      recordAuthRejected()
    }
    return response
  },
}

/**
 * Thin, typed wrapper around `openapi-fetch`'s client factory — this is
 * the "thin fetch wrapper" half of T7.1's chosen client-generation
 * approach (`openapi-typescript` for types + this file for the actual HTTP
 * calls; see web/README.md "Typed client generation" for the full
 * rationale).
 *
 * `Paths` is the `paths` type `openapi-typescript` emits for one bounded
 * context's swagger.json (booking/payments/socialplay — see
 * web/src/api/generated/, gitignored; run `npm run generate:client` to
 * produce it, mirroring internal/gen/**'s own convention per CLAUDE.md
 * rule 6). Kept generic and separate from any one generated module so
 * it's (a) unit-testable against a small fixture `paths` shape without
 * depending on generated output being present on disk, and (b) reused
 * identically by every bounded context's concrete client (see
 * bookingClient.ts, paymentsClient.ts, socialplayClient.ts) instead of
 * each one hand-rolling its own fetch call.
 */
export function createTypedClient<Paths extends object>(
  options: CreateTypedClientOptions = {},
): Client<Paths> {
  const client = createFetchClient<Paths>({
    baseUrl: options.baseUrl ?? DEFAULT_API_BASE_URL,
    fetch: options.fetch,
  })
  if (options.withoutAuth !== true) {
    client.use(authMiddleware)
  }
  return client
}
