// Concrete typed client for the Facilities bounded context (T7.3's real
// CreateFacility/GetFacility/ListFacilities/AddCourt/AddCameraLink REST
// surface). Mirrors bookingClient.ts's shape exactly.
//
// NOTE: this file only type-checks once `npm run generate:client` has been
// run (it needs `./generated/facilities` to exist — see web/README.md).
// `import type` is erased at build time (no runtime import), so — same as
// bookingClient.ts — this module is safe to import from component code that
// Vitest exercises on a fresh clone before `generate:client` has ever run;
// only `npm run build`'s type-check step needs the generated file present.
import type { paths as FacilitiesPaths } from './generated/facilities'
import { createTypedClient, type CreateTypedClientOptions } from './client'

export function createFacilitiesClient(options: CreateTypedClientOptions = {}) {
  return createTypedClient<FacilitiesPaths>(options)
}

export const facilitiesClient = createFacilitiesClient()

/** The shape a component accepts to allow injecting a mock client in tests
 * (see src/views/__tests__/FacilityOnboarding.spec.ts) without needing to
 * mock the ambient `fetch`. */
export type FacilitiesClient = ReturnType<typeof createFacilitiesClient>
