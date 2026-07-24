package host

import (
	"errors"
	"fmt"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

func (m *Manager) manualStart(target *serviceNode) {
	m.operationMu.Lock()

	m.mu.Lock()
	if m.state != managerRunning {
		m.mu.Unlock()
		m.operationMu.Unlock()
		m.publishAllWrapperStatuses()
		return
	}
	target.intent = intentRunning
	if target.phase == phaseFailed {
		target.phase = phaseStopped
		target.lastError = nil
		target.lastStatus.Running = false
		target.lastStatus.Error = nil
	}
	m.mu.Unlock()
	m.publishAllWrapperStatuses()

	candidates := m.collectStatefulAncestorsAndSelf(target)
	m.startStatefulCandidates(candidates, target)

	m.mu.RLock()
	running := m.nodeRunningLocked(target)
	m.mu.RUnlock()
	m.operationMu.Unlock()

	if running {
		m.requestReconcile(target.name)
	}
}

func (m *Manager) manualStop(target *serviceNode) {
	m.operationMu.Lock()
	defer m.operationMu.Unlock()

	m.mu.Lock()
	if m.state != managerRunning {
		m.mu.Unlock()
		m.publishAllWrapperStatuses()
		return
	}
	target.intent = intentStopped
	nodes := m.collectRunningStatefulDownstreamLocked(target, true)
	if _, included := nodes[target]; !included {
		target.phase = phaseStopped
		target.lastStatus.Running = false
		target.lastStatus.Error = nil
		target.lastError = nil
	}
	m.mu.Unlock()
	m.publishAllWrapperStatuses()

	m.runStatefulStopLayers(buildStopLayers(nodes), target)
}

func (m *Manager) manualRestart(target *serviceNode) {
	m.operationMu.Lock()

	m.mu.Lock()
	if m.state != managerRunning {
		m.mu.Unlock()
		m.operationMu.Unlock()
		m.publishAllWrapperStatuses()
		return
	}
	target.intent = intentRunning
	downstream := m.collectRunningStatefulDownstreamLocked(target, false)
	m.mu.Unlock()
	m.publishAllWrapperStatuses()

	m.runStatefulStopLayers(buildStopLayers(downstream), nil)
	ancestors := m.collectStatefulAncestorsAndSelf(target)
	m.refreshOrdinaryDependenciesForCandidates(ancestors)
	m.refreshStatefulCandidates(ancestors, target)

	m.mu.RLock()
	depsRunning := m.dependenciesRunningLocked(target)
	targetRunning := m.nodeRunningLocked(target)
	m.mu.RUnlock()
	if !depsRunning {
		if targetRunning {
			m.runStatefulStopLayers(buildStopLayers(map[*serviceNode]struct{}{target: {}}), nil)
		}
		m.mu.Lock()
		target.phase = phaseBlocked
		target.lastStatus.Running = false
		target.lastError = nil
		m.mu.Unlock()
		m.publishAllWrapperStatuses()
		m.operationMu.Unlock()
		return
	}

	m.runRestartNode(target)
	m.mu.RLock()
	running := m.nodeRunningLocked(target)
	m.mu.RUnlock()
	m.operationMu.Unlock()

	if running {
		m.requestReconcile(target.name)
	}
}

func (m *Manager) runRestartNode(target *serviceNode) {
	m.mu.Lock()
	target.phase = phaseStarting
	target.expected = expectedRestart
	target.startAttempted = true
	target.lastError = nil
	m.mu.Unlock()
	m.publishAllWrapperStatuses()

	callErr := safeCall("Restart service "+target.name, target.stateful.Restart)
	status := combineOperationStatus(safeStatus(target), callErr)

	m.mu.Lock()
	target.expected = expectedNone
	target.lastStatus = status
	if callErr == nil && status.Running {
		target.phase = phaseRunning
		target.lastError = status.Error
	} else if callErr == nil && status.Error == nil {
		target.phase = phaseStopped
		target.lastError = nil
	} else {
		target.phase = phaseFailed
		target.lastError = status.Error
		if target.lastError == nil {
			target.lastError = fmt.Errorf(
				"%w after restart: %s",
				ErrServiceDidNotStart,
				target.name,
			)
		}
	}
	m.mu.Unlock()
	m.publishAllWrapperStatuses()
}

func (m *Manager) collectStatefulAncestorsAndSelf(target *serviceNode) map[*serviceNode]struct{} {
	result := make(map[*serviceNode]struct{})
	seen := make(map[*serviceNode]struct{})
	stack := []*serviceNode{target}
	for len(stack) > 0 {
		last := len(stack) - 1
		node := stack[last]
		stack = stack[:last]
		if _, exists := seen[node]; exists {
			continue
		}
		seen[node] = struct{}{}
		if node.isStateful() {
			result[node] = struct{}{}
		}
		for _, dep := range node.deps {
			stack = append(stack, dep)
		}
	}
	return result
}

func (m *Manager) collectStatefulDescendantsLocked(roots []*serviceNode, includeRoots bool) map[*serviceNode]struct{} {
	result := make(map[*serviceNode]struct{})
	seen := make(map[*serviceNode]struct{})
	stack := append([]*serviceNode(nil), roots...)
	rootSet := make(map[*serviceNode]struct{}, len(roots))
	for _, root := range roots {
		rootSet[root] = struct{}{}
	}
	for len(stack) > 0 {
		last := len(stack) - 1
		node := stack[last]
		stack = stack[:last]
		if _, exists := seen[node]; exists {
			continue
		}
		seen[node] = struct{}{}
		_, isRoot := rootSet[node]
		if node.isStateful() && (includeRoots || !isRoot) {
			result[node] = struct{}{}
		}
		for _, dependent := range node.dependents {
			stack = append(stack, dependent)
		}
	}
	return result
}

func (m *Manager) collectRunningStatefulDownstreamLocked(root *serviceNode, includeRoot bool) map[*serviceNode]struct{} {
	all := m.collectStatefulDescendantsLocked([]*serviceNode{root}, includeRoot)
	for node := range all {
		if !m.nodeRunningLocked(node) {
			delete(all, node)
		}
	}
	return all
}

func (m *Manager) startStatefulCandidates(candidates map[*serviceNode]struct{}, explicitTarget *serviceNode) {
	m.refreshOrdinaryDependenciesForCandidates(candidates)
	m.refreshStatefulCandidates(candidates, explicitTarget)
	for _, globalLayer := range m.startLayers {
		batch := make([]*serviceNode, 0)
		m.mu.Lock()
		for _, node := range globalLayer {
			if _, selected := candidates[node]; !selected || !node.isStateful() {
				continue
			}
			if m.nodeRunningLocked(node) || node.phase == phaseStarting {
				continue
			}
			if node.intent == intentStopped {
				continue
			}
			if node.phase == phaseFailed && node != explicitTarget {
				continue
			}
			if !m.dependenciesRunningLocked(node) {
				node.phase = phaseBlocked
				node.lastStatus.Running = false
				node.lastError = nil
				continue
			}
			batch = append(batch, node)
		}
		m.mu.Unlock()
		m.publishAllWrapperStatuses()
		m.runStartBatch(batch)
	}
}

func (m *Manager) startDownstreamFromRoots(roots []*serviceNode) {
	m.mu.RLock()
	candidates := m.collectStatefulDescendantsLocked(roots, false)
	m.mu.RUnlock()
	m.startStatefulCandidates(candidates, nil)
}

func (m *Manager) markFailureLocked(node *serviceNode, statusErr error) {
	node.phase = phaseFailed
	node.lastError = statusErr
	if node.lastError == nil {
		node.lastError = fmt.Errorf("%w: %s", ErrUnexpectedStop, node.name)
	}
	if !node.isStateful() {
		node.permanentFailure = true
	}
}

func (m *Manager) stopDescendantsOfFailedRoots(roots []*serviceNode) {
	m.mu.RLock()
	all := m.collectStatefulDescendantsLocked(roots, false)
	for node := range all {
		if !m.nodeRunningLocked(node) {
			delete(all, node)
		}
	}
	m.mu.RUnlock()
	m.runStatefulStopLayers(buildStopLayers(all), nil)
}

func (m *Manager) refreshStatefulCandidates(candidates map[*serviceNode]struct{}, explicitTarget *serviceNode) {
	type observation struct {
		node   *serviceNode
		status interactive.RunnerStatus
	}
	observations := make([]observation, 0, len(candidates))
	for node := range candidates {
		if node.isStateful() {
			observations = append(observations, observation{node: node, status: safeStatus(node)})
		}
	}

	failed := make([]*serviceNode, 0)
	forceStop := make([]*serviceNode, 0)
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
				forceStop = append(forceStop, node)
			}
			continue
		}
		if node == explicitTarget {
			if node.phase != phaseBlocked {
				node.phase = phaseStopped
			}
			node.lastError = nil
			continue
		}
		if node.intent == intentStopped {
			node.phase = phaseStopped
			node.lastError = nil
			continue
		}
		if previousPhase == phaseRunning || previousRunning {
			m.markFailureLocked(node, status.Error)
			failed = append(failed, node)
			continue
		}
		if previousPhase == phaseFailed && status.Error != nil {
			node.lastError = status.Error
		}
	}
	m.mu.Unlock()
	m.publishAllWrapperStatuses()

	if len(failed) > 0 {
		m.stopDescendantsOfFailedRoots(failed)
	}
	for _, root := range forceStop {
		m.mu.Lock()
		nodes := m.collectRunningStatefulDownstreamLocked(root, true)
		m.mu.Unlock()
		m.runStatefulStopLayers(buildStopLayers(nodes), root)
	}
}

