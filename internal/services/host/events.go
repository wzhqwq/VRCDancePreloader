package host

import (
	"fmt"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

func (m *Manager) startStatusInfrastructure() {
	m.mu.RLock()
	ctx := m.listenerCtx
	nodes := append([]*serviceNode(nil), m.order...)
	m.mu.RUnlock()

	m.listenerWG.Add(1)
	go m.reconcileLoop(ctx)

	for _, node := range nodes {
		if !node.isStateful() {
			continue
		}
		subscriber, err := safeSubscribe(node)
		if err != nil || subscriber == nil {
			m.mu.Lock()
			node.phase = phaseFailed
			node.lastError = err
			if node.lastError == nil {
				node.lastError = fmt.Errorf("service %q returned a nil status subscriber", node.name)
			}
			m.mu.Unlock()
			continue
		}
		m.mu.Lock()
		node.subscriber = subscriber
		m.mu.Unlock()
		m.listenerWG.Add(1)
		go m.listenServiceStatus(ctx, node, subscriber)
	}
	m.publishAllWrapperStatuses()
}

func safeSubscribe(node *serviceNode) (subscriber *utils.EventSubscriber[interactive.RunnerStatus], err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = panicError("SubscribeStatus service "+node.name, recovered)
		}
	}()
	return node.stateful.SubscribeStatus(), nil
}

func (m *Manager) listenServiceStatus(
	ctx interface{ Done() <-chan struct{} },
	node *serviceNode,
	subscriber *utils.EventSubscriber[interactive.RunnerStatus],
) {
	defer m.listenerWG.Done()
	defer subscriber.Close()
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-subscriber.Channel:
			if !ok {
				return
			}
			m.requestReconcile(node.name)
		}
	}
}

func (m *Manager) requestReconcile(name string) {
	m.pendingMu.Lock()
	m.pendingRoots[name] = struct{}{}
	m.pendingMu.Unlock()
	select {
	case m.reconcileWake <- struct{}{}:
	default:
	}
}

func (m *Manager) takePendingRoots() []string {
	m.pendingMu.Lock()
	defer m.pendingMu.Unlock()
	if len(m.pendingRoots) == 0 {
		return nil
	}
	roots := make([]string, 0, len(m.pendingRoots))
	for name := range m.pendingRoots {
		roots = append(roots, name)
	}
	clear(m.pendingRoots)
	return roots
}

func (m *Manager) reconcileLoop(ctx interface{ Done() <-chan struct{} }) {
	defer m.listenerWG.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.reconcileWake:
			for {
				roots := m.takePendingRoots()
				if len(roots) == 0 {
					break
				}
				m.reconcileRoots(roots)
			}
		}
	}
}

func (m *Manager) reconcileRoots(names []string) {
	m.operationMu.Lock()
	defer m.operationMu.Unlock()

	m.mu.RLock()
	if m.state != managerRunning {
		m.mu.RUnlock()
		return
	}
	roots := make([]*serviceNode, 0, len(names))
	for _, name := range names {
		if node := m.nodes[name]; node != nil && node.isStateful() {
			roots = append(roots, node)
		}
	}
	m.mu.RUnlock()
	if len(roots) == 0 {
		return
	}

	type observed struct {
		node   *serviceNode
		status interactive.RunnerStatus
	}
	observations := make([]observed, 0, len(roots))
	for _, node := range roots {
		observations = append(observations, observed{node: node, status: safeStatus(node)})
	}

	failureRoots := make([]*serviceNode, 0)
	recoveryRoots := make([]*serviceNode, 0)
	forceStopRoots := make([]*serviceNode, 0)
	m.mu.Lock()
	for _, observation := range observations {
		node := observation.node
		status := observation.status
		previousPhase := node.phase
		previousRunning := node.lastStatus.Running
		node.lastStatus = status

		if status.Running {
			node.phase = phaseRunning
			node.lastError = status.Error
			if node.intent == intentStopped {
				forceStopRoots = append(forceStopRoots, node)
			} else {
				recoveryRoots = append(recoveryRoots, node)
			}
			continue
		}

		if node.intent == intentStopped {
			node.phase = phaseStopped
			node.lastError = nil
			continue
		}
		if previousPhase == phaseBlocked || previousPhase == phaseStopped || previousPhase == phaseFailed {
			if previousPhase == phaseFailed && status.Error != nil {
				node.lastError = status.Error
			}
			continue
		}
		if previousPhase == phaseRunning || previousRunning {
			m.markFailureLocked(node, status.Error)
			failureRoots = append(failureRoots, node)
		}
	}
	m.mu.Unlock()
	m.publishAllWrapperStatuses()

	if len(failureRoots) > 0 {
		m.stopDescendantsOfFailedRoots(failureRoots)
	}
	for _, root := range forceStopRoots {
		m.mu.Lock()
		nodes := m.collectRunningStatefulDownstreamLocked(root, true)
		m.mu.Unlock()
		m.runStatefulStopLayers(buildStopLayers(nodes), root)
	}
	if len(recoveryRoots) > 0 {
		m.startDownstreamFromRoots(recoveryRoots)
	}
}
