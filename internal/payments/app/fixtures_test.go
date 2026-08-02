package app_test

// fixedIDs is a deterministic port.IDGenerator test double: it returns
// each of the seeded IDs in order, then panics if asked for more than
// were seeded (a test bug, not a runtime concern) — the same pattern used
// by the Booking context's app-layer tests so behaviour stays
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
