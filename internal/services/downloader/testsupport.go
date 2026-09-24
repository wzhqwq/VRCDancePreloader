package downloader

import "github.com/samber/lo"

// Test support.
//
// The queue's decisions are only observable from inside this package: the
// manager, its order and the permits it publishes are unexported, and that is
// deliberate — nothing outside the downloader may reason about the queue.
// Packages that do have to assert on it (the preloader's request path is the one
// that matters, because that is where the order is decided) get exactly the two
// handles below and nothing more.
//
// These are not part of the service API and must not be used by production code.

// ManagersForTest replaces the service's managers with one manager under the
// given name.
//
// It exists so a test can have a queue without running ServiceStart, which would
// also start a retry ticker. The manager is fully functional: its queue, its
// permits and its retry table all work, and download loops are never started
// because nothing in these tests calls Download.
func (s *Service) ManagersForTest(name string) {
	s.managers = map[string]*downloadManager{
		name: newDownloadManager(1, nil),
	}
}

// InstallManagersForTest replaces the service's managers with one per name, so
// that a test which actually starts a download does not depend on which name
// Service.findManager routes its id to (pypy ids go to "pypy", everything else
// to "default").
func (s *Service) InstallManagersForTest(names ...string) {
	s.managers = make(map[string]*downloadManager, len(names))

	for _, name := range names {
		s.managers[name] = newDownloadManager(1, nil)
	}
}

// ManagerForTest returns the manager the service routes id to. Unlike
// ManagersForTest this keeps the real managers, and therefore the real
// scheduler, which a test that actually starts a download needs.
func (s *Service) ManagerForTest(id string) *downloadManager {
	return s.findManager(id)
}

// ManagerNamedForTest returns the manager installed under the given name, which
// is what a test that replaced the managers with ManagersForTest needs.
func (s *Service) ManagerNamedForTest(name string) *downloadManager {
	return s.managers[name]
}

// HoldsTaskForTest reports whether the manager still holds a download task for
// id, either in its task table or in its queue.
//
// Completing a download is not the same thing as leaving the manager: a task
// that ran and returned stays in dm.tasks and in dm.queue until something
// cancels it, and while it is there it keeps holding one of the maxParallel
// slots. This is the handle a test needs to tell those two apart from outside
// the package.
func (s *Service) HoldsTaskForTest(id string) bool {
	for _, dm := range s.managers {
		dm.Lock()

		_, held := dm.tasks[id]
		inQueue := lo.Contains(dm.queue, id)

		dm.Unlock()

		if held || inQueue {
			return true
		}
	}

	return false
}

// PermitForTest is the queue's decision about one task: where it sits and
// whether it may download.
type PermitForTest struct {
	ID string

	Position int
	Allowed  bool
}

// QueueOrderForTest returns the queue order across every manager, which is what
// a caller that prioritized ids needs to observe.
func (s *Service) QueueOrderForTest() []string {
	var order []string

	for _, dm := range s.managers {
		dm.Lock()
		order = append(order, dm.queue...)
		dm.Unlock()
	}

	return order
}

// PermitsForTest reads the current queue order together with the permit
// published for each task, in one critical section so that the two cannot come
// from different publications.
func (dm *downloadManager) PermitsForTest() []PermitForTest {
	dm.Lock()
	defer dm.Unlock()

	permits := make([]PermitForTest, 0, len(dm.queue))

	for i, id := range dm.queue {
		permit := PermitForTest{ID: id, Position: i}

		if t := dm.tasks[id]; t != nil {
			permit.Allowed = t.publishedPermit()
		}

		permits = append(permits, permit)
	}

	return permits
}

// publishedPermit reads the permit the queue last published for this task.
func (t *ManagedTask) publishedPermit() bool {
	traffic, ok := t.Traffic.(*traffic)
	if !ok {
		return false
	}

	return traffic.allowed.Load()
}
