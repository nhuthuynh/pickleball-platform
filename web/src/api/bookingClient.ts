// Concrete typed client for the Booking bounded context.
//
// NOTE: this file only type-checks once `npm run generate:client` has been
// run (it needs `./generated/booking` to exist — see web/README.md). This
// mirrors the backend's own internal/gen/** convention (CLAUDE.md rule 6):
// generated output is gitignored, and code that depends on it simply
// doesn't compile until `make generate`/`npm run generate:client` has run.
import type { paths as BookingPaths } from './generated/booking'
import { createTypedClient, type CreateTypedClientOptions } from './client'

export function createBookingClient(options: CreateTypedClientOptions = {}) {
  return createTypedClient<BookingPaths>(options)
}

export const bookingClient = createBookingClient()
