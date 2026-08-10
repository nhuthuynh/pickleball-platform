package domain

// SelfReportedStartingLevel is the raw, player-entered starting-level
// self-assessment on a bounded 1..5 scale — the exact mechanism the locked
// CLAUDE.md decision names for cold-start seeding ("new players seeded by a
// self-reported starting level").
//
// This is deliberately NOT an answer to ADR-0012's still-open Q1 (how the
// tenure+win-rate-weighted, history-derived Player Level formula is
// weighted). SelfReportedStartingLevel is the raw input a player gives at
// signup, before any Match history exists; Q1 is about a different,
// computed value this package does not build (see User's doc comment). A
// future reader must not conflate the two: this field answers "what did the
// player claim about themselves before we had any data," not "what score
// does the platform now compute for them."
type SelfReportedStartingLevel int

const (
	MinSelfReportedStartingLevel SelfReportedStartingLevel = 1
	MaxSelfReportedStartingLevel SelfReportedStartingLevel = 5
)

// IsValid reports whether l falls within the closed
// [MinSelfReportedStartingLevel, MaxSelfReportedStartingLevel] range.
func (l SelfReportedStartingLevel) IsValid() bool {
	return l >= MinSelfReportedStartingLevel && l <= MaxSelfReportedStartingLevel
}
