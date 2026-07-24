package host

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/wzhqwq/VRCDancePreloader/internal/services/service"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

var (
	ErrHostNotStarted     = errors.New("service host has not started")
	ErrHostStarting       = errors.New("service host is starting")
	ErrHostStopping       = errors.New("service host is stopping")
	ErrHostStopped        = errors.New("service host is stopped")
	ErrServiceDidNotStart = errors.New("service did not enter running state")
	ErrServiceDidNotStop  = errors.New("service did not enter stopped state")
	ErrUnexpectedStop     = errors.New("service stopped unexpectedly")
	ErrRegistrationClosed = errors.New("service registration is closed")
)

type managerState uint8

const (
	managerCreated managerState = iota
	managerStarting
	managerRunning
	managerStopping
	managerStopped
	managerInvalid
)

type lifecyclePhase uint8

const (
	phaseStopped lifecyclePhase = iota
	phaseStarting
	phaseRunning
	phaseStopping
	phaseFailed
	phaseBlocked
)

type runIntent uint8

const (
	intentAuto runIntent = iota
	intentRunning
	intentStopped
)

type expectedOperation uint8

const (
	expectedNone expectedOperation = iota
	expectedStart
	expectedManualStop
	expectedDependencyStop
	expectedRestart
	expectedShutdown
)

type serviceNode struct {
	name       string
	order      int
	service    service.Service
	stateful   interactive.StatefulService
	depNames   []string
	deps       []*serviceNode
	dependents []*serviceNode
	wrapper    *wrappedService

	phase            lifecyclePhase
	intent           runIntent
	expected         expectedOperation
	lastStatus       interactive.RunnerStatus
	lastError        error
	startAttempted   bool
	permanentFailure bool
	subscriber       *utils.EventSubscriber[interactive.RunnerStatus]
}

func (n *serviceNode) isStateful() bool {
	return n.stateful != nil
}

func (m *Manager) aggregateNodeErrorsLocked() error {
	errs := make([]error, 0)
	for _, node := range m.order {
		if node.phase == phaseFailed && node.lastError != nil {
			errs = append(errs, fmt.Errorf("service %q: %w", node.name, node.lastError))
		}
	}
	return errors.Join(errs...)
}

func (m *Manager) wrappedStatusLocked(node *serviceNode) interactive.RunnerStatus {
	switch m.state {
	case managerCreated:
		return interactive.RunnerStatus{Error: ErrHostNotStarted}
	case managerStarting:
		return interactive.RunnerStatus{Error: ErrHostStarting}
	case managerStopping:
		return interactive.RunnerStatus{Error: ErrHostStopping}
	case managerStopped:
		return interactive.RunnerStatus{Error: ErrHostStopped}
	case managerInvalid:
		return interactive.RunnerStatus{Error: m.validationErr}
	default:
	}

	if node.phase == phaseFailed {
		return interactive.RunnerStatus{Running: node.lastStatus.Running, Error: node.lastError}
	}
	if node.intent == intentStopped {
		return interactive.RunnerStatus{}
	}
	switch node.phase {
	case phaseRunning:
		return node.lastStatus
	case phaseBlocked:
		return interactive.RunnerStatus{Error: m.dependencyErrorLocked(node)}
	case phaseStarting, phaseStopping, phaseStopped:
		return interactive.RunnerStatus{Error: node.lastError}
	default:
		return interactive.RunnerStatus{Error: node.lastError}
	}
}

func (m *Manager) dependencyErrorLocked(node *serviceNode) error {
	issues := make([]string, 0)
	for _, dep := range node.deps {
		if m.nodeRunningLocked(dep) {
			continue
		}
		reason := "not running"
		switch {
		case dep.intent == intentStopped:
			reason = "user-stopped"
		case dep.phase == phaseFailed:
			reason = "failed"
		case dep.phase == phaseBlocked:
			reason = "blocked"
		}
		issues = append(issues, fmt.Sprintf("%s (%s)", dep.name, reason))
	}
	if len(issues) == 0 {
		return node.lastError
	}
	sort.Strings(issues)
	return fmt.Errorf("service %q is blocked by dependencies: %s", node.name, strings.Join(issues, ", "))
}

func (m *Manager) nodeRunningLocked(node *serviceNode) bool {
	return node.phase == phaseRunning && node.lastStatus.Running
}

func (m *Manager) dependenciesRunningLocked(node *serviceNode) bool {
	for _, dep := range node.deps {
		if !m.nodeRunningLocked(dep) {
			return false
		}
	}
	return true
}

func (m *Manager) publishAllWrapperStatuses() {
	m.mu.RLock()
	updates := make([]struct {
		wrapper *wrappedService
		status  interactive.RunnerStatus
	}, 0)
	for _, node := range m.order {
		if node.wrapper == nil {
			continue
		}
		updates = append(updates, struct {
			wrapper *wrappedService
			status  interactive.RunnerStatus
		}{node.wrapper, m.wrappedStatusLocked(node)})
	}
	m.mu.RUnlock()

	for _, update := range updates {
		update.wrapper.publish(update.status)
	}
}
