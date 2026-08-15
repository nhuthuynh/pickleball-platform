package port

import "context"

// CompetitionAdminReader is Payments' inbound read port onto Competitions'
// durable Competition-Admin store (T15.3's competition_admins table, reached
// via competitions/port.CompetitionAdminRepository) — the Competitions
// mirror of GameAdminReader, built for the identical reason (T15.5's §A12
// GAP C). internal/payments/adapter/competitions.CompetitionAdminReader
// implements it against the real
// competitionsapp.Service.ListCompetitionAdmins (exported at
// service.go:803, T15.3, specifically so T15.4 and T15.5 could each consume
// it — see that method's own doc comment).
//
// Returns bare subject strings, not competitionsdomain.CompetitionAdmin
// values, for the identical reason GameAdminReader's doc comment gives:
// Payments needs only the membership question
// competitionsdomain.HasCompetitionAdmin already answers on the Competitions
// side, not AssignedBy/AssignedAt.
//
// # Wired into internal/payments/app.Service as of T16.2 — read
// GameAdminReader's doc comment first
//
// This port and internal/payments/adapter/competitions.CompetitionAdminReader
// were built and tested against Competitions' real app.Service in T15.5, but
// were deliberately left unwired for the exact same structural reason
// GameAdminReader's doc comment documents at length:
// ListCompetitionAdmins(ctx, competitionID) needs a competitionID, and
// neither RecordOfflinePaymentInput nor RefundPaymentInput ever carried one
// for a PayableTypeCompetitionEntry payable — only PayableID (the
// CompetitionEntry's own id) and the caller-supplied EntrantPlayerID.
//
// T16.2 closes the gap: internal/payments/port.EntryLookup resolves a
// CompetitionEntry's own id to both its CompetitionID and PlayerID in one
// call (competitionsapp.Service.GetEntryByID), and this port then resolves
// that CompetitionID to its current admin set. Both are now wired into
// authorizeOfflineRecording's PayableTypeCompetitionEntry branch via
// internal/payments/app.Service. See GameAdminReader's doc comment for the
// remaining, deliberately out-of-scope gap this ticket leaves (#149,
// BookingHostID only).
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
