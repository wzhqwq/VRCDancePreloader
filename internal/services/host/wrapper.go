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
	events  *utils.EventManager[interactive.RunnerStatus]

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
		events:  utils.NewEventManager[interactive.RunnerStatus](),
	}
}

func (w *wrappedService) SubscribeStatus() *utils.EventSubscriber[interactive.RunnerStatus] {
	return w.events.SubscribeEvent()
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
	w.events.NotifySubscribers(status)
}
