package domain

// Format is the play format a Host advertises for a Competition: singles or
// doubles.
//
// **Format is descriptive, not enforcing.** Nothing in this domain — not
// NewCompetition, not Enter, not the capacity rule — validates entry counts,
// pairings, or anything else against it. An entrant may enter a
// FormatDoubles Competition alone, and the domain will accept it; partner
// pairing is not modelled in T9 at all (brackets, rounds, seeding, and
// results are explicitly out of scope, docs/process/t9-sprint-plan.md §A4).
// The value is stored and displayed, nothing more.
//
// Why that is acceptable here when T8's kickoff note dropped Social Play's
// `CancellationCutoff` on the rule "shipping a cosmetic text field that
// enforces nothing would be actively misleading" (§A4, resolved by the team
// rather than left as an apparent inconsistency): the test is whether a
// field *implies enforcement*, not whether it has code behind it. A
// cancellation cutoff is a rule the Host asserts over other people's
// behavior — a Host who sets one reasonably expects late cancellations to
// be blocked, and a field that silently doesn't is a broken promise. A
// format label is descriptive information for players ("this is a doubles
// competition"), which is complete and honest the moment it is displayed.
// If a future ticket ever makes Format load-bearing (e.g. enforcing paired
// entries), that is a new invariant with its own tests — not a quiet
// reinterpretation of this field.
type Format string

const (
	FormatSingles Format = "singles"
	FormatDoubles Format = "doubles"
)

// IsValid reports whether f is one of Format's closed enum values.
// NewCompetition uses this so an arbitrary string can never reach the
// field: descriptive-only does not mean unvalidated — a garbage value would
// still be displayed to players as if it meant something.
func (f Format) IsValid() bool {
	switch f {
	case FormatSingles, FormatDoubles:
		return true
	default:
		return false
	}
}
