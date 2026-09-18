package downloader

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/services/downloader/task"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

// T3 / T4 / T5 / T11 — the download queue gate.
//
// The gate is the "may this task download right now" decision. It is level
// triggered: the manager publishes position + allowed as state, and every gate
// entry re-reads it. These tests pin that property, including the case the old
// edge triggered design could not express: a task that re-enters the gate after
// an attempt restart must be blocked when it is outside the allowed prefix.

// gateWait bounds the negative assertions ("this task must not issue a request")
// below. It must comfortably exceed the 100ms body request pacing, because a
// bypass would show up right after an attempt restart.
const gateWait = 400 * time.Millisecond

// errGateFailed stands in for the CDN errors that end a download loop for good
// (the loop only restarts itself for EOF, fragments, restarts and client
// changes; anything else stops the task, which is when a retry becomes the
// manager's business).
var errGateFailed = errors.New("gate: simulated cdn failure")

// gateRemote parks inside GetDownloadStream until it is released or the attempt
// scope is aborted. A parked task stays "downloading", so it keeps whatever
// slot it holds and the test can observe who is allowed to run.
type gateRemote struct {
	resolveCalls atomic.Int64
	streamCalls  atomic.Int64

	entered chan struct{}

	// failures is how many leading GetDownloadStream calls fail with
	// errGateFailed instead of parking. It is read by the download goroutine, so
	// it has to be set before the task is started.
	failures int64

	release     chan struct{}
	releaseOnce sync.Once
}

func newGateRemote() *gateRemote {
	return &gateRemote{
		entered: make(chan struct{}, 32),
		release: make(chan struct{}),
	}
}

func (g *gateRemote) WaitResolving(context.Context, func(interactive.RemoteStatus)) (int64, error) {
	g.resolveCalls.Add(1)
	return 1024, nil
}

func (g *gateRemote) GetDownloadStream(_ int64, ctx context.Context) (task.StreamInfo, error) {
	n := g.streamCalls.Add(1)

	select {
	case g.entered <- struct{}{}:
	default:
	}

	if n <= g.failures {
		return task.StreamInfo{}, errGateFailed
	}

	select {
	case <-ctx.Done():
		return task.StreamInfo{}, context.Cause(ctx)
	case <-g.release:
		// An empty body ends the attempt cleanly, so the loop marks the task as
		// completed and the manager hands the slot to the next task.
		return task.StreamInfo{
			Rc:             io.NopCloser(strings.NewReader("")),
			Length:         0,
			RangeAvailable: true,
		}, nil
	}
}

func (g *gateRemote) calls() int64 { return g.streamCalls.Load() }

func (g *gateRemote) unblock() {
	g.releaseOnce.Do(func() { close(g.release) })
}

type gateHarness struct {
	t    *testing.T
	dm   *downloadManager
	dir  string
	rem  map[string]*gateRemote
	file map[string]*os.File
	done map[string]<-chan struct{}
}

func newGateHarness(t *testing.T, maxParallel int) *gateHarness {
	t.Helper()

	h := &gateHarness{
		t:    t,
		dm:   newDownloadManager(maxParallel, utils.NewBasicScheduler()),
		dir:  t.TempDir(),
		rem:  map[string]*gateRemote{},
		file: map[string]*os.File{},
		done: map[string]<-chan struct{}{},
	}

	// The temp dir is allocated above on purpose: t.Cleanup runs in LIFO order,
	// so taking it before this registration makes the file release below run
	// first. On Windows a still open handle makes testing.TempDir's RemoveAll
	// fail the test. Downloads are parked in gateRemote.GetDownloadStream and
	// never return on their own, so release them before waiting.
	t.Cleanup(func() {
		for _, remote := range h.rem {
			remote.unblock()
		}
		for _, done := range h.done {
			select {
			case <-done:
			case <-time.After(5 * time.Second):
			}
		}
		for _, f := range h.file {
			_ = f.Close()
		}
	})

	return h
}

// requireTask registers a task through the same entry point the downloader
// service uses, with a real (file backed) LocalProvider.
func (h *gateHarness) requireTask(id string) *gateRemote {
	h.t.Helper()

	remote := h.rem[id]
	if remote == nil {
		remote = newGateRemote()
		h.rem[id] = remote

		f, err := os.CreateTemp(h.dir, "vrcdp-"+id)
		if err != nil {
			h.t.Fatalf("create temp file: %v", err)
		}
		h.file[id] = f
	}

	got := h.dm.CreateOrGetPausedTask(id,
		func() task.RemoteProvider { return remote },
		func(utils.LoggerImpl) task.LocalProvider { return task.NewLocalFileProvider(h.file[id]) },
	)
	if got == nil {
		h.t.Fatalf("task %q was not created", id)
	}

	return remote
}

