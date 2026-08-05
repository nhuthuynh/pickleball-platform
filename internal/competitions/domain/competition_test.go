package domain_test

import (
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/competitions/domain"
)

// validSessions returns a single-session fixture: one court, one two-hour
// slot. Built fresh per call so a test mutating it can't leak into another.
func validSessions(t *testing.T) []domain.Session {
	t.Helper()
	return []domain.Session{{
		Range:    mustRange(t, "2026-09-05T09:00:00Z", "2026-09-05T11:00:00Z"),
		CourtIDs: []string{"court-1"},
	}}
}

// validFee is a well-formed non-zero entry fee.
func validFee() domain.Money {
	return domain.Money{AmountCents: 2000, CurrencyCode: "GBP"}
}

// newValidCompetition builds the canonical happy-path Competition used as a
// fixture by the entry tests, failing the test if construction breaks.
func newValidCompetition(t *testing.T, capacity, guestAllowance int) domain.Competition {
	t.Helper()
	c, err := domain.NewCompetition(
		"comp-1", "host-1", "Autumn Doubles Open", "venue-1",
		validSessions(t), capacity, guestAllowance,
		domain.PaymentMethodEither, validFee(), domain.FormatDoubles, "tok-1",
	)
	if err != nil {
		t.Fatalf("unexpected err building competition fixture: %v", err)
	}
	return c
}

// TestNewCompetition_Valid proves the happy path: a well-formed Competition
// is constructed with Status starting at "scheduled" and every field stored
// as given.
func TestNewCompetition_Valid(t *testing.T) {
	t.Parallel()

	sessions := []domain.Session{
		{Range: mustRange(t, "2026-09-05T09:00:00Z", "2026-09-05T11:00:00Z"), CourtIDs: []string{"court-1", "court-2"}},
		{Range: mustRange(t, "2026-09-06T09:00:00Z", "2026-09-06T11:00:00Z"), CourtIDs: []string{"court-1"}},
	}

	c, err := domain.NewCompetition(
		"comp-1", "host-1", "Autumn Doubles Open", "venue-1",
		sessions, 16, 2,
		domain.PaymentMethodOnline, domain.Money{AmountCents: 2500, CurrencyCode: "GBP"},
		domain.FormatDoubles, "tok-abc",
	)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if c.Status != domain.StatusScheduled {
		t.Fatalf("status = %v, want %v", c.Status, domain.StatusScheduled)
	}
	if c.ID != "comp-1" || c.HostID != "host-1" || c.Name != "Autumn Doubles Open" || c.VenueFacilityID != "venue-1" {
		t.Fatalf("unexpected identity fields: %+v", c)
	}
	if len(c.Sessions) != 2 {
		t.Fatalf("sessions = %d, want 2", len(c.Sessions))
	}
	if c.Capacity != 16 || c.GuestAllowance != 2 {
		t.Fatalf("capacity/guest allowance = %d/%d, want 16/2", c.Capacity, c.GuestAllowance)
	}
	if c.PaymentMethod != domain.PaymentMethodOnline {
		t.Fatalf("payment method = %v, want %v", c.PaymentMethod, domain.PaymentMethodOnline)
	}
	if c.EntryFee != (domain.Money{AmountCents: 2500, CurrencyCode: "GBP"}) {
		t.Fatalf("entry fee = %+v, want 2500 GBP", c.EntryFee)
	}
	if c.Format != domain.FormatDoubles {
		t.Fatalf("format = %v, want %v", c.Format, domain.FormatDoubles)
	}
	if c.ShareToken != "tok-abc" {
		t.Fatalf("share token = %q, want %q", c.ShareToken, "tok-abc")
	}
}

// TestNewCompetition_EmptyVenueFacilityIDIsLegal proves an unset venue is
// accepted — same semantics as domain.Game.VenueFacilityID: the field is
// stored as-is here and its existence is the app layer's job to check via a
// port, since this package may never import the Facilities context.
func TestNewCompetition_EmptyVenueFacilityIDIsLegal(t *testing.T) {
	t.Parallel()

	c, err := domain.NewCompetition(
		"comp-1", "host-1", "Unsited Open", "",
		validSessions(t), 8, 0,
		domain.PaymentMethodCash, domain.Money{}, domain.FormatSingles, "",
	)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if c.VenueFacilityID != "" {
		t.Fatalf("venue facility id = %q, want empty", c.VenueFacilityID)
	}
}

