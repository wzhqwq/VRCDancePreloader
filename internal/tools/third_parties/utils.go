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
// permission and the current external conditions, and is what
// interactive.RemoteManager.BindAvailability consumes.
//
// It is a utils.LevelSource[bool]: the manager reads Current when it is woken
// instead of trusting the wake-up, so a wake-up that is never delivered (or
// delivered twice) cannot leave the manager believing something that is no
// longer true.
//
// Update/Retry are called from the owning provider's loop goroutine, which is
// also the only writer of the derived state.
type ResourceGate struct {
	level   *utils.Level[bool]
	watcher *utils.LevelWatcher[bool]

	allowed   bool
	available bool
}

func newResourceGate() *ResourceGate {
	level := utils.NewLevel(false)

	return &ResourceGate{
		level:   level,
		watcher: level.Subscribe(),
	}
}

// Current is the level the manager reads; it is also the read side of the gate.
func (g *ResourceGate) Current() bool {
	return g.level.Current()
}

// Wakes implements utils.LevelSource (the gate has exactly one consumer: the
// availability loop of the manager it is bound to).
func (g *ResourceGate) Wakes() <-chan struct{} {
	return g.watcher.Wakes()
}

// Update recomputes the gate from both of its inputs and wakes the manager.
func (g *ResourceGate) Update(allowed ResourceAllowed, available ResourceAvailable) {
	g.allowed = bool(allowed)
	g.available = bool(available)

	g.level.Store(g.allowed && g.available)
}

// Retry wakes the manager again for the callers that know an external condition
// changed while the gate's own inputs did not: a replaced HTTP client, for
// example, which makes the fetches that failed with the old one worth another
// try. Whether the gate is usable is not decided here — a wake-up only means
// "read the level again", and an unusable gate tells the manager nothing to do.
func (g *ResourceGate) Retry() {
	g.level.Wake()
}
