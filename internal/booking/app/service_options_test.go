// Tests for ServiceOptions, the named-field replacement for what was a
// seven-argument positional NewService (T13.8, closing issue #123).
//
// **Why this file exists at all.** A positional constructor makes omitting a
// dependency a compile error: seven parameters means seven arguments or the
// build fails. An options struct gives that up — every field a caller forgets
// is silently the zero value, and for an interface field the zero value is nil.
// Without a guard the refactor would trade "unbuildable" for "builds fine, then
// panics on some later request with a nil-pointer dereference and no hint about
// which of seven dependencies was never wired". That is a strictly worse
// failure mode than the one being replaced, and it is what these tests exist to
// prevent: the property is preserved, moved from the compiler to construction
// time, not dropped.
//
// Construction time is the right moment because that is where the wiring
// mistake is actually made — cmd/server builds every Service before it serves
// its first request, so a missing dependency stops the process at boot rather
// than at 03:00 on the one endpoint that needed it. Same argument
// internal/platform/auth.EnsureVerifierConfigured makes for a nil TokenVerifier
// (T13.5, issue #136); this is that reasoning applied to service dependencies.
package app_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/nhuthuynh/white-label/internal/booking/app"
)

// completeOptions returns a ServiceOptions with every required dependency
// supplied, so each test below can blank exactly one field and attribute the
// resulting failure to that field alone.
func completeOptions() app.ServiceOptions {
	return app.ServiceOptions{
		Bookings:       newInMemoryRepo(),
		PricingRules:   &fakePricingRepo{},
		DiscountRules:  newFakeDiscountRepo(),
		RecurringHires: newFakeRecurringHireRepo(),
		Facilities:     &fakeFacilityLookup{},
		Identity:       &fakeIdentityLookup{},
		IDs:            &sequentialIDs{},
	}
}

// TestNewServicePanicsOnMissingRequiredDependency is the guard's core table:
// every field of ServiceOptions is required, and leaving any one of them unset
// fails at construction with a message that names it.
//
// The message content is asserted, not just the panic: "something was nil" is
// barely more useful than the nil-dereference it replaces. Naming the field is
// the whole point — it turns a wiring bug into a one-line fix.
func TestNewServicePanicsOnMissingRequiredDependency(t *testing.T) {
	tests := []struct {
		name  string
		blank func(*app.ServiceOptions)
		want  string
	}{
		{"Bookings", func(o *app.ServiceOptions) { o.Bookings = nil }, "Bookings"},
		{"PricingRules", func(o *app.ServiceOptions) { o.PricingRules = nil }, "PricingRules"},
		{"DiscountRules", func(o *app.ServiceOptions) { o.DiscountRules = nil }, "DiscountRules"},
		{"RecurringHires", func(o *app.ServiceOptions) { o.RecurringHires = nil }, "RecurringHires"},
		{"Facilities", func(o *app.ServiceOptions) { o.Facilities = nil }, "Facilities"},
		{"Identity", func(o *app.ServiceOptions) { o.Identity = nil }, "Identity"},
		{"IDs", func(o *app.ServiceOptions) { o.IDs = nil }, "IDs"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := completeOptions()
			tt.blank(&opts)

			defer func() {
				r := recover()
				if r == nil {
					t.Fatalf("NewService with a nil %s did not panic; a missing required dependency must fail at construction, not on first use", tt.want)
				}
				msg, ok := r.(string)
				if !ok {
					t.Fatalf("NewService panicked with %T (%v), want a string message naming the missing field", r, r)
				}
				if !strings.Contains(msg, tt.want) {
					t.Errorf("panic message %q does not name the missing field %q", msg, tt.want)
				}
			}()

			_ = app.NewService(opts)
		})
	}
}

// TestNewServiceAcceptsCompleteOptions is the other half of the table above:
// the guard must not fire on a correctly wired Service. Without this, a
// Validate that always returned an error would pass every test above.
func TestNewServiceAcceptsCompleteOptions(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("NewService with every dependency supplied panicked: %v", r)
		}
	}()

	if svc := app.NewService(completeOptions()); svc == nil {
		t.Fatal("NewService returned nil for complete options")
	}
}

// TestValidateReportsEveryMissingDependency pins that Validate reports the
// whole set at once rather than the first one it finds.
//
// Fixing seven wiring mistakes one boot at a time is exactly the loop this
// ticket is meant to end, so "reports all of them" is a real requirement and
// not an incidental detail of the implementation.
func TestValidateReportsEveryMissingDependency(t *testing.T) {
	err := app.ServiceOptions{}.Validate()
	if err == nil {
		t.Fatal("(ServiceOptions{}).Validate() = nil, want an error naming every missing dependency")
	}
	if !errors.Is(err, app.ErrMissingDependency) {
		t.Errorf("Validate() error does not wrap ErrMissingDependency: %v", err)
	}

	for _, field := range []string{"Bookings", "PricingRules", "DiscountRules", "RecurringHires", "Facilities", "Identity", "IDs"} {
		if !strings.Contains(err.Error(), field) {
			t.Errorf("Validate() error %q does not name missing field %q", err, field)
		}
	}
}

// TestValidateAcceptsCompleteOptions is the Validate-level companion to
// TestNewServiceAcceptsCompleteOptions.
func TestValidateAcceptsCompleteOptions(t *testing.T) {
	if err := completeOptions().Validate(); err != nil {
		t.Fatalf("complete ServiceOptions.Validate() = %v, want nil", err)
	}
}

// TestNewServiceRejectsTypedNilDependency covers the half a plain `== nil`
// check misses: an interface field holding a nil *pointer* is not a nil
// interface, so it passes the easy check and then panics on the first method
// call anyway — the precise fail-late shape this guard exists to stop.
//
// Same reasoning, and the same reflect-based remedy, as
// internal/platform/auth.verifierIsNil (T13.5). A guard that is narrower than
// the failure it protects against is not a guard.
func TestNewServiceRejectsTypedNilDependency(t *testing.T) {
	opts := completeOptions()
	opts.PricingRules = (*fakePricingRepo)(nil)

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("NewService with a typed-nil PricingRules did not panic; an interface holding a nil pointer is as unusable as a nil interface")
		}
		if msg, ok := r.(string); ok && !strings.Contains(msg, "PricingRules") {
			t.Errorf("panic message %q does not name the unusable field %q", msg, "PricingRules")
		}
	}()

	_ = app.NewService(opts)
}
