// Package resolver provides dependency resolution for dex packages.
package resolver

import (
	"fmt"
	"sort"
)

// DepGraph represents a dependency graph.
type DepGraph struct {
	nodes map[string]*DepNode
}

// DepNode represents a node in the dependency graph.
type DepNode struct {
	// Name is the package name
	Name string

	// Version is the resolved version
	Version string

	// Constraint is the original version constraint
	Constraint string

	// Dependencies maps dependency names to version constraints
	Dependencies map[string]string

	// Dependents lists packages that depend on this one
	Dependents []string

	// Registry is the registry to fetch from
	Registry string

	// Source is the direct source URL
	Source string
}

// NewDepGraph creates a new empty dependency graph.
func NewDepGraph() *DepGraph {
	return &DepGraph{
		nodes: make(map[string]*DepNode),
	}
}

// AddNode adds a node to the graph. Returns the node for further modification.
func (g *DepGraph) AddNode(name string) *DepNode {
	if existing, ok := g.nodes[name]; ok {
		return existing
	}
	node := &DepNode{
		Name:         name,
		Dependencies: make(map[string]string),
		Dependents:   []string{},
	}
	g.nodes[name] = node
	return node
}

// GetNode returns a node by name, or nil if not found.
func (g *DepGraph) GetNode(name string) *DepNode {
	return g.nodes[name]
}

// HasNode returns true if the graph has a node with the given name.
func (g *DepGraph) HasNode(name string) bool {
	_, ok := g.nodes[name]
	return ok
}

// AddDependency adds a dependency edge from parent to child with a version constraint.
func (g *DepGraph) AddDependency(parent, child, constraint string) {
	parentNode := g.AddNode(parent)
	childNode := g.AddNode(child)

	parentNode.Dependencies[child] = constraint
	childNode.Dependents = append(childNode.Dependents, parent)
}

// TopologicalSort returns the nodes in topological order (dependencies first).
// Returns an error if the graph contains a cycle.
func (g *DepGraph) TopologicalSort() ([]string, error) {
	// Kahn's algorithm
	inDegree := make(map[string]int)
	for name := range g.nodes {
		inDegree[name] = 0
	}

	// Calculate in-degrees
	for _, node := range g.nodes {
		for dep := range node.Dependencies {
			inDegree[dep]++
		}
	}

	// Start with nodes that have no incoming edges (no dependents)
	var queue []string
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	// Sort for deterministic output
	sort.Strings(queue)

	var result []string
	for len(queue) > 0 {
		// Dequeue
		name := queue[0]
		queue = queue[1:]
		result = append(result, name)

		// Decrease in-degree for dependencies
		node := g.nodes[name]
		for dep := range node.Dependencies {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
				sort.Strings(queue)
			}
		}
	}

	// Check for cycles
	if len(result) != len(g.nodes) {
		// Find nodes in the cycle for error message
		var cycleNodes []string
		for name, degree := range inDegree {
			if degree > 0 {
				cycleNodes = append(cycleNodes, name)
			}
		}
		sort.Strings(cycleNodes)
		return nil, &CycleError{Packages: cycleNodes}
	}

	// Reverse to get install order (dependencies first)
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return result, nil
}

// FindDependents returns all packages that depend on the given package.
func (g *DepGraph) FindDependents(name string) []string {
	node := g.nodes[name]
	if node == nil {
		return nil
	}

	// Return a copy to prevent modification
	result := make([]string, len(node.Dependents))
	copy(result, node.Dependents)
	sort.Strings(result)
	return result
}

// AllNodes returns all node names in the graph.
func (g *DepGraph) AllNodes() []string {
	names := make([]string, 0, len(g.nodes))
	for name := range g.nodes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// CycleError indicates a circular dependency was detected.
type CycleError struct {
	Packages []string
}

func (e *CycleError) Error() string {
	return fmt.Sprintf("circular dependency detected involving: %v", e.Packages)
}
