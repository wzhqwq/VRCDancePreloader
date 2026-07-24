package host

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

func (m *Manager) validateAndBuildGraph() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureInitializedLocked()

	errs := append([]error(nil), m.registrationErrors...)
	for _, node := range m.order {
		node.deps = nil
		node.dependents = nil
	}

	for _, node := range m.order {
		for _, depName := range node.depNames {
			dep := m.nodes[depName]
			if dep == nil {
				errs = append(errs, fmt.Errorf("service %q depends on missing service %q", node.name, depName))
				continue
			}
			node.deps = append(node.deps, dep)
			dep.dependents = append(dep.dependents, node)
			if !node.isStateful() && dep.isStateful() {
				errs = append(errs, fmt.Errorf(
					"non-stateful service %q cannot depend on stateful service %q",
					node.name,
					dep.name,
				))
			}
		}
	}

	indegree := make(map[*serviceNode]int, len(m.order))
	current := make([]*serviceNode, 0)
	for _, node := range m.order {
		indegree[node] = len(node.deps)
		if len(node.deps) == 0 {
			current = append(current, node)
		}
	}
	sortNodes(current)

	layers := make([][]*serviceNode, 0)
	processed := 0
	for len(current) > 0 {
		layer := append([]*serviceNode(nil), current...)
		layers = append(layers, layer)
		processed += len(layer)
		next := make([]*serviceNode, 0)
		for _, node := range layer {
			for _, dependent := range node.dependents {
				indegree[dependent]--
				if indegree[dependent] == 0 {
					next = append(next, dependent)
				}
			}
		}
		sortNodes(next)
		current = next
	}

	if processed != len(m.order) {
		cycleNodes := make([]string, 0)
		for _, node := range m.order {
			if indegree[node] > 0 {
				cycleNodes = append(cycleNodes, node.name)
			}
		}
		sort.Strings(cycleNodes)
		errs = append(errs, fmt.Errorf("service dependency graph contains a cycle involving: %s", strings.Join(cycleNodes, ", ")))
	}

	if err := errors.Join(errs...); err != nil {
		return err
	}
	m.startLayers = layers
	allNodes := make(map[*serviceNode]struct{}, len(m.order))
	for _, node := range m.order {
		allNodes[node] = struct{}{}
	}
	m.stopLayers = buildStopLayers(allNodes)
	return nil
}

func sortNodes(nodes []*serviceNode) {
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].order == nodes[j].order {
			return nodes[i].name < nodes[j].name
		}
		return nodes[i].order < nodes[j].order
	})
}

func buildStopLayers(nodes map[*serviceNode]struct{}) [][]*serviceNode {
	if len(nodes) == 0 {
		return nil
	}
	remaining := make(map[*serviceNode]int, len(nodes))
	current := make([]*serviceNode, 0)
	for node := range nodes {
		for _, dependent := range node.dependents {
			if _, ok := nodes[dependent]; ok {
				remaining[node]++
			}
		}
	}
	for node := range nodes {
		if remaining[node] == 0 {
			current = append(current, node)
		}
	}
	sortNodes(current)

	layers := make([][]*serviceNode, 0)
	seen := 0
	for len(current) > 0 {
		layer := append([]*serviceNode(nil), current...)
		layers = append(layers, layer)
		seen += len(layer)
		next := make([]*serviceNode, 0)
		for _, stopped := range layer {
			for _, dependency := range stopped.deps {
				if _, ok := nodes[dependency]; !ok {
					continue
				}
				remaining[dependency]--
				if remaining[dependency] == 0 {
					next = append(next, dependency)
				}
			}
		}
		sortNodes(next)
		current = next
	}
	if seen != len(nodes) {
		panic("validated DAG produced cyclic stop subgraph")
	}
	return layers
}
