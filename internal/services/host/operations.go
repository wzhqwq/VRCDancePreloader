package host

import (
	"errors"
	"fmt"
	"runtime/debug"
	"sync"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

type nodeOperationResult struct {
	node    *serviceNode
	status  interactive.RunnerStatus
	callErr error
}

func panicError(operation string, recovered any) error {
	return fmt.Errorf("%s panicked: %v\n%s", operation, recovered, debug.Stack())
}

func safeCall(operation string, fn func()) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = panicError(operation, recovered)
		}
	}()
	fn()
	return nil
}

func safeCallError(operation string, fn func() error) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = panicError(operation, recovered)
		}
	}()
	return fn()
}

func safeStatus(node *serviceNode) (status interactive.RunnerStatus) {
	defer func() {
		if recovered := recover(); recovered != nil {
			status = interactive.RunnerStatus{Error: panicError("Status", recovered)}
		}
	}()
	return node.service.Status()
}

func combineOperationStatus(status interactive.RunnerStatus, callErr error) interactive.RunnerStatus {
	status.Error = errors.Join(callErr, status.Error)
	return status
}

func (m *Manager) runStartBatch(nodes []*serviceNode) {
	if len(nodes) == 0 {
		return
	}
	m.mu.Lock()
	for _, node := range nodes {
		node.phase = phaseStarting
		node.expected = expectedStart
		node.startAttempted = true
		node.lastError = nil
	}
	m.mu.Unlock()
	m.publishAllWrapperStatuses()

	results := make(chan nodeOperationResult, len(nodes))
	var wg sync.WaitGroup
	for _, node := range nodes {
		wg.Add(1)
		go func(node *serviceNode) {
			defer wg.Done()
			callErr := safeCall("Start service "+node.name, node.service.Start)
			status := combineOperationStatus(safeStatus(node), callErr)
			results <- nodeOperationResult{node: node, status: status, callErr: callErr}
		}(node)
	}
	wg.Wait()
	close(results)

	m.mu.Lock()
	for result := range results {
		node := result.node
		node.expected = expectedNone
		node.lastStatus = result.status

		if result.callErr == nil && result.status.Running {
			node.phase = phaseRunning
			node.lastError = result.status.Error
			continue
		}

		if node.isStateful() &&
			result.callErr == nil &&
			result.status.Error == nil {
			node.phase = phaseStopped
			node.lastError = nil
			continue
		}

		node.phase = phaseFailed
		node.lastError = result.status.Error
		if node.lastError == nil {
			node.lastError = fmt.Errorf("%w: %s", ErrServiceDidNotStart, node.name)
		}
		if !node.isStateful() {
			node.permanentFailure = true
		}
	}
	m.mu.Unlock()
	m.publishAllWrapperStatuses()
}

func (m *Manager) runShutdownBatch(nodes []*serviceNode) []error {
	if len(nodes) == 0 {
		return nil
	}
	m.mu.Lock()
	for _, node := range nodes {
		node.phase = phaseStopping
		node.expected = expectedShutdown
	}
	m.mu.Unlock()
	m.publishAllWrapperStatuses()

	results := make(chan nodeOperationResult, len(nodes))
	var wg sync.WaitGroup
	for _, node := range nodes {
		wg.Add(1)
		go func(node *serviceNode) {
			defer wg.Done()
			callErr := safeCallError("Shutdown service "+node.name, node.service.Shutdown)
			status := combineOperationStatus(safeStatus(node), callErr)
			results <- nodeOperationResult{node: node, status: status, callErr: callErr}
		}(node)
	}
	wg.Wait()
	close(results)

	errs := make([]error, 0)
	m.mu.Lock()
	for result := range results {
		node := result.node
		node.expected = expectedNone
		node.lastStatus = result.status
		if result.callErr != nil {
			errs = append(errs, fmt.Errorf("shutdown service %q: %w", node.name, result.callErr))
		}
		if result.status.Running {
			node.phase = phaseFailed
			node.lastError = errors.Join(result.status.Error, fmt.Errorf("%w: %s", ErrServiceDidNotStop, node.name))
			errs = append(errs, fmt.Errorf("shutdown service %q: %w", node.name, node.lastError))
		} else {
			node.phase = phaseStopped
			node.lastError = result.status.Error
		}
	}
	m.mu.Unlock()
	m.publishAllWrapperStatuses()
	return errs
}

type stopDisposition uint8

const (
	stopAsDependency stopDisposition = iota
	stopAsUser
)

func (m *Manager) runStatefulStopLayers(layers [][]*serviceNode, userRoot *serviceNode) {
	for _, layer := range layers {
		m.mu.Lock()
		for _, node := range layer {
			node.phase = phaseStopping
			if node == userRoot {
				node.expected = expectedManualStop
			} else {
				node.expected = expectedDependencyStop
			}
		}
		m.mu.Unlock()
		m.publishAllWrapperStatuses()

		results := make(chan nodeOperationResult, len(layer))
		var wg sync.WaitGroup
		for _, node := range layer {
			wg.Add(1)
			go func(node *serviceNode) {
				defer wg.Done()
				callErr := safeCall("Stop service "+node.name, node.stateful.Stop)
				status := combineOperationStatus(safeStatus(node), callErr)
				results <- nodeOperationResult{node: node, status: status, callErr: callErr}
			}(node)
		}
		wg.Wait()
		close(results)

		m.mu.Lock()
		for result := range results {
			node := result.node
			node.expected = expectedNone
			node.lastStatus = result.status
			if result.callErr != nil || result.status.Running {
				node.phase = phaseFailed
				node.lastError = result.status.Error
				if result.status.Running {
					node.lastError = errors.Join(node.lastError, fmt.Errorf("%w: %s", ErrServiceDidNotStop, node.name))
				}
				continue
			}
			if node == userRoot {
				node.phase = phaseStopped
				node.lastError = result.status.Error
			} else {
				node.phase = phaseBlocked
				node.lastError = nil
			}
		}
		m.mu.Unlock()
		m.publishAllWrapperStatuses()
	}
}
