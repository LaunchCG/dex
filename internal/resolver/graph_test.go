package resolver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDepGraph(t *testing.T) {
	g := NewDepGraph()
	assert.NotNil(t, g)
	assert.Empty(t, g.AllNodes())
}

func TestDepGraph_AddNode(t *testing.T) {
	g := NewDepGraph()

	node := g.AddNode("pkg-a")
	assert.NotNil(t, node)
	assert.Equal(t, "pkg-a", node.Name)
	assert.NotNil(t, node.Dependencies)
	assert.Empty(t, node.Dependents)

	// Adding same node again returns existing
	node2 := g.AddNode("pkg-a")
	assert.Same(t, node, node2)
}

func TestDepGraph_GetNode(t *testing.T) {
	g := NewDepGraph()

	// Non-existent node
	assert.Nil(t, g.GetNode("nonexistent"))

	// Add and get
	g.AddNode("pkg-a")
	node := g.GetNode("pkg-a")
	assert.NotNil(t, node)
	assert.Equal(t, "pkg-a", node.Name)
}

func TestDepGraph_HasNode(t *testing.T) {
	g := NewDepGraph()

	assert.False(t, g.HasNode("pkg-a"))
	g.AddNode("pkg-a")
	assert.True(t, g.HasNode("pkg-a"))
}

func TestDepGraph_AddDependency(t *testing.T) {
	g := NewDepGraph()

	g.AddDependency("app", "lib", "^1.0.0")

	// Both nodes should exist
	assert.True(t, g.HasNode("app"))
	assert.True(t, g.HasNode("lib"))

	// Check dependency was recorded
	appNode := g.GetNode("app")
	assert.Equal(t, "^1.0.0", appNode.Dependencies["lib"])

	// Check dependent was recorded
	libNode := g.GetNode("lib")
	assert.Contains(t, libNode.Dependents, "app")
}

func TestDepGraph_TopologicalSort_Simple(t *testing.T) {
	g := NewDepGraph()

	// app -> lib -> core
	g.AddDependency("app", "lib", "^1.0.0")
	g.AddDependency("lib", "core", "^2.0.0")

	order, err := g.TopologicalSort()
	require.NoError(t, err)
	assert.Len(t, order, 3)

	// core should come before lib, lib before app
	coreIdx := indexOf(order, "core")
	libIdx := indexOf(order, "lib")
	appIdx := indexOf(order, "app")

	assert.Less(t, coreIdx, libIdx, "core should come before lib")
	assert.Less(t, libIdx, appIdx, "lib should come before app")
}

func TestDepGraph_TopologicalSort_Diamond(t *testing.T) {
	g := NewDepGraph()

	// Diamond dependency:
	//     app
	//    /   \
	//   a     b
	//    \   /
	//    core
	g.AddDependency("app", "a", "^1.0.0")
	g.AddDependency("app", "b", "^1.0.0")
	g.AddDependency("a", "core", "^1.0.0")
	g.AddDependency("b", "core", "^1.0.0")

	order, err := g.TopologicalSort()
	require.NoError(t, err)
	assert.Len(t, order, 4)

	// core must come before a and b, which must come before app
	coreIdx := indexOf(order, "core")
	aIdx := indexOf(order, "a")
	bIdx := indexOf(order, "b")
	appIdx := indexOf(order, "app")

	assert.Less(t, coreIdx, aIdx)
	assert.Less(t, coreIdx, bIdx)
	assert.Less(t, aIdx, appIdx)
	assert.Less(t, bIdx, appIdx)
}

func TestDepGraph_TopologicalSort_Cycle(t *testing.T) {
	g := NewDepGraph()

	// a -> b -> c -> a (cycle)
	g.AddDependency("a", "b", "^1.0.0")
	g.AddDependency("b", "c", "^1.0.0")
	g.AddDependency("c", "a", "^1.0.0")

	order, err := g.TopologicalSort()
	assert.Nil(t, order)
	require.Error(t, err)

	cycleErr, ok := err.(*CycleError)
	require.True(t, ok, "error should be CycleError")
	assert.Len(t, cycleErr.Packages, 3)
	assert.Contains(t, cycleErr.Packages, "a")
	assert.Contains(t, cycleErr.Packages, "b")
	assert.Contains(t, cycleErr.Packages, "c")
}

