package third_parties

import "sync/atomic"

// ResourceAvailable is "the current external conditions allow fetching this
// resource" (the fetch mode, an api key, yt-dlp, or the dance room catalog
// being loaded). ResourceAllowed is "the user's configuration allows fetching
// this resource".
//
// Both are named types on purpose. As a pair of plain bools the two arguments of
// the gate below could be swapped without the compiler noticing, which is
// exactly the bug 0eed86d fixed (see review/09 §6).
type ResourceAvailable bool
type ResourceAllowed bool

// ResourceGate derives "this resource may be fetched right now" from the user's
// permission and the current external conditions, and is what
// interactive.RemoteManager.BindAvailability consumes.
//
// It is an AvailabilitySource: the manager reads Available when it is woken
// instead of trusting the wake-up, so a wake-up that is never delivered (or
// delivered twice) cannot leave the manager believing something that is no
// longer true.
//
// Update/Retry are called from the owning provider's loop goroutine, which is
// also the only reader of the derived state.
type ResourceGate struct {
	// wake has room for exactly one pending wake-up: the receiver re-reads the
	// level, so notifying it twice is the same as notifying it once.
	wake chan struct{}

	// satisfied crosses goroutines: it is written by the owning provider's loop
	// and read by the manager's availability goroutine (Available). It has to be
	// atomic because the wake-up channel only orders the reads that follow a
	// *delivered* wake-up, and a coalesced one carries no such edge.
	satisfied atomic.Bool

	// allowed and available are only touched by the owning provider's loop.
	allowed   bool
	available bool
}

func newResourceGate() *ResourceGate {
	return &ResourceGate{wake: make(chan struct{}, 1)}
}

// Available is the level the manager reads; it is also the read side of the gate.
func (g *ResourceGate) Available() bool {
	return g.satisfied.Load()
}

// Wakes implements interactive.AvailabilitySource.
func (g *ResourceGate) Wakes() <-chan struct{} {
	return g.wake
}

// Update recomputes the gate from both of its inputs.
func (g *ResourceGate) Update(allowed ResourceAllowed, available ResourceAvailable) {
	g.allowed = bool(allowed)
	g.available = bool(available)
	g.satisfied.Store(g.allowed && g.available)

	g.NotifyIfSatisfied()
}

// Retry wakes the manager again for the callers that know an external condition
// changed while the gate's own inputs did not: a replaced HTTP client, for
// example, which makes the fetches that failed with the old one worth another
// try.
func (g *ResourceGate) Retry() {
	g.NotifyIfSatisfied()
}

func (g *ResourceGate) NotifyIfSatisfied() {
	if !g.satisfied.Load() {
		return
	}

	select {
	case g.wake <- struct{}{}:
	default:
		// A wake-up is already pending and the receiver will read the level that
		// is set right now, so this one adds nothing.
	}
}
