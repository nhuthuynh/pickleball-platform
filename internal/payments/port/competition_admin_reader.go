package port

import "context"

// CompetitionAdminReader is Payments' inbound read port onto Competitions'
// durable Competition-Admin store (T15.3's competition_admins table, reached
// via competitions/port.CompetitionAdminRepository) — the Competitions
// mirror of GameAdminReader, built for the identical reason (T15.5's §A12
// GAP C). internal/payments/adapter/competitions.CompetitionAdminReader
// implements it against the real
// competitionsapp.Service.ListCompetitionAdmins (exported at
// service.go:760, T15.3, specifically so T15.4 and T15.5 could each consume
// it — see that method's own doc comment).
//
// Returns bare subject strings, not competitionsdomain.CompetitionAdmin
// values, for the identical reason GameAdminReader's doc comment gives:
// Payments needs only the membership question
// competitionsdomain.HasCompetitionAdmin already answers on the Competitions
// side, not AssignedBy/AssignedAt.
//
// # NOT wired into internal/payments/app.Service as of T15.5 — read
// GameAdminReader's doc comment first
//
// This port and internal/payments/adapter/competitions.CompetitionAdminReader
// are built and tested against Competitions' real app.Service, but are
// deliberately NOT wired into app.Service or authorizeOfflineRecording, for
// the exact same structural reason GameAdminReader's doc comment documents at
// length: ListCompetitionAdmins(ctx, competitionID) needs a competitionID,
// and neither RecordOfflinePaymentInput nor RefundPaymentInput ever carries
// one for a PayableTypeCompetitionEntry payable — only PayableID (the
// CompetitionEntry's own id) and the caller-supplied EntrantPlayerID.
// competitionsapp.Service exposes no GetCompetitionEntry-shaped read that
// would resolve a CompetitionEntry's owning Competition id, so this port
// cannot correctly be invoked for that payable type today either. See
// GameAdminReader's doc comment and the T15.5 PR description for the full
// writeup — the finding is one gap, disclosed once, that happens to recur
// identically in both contexts Payments depends on.
type CompetitionAdminReader interface {
	// ListCompetitionAdmins returns the user ids currently holding
	// Competition-Admin authority over competitionID, oldest assignment
	// first — competitions/port.CompetitionAdminRepository.
	// ListCompetitionAdmins' order, unchanged; this port is a thin
	// read-through.
	//
	// An unknown or malformed competitionID returns an empty slice, not an
	// error — the same deliberate widening GameAdminReader.ListGameAdmins
	// documents, for the identical reason: a pre-check has one safe answer
	// for "no such Competition" and "no admins", and this port's only
	// intended caller is a pre-check, never a response body.
	ListCompetitionAdmins(ctx context.Context, competitionID string) ([]string, error)
}
