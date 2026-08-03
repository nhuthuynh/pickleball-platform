import { describe, it, expect } from 'vitest'
import { mapToFacilitySummary, mapToFacilityDetail, type RawFacility, type RawCourt } from '../facility'

/**
 * A raw Facility as the API actually returns it — including camera-link
 * data, per T7.5's instructions ("even if the API response includes
 * camera-link data ... this screen must never render it"). URL/CourtID
 * values are distinctive so a leak is easy to spot in a failure message.
 */
const rawFacilityWithCameraLinks: RawFacility = {
  id: 'facility-1',
  ownerId: 'owner-1',
  name: 'Riverside Courts',
  description: 'Six outdoor courts by the river.',
  address: '100 River Rd, Springfield',
  photoUrls: ['https://cdn.example.test/riverside-1.jpg'],
  cameraConsentAttested: true,
  cameraLinks: [{ url: 'https://cameras.example.test/riverside/secret-feed', courtId: '' }],
}

describe('mapToFacilitySummary', () => {
  it('maps the list-row fields', () => {
    expect(mapToFacilitySummary(rawFacilityWithCameraLinks)).toEqual({
      id: 'facility-1',
      name: 'Riverside Courts',
      description: 'Six outdoor courts by the river.',
      address: '100 River Rd, Springfield',
    })
  })

  it('never carries camera-link data onto the view model, even when present on the raw response', () => {
    const summary = mapToFacilitySummary(rawFacilityWithCameraLinks)
    expect(summary).not.toHaveProperty('cameraLinks')
    expect(summary).not.toHaveProperty('cameraConsentAttested')
    expect(JSON.stringify(summary)).not.toContain('cameras.example.test')
  })

  it('defaults missing optional fields to empty strings', () => {
    expect(mapToFacilitySummary({})).toEqual({ id: '', name: '', description: '', address: '' })
  })
})

describe('mapToFacilityDetail', () => {
  it('maps description/photos/address', () => {
    const detail = mapToFacilityDetail(rawFacilityWithCameraLinks)
    expect(detail.id).toBe('facility-1')
    expect(detail.name).toBe('Riverside Courts')
    expect(detail.description).toBe('Six outdoor courts by the river.')
    expect(detail.address).toBe('100 River Rd, Springfield')
    expect(detail.photoUrls).toEqual(['https://cdn.example.test/riverside-1.jpg'])
  })

  it('never carries camera-link data onto the view model, even when present on the raw response', () => {
    const detail = mapToFacilityDetail(rawFacilityWithCameraLinks)
    expect(detail).not.toHaveProperty('cameraLinks')
    expect(detail).not.toHaveProperty('cameraConsentAttested')
    expect(JSON.stringify(detail)).not.toContain('cameras.example.test')
  })

  it('returns an empty courts list when no rawCourts argument is given (e.g. a CreateFacility/AddCourt response, which carries no courts list)', () => {
    const detail = mapToFacilityDetail(rawFacilityWithCameraLinks)
    expect(detail.courts).toEqual([])
  })

  it('returns an empty courts list for a facility with none (GetFacilityResponse.courts === [])', () => {
    const detail = mapToFacilityDetail(rawFacilityWithCameraLinks, [])
    expect(detail.courts).toEqual([])
  })

  // T8.2: GetFacilityResponse.courts closes the gap the previous version of
  // this test documented (no endpoint listed a facility's courts back) —
  // this proves the populated case maps correctly, not just the empty one.
  it('maps a populated rawCourts list onto FacilityDetail.courts', () => {
    const rawCourts: RawCourt[] = [
      { id: 'court-1', facilityId: 'facility-1', name: 'Court 1' },
      { id: 'court-2', facilityId: 'facility-1', name: 'Court 2' },
    ]
    const detail = mapToFacilityDetail(rawFacilityWithCameraLinks, rawCourts)
    expect(detail.courts).toEqual([
      { id: 'court-1', name: 'Court 1' },
      { id: 'court-2', name: 'Court 2' },
    ])
  })

  it('defaults a missing court id/name to empty strings', () => {
    const detail = mapToFacilityDetail(rawFacilityWithCameraLinks, [{}])
    expect(detail.courts).toEqual([{ id: '', name: '' }])
  })

  it('defaults missing optional fields to empty strings/arrays', () => {
    expect(mapToFacilityDetail({})).toEqual({
      id: '',
      name: '',
      description: '',
      address: '',
      photoUrls: [],
      courts: [],
    })
  })
})
