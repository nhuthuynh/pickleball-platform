package app_test

import (
	"context"
	"sync/atomic"

	competitionsdomain "github.com/nhuthuynh/white-label/internal/competitions/domain"
	"github.com/nhuthuynh/white-label/internal/payments/domain"
	socialplaydomain "github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// fixedIDs is a deterministic port.IDGenerator test double: it returns
// each of the seeded IDs in order, then panics if asked for more than were
// seeded (a test bug, not a runtime concern) — the same pattern used by
// the Booking and Social Play contexts' app-layer tests so behaviour stays
// deterministic without a real UUID library.
type fixedIDs struct {
	ids  []string
	next int
}

func (f *fixedIDs) NewID() string {
	if f.next >= len(f.ids) {
		panic("fixedIDs: ran out of seeded ids")
	}
	id := f.ids[f.next]
	f.next++
	return id
}

// fakeRepository is an in-memory port.Repository test double (T6.4). It
// enforces the same "one Payment per (PayableType, PayableID)" distinctness
// invariant the Postgres adapter's UNIQUE (payable_type, payable_id)
// constraint enforces for real, returning domain.ErrPaymentAlreadyRecorded
// on a second Create for the same payable action — so app-layer tests can
// exercise the T6.4 AC (duplicate offline recording is rejected) without a
// real database, mirroring the fixedIDs/stripestub.Processor pattern of
// deterministic in-memory doubles standing in for infrastructure.
type fakeRepository struct {
	byID      map[string]domain.Payment
	byPayable map[string]string // "payableType:payableID" -> payment id

	// createCalls/getByIDCalls count real invocations of Create/GetByID, so
	// a test can prove a malformed-shape PayableID/id never reaches the
	// repository at all (T10.7, closing issue #97) — the in-memory maps
	// here can't reproduce Postgres rejecting a non-UUID against a `uuid`
	// column, so the app-layer shape-check short-circuit can only be proven
	// by observing the call itself never happens, not by the return value
	// alone. atomic.Int64, mirroring internal/booking/app/service_test.go's
	// inMemoryRepo.listActiveForCourtCalls exactly, for the same reason:
	// safe even if a future test shares one fixture across t.Parallel()
	// subtests.
	createCalls  atomic.Int64
	getByIDCalls atomic.Int64

	// updateErr, when non-nil, makes Update fail *without* mutating byID —
	// the persistence-failure injection T12.3's RefundPayment tests need to
	// prove a refund that succeeded at the processor but failed to persist
	// is reported to the caller as an error rather than as a success (the
	// ticket's explicit non-functional requirement). Deliberately a plain
	// field rather than a function hook: no existing test needs
	// per-call-varying behaviour, and the simplest double that proves the
	// requirement is the right one.
	updateErr error
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		byID:      map[string]domain.Payment{},
		byPayable: map[string]string{},
	}
}

func payableKey(t domain.PayableType, id string) string {
	return string(t) + ":" + id
}

func (r *fakeRepository) Create(_ context.Context, p domain.Payment) (domain.Payment, error) {
	r.createCalls.Add(1)
	key := payableKey(p.PayableType, p.PayableID)
	if _, exists := r.byPayable[key]; exists {
		return domain.Payment{}, domain.ErrPaymentAlreadyRecorded
	}
	r.byPayable[key] = p.ID
	r.byID[p.ID] = p
	return p, nil
}

func (r *fakeRepository) GetByID(_ context.Context, id string) (domain.Payment, error) {
	r.getByIDCalls.Add(1)
	p, ok := r.byID[id]
	if !ok {
		return domain.Payment{}, domain.ErrPaymentNotFound
	}
	return p, nil
}

func (r *fakeRepository) Update(_ context.Context, p domain.Payment) (domain.Payment, error) {
	if _, ok := r.byID[p.ID]; !ok {
		return domain.Payment{}, domain.ErrPaymentNotFound
	}
	// Checked after the existence lookup but before the write, so an
	// injected failure leaves the stored Payment exactly as it was — the
	// point of the injection is to prove the caller doesn't report success,
	// which is only meaningful if the store genuinely didn't change.
	if r.updateErr != nil {
		return domain.Payment{}, r.updateErr
	}
	r.byID[p.ID] = p
	return p, nil
}

// registrationUpdateCall records one call to
// fakeRegistrationUpdater.UpdatePaymentStatus, for asserting exact
// call-count/argument expectations (T6.5's required test).
type registrationUpdateCall struct {
	registrationID string
	status         socialplaydomain.PaymentStatus
}

// fakeRegistrationUpdater is an in-memory
// socialplayport.RegistrationPaymentUpdater test double (T6.5's required
// test): it records every call it receives rather than talking to a real
// (or fake) Social Play repository, so tests can assert exactly how many
// times it was called and with what arguments — including the "not called
// at all for a booking-payable Payment" negative case, which a
// state-mutating fake alone couldn't prove as directly.
type fakeRegistrationUpdater struct {
	calls []registrationUpdateCall
	err   error // optional: simulate the port call itself failing
}

func (u *fakeRegistrationUpdater) UpdatePaymentStatus(_ context.Context, registrationID string, status socialplaydomain.PaymentStatus) error {
	u.calls = append(u.calls, registrationUpdateCall{registrationID: registrationID, status: status})
	return u.err
}

// entryUpdateCall records one call to
// fakeCompetitionEntryUpdater.UpdatePaymentStatus, for asserting exact
// call-count/argument expectations (T10.6's required test, mirroring T6.5's
// registrationUpdateCall).
type entryUpdateCall struct {
	entryID string
	status  competitionsdomain.PaymentStatus
}