// TestNewCompetition_Validation is the required table-driven boundary
// coverage for construction: capacity, sessions, court IDs, ranges, enum
// values, guest allowance, and entry fee.
func TestNewCompetition_Validation(t *testing.T) {
	t.Parallel()

	// A zero-duration range built by bypassing NewTimeRange (which itself
	// rejects it) via the exported struct fields — this is exactly how a
	// caller could hand NewCompetition an already-invalid range, so
	// NewCompetition must re-validate rather than trust its input (the same
	// reasoning domain.NewGame documents).
	sameInstant := mustTime(t, "2026-09-05T09:00:00Z")
	zeroDuration := domain.TimeRange{Start: sameInstant, End: sameInstant}
	inverted := domain.TimeRange{Start: mustTime(t, "2026-09-05T11:00:00Z"), End: mustTime(t, "2026-09-05T09:00:00Z")}

	tests := []struct {
		name           string
		sessions       []domain.Session
		capacity       int
		guestAllowance int
		paymentMethod  domain.PaymentMethod
		entryFee       domain.Money
		format         domain.Format
		wantErr        error
	}{
		{
			name: "valid inputs accepted", sessions: validSessions(t), capacity: 8, guestAllowance: 1,
			paymentMethod: domain.PaymentMethodEither, entryFee: validFee(), format: domain.FormatSingles,
		},
		{
			name: "capacity zero is rejected", sessions: validSessions(t), capacity: 0, guestAllowance: 0,
			paymentMethod: domain.PaymentMethodEither, entryFee: validFee(), format: domain.FormatSingles,
			wantErr: domain.ErrInvalidCapacity,
		},
		{
			name: "negative capacity is rejected", sessions: validSessions(t), capacity: -1, guestAllowance: 0,
			paymentMethod: domain.PaymentMethodEither, entryFee: validFee(), format: domain.FormatSingles,
			wantErr: domain.ErrInvalidCapacity,
		},
		{
			name: "nil sessions rejected", sessions: nil, capacity: 8, guestAllowance: 0,
			paymentMethod: domain.PaymentMethodEither, entryFee: validFee(), format: domain.FormatSingles,
			wantErr: domain.ErrEmptySessions,
		},
		{
			name: "zero sessions rejected", sessions: []domain.Session{}, capacity: 8, guestAllowance: 0,
			paymentMethod: domain.PaymentMethodEither, entryFee: validFee(), format: domain.FormatSingles,
			wantErr: domain.ErrEmptySessions,
		},
		{
			name: "session with nil court ids rejected",
			sessions: []domain.Session{
				{Range: mustRange(t, "2026-09-05T09:00:00Z", "2026-09-05T11:00:00Z"), CourtIDs: nil},
			},
			capacity: 8, guestAllowance: 0,
			paymentMethod: domain.PaymentMethodEither, entryFee: validFee(), format: domain.FormatSingles,
			wantErr: domain.ErrEmptyCourtIDs,
		},
		{
			name: "second session with empty court ids rejected",
			sessions: []domain.Session{
				{Range: mustRange(t, "2026-09-05T09:00:00Z", "2026-09-05T11:00:00Z"), CourtIDs: []string{"court-1"}},
				{Range: mustRange(t, "2026-09-06T09:00:00Z", "2026-09-06T11:00:00Z"), CourtIDs: []string{}},
			},
			capacity: 8, guestAllowance: 0,
			paymentMethod: domain.PaymentMethodEither, entryFee: validFee(), format: domain.FormatSingles,
			wantErr: domain.ErrEmptyCourtIDs,
		},
		{
			name:     "zero-duration session range rejected",
			sessions: []domain.Session{{Range: zeroDuration, CourtIDs: []string{"court-1"}}},
			capacity: 8, guestAllowance: 0,
			paymentMethod: domain.PaymentMethodEither, entryFee: validFee(), format: domain.FormatSingles,
			wantErr: domain.ErrInvalidTimeRange,
		},
		{
			name:     "inverted session range rejected",
			sessions: []domain.Session{{Range: inverted, CourtIDs: []string{"court-1"}}},
			capacity: 8, guestAllowance: 0,
			paymentMethod: domain.PaymentMethodEither, entryFee: validFee(), format: domain.FormatSingles,
			wantErr: domain.ErrInvalidTimeRange,
		},
		{
			name: "negative guest allowance is rejected", sessions: validSessions(t), capacity: 8, guestAllowance: -1,
			paymentMethod: domain.PaymentMethodEither, entryFee: validFee(), format: domain.FormatSingles,
			wantErr: domain.ErrInvalidGuestAllowance,
		},
		{
			name: "zero guest allowance accepted", sessions: validSessions(t), capacity: 8, guestAllowance: 0,
			paymentMethod: domain.PaymentMethodEither, entryFee: validFee(), format: domain.FormatSingles,
		},
		{
			name: "invalid payment method rejected", sessions: validSessions(t), capacity: 8, guestAllowance: 0,
			paymentMethod: domain.PaymentMethod("crypto"), entryFee: validFee(), format: domain.FormatSingles,
			wantErr: domain.ErrInvalidPaymentMethod,
		},
		{
			name: "empty payment method rejected", sessions: validSessions(t), capacity: 8, guestAllowance: 0,
			paymentMethod: domain.PaymentMethod(""), entryFee: validFee(), format: domain.FormatSingles,
			wantErr: domain.ErrInvalidPaymentMethod,
		},
		{
			name: "payment method cash accepted", sessions: validSessions(t), capacity: 8, guestAllowance: 0,
			paymentMethod: domain.PaymentMethodCash, entryFee: validFee(), format: domain.FormatSingles,
		},
		{
			name: "invalid format rejected", sessions: validSessions(t), capacity: 8, guestAllowance: 0,
			paymentMethod: domain.PaymentMethodEither, entryFee: validFee(), format: domain.Format("mixed"),
			wantErr: domain.ErrInvalidFormat,
		},
		{
			name: "empty format rejected", sessions: validSessions(t), capacity: 8, guestAllowance: 0,
			paymentMethod: domain.PaymentMethodEither, entryFee: validFee(), format: domain.Format(""),
			wantErr: domain.ErrInvalidFormat,
		},
		{
			name: "format doubles accepted", sessions: validSessions(t), capacity: 8, guestAllowance: 0,
			paymentMethod: domain.PaymentMethodEither, entryFee: validFee(), format: domain.FormatDoubles,
		},
		{
			name: "zero entry fee accepted as a free competition", sessions: validSessions(t), capacity: 8, guestAllowance: 0,
			paymentMethod: domain.PaymentMethodEither, entryFee: domain.Money{}, format: domain.FormatSingles,
		},
		{
			name: "non-empty amount with empty currency rejected", sessions: validSessions(t), capacity: 8, guestAllowance: 0,
			paymentMethod: domain.PaymentMethodEither, entryFee: domain.Money{AmountCents: 500}, format: domain.FormatSingles,
			wantErr: domain.ErrInvalidMoney,
		},
		{
			name: "negative entry fee rejected", sessions: validSessions(t), capacity: 8, guestAllowance: 0,
			paymentMethod: domain.PaymentMethodEither, entryFee: domain.Money{AmountCents: -500, CurrencyCode: "GBP"}, format: domain.FormatSingles,
			wantErr: domain.ErrInvalidMoney,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := domain.NewCompetition(
				"comp-1", "host-1", "Autumn Open", "venue-1",
				tt.sessions, tt.capacity, tt.guestAllowance,
				tt.paymentMethod, tt.entryFee, tt.format, "tok-1",
			)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("got err %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
		})
	}
}

