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

	em     *utils.EventManager[TesterStatus]
	mu     sync.Mutex
	status TesterStatus
}

func NewTester(testFn func() error) *Tester {
	return &Tester{
		testFn: testFn,

		em:     utils.NewEventManager[TesterStatus](),
		status: InitialTester(),
	}
}

func (t *Tester) Test() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.status.Running {
		t.updateStatus(Testing())
		go t.updateStatus(Tested(t.testFn()))
	}
}

func (t *Tester) updateStatus(status TesterStatus) {
	t.status = status
	t.em.NotifySubscribers(status)
}

func (t *Tester) Status() TesterStatus {
	return t.status
}

func (t *Tester) Reset() {
	t.updateStatus(InitialTester())
}

func (t *Tester) SubscribeStatus() *utils.EventSubscriber[TesterStatus] {
	return t.em.SubscribeEvent()
}
