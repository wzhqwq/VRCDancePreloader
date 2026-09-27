package interactive

import (
	"sync"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type TesterStatus struct {
	Running bool
	Stale   bool
	Error   error
}

func InitialTester() TesterStatus {
	return TesterStatus{
		Stale: true,
	}
}

func Testing() TesterStatus {
	return TesterStatus{
		Running: true,
	}
}

func Tested(err error) TesterStatus {
	return TesterStatus{
		Error: err,
	}
}

type Tester struct {
	testFn func() error

	// status is a level, not an emitter: it is written by Test/Reset (one of them
	// from a goroutine of its own) and read by Status/WatchStatus consumers, so
	// the atomics inside Level replace the unlocked field this used to be.
	status *utils.Level[TesterStatus]
	mu     sync.Mutex
}

func NewTester(testFn func() error) *Tester {
	return &Tester{
		testFn: testFn,

		status: utils.NewLevel(InitialTester()),
	}
}

func (t *Tester) Test() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.status.Current().Running {
		t.updateStatus(Testing())
		go t.updateStatus(Tested(t.testFn()))
	}
}

func (t *Tester) updateStatus(status TesterStatus) {
	t.status.Store(status)
}

func (t *Tester) Status() TesterStatus {
	return t.status.Current()
}

func (t *Tester) Reset() {
	t.updateStatus(InitialTester())
}

// WatchStatus hands the caller its own watcher of the status level: wait on
// Wakes, then read Status (or the watcher's Current) for the value.
func (t *Tester) WatchStatus() *utils.LevelWatcher[TesterStatus] {
	return t.status.Subscribe()
}
