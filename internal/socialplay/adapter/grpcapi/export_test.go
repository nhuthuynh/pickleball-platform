// T14.7 (#158) — test-only seam onto this package's single error mapper.
//
// error_mapping_test.go pins one gRPC code per domain sentinel. Most rows are
// additionally driven through a real RPC, which is the stronger assertion and
// the one T13.9's payments template uses throughout. But Social Play's domain
// declares four sentinels that no client-facing RPC can surface — see the
// "deliberately unreachable" rows in that file — and a table that silently
// skipped them would let the very thing guard 4 exists to prevent (a sentinel
// with no pinned code, falling through toStatus's default to Internal) back in
// through the gap.
//
// Exposing toStatus to the test binary lets every sentinel be pinned directly
// while the reachable ones are *also* driven end-to-end. It is a test file, so
// nothing here is compiled into the production package.
package grpcapi

// ToStatusForTest is toStatus, the one error mapper in this package.
// TestSocialPlayService_HasExactlyOneErrorMapper enforces that "one".
var ToStatusForTest = toStatus