// TestNewCompetition_OverlappingSessions is the ticket's Given/When/Then:
// two sessions of the same Competition reserving the same court at
// overlapping times are rejected at construction, because the Booking-level
// invariant would otherwise reject the second reservation mid-way through
// T9.3's reservation loop and force a rollback for a mistake detectable up
// front. Back-to-back sessions on one court are NOT an overlap — ranges are
// half-open [start, end), the same rule as Booking.
func TestNewCompetition_OverlappingSessions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		sessions []domain.Session
		wantErr  error
	}{
		{
			name: "same court overlapping sessions rejected",
			sessions: []domain.Session{
				{Range: mustRange(t, "2026-09-05T10:00:00Z", "2026-09-05T12:00:00Z"), CourtIDs: []string{"court-1"}},
				{Range: mustRange(t, "2026-09-05T11:00:00Z", "2026-09-05T13:00:00Z"), CourtIDs: []string{"court-1"}},
			},
			wantErr: domain.ErrOverlappingSessions,
		},
		{
			name: "overlap on any shared court rejected even when other courts differ",
			sessions: []domain.Session{
				{Range: mustRange(t, "2026-09-05T10:00:00Z", "2026-09-05T12:00:00Z"), CourtIDs: []string{"court-1", "court-2"}},
				{Range: mustRange(t, "2026-09-05T11:00:00Z", "2026-09-05T13:00:00Z"), CourtIDs: []string{"court-2", "court-3"}},
			},
			wantErr: domain.ErrOverlappingSessions,
		},
		{
			name: "identical sessions on the same court rejected",
			sessions: []domain.Session{
				{Range: mustRange(t, "2026-09-05T10:00:00Z", "2026-09-05T12:00:00Z"), CourtIDs: []string{"court-1"}},
				{Range: mustRange(t, "2026-09-05T10:00:00Z", "2026-09-05T12:00:00Z"), CourtIDs: []string{"court-1"}},
			},
			wantErr: domain.ErrOverlappingSessions,
		},
		{
			name: "a court repeated within one session's own court list overlaps itself",
			sessions: []domain.Session{
				{Range: mustRange(t, "2026-09-05T10:00:00Z", "2026-09-05T12:00:00Z"), CourtIDs: []string{"court-1", "court-1"}},
			},
			wantErr: domain.ErrOverlappingSessions,
		},
		{
			name: "back-to-back sessions on the same court allowed",
			sessions: []domain.Session{
				{Range: mustRange(t, "2026-09-05T10:00:00Z", "2026-09-05T12:00:00Z"), CourtIDs: []string{"court-1"}},
				{Range: mustRange(t, "2026-09-05T12:00:00Z", "2026-09-05T14:00:00Z"), CourtIDs: []string{"court-1"}},
			},
		},
		{
			name: "overlapping sessions on different courts allowed",
			sessions: []domain.Session{
				{Range: mustRange(t, "2026-09-05T10:00:00Z", "2026-09-05T12:00:00Z"), CourtIDs: []string{"court-1"}},
				{Range: mustRange(t, "2026-09-05T11:00:00Z", "2026-09-05T13:00:00Z"), CourtIDs: []string{"court-2"}},
			},
		},
		{
			name: "sessions across different dates on the same court allowed",
			sessions: []domain.Session{
				{Range: mustRange(t, "2026-09-05T10:00:00Z", "2026-09-05T12:00:00Z"), CourtIDs: []string{"court-1"}},
				{Range: mustRange(t, "2026-09-06T10:00:00Z", "2026-09-06T12:00:00Z"), CourtIDs: []string{"court-1"}},
				{Range: mustRange(t, "2026-09-07T10:00:00Z", "2026-09-07T12:00:00Z"), CourtIDs: []string{"court-1"}},
			},
		},
		{
			name: "three sessions where only the first and last overlap is still rejected",
			sessions: []domain.Session{
				{Range: mustRange(t, "2026-09-05T10:00:00Z", "2026-09-05T12:00:00Z"), CourtIDs: []string{"court-1"}},
				{Range: mustRange(t, "2026-09-06T10:00:00Z", "2026-09-06T12:00:00Z"), CourtIDs: []string{"court-1"}},
				{Range: mustRange(t, "2026-09-05T11:00:00Z", "2026-09-05T13:00:00Z"), CourtIDs: []string{"court-1"}},
			},
			wantErr: domain.ErrOverlappingSessions,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := domain.NewCompetition(
				"comp-1", "host-1", "Autumn Open", "venue-1",
				tt.sessions, 8, 0,
				domain.PaymentMethodEither, validFee(), domain.FormatSingles, "tok-1",
			)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("got err %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
		})
	}
}

