package resolver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dex-tools/dex/internal/config"
	"github.com/dex-tools/dex/internal/lockfile"
)

func TestNewResolver(t *testing.T) {
	project := &config.ProjectConfig{
		Project: config.ProjectBlock{
			Name:            "test",
			AgenticPlatform: "claude-code",
		},
	}
	lock := &lockfile.LockFile{
		Plugins: make(map[string]*lockfile.LockedPlugin),
	}

	r := NewResolver(project, lock)
	assert.NotNil(t, r)
	assert.Equal(t, project, r.project)
	assert.Equal(t, lock, r.lock)
}

func TestResolver_ResolveForUpdate_EmptyNames(t *testing.T) {
	project := &config.ProjectConfig{
		Project: config.ProjectBlock{
			Name:            "test",
			AgenticPlatform: "claude-code",
		},
		Plugins: []config.PluginBlock{
			{Name: "plugin-a", Source: "file:///tmp/a"},
		},
	}
	lock := &lockfile.LockFile{
		Plugins: map[string]*lockfile.LockedPlugin{
			"plugin-a": {Version: "1.0.0"},
			"plugin-b": {Version: "2.0.0"},
		},
	}

	r := NewResolver(project, lock)

	// When names is empty, it should update all locked packages
	// This will fail because registries aren't configured, but we can
	// verify the correct packages are being targeted
	_, err := r.ResolveForUpdate(nil)
	assert.Error(t, err) // Expected to fail without proper registry

	// The error should be about the plugin resolution
	assert.Contains(t, err.Error(), "plugin")
}

func TestResolution_InstallOrder(t *testing.T) {
	// Create a simple resolution to test the structure
	res := &Resolution{
		InstallOrder: []string{"core", "utils", "app"},
		Resolved: map[string]*ResolvedDep{
			"core":  {Name: "core", Version: "1.0.0"},
			"utils": {Name: "utils", Version: "1.0.0"},
			"app":   {Name: "app", Version: "1.0.0"},
		},
		Graph: NewDepGraph(),
	}

	assert.Len(t, res.InstallOrder, 3)
	assert.Equal(t, "core", res.InstallOrder[0])
	assert.Equal(t, "utils", res.InstallOrder[1])
	assert.Equal(t, "app", res.InstallOrder[2])

	assert.NotNil(t, res.Resolved["core"])
	assert.Equal(t, "1.0.0", res.Resolved["core"].Version)
}

func TestResolvedDep_Fields(t *testing.T) {
	dep := &ResolvedDep{
		Name:       "test-pkg",
		Version:    "1.2.3",
		Source:     "file:///tmp/test",
		Registry:   "local",
		Constraint: "^1.0.0",
	}

	assert.Equal(t, "test-pkg", dep.Name)
	assert.Equal(t, "1.2.3", dep.Version)
	assert.Equal(t, "file:///tmp/test", dep.Source)
	assert.Equal(t, "local", dep.Registry)
	assert.Equal(t, "^1.0.0", dep.Constraint)
}

func TestPluginSpec_Fields(t *testing.T) {
	spec := PluginSpec{
		Name:     "test-plugin",
		Version:  "^2.0.0",
		Source:   "git+https://github.com/test/plugin",
		Registry: "npm",
	}

	assert.Equal(t, "test-plugin", spec.Name)
	assert.Equal(t, "^2.0.0", spec.Version)
	assert.Equal(t, "git+https://github.com/test/plugin", spec.Source)
	assert.Equal(t, "npm", spec.Registry)
}

func TestResolver_detectConflicts_NoConflicts(t *testing.T) {
	project := &config.ProjectConfig{}
	lock := &lockfile.LockFile{Plugins: make(map[string]*lockfile.LockedPlugin)}
	r := NewResolver(project, lock)

	graph := NewDepGraph()
	graph.AddDependency("app", "lib", "^1.0.0")
	graph.GetNode("lib").Version = "1.5.0"

	resolved := map[string]*ResolvedDep{
		"app": {Name: "app", Version: "1.0.0"},
		"lib": {Name: "lib", Version: "1.5.0"},
	}

	conflicts := r.detectConflicts(graph, resolved)
	assert.Empty(t, conflicts, "should have no conflicts when versions match constraints")
}

func TestResolver_detectConflicts_WithConflict(t *testing.T) {
	project := &config.ProjectConfig{}
	lock := &lockfile.LockFile{Plugins: make(map[string]*lockfile.LockedPlugin)}
	r := NewResolver(project, lock)

	graph := NewDepGraph()
	graph.AddDependency("app", "lib", "^2.0.0") // app wants lib ^2.0.0
	graph.GetNode("lib").Version = "1.5.0"       // but lib is 1.5.0

	resolved := map[string]*ResolvedDep{
		"app": {Name: "app", Version: "1.0.0"},
		"lib": {Name: "lib", Version: "1.5.0"},
	}

	conflicts := r.detectConflicts(graph, resolved)
	require.Len(t, conflicts, 1, "should detect one conflict")
	assert.Equal(t, "lib", conflicts[0].Package)
	assert.Contains(t, conflicts[0].Required[0], "app requires lib@^2.0.0")
}

func TestResolver_loadPackageDependencies_FromLock(t *testing.T) {
	project := &config.ProjectConfig{}
	lock := &lockfile.LockFile{
		Plugins: map[string]*lockfile.LockedPlugin{
			"my-plugin": {
				Version: "1.0.0",
				Dependencies: map[string]string{
					"dep-a": "^1.0.0",
					"dep-b": ">=2.0.0",
				},
			},
		},
	}
	r := NewResolver(project, lock)

	dep := &ResolvedDep{Name: "my-plugin", Version: "1.0.0"}
	deps, err := r.loadPackageDependencies(dep)

	require.NoError(t, err)
	assert.Len(t, deps, 2)

	// Check that dependencies are loaded
	depNames := make(map[string]string)
	for _, d := range deps {
		depNames[d.Name] = d.Version
	}
	assert.Equal(t, "^1.0.0", depNames["dep-a"])
	assert.Equal(t, ">=2.0.0", depNames["dep-b"])
}
