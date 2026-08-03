// Concrete typed client for the Social Play bounded context (T8.9: `ListGames`
// for the Discover & Join Games browse/filter read, `RegisterForGame`, and
// `JoinWaitlist`). Mirrors bookingClient.ts/facilitiesClient.ts's shape
// exactly.
//
// This file didn't exist on the base branch T8.9 forked from — T8.8 (Social
// Game creation, in flight on a separate branch at the time of writing) may
// add its own version independently; per this ticket's dispatch, a later
// merge is expected to reconcile the two mechanically rather than either
// side waiting on the other.
//
// NOTE: this file only type-checks once `npm run generate:client` has been
// run (it needs `./generated/socialplay` to exist — see web/README.md).
// `import type` is erased at build time (no runtime import), so — same as
// bookingClient.ts/facilitiesClient.ts — this module is safe to import from
// component code that Vitest exercises on a fresh clone before
// `generate:client` has ever run; only `npm run build`'s type-check step
// needs the generated file present.
import type { paths as SocialPlayPaths } from './generated/socialplay'
import { createTypedClient, type CreateTypedClientOptions } from './client'

export function createSocialPlayClient(options: CreateTypedClientOptions = {}) {
  return createTypedClient<SocialPlayPaths>(options)
}

export const socialplayClient = createSocialPlayClient()

/** The shape a component accepts to allow injecting a mock client in tests
 * without needing to mock the ambient `fetch` — mirrors
 * `FacilitiesClient`/`BookingClient`'s identical role. */
export type SocialPlayClient = ReturnType<typeof createSocialPlayClient>