// TestCompetition_EnsureHost proves the object-level (BOLA-shaped)
// authorization check T9.3's CancelCompetition depends on — mirrors
// facilities.Facility.EnsureOwner, including rejecting an empty actor.
func TestCompetition_EnsureHost(t *testing.T) {
	t.Parallel()

	c := newValidCompetition(t, 8, 0)

	tests := []struct {
		name        string
		actorUserID string
		wantErr     error
	}{
		{"the host is allowed", "host-1", nil},
		{"a different user is rejected", "host-2", domain.ErrNotCompetitionHost},
		{"an empty actor is rejected", "", domain.ErrNotCompetitionHost},
		{"a case-mismatched host id is rejected", "HOST-1", domain.ErrNotCompetitionHost},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := c.EnsureHost(tt.actorUserID)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("got err %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
		})
	}
}

// TestCompetition_EnsureHost_EmptyHostIDStillRejectsEmptyActor guards the
// degenerate case where a Competition somehow carries an empty HostID: an
// empty actor must not be accepted just because both sides are empty.
func TestCompetition_EnsureHost_EmptyHostIDStillRejectsEmptyActor(t *testing.T) {
	t.Parallel()

	c := domain.Competition{HostID: ""}
	if err := c.EnsureHost(""); !errors.Is(err, domain.ErrNotCompetitionHost) {
		t.Fatalf("got err %v, want %v", err, domain.ErrNotCompetitionHost)
	}
}

