package port

// IDGenerator produces new identifiers for aggregates. Kept as a port so the
// app layer stays deterministic and testable — tests inject a fixed-sequence
// generator instead of depending on a real UUID library.
type IDGenerator interface {
	NewID() string
}
