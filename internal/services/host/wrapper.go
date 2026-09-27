package host

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

type wrappedService struct {
	manager *Manager
	node    *serviceNode
	status  *utils.Level[interactive.RunnerStatus]

	publishMu    sync.Mutex
	hasPublished bool
	last         statusFingerprint
}

type statusFingerprint struct {
	running   bool
	errorType string
	errorText string
}

func newWrappedService(manager *Manager, node *serviceNode) *wrappedService {
	return &wrappedService{
		manager: manager,
		node:    node,
		status:  utils.NewLevel(interactive.RunnerStatus{}),
	}
}

// WatchStatus hands the caller its own watcher of the wrapper's status level.
func (w *wrappedService) WatchStatus() *utils.LevelWatcher[interactive.RunnerStatus] {
	return w.status.Subscribe()
}

func (w *wrappedService) Status() interactive.RunnerStatus {
	w.manager.mu.RLock()
	defer w.manager.mu.RUnlock()
	return w.manager.wrappedStatusLocked(w.node)
}

func (w *wrappedService) Start() {
	w.manager.manualStart(w.node)
}

func (w *wrappedService) Stop() {
	w.manager.manualStop(w.node)
}

func (w *wrappedService) Restart() {
	w.manager.manualRestart(w.node)
}

func fingerprintStatus(status interactive.RunnerStatus) statusFingerprint {
	fingerprint := statusFingerprint{running: status.Running}
	if status.Error != nil {
		fingerprint.errorType = fmt.Sprintf("%v", reflect.TypeOf(status.Error))
		fingerprint.errorText = status.Error.Error()
	}
	return fingerprint
}

func (w *wrappedService) publish(status interactive.RunnerStatus) {
	fingerprint := fingerprintStatus(status)
	w.publishMu.Lock()
	if w.hasPublished && w.last == fingerprint {
		w.publishMu.Unlock()
		return
	}
	w.hasPublished = true
	w.last = fingerprint
	w.publishMu.Unlock()
	w.status.Store(status)
}
