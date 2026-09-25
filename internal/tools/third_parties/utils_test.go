package third_parties

import (
	"testing"
)

// allow / ready keep the steps below readable while still typing the two inputs
// differently: swapping them is a compile error (see ResourceGate).
func allow(v bool) ResourceAllowed   { return ResourceAllowed(v) }
func ready(v bool) ResourceAvailable { return ResourceAvailable(v) }

// pendingWakes counts (and drains) the wake-ups the gate has signalled.
func pendingWakes(g *ResourceGate) int {
	woken := 0

drain:
	for {
		select {
		case <-g.Wakes():
			woken++
		default:
			break drain
		}
	}

	return woken
}

// gateStep is one Update/Retry call plus what the manager must be able to see
// because of it: whether a wake-up has to be waiting, and the level at that
// moment.
type gateStep struct {
	name  string
	apply func(*ResourceGate)
	wake  bool
	avail bool
}

func assertSteps(t *testing.T, steps []gateStep) {
	t.Helper()

	gate := newResourceGate()

	for i, step := range steps {
		step.apply(gate)

		woken := pendingWakes(gate)
		if (woken > 0) != step.wake {
			t.Fatalf("step %d (%s): got %d wake-ups, want wake=%v", i, step.name, woken, step.wake)
		}
		if got := gate.Available(); got != step.avail {
			t.Fatalf("step %d (%s): Available()=%v, want %v", i, step.name, got, step.avail)
		}
	}
}

// TestResourceGateWakesWhenItBecomesSatisfied covers the two ways a gate can
// turn usable, including the one that 0eed86d fixed: the permission was granted
// long before the dance room catalog finished loading.
func TestResourceGateWakesWhenItBecomesSatisfied(t *testing.T) {
	assertSteps(t, []gateStep{
		{"permission only", func(g *ResourceGate) { g.Update(allow(true), ready(false)) }, false, false},
		{"the catalog data arrives", func(g *ResourceGate) { g.Update(allow(true), ready(true)) }, true, true},
		{"the data goes away again", func(g *ResourceGate) { g.Update(allow(true), ready(false)) }, false, false},
	})

	assertSteps(t, []gateStep{
		{"data only", func(g *ResourceGate) { g.Update(allow(false), ready(true)) }, false, false},
		{"the user enables the resource", func(g *ResourceGate) { g.Update(allow(true), ready(true)) }, true, true},
	})
}

// TestResourceGateStaysSilentWhileUnsatisfied pins the fact that the manager is
// never woken while the gate is unusable.
func TestResourceGateStaysSilentWhileUnsatisfied(t *testing.T) {
	assertSteps(t, []gateStep{
		{"neither", func(g *ResourceGate) { g.Update(allow(false), ready(false)) }, false, false},
		{"data only", func(g *ResourceGate) { g.Update(allow(false), ready(true)) }, false, false},
		{"permission revoked", func(g *ResourceGate) { g.Update(allow(false), ready(false)) }, false, false},
		{"data gone", func(g *ResourceGate) { g.Update(allow(true), ready(false)) }, false, false},
		{"retry while unsatisfied", func(g *ResourceGate) { g.Retry() }, false, false},
	})
}

// TestResourceGateRetryWakesWhileSatisfied covers the "a new HTTP client
// replaced the old one" path: the inputs did not change, but the fetches that
// failed with the old client are worth another try.
func TestResourceGateRetryWakesWhileSatisfied(t *testing.T) {
	assertSteps(t, []gateStep{
		{"satisfied", func(g *ResourceGate) { g.Update(allow(true), ready(true)) }, true, true},
		{"retry", func(g *ResourceGate) { g.Retry() }, true, true},
		{"permission lost", func(g *ResourceGate) { g.Update(allow(false), ready(true)) }, false, false},
		{"retry while unsatisfied", func(g *ResourceGate) { g.Retry() }, false, false},
		{"permission back", func(g *ResourceGate) { g.Update(allow(true), ready(true)) }, true, true},
	})
}

// TestResourceGateCoalescesWakes pins the property the manager relies on: one
// pending wake-up is enough, because handling it re-reads the level. A wake-up
// that is dropped because one is already pending therefore loses nothing.
func TestResourceGateCoalescesWakes(t *testing.T) {
	gate := newResourceGate()

	gate.Update(allow(true), ready(true))
	gate.Update(allow(true), ready(true))
	gate.Retry()

	if woken := pendingWakes(gate); woken != 1 {
		t.Fatalf("got %d pending wake-ups after three satisfied updates, want 1", woken)
	}
	if woken := pendingWakes(gate); woken != 0 {
		t.Fatalf("got %d wake-ups after the channel was drained, want 0", woken)
	}

	gate.Update(allow(true), ready(true))
	if woken := pendingWakes(gate); woken != 1 {
		t.Fatalf("got %d wake-ups after the channel was drained, want 1", woken)
	}
}

// TestResourceGateAvailableIsTheLevelTheManagerReads is the read side of the
// gate: it is what the manager consults when it is woken, so it has to track the
// inputs even when no wake-up is signalled.
func TestResourceGateAvailableIsTheLevelTheManagerReads(t *testing.T) {
	gate := newResourceGate()
	if gate.Available() {
		t.Fatal("a fresh gate must not be available")
	}

	// Unsatisfied updates must still move the level, so that a manager woken by
	// something else reads the truth.
	gate.Update(allow(true), ready(false))
	if gate.Available() {
		t.Fatal("the gate must not be available while the condition is missing")
	}

	gate.Update(allow(false), ready(true))
	if gate.Available() {
		t.Fatal("the gate must not be available while the permission is missing")
	}

	gate.Update(allow(true), ready(true))
	if !gate.Available() {
		t.Fatal("the gate has to be available after allowed && available")
	}
}
