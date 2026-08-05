// k6 load test: the booking write path + the quote read path.
//
// Run against a LOCAL stack only (`make up`), never a shared environment —
// this test writes real bookings.
//
//   make up
//   k6 run loadtest/booking-quote.js
//   BASE_URL=http://localhost:8080 k6 run loadtest/booking-quote.js
//
// See loadtest/README.md for why k6, what the thresholds mean, and how the
// Jenkins stage is triggered (it is opt-in, not a per-PR gate).

import http from 'k6/http'
import { check, group } from 'k6'
import { Counter, Rate } from 'k6/metrics'

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080'

// Seeded by db/migrations/0002_seed.sql. Court 1 is the one with pricing
// rules (0004_seed_pricing.sql), so it is the only court a quote resolves
// for — court 2 would legitimately return an error and is not used here.
const COURT_ID = __ENV.COURT_ID || '11111111-1111-1111-1111-111111111111'

// A weekday 09:00–10:00 UTC slot that falls inside the seeded "weekday"
// pricing band (Mon–Fri, minute 360–1020, i.e. 06:00–17:00). 2026-08-10 is
// a Monday. The quote path is a pure read, so every VU can reuse this slot.
const QUOTE_SLOT = { starts_at: '2026-08-10T09:00:00Z', ends_at: '2026-08-10T10:00:00Z' }

// Booking creation, by contrast, must NOT collide: the whole point of the
// system is that two bookings cannot overlap on one court, so a naive load
// test would mostly measure 409s. Each (VU, iteration) pair therefore gets
// its own one-hour slot, far enough out to never touch seeded data.
const BOOKING_EPOCH = Date.UTC(2027, 0, 1, 0, 0, 0)
const HOUR_MS = 60 * 60 * 1000

const conflicts = new Counter('booking_conflicts')
const bookingSuccess = new Rate('booking_success_rate')
const quoteSuccess = new Rate('quote_success_rate')

export const options = {
  // Deliberately small: this is a smoke-scale load profile meant to catch
  // gross regressions and obvious contention, not a capacity-planning run.
  vus: Number(__ENV.VUS || 20),
  duration: __ENV.DURATION || '30s',

  // Thresholds are what make this pass/fail rather than a wall of numbers.
  //
  // HONESTY NOTE (CLAUDE.md rule 10): these are *starting* values chosen to
  // be obviously-broken detectors, not validated SLOs. Nobody has run this
  // against a real stack yet — see loadtest/README.md. Tune them from the
  // first few real runs before treating a breach as meaningful.
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<500'],
    'quote_success_rate': ['rate>0.99'],
    'booking_success_rate': ['rate>0.99'],
  },
}

function uniqueSlot() {
  // __VU is 1-based and __ITER is 0-based. __VU*100000 buckets each VU into
  // its own block of the index space; collisions are avoided as long as
  // __ITER stays below 100000 for any single VU (that's the constraint —
  // VU count itself has no such ceiling from this formula).
  const index = __VU * 100000 + __ITER
  const start = new Date(BOOKING_EPOCH + index * HOUR_MS)
  const end = new Date(start.getTime() + HOUR_MS)
  return { starts_at: start.toISOString(), ends_at: end.toISOString() }
}

const JSON_HEADERS = { headers: { 'Content-Type': 'application/json' } }

export default function () {
  group('quote', function () {
    const res = http.post(
      `${BASE_URL}/v1/quotes`,
      JSON.stringify({ court_id: COURT_ID, ...QUOTE_SLOT }),
      { ...JSON_HEADERS, tags: { endpoint: 'GetQuote' } },
    )

    const ok = check(res, {
      'quote returns 200': (r) => r.status === 200,
      'quote body carries a price': (r) => {
        if (r.status !== 200) return false
        try {
          const body = r.json()
          // grpc-gateway emits lowerCamelCase JSON by default.
          const cents = body.priceCents ?? body.price_cents
          return cents !== undefined && Number(cents) > 0
        } catch {
          return false
        }
      },
    })
    quoteSuccess.add(ok)
  })

  group('create booking', function () {
    const slot = uniqueSlot()
    const res = http.post(
      `${BASE_URL}/v1/bookings`,
      JSON.stringify({ court_id: COURT_ID, source: 'SOURCE_INDIVIDUAL', ...slot }),
      { ...JSON_HEADERS, tags: { endpoint: 'CreateBooking' } },
    )

    // A 409 here means the slot-allocation scheme above collided, which is
    // a bug in this script rather than in the server. Counted separately so
    // the two causes never get confused in a run report.
    if (res.status === 409) conflicts.add(1)

    const ok = check(res, {
      'booking returns 200': (r) => r.status === 200,
      'booking body carries an id': (r) => {
        if (r.status !== 200) return false
        try {
          const body = r.json()
          const booking = body.booking ?? body
          return Boolean(booking.id)
        } catch {
          return false
        }
      },
    })
    bookingSuccess.add(ok)
  })

  group('list court bookings', function () {
    const res = http.get(
      `${BASE_URL}/v1/courts/${COURT_ID}/bookings?from=2026-08-10T00:00:00Z&to=2026-08-11T00:00:00Z`,
      { tags: { endpoint: 'ListCourtBookings' } },
    )
    check(res, { 'list returns 200': (r) => r.status === 200 })
  })
}
