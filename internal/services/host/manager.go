package host

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/wzhqwq/VRCDancePreloader/internal/config"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/service"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

var logger = utils.NewLogger("VRCDP Host")

type serviceDependency struct {
	name string
	s    service.Service
	deps []string
}

type Manager struct {
	operationMu sync.Mutex
	mu          sync.RWMutex

	cfg         config.Config
	cfgMgr      *config.Manager
	cfgUpdaters map[string]CfgApplier

	state managerState
	nodes map[string]*serviceNode
	order []*serviceNode

	registrationErrors []error
	validationErr      error
	shutdownErr        error
	startLayers        [][]*serviceNode
	stopLayers         [][]*serviceNode

	listenerCtx    context.Context
	listenerCancel context.CancelFunc
	listenerWG     sync.WaitGroup

	pendingMu     sync.Mutex
	pendingRoots  map[string]struct{}
	reconcileWake chan struct{}
}

func NewManager(cfg config.Config) *Manager {
	return &Manager{
		cfg:         cfg,
		cfgUpdaters: make(map[string]CfgApplier),

		state:         managerCreated,
		nodes:         make(map[string]*serviceNode),
		pendingRoots:  make(map[string]struct{}),
		reconcileWake: make(chan struct{}, 1),
	}
}

func (m *Manager) ensureInitializedLocked() {
	if m.nodes == nil {
		m.nodes = make(map[string]*serviceNode)
	}
	if m.pendingRoots == nil {
		m.pendingRoots = make(map[string]struct{})
	}
	if m.reconcileWake == nil {
		m.reconcileWake = make(chan struct{}, 1)
	}
}

func isNilInterface(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

func (m *Manager) RegisterService(name string, s service.Service, deps ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureInitializedLocked()

	if m.state != managerCreated {
		m.registrationErrors = append(m.registrationErrors,
			fmt.Errorf("register service %q: %w", name, ErrRegistrationClosed))
		return
	}
	if strings.TrimSpace(name) == "" {
		m.registrationErrors = append(m.registrationErrors, errors.New("service name cannot be empty"))
		return
	}
	if isNilInterface(s) {
		m.registrationErrors = append(m.registrationErrors,
			fmt.Errorf("service %q is nil", name))
		return
	}
	if _, exists := m.nodes[name]; exists {
		m.registrationErrors = append(m.registrationErrors,
			fmt.Errorf("service %q is registered more than once", name))
		return
	}

	seenDeps := make(map[string]struct{}, len(deps))
	cleanDeps := make([]string, 0, len(deps))
	for _, dep := range deps {
		if dep == name {
			m.registrationErrors = append(m.registrationErrors,
				fmt.Errorf("service %q cannot depend on itself", name))
			continue
		}
		if _, exists := seenDeps[dep]; exists {
			m.registrationErrors = append(m.registrationErrors,
				fmt.Errorf("service %q declares dependency %q more than once", name, dep))
			continue
		}
		seenDeps[dep] = struct{}{}
		cleanDeps = append(cleanDeps, dep)
	}

	node := &serviceNode{
		name:     name,
		order:    len(m.order),
		service:  s,
		depNames: cleanDeps,
		phase:    phaseStopped,
		intent:   intentAuto,
	}
	if stateful, ok := s.(interactive.StatefulService); ok && !isNilInterface(stateful) {
		node.stateful = stateful
		node.wrapper = newWrappedService(m, node)
	}
	m.nodes[name] = node
	m.order = append(m.order, node)
}

func (m *Manager) WrappedService(name string) interactive.StatefulService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if node := m.nodes[name]; node != nil && node.wrapper != nil {
		return node.wrapper
	}
	return nil
}

func (m *Manager) Status() interactive.RunnerStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	switch m.state {
	case managerCreated:
		return interactive.RunnerStatus{Error: ErrHostNotStarted}
	case managerStarting:
		return interactive.RunnerStatus{Error: ErrHostStarting}
	case managerRunning:
		return interactive.RunnerStatus{
			Running: true,
			Error:   m.aggregateNodeErrorsLocked(),
		}
	case managerStopping:
		return interactive.RunnerStatus{Error: errors.Join(ErrHostStopping, m.shutdownErr)}
	case managerStopped:
		return interactive.RunnerStatus{Error: errors.Join(ErrHostStopped, m.shutdownErr)}
	case managerInvalid:
		return interactive.RunnerStatus{Error: m.validationErr}
	default:
		return interactive.RunnerStatus{Error: ErrHostStopped}
	}
}

func (m *Manager) ServiceStatuses() map[string]interactive.RunnerStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]interactive.RunnerStatus, len(m.nodes))
	for name, node := range m.nodes {
		if node.wrapper != nil {
			result[name] = m.wrappedStatusLocked(node)
		} else {
			result[name] = node.lastStatus
			if node.phase == phaseFailed {
				result[name] = interactive.RunnerStatus{Running: node.lastStatus.Running, Error: node.lastError}
			}
		}
	}
	return result
}

func (m *Manager) getService(name string) service.Service {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if node := m.nodes[name]; node != nil && node.wrapper != nil {
		return node.service
	}
	return nil
}

func (m *Manager) Config() *config.Manager {
	return m.cfgMgr
}