func TestDepGraph_TopologicalSort_NoDependencies(t *testing.T) {
	g := NewDepGraph()

	g.AddNode("a")
	g.AddNode("b")
	g.AddNode("c")

	order, err := g.TopologicalSort()
	require.NoError(t, err)
	assert.Len(t, order, 3)
	// All nodes should be present (order doesn't matter since no deps)
	assert.Contains(t, order, "a")
	assert.Contains(t, order, "b")
	assert.Contains(t, order, "c")
}

func TestDepGraph_FindDependents(t *testing.T) {
	g := NewDepGraph()

	g.AddDependency("app1", "lib", "^1.0.0")
	g.AddDependency("app2", "lib", "^2.0.0")
	g.AddDependency("lib", "core", "^1.0.0")

	// lib has two dependents
	libDeps := g.FindDependents("lib")
	assert.Len(t, libDeps, 2)
	assert.Contains(t, libDeps, "app1")
	assert.Contains(t, libDeps, "app2")

	// core has one dependent
	coreDeps := g.FindDependents("core")
	assert.Len(t, coreDeps, 1)
	assert.Contains(t, coreDeps, "lib")

	// apps have no dependents
	assert.Empty(t, g.FindDependents("app1"))
	assert.Empty(t, g.FindDependents("app2"))

	// Non-existent node
	assert.Nil(t, g.FindDependents("nonexistent"))
}

func TestDepGraph_AllNodes(t *testing.T) {
	g := NewDepGraph()

	g.AddNode("c")
	g.AddNode("a")
	g.AddNode("b")

	nodes := g.AllNodes()
	assert.Len(t, nodes, 3)
	// Should be sorted alphabetically
	assert.Equal(t, []string{"a", "b", "c"}, nodes)
}

func TestCycleError_Error(t *testing.T) {
	err := &CycleError{Packages: []string{"a", "b", "c"}}
	errMsg := err.Error()
	assert.Contains(t, errMsg, "circular dependency")
	assert.Contains(t, errMsg, "a")
	assert.Contains(t, errMsg, "b")
	assert.Contains(t, errMsg, "c")
}

// Helper function to find index of element in slice
func indexOf(slice []string, item string) int {
	for i, s := range slice {
		if s == item {
			return i
		}
	}
	return -1
}

// TestDepGraph_TopologicalSort_LongChain tests a deep linear dependency chain.
// Chain: app -> layer5 -> layer4 -> layer3 -> layer2 -> layer1 -> core
func TestDepGraph_TopologicalSort_LongChain(t *testing.T) {
	g := NewDepGraph()

	g.AddDependency("app", "layer5", "^1.0.0")
	g.AddDependency("layer5", "layer4", "^1.0.0")
	g.AddDependency("layer4", "layer3", "^1.0.0")
	g.AddDependency("layer3", "layer2", "^1.0.0")
	g.AddDependency("layer2", "layer1", "^1.0.0")
	g.AddDependency("layer1", "core", "^1.0.0")

	order, err := g.TopologicalSort()
	require.NoError(t, err)
	assert.Len(t, order, 7)

	// Verify order: core must come before layer1, layer1 before layer2, etc.
	for i := 0; i < len(order)-1; i++ {
		// Each layer should come before the one that depends on it
		if order[i] == "core" {
			assert.Less(t, i, indexOf(order, "layer1"))
		}
	}

	coreIdx := indexOf(order, "core")
	layer1Idx := indexOf(order, "layer1")
	layer2Idx := indexOf(order, "layer2")
	layer3Idx := indexOf(order, "layer3")
	layer4Idx := indexOf(order, "layer4")
	layer5Idx := indexOf(order, "layer5")
	appIdx := indexOf(order, "app")

	assert.Less(t, coreIdx, layer1Idx)
	assert.Less(t, layer1Idx, layer2Idx)
	assert.Less(t, layer2Idx, layer3Idx)
	assert.Less(t, layer3Idx, layer4Idx)
	assert.Less(t, layer4Idx, layer5Idx)
	assert.Less(t, layer5Idx, appIdx)
}

