// Tests for ServiceOptions, the named-field replacement for what was a
// five-argument positional NewService (T13.8, closing issue #123).
//
// The reasoning is identical to internal/booking/app's copy of this file, and
// the canonical write-up lives there: an options struct trades "omitting a
// dependency is a compile error" for "omitting a dependency is a silent nil",
// so the guard tested here moves that property from the compiler to
// construction time rather than dropping it.
package app_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/nhuthuynh/white-label/internal/socialplay/app"
)

// completeOptions returns a ServiceOptions with every required dependency
// supplied, so each test below can blank exactly one field and attribute the
// resulting failure to that field alone.
func completeOptions() app.ServiceOptions {
	return app.ServiceOptions{
		IDs:           &sequentialIDs{},
		Games:         newFakeGameRepository(),
		Registrations: newFakeRegistrationRepository(),
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
	}
}

// TestNewServicePanicsOnMissingRequiredDependency: every field of
// ServiceOptions is required, and leaving any one of them unset fails at
// construction with a message that names it.
func TestNewServicePanicsOnMissingRequiredDependency(t *testing.T) {
	tests := []struct {
		name  string
		blank func(*app.ServiceOptions)
		want  string
	}{
		{"IDs", func(o *app.ServiceOptions) { o.IDs = nil }, "IDs"},
		{"Games", func(o *app.ServiceOptions) { o.Games = nil }, "Games"},
		{"Registrations", func(o *app.ServiceOptions) { o.Registrations = nil }, "Registrations"},
		{"Waitlist", func(o *app.ServiceOptions) { o.Waitlist = nil }, "Waitlist"},
		{"Matches", func(o *app.ServiceOptions) { o.Matches = nil }, "Matches"},
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

// TestNewServiceAcceptsCompleteOptions: the guard must not fire on a correctly
// wired Service, or every test above would pass against a Validate that always
// returned an error.
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
func TestValidateReportsEveryMissingDependency(t *testing.T) {
	err := app.ServiceOptions{}.Validate()
	if err == nil {
		t.Fatal("(ServiceOptions{}).Validate() = nil, want an error naming every missing dependency")
	}
	if !errors.Is(err, app.ErrMissingDependency) {
		t.Errorf("Validate() error does not wrap ErrMissingDependency: %v", err)
	}

	for _, field := range []string{"IDs", "Games", "Registrations", "Waitlist", "Matches"} {
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
// check misses — see internal/booking/app/service_options_test.go's copy for
// the full write-up, and internal/platform/auth.verifierIsNil (T13.5) for the
// precedent.
func TestNewServiceRejectsTypedNilDependency(t *testing.T) {
	opts := completeOptions()
	opts.Games = (*fakeGameRepository)(nil)

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("NewService with a typed-nil Games did not panic; an interface holding a nil pointer is as unusable as a nil interface")
		}
		if msg, ok := r.(string); ok && !strings.Contains(msg, "Games") {
			t.Errorf("panic message %q does not name the unusable field %q", msg, "Games")
		}
	}()

	_ = app.NewService(opts)
}
