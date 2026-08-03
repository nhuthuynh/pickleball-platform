package port

// IDGenerator produces new identifiers for aggregates. Mirrors
// internal/booking/port.IDGenerator — kept as its own per-context interface
// (not a shared import) so each bounded context stays independently
// testable, even though internal/platform/idgen.UUID satisfies all of them
// structurally (see cmd/server/main.go's wiring).
type IDGenerator interface {
	NewID() string
}
