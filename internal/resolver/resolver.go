package resolver

import (
	"fmt"

	"github.com/launchcg/dex/internal/config"
	"github.com/launchcg/dex/internal/lockfile"
	"github.com/launchcg/dex/internal/registry"
	"github.com/launchcg/dex/pkg/version"
)

// Resolver handles dependency resolution for package installation.
type Resolver struct {
	project    *config.ProjectConfig
	lock       *lockfile.LockFile
	registries map[string]registry.Registry
}

// Resolution contains the results of dependency resolution.
type Resolution struct {
	// InstallOrder is the topological order of packages to install (dependencies first)
	InstallOrder []string

	// Resolved maps package name to resolved version info
	Resolved map[string]*ResolvedDep

	// Graph is the dependency graph
	Graph *DepGraph
}

// ResolvedDep represents a resolved dependency.
type ResolvedDep struct {
	Name       string
	Version    string
	Source     string
	Registry   string
	Constraint string
}

// PluginSpec specifies a plugin to install (mirrors installer.PluginSpec for decoupling).
type PluginSpec struct {
	Name     string
	Version  string
	Source   string
	Registry string
}

// NewResolver creates a new resolver.
func NewResolver(project *config.ProjectConfig, lock *lockfile.LockFile) *Resolver {
	return &Resolver{
		project:    project,
		lock:       lock,
		registries: make(map[string]registry.Registry),
	}
}

// Resolve performs dependency resolution for the given specs.
// It builds a dependency graph, detects conflicts, and returns the install order.
func (r *Resolver) Resolve(specs []PluginSpec) (*Resolution, error) {
	graph := NewDepGraph()
	resolved := make(map[string]*ResolvedDep)

	// Process each spec and its transitive dependencies
	var queue []PluginSpec
	queue = append(queue, specs...)
	visited := make(map[string]bool)

	for len(queue) > 0 {
		spec := queue[0]
		queue = queue[1:]

		if visited[spec.Name] {
			continue
		}
		visited[spec.Name] = true

		// Resolve the package
		dep, err := r.resolvePackage(spec)
		if err != nil {
			return nil, err
		}

		resolved[spec.Name] = dep
		node := graph.AddNode(spec.Name)
		node.Version = dep.Version
		node.Constraint = dep.Constraint
		node.Registry = dep.Registry
		node.Source = dep.Source

		// Load package config to get dependencies
		pkgDeps, err := r.loadPackageDependencies(dep)
		if err != nil {
			// If we can't load dependencies, the package might not be fetched yet
			// This is fine for the first pass - we'll handle it during install
			continue
		}

		// Add dependencies to the graph and queue
		for _, pkgDep := range pkgDeps {
			graph.AddDependency(spec.Name, pkgDep.Name, pkgDep.Version)
			if !visited[pkgDep.Name] {
				queue = append(queue, PluginSpec{
					Name:     pkgDep.Name,
					Version:  pkgDep.Version,
					Source:   pkgDep.Source,
					Registry: pkgDep.Registry,
				})
			}
		}
	}

	// Check for version conflicts
	conflicts := r.detectConflicts(graph, resolved)
	if len(conflicts) > 0 {
		return nil, &ConflictError{Conflicts: conflicts}
	}

	// Compute topological sort
	order, err := graph.TopologicalSort()
	if err != nil {
		return nil, err
	}

	return &Resolution{
		InstallOrder: order,
		Resolved:     resolved,
		Graph:        graph,
	}, nil
}

// resolvePackage resolves a package to a specific version.
func (r *Resolver) resolvePackage(spec PluginSpec) (*ResolvedDep, error) {
	// Check if already locked and no explicit version specified
	if spec.Version == "" {
		if locked := r.lock.Get(spec.Name); locked != nil {
			return &ResolvedDep{
				Name:       spec.Name,
				Version:    locked.Version,
				Source:     locked.Resolved,
				Constraint: "locked",
			}, nil
		}
	}

	// Find in project plugins for registry/source info
	var pluginCfg *config.PluginBlock
	for i := range r.project.Plugins {
		if r.project.Plugins[i].Name == spec.Name {
			pluginCfg = &r.project.Plugins[i]
			break
		}
	}

	// Determine source/registry
	source := spec.Source
	registryName := spec.Registry
	if pluginCfg != nil {
		if source == "" {
			source = pluginCfg.Source
		}
		if registryName == "" {
			registryName = pluginCfg.Registry
		}
	}

	// Get the registry
	reg, err := r.getRegistry(source, registryName)
	if err != nil {
		return nil, &PackageNotFoundError{Package: spec.Name, Registry: registryName}
	}

	// Resolve version
	versionConstraint := spec.Version
	if versionConstraint == "" && pluginCfg != nil {
		versionConstraint = pluginCfg.Version
	}
	if versionConstraint == "" {
		versionConstraint = "latest"
	}

	resolved, err := reg.ResolvePackage(spec.Name, versionConstraint)
	if err != nil {
		return nil, &VersionNotFoundError{
			Package:    spec.Name,
			Constraint: versionConstraint,
		}
	}

	return &ResolvedDep{
		Name:       spec.Name,
		Version:    resolved.Version,
		Source:     resolved.URL,
		Registry:   registryName,
		Constraint: versionConstraint,
	}, nil
}