func (m *Manager) refreshOrdinaryDependenciesForCandidates(candidates map[*serviceNode]struct{}) {
	ordinary := make(map[*serviceNode]struct{})
	seen := make(map[*serviceNode]struct{})
	stack := make([]*serviceNode, 0, len(candidates))
	for node := range candidates {
		stack = append(stack, node)
	}
	for len(stack) > 0 {
		last := len(stack) - 1
		node := stack[last]
		stack = stack[:last]
		if _, exists := seen[node]; exists {
			continue
		}
		seen[node] = struct{}{}
		for _, dep := range node.deps {
			if dep.isStateful() {
				stack = append(stack, dep)
				continue
			}
			ordinary[dep] = struct{}{}
			stack = append(stack, dep)
		}
	}

	failed := make([]*serviceNode, 0)
	for node := range ordinary {
		m.mu.RLock()
		permanentFailure := node.permanentFailure
		m.mu.RUnlock()
		if permanentFailure {
			continue
		}
		status := safeStatus(node)
		m.mu.Lock()
		if node.permanentFailure {
			m.mu.Unlock()
			continue
		}
		node.lastStatus = status
		if status.Running {
			node.phase = phaseRunning
			node.lastError = status.Error
			m.mu.Unlock()
			continue
		}
		m.markFailureLocked(node, errors.Join(
			status.Error,
			fmt.Errorf("ordinary service %q is unavailable and will not be restarted", node.name),
		))
		failed = append(failed, node)
		m.mu.Unlock()
	}
	m.publishAllWrapperStatuses()
	if len(failed) > 0 {
		m.stopDescendantsOfFailedRoots(failed)
	}
}
