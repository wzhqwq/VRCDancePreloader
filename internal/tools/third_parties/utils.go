package third_parties

import "github.com/wzhqwq/VRCDancePreloader/internal/utils"

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
// permission and the current external conditions.
//
// Its EventManager is what interactive.RemoteManager.BindAvailability listens
// to, and its `true` payload is a wake-up call rather than a piece of state
// (the manager answers it with retryUnavailableEntries).
//
// Update/Retry are called from the owning provider's loop goroutine, which is
// also the only reader of the derived state.
type ResourceGate struct {
	em *utils.EventManager[bool]

	available bool
	allowed   bool
	satisfied bool
}

func newResourceGate() *ResourceGate {
	return &ResourceGate{em: utils.NewEventManager[bool]()}
}

// Subscribe hands the gate to interactive.RemoteManager.BindAvailability.
func (g *ResourceGate) Subscribe() *utils.EventSubscriber[bool] {
	return g.em.SubscribeEvent()
}

// Update recomputes the gate from both of its inputs.
//
// While the gate is satisfied it publishes `true` on every update instead of
// only on the false->true edge: the subscriber's reaction is idempotent, and an
// edge-only policy would turn a single dropped event (utils/event.go drops when
// a subscriber's buffer is full) into a resource that stays unavailable until
// something else happens to move the gate — see review/09 §2.5 and review/02 A7.
func (g *ResourceGate) Update(allowed ResourceAllowed, available ResourceAvailable) {
	g.allowed = bool(allowed)
	g.available = bool(available)
	g.satisfied = g.allowed && g.available

	g.NotifyIfSatisfied()
}

// Retry publishes `true` again for the callers that know an external condition
// changed while the gate's own inputs did not: a replaced HTTP client, for
// example, which makes the fetches that failed with the old one worth another
// try.
func (g *ResourceGate) Retry() {
	g.NotifyIfSatisfied()
}

// BothSatisfied reports the state computed by the last Update. Nothing calls it
// today: it is the read side of the gate, kept deliberately (AGENTS.md §7).
func (g *ResourceGate) BothSatisfied() bool {
	return g.satisfied
}

func (g *ResourceGate) NotifyIfSatisfied() {
	if g.satisfied {
		g.em.NotifySubscribers(true)
	}
}
