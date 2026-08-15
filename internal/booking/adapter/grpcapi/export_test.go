// T14.7 (#158) — test-only seam onto this package's single error mapper.
//
// error_mapping_test.go pins one gRPC code per domain sentinel. Most rows are
// additionally driven through a real RPC, which is the stronger assertion and
// the one T13.9's payments template uses throughout. But Booking's domain
// declares sentinels that no current RPC can surface (a constructor guard
// whose only reachable caller validates the field earlier, for instance), and
// a table that silently skipped those would let the very thing guard 4 exists
// to prevent — a sentinel with no pinned code, falling through toStatus's
// default to Internal — back in through the gap.
//
// Exposing toStatus to the test binary lets every sentinel be pinned directly
// while the reachable ones are *also* driven end-to-end. It is a test file, so
// nothing here is compiled into the production package.
package grpcapi

// ToStatusForTest is toStatus, the one error mapper in this package.
// TestBookingService_HasExactlyOneErrorMapper enforces that "one".
var ToStatusForTest = toStatus
