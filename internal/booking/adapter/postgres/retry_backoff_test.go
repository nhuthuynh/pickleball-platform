package postgres

import (
	"testing"
	"time"
)

// These tests exist because the thing they pin was REMOVED once already --
// in effect, by never being there. retryBackoff returned attempt*5ms, the
// same value for every caller, and nothing in the suite noticed that a
// concurrent burst would therefore retry in lockstep. Issue #375 is what that
// cost: on a loaded CI runner all 19 losers of a 20-way burst returned 40P01
// with conflicts = 0.
//
// Deliberately NOT integration tests. The end-to-end flake reproduces about
// once in fifteen runs under CPU saturation (measured T68), which is far too
// rare to gate on -- twenty green runs would happen ~23% of the time with no
// fix at all. These are deterministic tests of the MECHANISM, and they fail
// immediately if the jitter is deleted.

func TestRetryBackoffIsJittered(t *testing.T) {
	// A deterministic backoff synchronises the herd it is meant to break up.
	// If someone replaces the body with `attempt * 5 * time.Millisecond`,
	// every call returns the same value and this test fails.
	for _, attempt := range []int{1, 2, 3, 6} {
		seen := map[time.Duration]bool{}
		for i := 0; i < 200; i++ {
			seen[retryBackoff(attempt)] = true
		}
		if len(seen) < 2 {
			t.Errorf("attempt %d: retryBackoff returned %d distinct value(s) over 200 calls; "+
				"a constant backoff makes every contender in a burst retry at the same instant, "+
				"which is the cause of #375", attempt, len(seen))
		}
	}
}

func TestRetryBackoffStaysWithinEqualJitterBounds(t *testing.T) {
	// Equal jitter: at least half the base delay, never more than the base.
	// The floor matters -- a near-zero wait retries before the winning
	// transaction has committed, which is why this is not full jitter.
	for _, attempt := range []int{1, 2, 3, 6} {
		base := time.Duration(attempt) * 5 * time.Millisecond
		lo, hi := base/2, base
		for i := 0; i < 200; i++ {
			got := retryBackoff(attempt)
			if got < lo || got > hi {
				t.Fatalf("attempt %d: retryBackoff = %v, want within [%v, %v]", attempt, got, lo, hi)
			}
		}
	}
}

func TestRetryBackoffGrowsWithAttempt(t *testing.T) {
	// Each attempt's floor must clear the previous attempt's ceiling is too
	// strong for equal jitter; what must hold is that the floor increases, so
	// a burst that is still contending waits longer each round.
	var prevFloor time.Duration
	for attempt := 1; attempt <= createBookingMaxAttempts; attempt++ {
		floor := time.Duration(attempt) * 5 * time.Millisecond / 2
		if attempt > 1 && floor <= prevFloor {
			t.Errorf("attempt %d floor %v does not exceed attempt %d floor %v", attempt, floor, attempt-1, prevFloor)
		}
		min := time.Duration(1<<62 - 1)
		for i := 0; i < 200; i++ {
			if d := retryBackoff(attempt); d < min {
				min = d
			}
		}
		if min < floor {
			t.Errorf("attempt %d: observed minimum %v below floor %v", attempt, min, floor)
		}
		prevFloor = floor
	}
}

func TestRetryBudgetIsWideEnoughForATwentyWayBurst(t *testing.T) {
	// The old budget was 3 attempts of un-jittered 5/10ms: 15ms total, which
	// #375 shows is not enough for a 20-way burst on a loaded runner. This
	// pins the budget itself, so shrinking it back is a visible failure
	// rather than a silent regression.
	if createBookingMaxAttempts < 5 {
		t.Errorf("createBookingMaxAttempts = %d; #375 needed more than the original 3", createBookingMaxAttempts)
	}
	var worst time.Duration
	for attempt := 1; attempt < createBookingMaxAttempts; attempt++ {
		worst += time.Duration(attempt) * 5 * time.Millisecond
	}
	if worst < 50*time.Millisecond {
		t.Errorf("worst-case total retry window is %v; too short to drain a 20-way burst", worst)
	}
}
