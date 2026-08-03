// Player-facing view models for the Discover & browse facilities screen
// (T7.5, docs/process/t7-sprint-plan.md), plus the mapping functions that
// build them from the raw Facilities API response
// (web/src/api/generated/facilities.d.ts's `components['schemas']['v1Facility']`,
// produced from proto/pickleball/facilities/v1/facilities.proto).
//
// IMPORTANT — camera links must never reach this screen (T7.5 instructions,
// T7.2's domain note on internal/facilities/domain/facility.go: camera
// links are host-facing only, no player-facing surface). `RawFacility`
// includes `cameraLinks`/`cameraConsentAttested` because the API returns
// the full `Facility` message to every caller (there is no player-scoped
// response shape on the backend) — so the guarantee has to be enforced
// here, on the client, by construction: `FacilitySummary` and
// `FacilityDetail` below simply have no field that could hold camera-link
// data. `mapToFacilitySummary`/`mapToFacilityDetail` only ever copy the
// specific fields listed — they do not spread `raw` — so there is no way
// for a camera link to end up on the object handed to a component, let
// alone a template. See `__tests__/facility.spec.ts` for the regression
// test asserting this.
import type { components } from '../api/generated/facilities'

export type RawFacility = components['schemas']['v1Facility']

/** A single Court within a Facility, as shown to a Player browsing courts. */
export interface FacilityCourt {
  id: string
  name: string
}

/** One row in the facility list/search view. */
export interface FacilitySummary {
  id: string
  name: string
  description: string
  address: string
}

/**
 * The facility detail view's data: description, photos, address, and its
 * courts list (T7.5 requirement #1) — deliberately with no field that
 * could carry camera-link data (see file header).
 */
export interface FacilityDetail {
  id: string
  name: string
  description: string
  address: string
  photoUrls: string[]
  courts: FacilityCourt[]
}

export function mapToFacilitySummary(raw: RawFacility): FacilitySummary {
  return {
    id: raw.id ?? '',
    name: raw.name ?? '',
    description: raw.description ?? '',
    address: raw.address ?? '',
  }
}

/**
 * Maps a raw API `Facility` to the detail view model.
 *
 * `courts` is always `[]` today: the merged Facilities API (T7.3) has
 * `AddCourt` to create a Court but no endpoint that lists a Facility's
 * Courts back (`GetFacility`/`ListFacilities` return a bare `Facility`
 * with no `courts` field — confirmed against the generated OpenAPI types,
 * not assumed). So every facility genuinely has an empty courts list from
 * this screen's point of view right now, and the "zero courts" empty
 * state (T7.5 requirement #2) is not just a rare edge case to handle, it
 * is the *only* case this screen can currently observe. `courts` is kept
 * as a real field (rather than deleting it from the view model) so a
 * follow-up ticket that adds a courts-listing capability to the Facilities
 * API only has to change this one line, not this screen's components.
 */
export function mapToFacilityDetail(raw: RawFacility): FacilityDetail {
  return {
    id: raw.id ?? '',
    name: raw.name ?? '',
    description: raw.description ?? '',
    address: raw.address ?? '',
    photoUrls: raw.photoUrls ?? [],
    courts: [],
  }
}
