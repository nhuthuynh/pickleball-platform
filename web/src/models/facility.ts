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

/**
 * A raw Court as GetFacilityResponse.courts returns it (T8.2) — see
 * `mapToFacilityDetail`'s doc comment for why `courts` lives on the
 * response wrapper rather than on `RawFacility`/`v1Facility` itself.
 */
export type RawCourt = components['schemas']['v1Court']

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
 * Maps a raw API `Facility` (plus its sibling `courts` list) to the detail
 * view model.
 *
 * `rawCourts` (T8.2) closes the gap this function's previous version
 * documented: the merged Facilities API (T7.3) had `AddCourt` to create a
 * Court but no endpoint that listed a Facility's Courts back, so this
 * always mapped `courts: []` regardless of what a Facility Owner had
 * actually added. `GetFacilityResponse` now carries `courts` (see
 * proto/pickleball/facilities/v1/facilities.proto's doc comment on that
 * message for why it's a field on the *response*, not on `RawFacility`/
 * `v1Facility` itself) — `rawCourts` is that sibling list, passed in by the
 * caller (`useFacilityDetail.ts`) alongside `raw.facility`, not read off
 * `raw`. Defaults to `[]` (not required) so existing call sites/tests that
 * only have a bare `RawFacility` (e.g. a `CreateFacility`/`AddCourt`
 * response, neither of which carries a courts list) keep working.
 */
export function mapToFacilityDetail(raw: RawFacility, rawCourts: RawCourt[] = []): FacilityDetail {
  return {
    id: raw.id ?? '',
    name: raw.name ?? '',
    description: raw.description ?? '',
    address: raw.address ?? '',
    photoUrls: raw.photoUrls ?? [],
    courts: rawCourts.map((c) => ({ id: c.id ?? '', name: c.name ?? '' })),
  }
}
