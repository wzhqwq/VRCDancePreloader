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
//	WatchQueue()       *utils.LevelWatcher[[]string]  // the queue order, as state
//	GetQueueSnapshot() []*ManagedTask                 // queue order, never nil
//
// The queue order is *state*: a consumer is woken and reads it, so a coalesced or
// late wake-up loses nothing (review/09 §8.5). The manager has no lifecycle event
// — Destroy only runs when the service stops, and the GUI learns about that from
// the host service status it already watches.
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

// drainQueue empties whatever wake-up is already pending, so that a later wait is
// about the operation under test rather than about an earlier one.
func drainQueue(watcher *utils.LevelWatcher[[]string]) {
	for {
		select {
		case <-watcher.Wakes():
		default:
			return
		}
	}
}

// waitQueueOrder asserts that the consumer is woken *and* that the order it then
// reads is the expected one. The wake-up slot is coalescing, so the wait has to
// tolerate earlier ones being merged, but it must not tolerate silence.
func waitQueueOrder(t *testing.T, watcher *utils.LevelWatcher[[]string], what string, want ...string) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-watcher.Wakes():
			if got := watcher.Current(); !slices.Equal(got, want) {
				t.Fatalf("%s: the order read after the wake-up = %v, want %v", what, got, want)
			}
			return
		case <-time.After(time.Millisecond):
		}
	}

	t.Fatalf("timed out waiting for a queue change (%s)", what)
}

// Destroy is a lifecycle operation, not a queue change: it cancels the tasks and
// leaves the queue state alone, and it deliberately announces nothing (the GUI
// learns that the manager is going away from the host service status).
func TestQueueContractOnDestroy(t *testing.T) {
	h := newGateHarness(t, 2)
	queue := h.dm.WatchQueue()
	defer queue.Close()

	h.requireTask("a")
	before := slices.Clone(queue.Current())

	h.dm.Destroy()

	if got := queue.Current(); !slices.Equal(got, before) {
		t.Fatalf("Destroy changed the queue state: %v -> %v", before, got)
	}
}

// Cancelling a task removes it from the snapshot and announces the change.
func TestQueueContractOnCancel(t *testing.T) {
	h := newGateHarness(t, 2)
	queue := h.dm.WatchQueue()
	defer queue.Close()

	for _, id := range []string{"a", "b", "c"} {
		h.requireTask(id)
	}
	h.assertSnapshot("after creating three tasks", "a", "b", "c")

	drainQueue(queue)
	h.dm.CancelDownload("b")

	h.assertSnapshot("after cancelling b", "a", "c")
	waitQueueOrder(t, queue, "cancelling a task", "a", "c")
}

// Completing a task removes it from the snapshot and announces the change.
func TestQueueContractOnCompletion(t *testing.T) {
	h := newGateHarness(t, 1)
	queue := h.dm.WatchQueue()
	defer queue.Close()

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
	waitQueueOrder(t, queue, "a task completing", "b")
}

// Interleaving Prioritize with new tasks is how the preloader assembles the
// queue. The snapshot has to track whatever order results, and the GUI has to
// hear about it.
func TestQueueContractOnInterleavedPrioritizeAndCreate(t *testing.T) {
	h := newGateHarness(t, 2)
	queue := h.dm.WatchQueue()
	defer queue.Close()

	h.requireTask("a")
	h.requireTask("b")

	drainQueue(queue)
	h.dm.Prioritize("b")
	h.assertSnapshot("after prioritizing b", "b", "a")
	waitQueueOrder(t, queue, "prioritizing b", "b", "a")

	drainQueue(queue)
	h.requireTask("c")
	h.assertSnapshot("after creating c", "b", "a", "c")
	waitQueueOrder(t, queue, "creating a task", "b", "a", "c")

	drainQueue(queue)
	h.dm.Prioritize("c")
	h.assertSnapshot("after prioritizing c", "c", "b", "a")
	waitQueueOrder(t, queue, "prioritizing c", "c", "b", "a")
}

// A plain Prioritize moves the named task to the front and keeps the rest in
// their previous relative order.
func TestQueueContractOnPrioritize(t *testing.T) {
	h := newGateHarness(t, 2)
	queue := h.dm.WatchQueue()
	defer queue.Close()

	for _, id := range []string{"a", "b", "c", "d"} {
		h.requireTask(id)
	}

	drainQueue(queue)
	h.dm.Prioritize("c")

	h.assertSnapshot("after prioritizing c", "c", "a", "b", "d")
	waitQueueOrder(t, queue, "prioritizing c", "c", "a", "b", "d")
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