// fakeCompetitionEntryUpdater is an in-memory
// competitionsport.CompetitionEntryPaymentUpdater test double (T10.6's
// required test, mirroring fakeRegistrationUpdater exactly): it records
// every call it receives rather than talking to a real (or fake)
// Competitions repository, so tests can assert exactly how many times it
// was called and with what arguments — including the "not called at all
// for a non-competition_entry-payable Payment" negative case.
type fakeCompetitionEntryUpdater struct {
	calls []entryUpdateCall
	err   error // optional: simulate the port call itself failing
}

func (u *fakeCompetitionEntryUpdater) UpdatePaymentStatus(_ context.Context, entryID string, status competitionsdomain.PaymentStatus) error {
	u.calls = append(u.calls, entryUpdateCall{entryID: entryID, status: status})
	return u.err
}

// --- T16.2: authorization resolver port fakes (closes #168) ---------------
//
// These fake the five NEW ports (port.RegistrationLookup, GameLookup,
// GameAdminReader, EntryLookup, CompetitionAdminReader) at the same level
// fakeRegistrationUpdater/fakeCompetitionEntryUpdater above fake their own
// ports — a plain in-memory double of internal/payments' own port
// interface, not a fake of another context's app.Service. That is a
// deliberate choice, not an oversight of T14.8/T15.5's cross-context-fake
// warning: that warning is about testing an ADAPTER (whether
// internal/payments/adapter/socialplay.RegistrationLookup correctly
// translates a REAL socialplayapp.Service call) against a fake stand-in for
// the other context, which would prove nothing about the real seam — see
// internal/payments/adapter/socialplay/registration_lookup_test.go and
// internal/payments/adapter/socialplay/authorization_boundary_test.go
// (T16.2) for where that real-seam proof actually lives. This file tests
// internal/payments/app.Service's own ORCHESTRATION logic (which branch
// calls which port, in what order, with what fallback), which is properly
// tested against a fake of Payments' OWN port contract, exactly as
// RegistrationUpdater/CompetitionEntryUpdater already are above.
type fakeRegistrationLookup struct {
	gameByRegistration map[string]string
	err                error
}

func (f *fakeRegistrationLookup) GameIDForRegistration(_ context.Context, registrationID string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.gameByRegistration[registrationID], nil
}

type fakeGameLookup struct {
	hostByGame map[string]string
	err        error
}

func (f *fakeGameLookup) HostIDForGame(_ context.Context, gameID string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.hostByGame[gameID], nil
}

type fakeGameAdminReader struct {
	adminsByGame map[string][]string
	err          error
}

func (f *fakeGameAdminReader) ListGameAdmins(_ context.Context, gameID string) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.adminsByGame[gameID], nil
}

type fakeEntryLookup struct {
	competitionByEntry map[string]string
	playerByEntry      map[string]string
	err                error
}

func (f *fakeEntryLookup) CompetitionIDAndPlayerIDForEntry(_ context.Context, entryID string) (string, string, error) {
	if f.err != nil {
		return "", "", f.err
	}
	return f.competitionByEntry[entryID], f.playerByEntry[entryID], nil
}

type fakeCompetitionAdminReader struct {
	adminsByCompetition map[string][]string
	err                 error
}

func (f *fakeCompetitionAdminReader) ListCompetitionAdmins(_ context.Context, competitionID string) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.adminsByCompetition[competitionID], nil
}

// newGameAuthzFixtures builds the three resolver fakes
// authorizeGameRecording (RecordOfflinePayment/RefundPayment's
// PayableTypeRegistration/PayableTypeNoShowFee branch) needs, pre-wired so
// fixtureRegistrationID resolves to fixtureGameID, hostID is that Game's
// Host, and admins is its current Game-Admin set — replacing what a test
// used to set directly via the now-deleted GameHostID/
// AssignedGameAdminUserIDs input fields.
func newGameAuthzFixtures(hostID string, admins ...string) (*fakeRegistrationLookup, *fakeGameLookup, *fakeGameAdminReader) {
	regs := &fakeRegistrationLookup{gameByRegistration: map[string]string{fixtureRegistrationID: fixtureGameID}}
	games := &fakeGameLookup{hostByGame: map[string]string{fixtureGameID: hostID}}
	gameAdmins := &fakeGameAdminReader{adminsByGame: map[string][]string{fixtureGameID: admins}}
	return regs, games, gameAdmins
}

// newEntryAuthzFixtures is newGameAuthzFixtures' Competitions mirror:
// pre-wires fixtureCompetitionEntryID to resolve to fixtureCompetitionID and
// playerID in one call, and admins as that Competition's current
// Competition-Admin set — replacing what a test used to set directly via
// the now-deleted EntrantPlayerID/AssignedCompetitionAdminUserIDs input
// fields.
func newEntryAuthzFixtures(playerID string, admins ...string) (*fakeEntryLookup, *fakeCompetitionAdminReader) {
	entries := &fakeEntryLookup{
		competitionByEntry: map[string]string{fixtureCompetitionEntryID: fixtureCompetitionID},
		playerByEntry:      map[string]string{fixtureCompetitionEntryID: playerID},
	}
	competitionAdmins := &fakeCompetitionAdminReader{adminsByCompetition: map[string][]string{fixtureCompetitionID: admins}}
	return entries, competitionAdmins
}

// fixtureOnlinePayerID is the verified actor the online-path tests create
// intents as, and therefore the only actor entitled to confirm them (T13.7,
// closes issue #148). Before that ticket CreateOnlinePayment stored no actor at
// all and ConfirmOnlinePayment checked none, so these fixtures could leave it
// unset; now an online Payment without an owner is one nobody may capture,
// which is the point.
const fixtureOnlinePayerID = "payer-online-1"
