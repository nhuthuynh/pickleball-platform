package port

import "time"

// LocationLoader resolves an IANA timezone identifier (e.g.
// "Asia/Ho_Chi_Minh") to a *time.Location (T68.1, ADR-0004, closes #356).
//
// # Why this is a port at all
//
// Because resolving a zone name is I/O. time.LoadLocation consults $ZONEINFO,
// then the host's tz database, then the copy embedded by `import _
// "time/tzdata"` — file reads, which internal/booking/domain must not do
// (CLAUDE.md rule 2) and which internal/booking/app reaches only through a
// port like every other outside-world dependency it has. The production
// implementation is internal/platform/tz.Loader, the same shape
// internal/platform/idgen.UUID has for IDGenerator: one tiny adapter shared
// by every context rather than a per-context dependency on the host.
//
// # What implementations must and must not do
//
// Return an error for a name that does not resolve. In particular, **do not
// map an empty name to UTC**, which is what time.LoadLocation itself does —
// `time.LoadLocation("")` returns UTC with no error, and a facility whose zone
// was never recorded would then be silently priced in UTC. That is the exact
// defect this ticket closes, so the empty case is an error at both ends: the
// app layer rejects "" before calling (app.Service.courtLocation) and
// tz.Loader rejects it too.
//
// A returned *time.Location must be non-nil whenever the error is nil;
// domain.ResolvePrice answers domain.ErrFacilityTimezoneUnset for a nil
// Location rather than assuming UTC, so a sloppy implementation fails loudly
// instead of mispricing.
//
// The method deliberately takes no context.Context: it is a pure, local,
// in-process lookup against a database compiled into the binary or sitting on
// disk, with nothing to cancel and no deadline to honour. Adding one would
// imply a remote call that no implementation makes.
type LocationLoader interface {
	LoadLocation(name string) (*time.Location, error)
}
