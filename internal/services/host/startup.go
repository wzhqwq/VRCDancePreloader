package host

import (
	"context"
	"errors"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

func (m *Manager) Start() {
	m.operationMu.Lock()
	defer m.operationMu.Unlock()

	m.mu.Lock()
	m.ensureInitializedLocked()
	if m.state != managerCreated {
		m.mu.Unlock()
		return
	}
	m.mu.Unlock()

	if err := m.validateAndBuildGraph(); err != nil {
		m.mu.Lock()
		m.validationErr = err
		m.state = managerInvalid
		m.mu.Unlock()
		m.publishAllWrapperStatuses()
		return
	}

	m.mu.Lock()
	m.state = managerStarting
	for _, node := range m.order {
		node.intent = intentAuto
		node.phase = phaseStopped
		node.expected = expectedNone
		node.lastStatus = interactive.RunnerStatus{}
		node.lastError = nil
		node.startAttempted = false
		node.permanentFailure = false
	}
	m.listenerCtx, m.listenerCancel = context.WithCancel(context.Background())
	m.mu.Unlock()
	m.publishAllWrapperStatuses()
	m.startStatusInfrastructure()

	for _, layer := range m.startLayers {
		batch := make([]*serviceNode, 0, len(layer))
		m.mu.Lock()
		for _, node := range layer {
			if node.phase == phaseFailed {
				continue
			}
			if m.dependenciesRunningLocked(node) {
				batch = append(batch, node)
				continue
			}
			node.phase = phaseBlocked
			node.lastStatus = interactive.RunnerStatus{}
			node.lastError = nil
		}
		m.mu.Unlock()
		m.publishAllWrapperStatuses()
		m.runStartBatch(batch)
	}

	m.mu.Lock()
	m.state = managerRunning
	m.mu.Unlock()
	m.publishAllWrapperStatuses()
}

func (m *Manager) GracefulShutdown() {
	m.operationMu.Lock()

	m.mu.Lock()
	switch m.state {
	case managerCreated:
		m.state = managerStopped
		m.mu.Unlock()
		m.operationMu.Unlock()
		m.publishAllWrapperStatuses()
		return
	case managerInvalid, managerStopped:
		m.mu.Unlock()
		m.operationMu.Unlock()
		return
	case managerStopping:
		m.mu.Unlock()
		m.operationMu.Unlock()
		return
	}
	m.state = managerStopping
	m.mu.Unlock()
	m.publishAllWrapperStatuses()

	errs := make([]error, 0)
	for _, layer := range m.stopLayers {
		m.mu.RLock()
		batch := make([]*serviceNode, 0, len(layer))
		for _, node := range layer {
			if node.startAttempted || node.lastStatus.Running {
				batch = append(batch, node)
			}
		}
		m.mu.RUnlock()
		errs = append(errs, m.runShutdownBatch(batch)...)
	}

	m.mu.Lock()
	m.shutdownErr = errors.Join(errs...)
	m.state = managerStopped
	cancel := m.listenerCancel
	m.mu.Unlock()
	m.publishAllWrapperStatuses()
	if cancel != nil {
		cancel()
	}
	m.operationMu.Unlock()
	m.listenerWG.Wait()
}
