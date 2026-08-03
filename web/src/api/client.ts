import createFetchClient, { type Client } from 'openapi-fetch'

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
  return createFetchClient<Paths>({
    baseUrl: options.baseUrl ?? DEFAULT_API_BASE_URL,
    fetch: options.fetch,
  })
}