// start runs the download loop the way Service.Download does, and returns a
// channel that is closed once the loop returned.
func (h *gateHarness) start(id string) <-chan struct{} {
	h.t.Helper()

	h.dm.Lock()
	mt := h.dm.tasks[id]
	h.dm.Unlock()

	if mt == nil {
		h.t.Fatalf("task %q is not registered", id)
	}

	done := make(chan struct{})
	h.done[id] = done

	go func() {
		defer close(done)
		mt.Download()
		h.dm.UpdatePriorities()
	}()

	return done
}

func (h *gateHarness) task(id string) *ManagedTask {
	h.t.Helper()

	h.dm.Lock()
	defer h.dm.Unlock()

	mt := h.dm.tasks[id]
	if mt == nil {
		h.t.Fatalf("task %q is not registered", id)
	}
	return mt
}

// permit reads the published permit. It is the queue's own view of the task, so
// asserting on it makes a failure easy to diagnose; the behavioural assertions
// below prove the task actually acts on it.
func (h *gateHarness) permit(id string) (position int, allowed bool) {
	h.t.Helper()

	h.dm.Lock()
	mt := h.dm.tasks[id]
	h.dm.Unlock()

	if mt == nil {
		return -1, false
	}

	p, ok := mt.Traffic.(*traffic)
	if !ok {
		h.t.Fatalf("task %q does not use the queue controlled permit", id)
	}

	return int(p.position.Load()), p.allowed.Load()
}

func (h *gateHarness) requireAllowed(id string) {
	h.t.Helper()

	if _, allowed := h.permit(id); !allowed {
		h.t.Fatalf("task %q should be allowed to download", id)
	}
}

func (h *gateHarness) requireBlocked(id string, why string) {
	h.t.Helper()

	if pos, allowed := h.permit(id); allowed {
		h.t.Fatalf("task %q must not be allowed to download (%s), position=%d", id, why, pos)
	}
}

func (h *gateHarness) waitCall(id string, what string) {
	h.t.Helper()

	select {
	case <-h.rem[id].entered:
	case <-time.After(10 * time.Second):
		h.t.Fatalf("timed out waiting for %s", what)
	}
}

// requireNoNewCall waits out the pacing window and asserts that the task did
// not issue another request. The counter is authoritative, so a stale token
// left in the channel cannot make this flaky.
func (h *gateHarness) requireNoNewCall(id string, what string) {
	h.t.Helper()

	before := h.rem[id].calls()
	time.Sleep(gateWait)

	if after := h.rem[id].calls(); after != before {
		h.t.Fatalf("%s issued %d extra request(s), want none", what, after-before)
	}
}

func (h *gateHarness) waitDone(id string, what string) {
	h.t.Helper()

	select {
	case <-h.done[id]:
	case <-time.After(10 * time.Second):
		h.t.Fatalf("timed out waiting for %s", what)
	}
}

// T3 — the queue order decides who downloads, and the slot moves on when a task
// finishes.
func TestQueueGateSerializesDownloads(t *testing.T) {
	h := newGateHarness(t, 1)

	h.requireTask("a")
	h.requireTask("b")
	h.requireTask("c")

	h.requireAllowed("a")
	h.requireBlocked("b", "it is second in a queue with one slot")
	h.requireBlocked("c", "it is third in a queue with one slot")

	h.start("a")
	h.start("b")
	h.start("c")

	h.waitCall("a", "the first task to start downloading")
	h.requireNoNewCall("b", "the second task while the slot is taken")
	h.requireNoNewCall("c", "the third task while the slot is taken")

	// Let the first task finish: the slot must move to the second one only.
	h.rem["a"].unblock()
	h.waitCall("b", "the second task after the first one completed")
	h.requireNoNewCall("c", "the third task while the second one holds the slot")

	h.rem["b"].unblock()
	h.waitCall("c", "the third task after the second one completed")
}

// T4 — shrinking the parallel limit republishes the permits immediately, without
// aborting what is already running, and growing it back lets the task through.
func TestQueueGateReactsToMaxParallelChange(t *testing.T) {
	h := newGateHarness(t, 2)

	for _, id := range []string{"a", "b", "c"} {
		h.requireTask(id)
	}

	h.start("a")
	h.start("b")
	h.start("c")

	h.waitCall("a", "the first task")
	h.waitCall("b", "the second task")
	h.requireNoNewCall("c", "the third task with two slots")

	h.dm.SetMaxParallel(1)

	h.requireAllowed("a")
	h.requireBlocked("b", "the limit shrank below its position")
	h.requireBlocked("c", "the limit shrank below its position")

	// A running task is not torn down by a limit change...
	if got := h.rem["a"].calls(); got != 1 {
		t.Fatalf("the first task issued %d requests, want 1: a limit change must not restart it", got)
	}

	// ...but once it re-enters the gate it has to respect the new limit.
	h.task("b").CloseConnection()
	h.requireNoNewCall("b", "the task that fell out of the allowed prefix")

	// Growing the limit again lets it through.
	h.dm.SetMaxParallel(2)

	h.requireAllowed("b")
	h.waitCall("b", "the task after the limit grew back")
}

