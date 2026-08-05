package port

// ShareTokenGenerator produces the opaque token behind a Competition's
// shareable registration link (domain.Competition.ShareToken).
//
// Declared in T9.3 although its real consumer is T9.5's share-link flow.
// That is deliberate sequencing, not speculation: ScheduleCompetition is
// the only write path that constructs a Competition, so a token generated
// anywhere else would need a *second* write path to back-fill tokens onto
// Competitions created without one — plus a decision about what a
// tokenless Competition means in the meantime. Generating it at
// construction means every Competition has a token from the moment it
// exists, and T9.5 reduces to a real implementation plus a read path.
//
// Why this is a port rather than a direct crypto/rand call in the app
// layer: a share token must be unguessable (an enumerable token would make
// every unpublished Competition readable by anyone who can count), which
// makes it a real infrastructure concern with a real security requirement
// — and it makes the app layer non-deterministic to test. T9.5 supplies
// the crypto/rand-backed implementation and owns the entropy decision.
type ShareTokenGenerator interface {
	// NewShareToken returns a fresh, URL-safe, unguessable token.
	//
	// Unlike IDGenerator.NewID, this returns an error. That difference is
	// intentional: NewID is backed by a UUID library the platform treats as
	// infallible, whereas a token generator reads from a real entropy
	// source that can, in principle, fail. Callers must treat a failure as
	// fatal to the operation rather than falling back to a weaker token —
	// a predictable token is worse than no Competition, so
	// app.Service.ScheduleCompetition aborts before touching any other
	// port.
	NewShareToken() (string, error)
}
