// Concrete typed client for the Social Play bounded context. T8.8's Social
// Game creation flow (`CreateGame`) is the first thing in this repo to need
// it — mirrors bookingClient.ts/facilitiesClient.ts's shape exactly.
//
// NOTE: this file only type-checks once `npm run generate:client` has been
// run (it needs `./generated/socialplay` to exist — see web/README.md).
// `import type` is erased at build time (no runtime import), so this module
// is safe to import from component code that Vitest exercises on a fresh
// clone before `generate:client` has ever run; only `npm run build`'s
// type-check step needs the generated file present.
import type { paths as SocialPlayPaths } from './generated/socialplay'
import { createTypedClient, type CreateTypedClientOptions } from './client'

export function createSocialPlayClient(options: CreateTypedClientOptions = {}) {
  return createTypedClient<SocialPlayPaths>(options)
}

export const socialplayClient = createSocialPlayClient()

/** The shape a component accepts to allow injecting a mock client in tests
 * (see src/views/__tests__/GameCreation.spec.ts) without needing to mock
 * the ambient `fetch` — mirrors FacilitiesClient/BookingClient. */
export type SocialPlayClient = ReturnType<typeof createSocialPlayClient>