// TestCompetition_Cancel proves scheduled -> cancelled is the only legal
// transition, and that cancelling twice is rejected rather than silently
// idempotent (mirrors booking.Booking.Cancel / socialplay.Game.Cancel).
func TestCompetition_Cancel(t *testing.T) {
	t.Parallel()

	t.Run("scheduled to cancelled succeeds", func(t *testing.T) {
		t.Parallel()

		c := newValidCompetition(t, 8, 0)
		if err := c.Cancel(); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if c.Status != domain.StatusCancelled {
			t.Fatalf("status = %v, want %v", c.Status, domain.StatusCancelled)
		}
	})

	t.Run("cancelling twice is rejected", func(t *testing.T) {
		t.Parallel()

		c := newValidCompetition(t, 8, 0)
		if err := c.Cancel(); err != nil {
			t.Fatalf("unexpected err on first cancel: %v", err)
		}
		if err := c.Cancel(); !errors.Is(err, domain.ErrIllegalStatusTransition) {
			t.Fatalf("got err %v, want %v", err, domain.ErrIllegalStatusTransition)
		}
		if c.Status != domain.StatusCancelled {
			t.Fatalf("status = %v, want %v", c.Status, domain.StatusCancelled)
		}
	})
}

// TestEnumIsValid keeps PaymentMethod and Format closed enums — the
// domain-side guard against arbitrary strings reaching the field.
func TestEnumIsValid(t *testing.T) {
	t.Parallel()

	t.Run("payment method", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			value domain.PaymentMethod
			want  bool
		}{
			{domain.PaymentMethodOnline, true},
			{domain.PaymentMethodCash, true},
			{domain.PaymentMethodEither, true},
			{domain.PaymentMethod("online "), false},
			{domain.PaymentMethod("Online"), false},
			{domain.PaymentMethod("crypto"), false},
			{domain.PaymentMethod(""), false},
		}
		for _, tt := range tests {
			if got := tt.value.IsValid(); got != tt.want {
				t.Fatalf("PaymentMethod(%q).IsValid() = %v, want %v", tt.value, got, tt.want)
			}
		}
	})

	t.Run("format", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			value domain.Format
			want  bool
		}{
			{domain.FormatSingles, true},
			{domain.FormatDoubles, true},
			{domain.Format("Doubles"), false},
			{domain.Format("mixed"), false},
			{domain.Format(""), false},
		}
		for _, tt := range tests {
			if got := tt.value.IsValid(); got != tt.want {
				t.Fatalf("Format(%q).IsValid() = %v, want %v", tt.value, got, tt.want)
			}
		}
	})
}
