package downloader

import (
	"testing"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/services/downloader/task"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

// T7 — retry scheduling belongs to the manager.
//
// It used to be ManagedTask.Retry(): a getter that scheduled a retry as a side
// effect of being read, spawned a goroutine per failure that nothing could
// cancel, and read a mutable package level delay. Now the delay lives on the
// manager, a failure only records an entry in its due table, and a single
// service ticker runs whatever is due.

// The production backoff must not change just because the mechanism did.
func TestRetryDelayKeepsTheOldThreeSeconds(t *testing.T) {
	dm := newDownloadManager(1, utils.NewBasicScheduler())

	if dm.retryDelay != 3*time.Second {
		t.Fatalf("retryDelay = %v, want 3s: this is the value ManagedTask.tempDelay had", dm.retryDelay)
	}
}

// A failed task is run again by the service ticker, with nothing else driving
// it, and the retry happens through the ordinary download loop.
func TestServiceTickerRetriesFailedTask(t *testing.T) {
	h := newGateHarness(t, 1)
	h.dm.retryDelay = 20 * time.Millisecond

	// The first attempt fails, the retry succeeds.
	remote := h.requireTask("r")
	remote.failures = 1

	h.start("r")
	h.waitCall("r", "the first attempt to reach the network")

	// The first attempt must have given up: that is the failure the song state
	// machine reacts to by scheduling a retry. There is no song in this package,
	// so the decision is made here, explicitly.
	waitUntil(t, func() bool { return h.task("r").Err() != nil }, "the first attempt to give up")

	at := h.task("r").ScheduleRetry()
	if d := time.Until(at); d <= 0 || d > time.Second {
		t.Fatalf("ScheduleRetry = %v from now, want the (shortened) retry delay", d)
	}

	// Nothing but the ticker runs it.
	svc := &Service{managers: map[string]*downloadManager{"default": h.dm}}
	stop := make(chan struct{})
	go svc.retryLoop(stop)
	defer close(stop)

	h.waitCall("r", "the scheduled retry to reach the network")

	// The retry is a normal attempt: it parks like any other until it is let go.
	h.rem["r"].unblock()

	waitUntil(t, func() bool { return h.task("r").State() == task.TaskCompleted },
		"the retried task to complete")
}

// A retry must not be a way around the parallel limit. The task is still the
// same queue entry it was, so the retry re-enters the ordinary gate.
func TestRetryGoesThroughTheQueueGate(t *testing.T) {
	h := newGateHarness(t, 1)
	h.dm.retryDelay = time.Millisecond

	// a holds the only slot.
	h.requireTask("a")
	h.start("a")
	h.waitCall("a", "the task that holds the only slot")

	// b is queued behind it and has a retry due. Nothing has started it yet, so
	// the retry is what will.
	h.requireTask("b")
	h.requireBlocked("b", "it is second in a queue with one slot")

	h.dm.ScheduleRetry("b")
	waitUntil(t, func() bool {
		h.dm.Lock()
		at, ok := h.dm.dueAt["b"]
		h.dm.Unlock()

		return ok && time.Now().After(at)
	}, "the retry to become due")

	h.dm.runDueRetries()

	// The retry was started, but it is in the gate: it must not issue a request
	// while a holds the only slot.
	h.requireNoNewCall("b", "the retried task while the only slot is taken")

	// Freeing the slot lets the retry through.
	h.rem["a"].unblock()
	h.waitCall("b", "the retried task after a slot freed up")
}

// A cancelled task must not come back when its scheduled retry falls due.
func TestScheduledRetryIsDroppedWhenTheTaskIsCancelled(t *testing.T) {
	h := newGateHarness(t, 1)
	h.dm.retryDelay = time.Millisecond

	h.requireTask("c")
	h.dm.ScheduleRetry("c")
	h.dm.CancelDownload("c")

	// The schedule has to be dropped outright, not merely made unreachable: a
	// task re-created under the same id would otherwise inherit it.
	h.dm.Lock()
	_, scheduled := h.dm.dueAt["c"]
	h.dm.Unlock()

	if scheduled {
		t.Fatal("CancelDownload left the scheduled retry behind")
	}

	// And once the delay has elapsed, nothing runs it either.
	time.Sleep(20 * time.Millisecond)
	h.dm.runDueRetries()

	if got := h.rem["c"].calls(); got != 0 {
		t.Fatalf("cancelled task issued %d request(s), want none", got)
	}
}

// waitUntil polls cond until it holds, or fails the test.
func waitUntil(t *testing.T, cond func() bool, what string) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for %s", what)
}
