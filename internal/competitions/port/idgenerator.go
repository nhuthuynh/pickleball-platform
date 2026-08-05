package port

// IDGenerator produces new identifiers for aggregates. Mirrors
// internal/booking/port.IDGenerator and internal/socialplay/port.
// IDGenerator's shape so the app layer stays deterministic and testable —
// tests inject a fixed-sequence generator instead of depending on a real
// UUID library. internal/platform/idgen.UUID already satisfies it.
//
// One generator serves both Competition and CompetitionEntry IDs, exactly
// as Social Play's serves both Game and Registration; there is no reason
// for the two aggregates' identifiers to come from different sources.
type IDGenerator interface {
	NewID() string
}
