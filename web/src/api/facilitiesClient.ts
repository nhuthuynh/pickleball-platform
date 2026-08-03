// Concrete typed client for the Facilities bounded context (T7.3's
// CreateFacility/GetFacility/ListFacilities/AddCourt/AddCameraLink API).
//
// Written for T7.5 (docs/process/t7-sprint-plan.md) because no
// `facilitiesClient.ts` existed yet on the base branch at the time this
// ticket was implemented — T7.4 (Facility onboarding UI, a separate,
// concurrently-developed ticket/branch) may add its own. This file
// mirrors `bookingClient.ts`'s pattern exactly (thin, generic
// `createTypedClient` + one bounded-context-specific type parameter) so
// the two are trivially reconcilable/mergeable in a later review rather
// than diverging in shape.
//
// NOTE: this file only type-checks once `npm run generate:client` has
// been run (it needs `./generated/facilities` to exist — see
// web/README.md). This mirrors the backend's own internal/gen/**
// convention (CLAUDE.md rule 6): generated output is gitignored, and code
// that depends on it simply doesn't compile until `make
// generate`/`npm run generate:client` has run.
import type { paths as FacilitiesPaths } from './generated/facilities'
import { createTypedClient, type CreateTypedClientOptions } from './client'

export function createFacilitiesClient(options: CreateTypedClientOptions = {}) {
  return createTypedClient<FacilitiesPaths>(options)
}

export type FacilitiesClient = ReturnType<typeof createFacilitiesClient>

export const facilitiesClient = createFacilitiesClient()
