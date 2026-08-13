// Concrete typed client for the Identity/Users bounded context (T10.2's
// real CreateUser/GetUser/UpdateSelfReportedLevel REST surface). Mirrors
// facilitiesClient.ts/socialplayClient.ts's shape exactly.
//
// NOTE: this file only type-checks once `npm run generate:client` has been
// run (it needs `./generated/identity` to exist — see web/README.md).
// `import type` is erased at build time (no runtime import), so — same as
// the other concrete clients — this module is safe to import from
// component code that Vitest exercises on a fresh clone before
// `generate:client` has ever run; only `npm run build`'s type-check step
// needs the generated file present.
import type { paths as IdentityPaths } from './generated/identity'
import { createTypedClient, type CreateTypedClientOptions } from './client'

export function createIdentityClient(options: CreateTypedClientOptions = {}) {
  return createTypedClient<IdentityPaths>(options)
}

export const identityClient = createIdentityClient()

/** The shape a component accepts to allow injecting a mock client in tests
 * (see src/components/identity/__tests__/DisplayName.spec.ts) without
 * needing to mock the ambient `fetch` — mirrors FacilitiesClient's
 * identical role. */
export type IdentityClient = ReturnType<typeof createIdentityClient>