// T5 — cancelling a task that is waiting in the gate returns at once and issues
// no request.
func TestQueueGateCancelWhilePending(t *testing.T) {
	h := newGateHarness(t, 1)

	h.requireTask("a")
	h.requireTask("b")

	h.start("a")
	h.start("b")

	h.waitCall("a", "the first task")
	h.requireNoNewCall("b", "the pending task")

	h.dm.CancelDownload("b")
	h.waitDone("b", "the pending task to return after being cancelled")

	if got := h.rem["b"].calls(); got != 0 {
		t.Fatalf("a cancelled pending task issued %d request(s), want none", got)
	}
}

// T11 — the gate is level triggered. A task that was allowed, then pushed out of
// the allowed prefix, must be blocked again when its attempt restarts: the gate
// re-reads the current permit instead of inferring it from "no message arrived".
func TestQueueGateIsLevelTriggeredAcrossRestarts(t *testing.T) {
	h := newGateHarness(t, 2)

	for _, id := range []string{"a", "b", "c"} {
		h.requireTask(id)
	}

	h.start("a")
	h.start("b")
	h.start("c")

	h.waitCall("a", "the first task")
	h.waitCall("b", "the second task")
	h.requireNoNewCall("c", "the third task with two slots")

	// Reordering pushes b out of the allowed prefix: the queue becomes [c, a, b].
	h.dm.Prioritize("c")

	h.requireAllowed("c")
	h.requireAllowed("a")
	h.requireBlocked("b", "it was pushed out of the allowed prefix")

	// b re-enters the gate after a connection timeout. With the old design the
	// gate could only ask "is a position message pending?", which says nothing
	// about the current queue state, so a re-entering task could slip through.
	h.task("b").CloseConnection()
	h.requireNoNewCall("b", "the task that re-entered the gate while it is not allowed")

	// Freeing a slot lets it through again.
	h.rem["a"].unblock()
	h.waitCall("b", "the task after a slot freed up")
}

// T12 — assembling a multi step queue change must not publish permits from the
// half built order.
//
// This is the regression guard for Service.WithFrozenQueue, and it mirrors what
// the preloader does: it creates a task per upcoming song (each creation appends
// to the queue) and only afterwards moves the current song to the front. The
// queue is therefore, for a moment, in an order nobody intended — and any queue
// change landing in that window (a task finishing, a video request, a cancel)
// re-publishes permits over it.
//
// Without the freeze, b is granted a slot here because it happens to sit at
// index 1 while it is still queued by creation order; it starts downloading and
// will not re-read its permit until its attempt restarts, so the wrong grant
// outlives the assembly that caused it.
func TestQueueGatePublishesOnlyTheFinalOrder(t *testing.T) {
	h := newGateHarness(t, 2)

	// a already holds one of the two slots.
	h.requireTask("a")
	h.start("a")
	h.waitCall("a", "the task that is already downloading")

	// The assembly begins: freeze, then create the tasks for the new songs.
	h.dm.freeze()

	h.requireTask("b")
	h.requireTask("c")
	h.start("b")
	h.start("c")

	// Creating a task must not have granted anything: b and c are only where
	// they were appended, not where they belong.
	h.requireBlocked("b", "its position is not decided yet")
	h.requireBlocked("c", "its position is not decided yet")

	// An unrelated queue change during the assembly (this is what a finishing
	// task or a video request does) must not publish the interim order either.
	h.dm.UpdatePriorities()

	h.requireBlocked("b", "an unrelated queue change published the interim order")
	h.requireBlocked("c", "an unrelated queue change published the interim order")

	// The intended order puts c first, so b has to be pushed back out of the
	// allowed prefix — and it must never have been let through meanwhile.
	h.dm.Prioritize("c")
	h.dm.thaw()

	h.requireAllowed("c")
	h.requireAllowed("a")
	h.requireBlocked("b", "the final order pushed it out of the allowed prefix")
	h.requireNoNewCall("b", "the task that was only in the interim allowed prefix")
}

// WithFrozenQueue must cover both managers, otherwise the songs handled by the
// other one keep being published mid assembly.
func TestServiceWithFrozenQueueFreezesBothManagers(t *testing.T) {
	svc := &Service{
		managers: map[string]*downloadManager{
			"pypy":    newDownloadManager(1, utils.NewBasicScheduler()),
			"default": newDownloadManager(1, utils.NewBasicScheduler()),
		},
	}

	frozen := func(dm *downloadManager) bool {
		dm.Lock()
		defer dm.Unlock()

		return dm.batch > 0
	}

	for name, dm := range svc.managers {
		if frozen(dm) {
			t.Fatalf("%s manager is frozen before WithFrozenQueue", name)
		}
	}

	svc.WithFrozenQueue(func() {
		for name, dm := range svc.managers {
			if !frozen(dm) {
				t.Fatalf("%s manager is not frozen inside WithFrozenQueue", name)
			}
		}
	})

	for name, dm := range svc.managers {
		if frozen(dm) {
			t.Fatalf("%s manager is still frozen after WithFrozenQueue", name)
		}
	}
}
