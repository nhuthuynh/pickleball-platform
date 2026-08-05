package booking_test

import (
	competitionsbooking "github.com/nhuthuynh/white-label/internal/competitions/adapter/booking"
	"github.com/nhuthuynh/white-label/internal/competitions/port"
)

// Compile-time proof that *competitionsbooking.Reservation satisfies
// port.CourtReservation — the whole point of this package. Worth asserting
// explicitly rather than relying on a call site to catch a mismatch,
// because T9.3 has no call site yet: cmd/server doesn't wire Competitions
// until T9.4, so without this line a signature drift between the port and
// the adapter would compile cleanly and stay hidden for a whole ticket.
// Mirrors internal/payments/adapter/socialplay's identical assertion.
//
// The adapter's *behaviour* is deliberately not unit-tested here. Every
// branch it has is a translation of a real bookingapp.Service outcome
// (bookingdomain.ErrCourtDoubleBooked -> domain.ErrCourtUnavailable, and
// the non-%w wrapping of everything else), so a test against a stubbed
// Booking service would prove only that the stub returns what it was told
// to. The behaviour that matters — that a genuine court conflict surfaces
// as domain.ErrCourtUnavailable through the real Postgres EXCLUDE
// constraint — is an integration concern, and T9.4 is the ticket that
// brings up the schema needed to test it honestly.
var _ port.CourtReservation = (*competitionsbooking.Reservation)(nil)
