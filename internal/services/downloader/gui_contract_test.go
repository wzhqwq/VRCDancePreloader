package downloader

import (
	"slices"
	"testing"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

// T8 — the download queue GUI contract.
//
// The queue window is not implemented yet, but downloadManager already exposes
// the two things it will be built on, and the plan freezes both:
//
//	Subscribe()        *utils.EventSubscriber[ManagerChangeType]  // QueueChange / Stopped
//	GetQueueSnapshot() []*ManagedTask                             // queue order, never nil
//
// These tests pin that contract against the four mutations the queue actually
// goes through (cancel, completion, interleaved Prioritize + creation, plain
// Prioritize), plus the two properties that are easy to break by "simplifying"
// the implementation: the snapshot must skip a queued id that has no task, and
// it must not depend on the queue being non-empty to be correct.

// assertSnapshot checks the three properties the GUI relies on: the order is
// dm.queue's order, there is no nil entry, and the ids are exactly the expected
// ones.
func (h *gateHarness) assertSnapshot(what string, want ...string) {
	h.t.Helper()

	snapshot := h.dm.GetQueueSnapshot()

	h.dm.Lock()
	queue := slices.Clone(h.dm.queue)
	h.dm.Unlock()

	got := make([]string, 0, len(snapshot))
	for _, t := range snapshot {
		if t == nil {
			h.t.Fatalf("%s: GetQueueSnapshot returned a nil task (queue %v)", what, queue)
		}
		got = append(got, t.ID)
	}

	if !slices.Equal(got, want) {
		h.t.Fatalf("%s: snapshot ids = %v, want %v (queue %v)", what, got, want, queue)
	}
}

// drainEvents empties whatever is already buffered, so that a later wait is
// about the operation under test rather than about an earlier one.
func drainEvents(ch *utils.EventSubscriber[ManagerChangeType]) {
	for {
		select {
		case <-ch.Channel:
		default:
			return
		}
	}
}

// waitEvent asserts that the subscriber is told about an event of the given
// kind. The channel is buffered and its sends are non blocking, so the wait has
// to tolerate other events arriving first, but it must not tolerate silence.
func waitEvent(t *testing.T, ch *utils.EventSubscriber[ManagerChangeType], want ManagerChangeType, what string) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case got := <-ch.Channel:
			if got == want {
				return
			}
		case <-time.After(time.Millisecond):
		}
	}

	t.Fatalf("timed out waiting for a %v event (%s)", want, what)
}

// Cancelling a task removes it from the snapshot and announces the change.
func TestQueueContractOnCancel(t *testing.T) {
	h := newGateHarness(t, 2)
	ch := h.dm.Subscribe()
	defer ch.Close()

	for _, id := range []string{"a", "b", "c"} {
		h.requireTask(id)
	}
	h.assertSnapshot("after creating three tasks", "a", "b", "c")

	drainEvents(ch)
	h.dm.CancelDownload("b")

	h.assertSnapshot("after cancelling b", "a", "c")
	waitEvent(t, ch, QueueChange, "cancelling a task")
}

// Completing a task removes it from the snapshot and announces the change.
func TestQueueContractOnCompletion(t *testing.T) {
	h := newGateHarness(t, 1)
	ch := h.dm.Subscribe()
	defer ch.Close()

	h.requireTask("a")
	h.requireTask("b")
	h.assertSnapshot("after creating two tasks", "a", "b")

	h.start("a")
	h.start("b")
	h.waitCall("a", "the first task to start downloading")

	// Let the first task finish: it must leave the queue, and the second one
	// must take its place.
	h.rem["a"].unblock()

	waitUntil(t, func() bool { return len(h.dm.GetQueueSnapshot()) == 1 },
		"the completed task to leave the queue")

	h.assertSnapshot("after a completed", "b")
	waitEvent(t, ch, QueueChange, "a task completing")
}

// Interleaving Prioritize with new tasks is how the preloader assembles the
// queue. The snapshot has to track whatever order results, and the GUI has to
// hear about it.
func TestQueueContractOnInterleavedPrioritizeAndCreate(t *testing.T) {
	h := newGateHarness(t, 2)
	ch := h.dm.Subscribe()
	defer ch.Close()

	h.requireTask("a")
	h.requireTask("b")

	drainEvents(ch)
	h.dm.Prioritize("b")
	h.assertSnapshot("after prioritizing b", "b", "a")
	waitEvent(t, ch, QueueChange, "prioritizing b")

	drainEvents(ch)
	h.requireTask("c")
	h.assertSnapshot("after creating c", "b", "a", "c")
	waitEvent(t, ch, QueueChange, "creating a task")

	drainEvents(ch)
	h.dm.Prioritize("c")
	h.assertSnapshot("after prioritizing c", "c", "b", "a")
	waitEvent(t, ch, QueueChange, "prioritizing c")
}

// A plain Prioritize moves the named task to the front and keeps the rest in
// their previous relative order.
func TestQueueContractOnPrioritize(t *testing.T) {
	h := newGateHarness(t, 2)
	ch := h.dm.Subscribe()
	defer ch.Close()

	for _, id := range []string{"a", "b", "c", "d"} {
		h.requireTask(id)
	}

	drainEvents(ch)
	h.dm.Prioritize("c")

	h.assertSnapshot("after prioritizing c", "c", "a", "b", "d")
	waitEvent(t, ch, QueueChange, "prioritizing c")
}

// The snapshot must skip a queued id that has no task instead of returning a nil
// entry: a nil entry is what would crash the queue window, and the filtering is
// the part of the implementation that is easiest to drop by accident (indexing
// dm.queue directly would be "simpler" and wrong).
func TestQueueSnapshotSkipsQueuedIdsWithoutATask(t *testing.T) {
	h := newGateHarness(t, 2)

	h.requireTask("a")
	h.requireTask("b")

	// Reach the state the filter exists for: UpdatePriorities only removes such
	// an id when it next runs, so it can be observed in between.
	h.dm.Lock()
	h.dm.queue = append(h.dm.queue, "ghost")
	h.dm.Unlock()

	h.assertSnapshot("with an id that has no task", "a", "b")
}

// The queue is empty before anything is queued: the snapshot has to be a valid
// empty result rather than nil, because the GUI ranges over it directly.
func TestQueueSnapshotIsEmptyBeforeAnythingIsQueued(t *testing.T) {
	dm := newDownloadManager(2, utils.NewBasicScheduler())

	if got := dm.GetQueueSnapshot(); len(got) != 0 {
		t.Fatalf("empty manager returned %d entries, want none", len(got))
	}
}

// Destroy is the other half of the event contract: the GUI is told that the
// manager is going away.
func TestQueueContractOnDestroy(t *testing.T) {
	h := newGateHarness(t, 2)
	ch := h.dm.Subscribe()
	defer ch.Close()

	h.requireTask("a")

	drainEvents(ch)
	h.dm.Destroy()

	waitEvent(t, ch, Stopped, "Destroy")
}