// getRegistry returns a registry for the given source or registry name.
func (r *Resolver) getRegistry(source, registryName string) (registry.Registry, error) {
	// Cache key
	key := source
	if key == "" {
		key = registryName
	}
	if key == "" {
		return nil, fmt.Errorf("no source or registry specified")
	}

	// Check cache
	if reg, ok := r.registries[key]; ok {
		return reg, nil
	}

	// Create registry
	var reg registry.Registry
	var err error

	if source != "" {
		reg, err = registry.NewRegistry(source, registry.ModePackage)
	} else {
		// Look up registry in project config
		for _, regCfg := range r.project.Registries {
			if regCfg.Name == registryName {
				if regCfg.Path != "" {
					reg, err = registry.NewRegistry("file:"+regCfg.Path, registry.ModeRegistry)
				} else if regCfg.URL != "" {
					reg, err = registry.NewRegistry(regCfg.URL, registry.ModeRegistry)
				}
				break
			}
		}
		if reg == nil && err == nil {
			return nil, fmt.Errorf("registry %q not found", registryName)
		}
	}

	if err != nil {
		return nil, err
	}

	r.registries[key] = reg
	return reg, nil
}

// loadPackageDependencies loads the dependencies declared in a package's config.
func (r *Resolver) loadPackageDependencies(dep *ResolvedDep) ([]config.DependencyBlock, error) {
	// Check if we have cached dependencies in the lockfile
	if locked := r.lock.Get(dep.Name); locked != nil && len(locked.Dependencies) > 0 {
		var deps []config.DependencyBlock
		for name, ver := range locked.Dependencies {
			deps = append(deps, config.DependencyBlock{
				Name:    name,
				Version: ver,
			})
		}
		return deps, nil
	}

	// Dependencies will be loaded during actual installation when the package is fetched
	// For now, return empty - this is handled by the installer
	return nil, nil
}

// detectConflicts checks for version conflicts in the dependency graph.
func (r *Resolver) detectConflicts(graph *DepGraph, resolved map[string]*ResolvedDep) []*Conflict {
	var conflicts []*Conflict

	// For each package, check if all dependents' constraints are satisfied
	for _, name := range graph.AllNodes() {
		node := graph.GetNode(name)
		if node == nil {
			continue
		}

		dep, ok := resolved[name]
		if !ok {
			continue
		}

		resolvedVersion, err := version.Parse(dep.Version)
		if err != nil {
			continue // Skip if version can't be parsed
		}

		var unsatisfied []string

		// Check each dependent's constraint
		for _, dependentName := range node.Dependents {
			dependentNode := graph.GetNode(dependentName)
			if dependentNode == nil {
				continue
			}

			constraint, ok := dependentNode.Dependencies[name]
			if !ok {
				continue
			}

			c, err := version.ParseConstraint(constraint)
			if err != nil {
				continue // Skip invalid constraints
			}

			if !c.Match(resolvedVersion) {
				unsatisfied = append(unsatisfied, fmt.Sprintf("%s requires %s@%s", dependentName, name, constraint))
			}
		}

		if len(unsatisfied) > 0 {
			conflicts = append(conflicts, &Conflict{
				Package:    name,
				Required:   unsatisfied,
				Resolution: fmt.Sprintf("Update dependencies to use compatible version constraints for %s", name),
			})
		}
	}

	return conflicts
}

// ResolveForUpdate resolves packages for an update operation.
// Unlike regular Resolve, this ignores locked versions and finds the latest matching versions.
func (r *Resolver) ResolveForUpdate(names []string) (*Resolution, error) {
	// If no names specified, update all locked packages
	if len(names) == 0 {
		for name := range r.lock.Plugins {
			names = append(names, name)
		}
	}

	// Build specs from names, using constraints from project config
	var specs []PluginSpec
	for _, name := range names {
		spec := PluginSpec{Name: name}

		// Get constraint from project config
		for _, p := range r.project.Plugins {
			if p.Name == name {
				spec.Version = p.Version
				spec.Source = p.Source
				spec.Registry = p.Registry
				break
			}
		}

		specs = append(specs, spec)
	}

	// Create a temporary lock to ignore current versions
	oldLock := r.lock
	r.lock = &lockfile.LockFile{Plugins: make(map[string]*lockfile.LockedPlugin)}
	defer func() { r.lock = oldLock }()

	return r.Resolve(specs)
}