// TestDepGraph_TopologicalSort_MultiRoot tests multiple independent root packages.
func TestDepGraph_TopologicalSort_MultiRoot(t *testing.T) {
	g := NewDepGraph()

	// Three independent apps sharing some common dependencies
	// app-a -> shared-lib -> core
	// app-b -> shared-lib -> core
	// app-c -> utils -> core
	g.AddDependency("app-a", "shared-lib", "^1.0.0")
	g.AddDependency("app-b", "shared-lib", "^1.0.0")
	g.AddDependency("app-c", "utils", "^1.0.0")
	g.AddDependency("shared-lib", "core", "^1.0.0")
	g.AddDependency("utils", "core", "^1.0.0")

	order, err := g.TopologicalSort()
	require.NoError(t, err)
	assert.Len(t, order, 6)

	// Core must come before shared-lib and utils
	coreIdx := indexOf(order, "core")
	sharedIdx := indexOf(order, "shared-lib")
	utilsIdx := indexOf(order, "utils")

	assert.Less(t, coreIdx, sharedIdx)
	assert.Less(t, coreIdx, utilsIdx)

	// shared-lib must come before app-a and app-b
	appAIdx := indexOf(order, "app-a")
	appBIdx := indexOf(order, "app-b")
	assert.Less(t, sharedIdx, appAIdx)
	assert.Less(t, sharedIdx, appBIdx)

	// utils must come before app-c
	appCIdx := indexOf(order, "app-c")
	assert.Less(t, utilsIdx, appCIdx)
}

// TestDepGraph_TopologicalSort_ComplexGraph tests a complex real-world-like dependency graph.
func TestDepGraph_TopologicalSort_ComplexGraph(t *testing.T) {
	g := NewDepGraph()

	// Complex graph simulating a real project:
	// main-app -> (api-client, ui-framework)
	// api-client -> (http-lib, json-parser, core)
	// ui-framework -> (dom-utils, event-system, core)
	// http-lib -> core
	// json-parser -> (core)
	// dom-utils -> core
	// event-system -> core

	g.AddDependency("main-app", "api-client", "^2.0.0")
	g.AddDependency("main-app", "ui-framework", "^3.0.0")
	g.AddDependency("api-client", "http-lib", "^1.0.0")
	g.AddDependency("api-client", "json-parser", "^1.0.0")
	g.AddDependency("api-client", "core", "^1.0.0")
	g.AddDependency("ui-framework", "dom-utils", "^1.0.0")
	g.AddDependency("ui-framework", "event-system", "^1.0.0")
	g.AddDependency("ui-framework", "core", "^1.0.0")
	g.AddDependency("http-lib", "core", "^1.0.0")
	g.AddDependency("json-parser", "core", "^1.0.0")
	g.AddDependency("dom-utils", "core", "^1.0.0")
	g.AddDependency("event-system", "core", "^1.0.0")

	order, err := g.TopologicalSort()
	require.NoError(t, err)
	assert.Len(t, order, 8)

	coreIdx := indexOf(order, "core")
	mainAppIdx := indexOf(order, "main-app")

	// core should be first (or early), main-app should be last
	assert.Less(t, coreIdx, mainAppIdx)

	// All packages that depend on core should come after core
	for _, pkg := range []string{"http-lib", "json-parser", "dom-utils", "event-system", "api-client", "ui-framework"} {
		assert.Less(t, coreIdx, indexOf(order, pkg), "%s should come after core", pkg)
	}

	// api-client and ui-framework should come before main-app
	apiIdx := indexOf(order, "api-client")
	uiIdx := indexOf(order, "ui-framework")
	assert.Less(t, apiIdx, mainAppIdx)
	assert.Less(t, uiIdx, mainAppIdx)
}

// TestDepGraph_TopologicalSort_SelfCycle tests self-referential dependency detection.
func TestDepGraph_TopologicalSort_SelfCycle(t *testing.T) {
	g := NewDepGraph()

	// Package depends on itself
	g.AddDependency("broken", "broken", "^1.0.0")

	order, err := g.TopologicalSort()
	assert.Nil(t, order)
	require.Error(t, err)

	cycleErr, ok := err.(*CycleError)
	require.True(t, ok)
	assert.Contains(t, cycleErr.Packages, "broken")
}

// TestDepGraph_TopologicalSort_IndirectCycle tests indirect cycle detection.
func TestDepGraph_TopologicalSort_IndirectCycle(t *testing.T) {
	g := NewDepGraph()

	// Long cycle: a -> b -> c -> d -> e -> a
	g.AddDependency("a", "b", "^1.0.0")
	g.AddDependency("b", "c", "^1.0.0")
	g.AddDependency("c", "d", "^1.0.0")
	g.AddDependency("d", "e", "^1.0.0")
	g.AddDependency("e", "a", "^1.0.0")

	order, err := g.TopologicalSort()
	assert.Nil(t, order)
	require.Error(t, err)

	cycleErr, ok := err.(*CycleError)
	require.True(t, ok)
	assert.Len(t, cycleErr.Packages, 5)
}
