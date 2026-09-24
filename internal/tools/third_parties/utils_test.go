package third_parties

import (
	"testing"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

// allow / ready keep the steps below readable while still typing the two inputs
// differently: swapping them is a compile error (see ResourceGate).
func allow(v bool) ResourceAllowed   { return ResourceAllowed(v) }
func ready(v bool) ResourceAvailable { return ResourceAvailable(v) }

// drain takes everything currently buffered on the subscriber.
func drain(ch *utils.EventSubscriber[bool]) []bool {
	var got []bool

drain:
	for {
		select {
		case v := <-ch.Channel:
			got = append(got, v)
		default:
			break drain
		}
	}

	return got
}

// gateStep is one Update/Retry call plus the values the subscriber is expected
// to have received because of it.
type gateStep struct {
	name  string
	apply func(*ResourceGate)
	want  []bool
}

func assertSteps(t *testing.T, steps []gateStep) {
	t.Helper()

	gate := newResourceGate()
	ch := gate.Subscribe()
	defer ch.Close()

	for i, step := range steps {
		step.apply(gate)

		got := drain(ch)
		if len(got) != len(step.want) {
			t.Fatalf("step %d (%s): got %v, want %v", i, step.name, got, step.want)
		}
		for j := range got {
			if got[j] != step.want[j] {
				t.Fatalf("step %d (%s): got %v, want %v", i, step.name, got, step.want)
			}
		}
	}
}

// TestResourceGatePublishesWhenItBecomesSatisfied covers the two ways a gate can
// turn usable, including the one that 0eed86d fixed: the permission was granted
// long before the dance room catalog finished loading.
func TestResourceGatePublishesWhenItBecomesSatisfied(t *testing.T) {
	assertSteps(t, []gateStep{
		{"permission only", func(g *ResourceGate) { g.Update(allow(true), ready(false)) }, nil},
		{"the catalog data arrives", func(g *ResourceGate) { g.Update(allow(true), ready(true)) }, []bool{true}},
	})

	assertSteps(t, []gateStep{
		{"data only", func(g *ResourceGate) { g.Update(allow(false), ready(true)) }, nil},
		{"the user enables the resource", func(g *ResourceGate) { g.Update(allow(true), ready(true)) }, []bool{true}},
	})
}

// TestResourceGateStaysSilentWhileUnsatisfied pins the fact that the downstream
// is never told anything while the gate is unusable.
func TestResourceGateStaysSilentWhileUnsatisfied(t *testing.T) {
	assertSteps(t, []gateStep{
		{"neither", func(g *ResourceGate) { g.Update(allow(false), ready(false)) }, nil},
		{"data only", func(g *ResourceGate) { g.Update(allow(false), ready(true)) }, nil},
		{"permission revoked", func(g *ResourceGate) { g.Update(allow(false), ready(false)) }, nil},
		{"data gone", func(g *ResourceGate) { g.Update(allow(true), ready(false)) }, nil},
		{"retry while unsatisfied", func(g *ResourceGate) { g.Retry() }, nil},
	})
}

// TestResourceGateRepublishesWhileSatisfied pins level semantics on purpose: a
// dropped event (utils/event.go) must not be able to leave the downstream
// permanently stuck, so every update of a satisfied gate re-publishes `true`
// (review/09 §2.5, review/02 A7).
func TestResourceGateRepublishesWhileSatisfied(t *testing.T) {
	assertSteps(t, []gateStep{
		{"satisfied", func(g *ResourceGate) { g.Update(allow(true), ready(true)) }, []bool{true}},
		{"another update", func(g *ResourceGate) { g.Update(allow(true), ready(true)) }, []bool{true}},
		{"retry", func(g *ResourceGate) { g.Retry() }, []bool{true}},
		{"permission lost", func(g *ResourceGate) { g.Update(allow(false), ready(true)) }, nil},
		{"retry while unsatisfied", func(g *ResourceGate) { g.Retry() }, nil},
		{"permission back", func(g *ResourceGate) { g.Update(allow(true), ready(true)) }, []bool{true}},
	})
}

func TestResourceGateBothSatisfied(t *testing.T) {
	gate := newResourceGate()
	if gate.BothSatisfied() {
		t.Fatal("a fresh gate must not be satisfied")
	}

	gate.Update(allow(true), ready(true))
	if !gate.BothSatisfied() {
		t.Fatal("the gate has to be satisfied after allowed && available")
	}

	gate.Update(allow(true), ready(false))
	if gate.BothSatisfied() {
		t.Fatal("the gate must not stay satisfied after the condition went away")
	}
}
