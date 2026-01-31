// Package installer handles the installation and uninstallation of dex plugins.
//
// The installer orchestrates the complete installation flow:
//   - Loading project configuration and lock file
//   - Resolving plugin versions from registries
//   - Fetching plugin packages
//   - Planning and executing installations via adapters
//   - Tracking installed files in the manifest
//   - Updating the lock file with resolved versions
package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/launchcg/dex/internal/adapter"
	"github.com/launchcg/dex/internal/config"
	"github.com/launchcg/dex/internal/errors"
	"github.com/launchcg/dex/internal/lockfile"
	"github.com/launchcg/dex/internal/manifest"
	"github.com/launchcg/dex/internal/registry"
	"github.com/launchcg/dex/internal/resolver"
	"github.com/launchcg/dex/pkg/version"
)

// Installer handles plugin installation for a project.
type Installer struct {
	projectRoot string
	project     *config.ProjectConfig
	adapter     adapter.Adapter
	manifest    *manifest.Manifest
	lock        *lockfile.LockFile
	force       bool // Overwrite non-managed files
	noLock      bool // Don't update lock file
	namespace   bool // Namespace resources with package name
}

// PluginSpec specifies a plugin to install.
type PluginSpec struct {
	// Name is the plugin name
	Name string

	// Version is the version constraint (empty = latest, or use lock file)
	Version string

	// Source is a direct source URL (file://, git+, etc.)
	Source string

	// Registry is the registry name to use
	Registry string

	// Config provides plugin-specific configuration values
	Config map[string]string
}

// InstalledPlugin contains information about an installed plugin.
type InstalledPlugin struct {
	// Name is the plugin name from package.hcl
	Name string

	// Version is the resolved version
	Version string

	// Source is the source URL used
	Source string
}

// NewInstaller creates a new installer for the given project root.
// It loads the project configuration, manifest, and lock file.
func NewInstaller(projectRoot string) (*Installer, error) {
	// Load project config
	project, err := config.LoadProject(projectRoot)
	if err != nil {
		return nil, errors.NewConfigError(
			filepath.Join(projectRoot, "dex.hcl"),
			0, 0,
			"failed to load project config",
			err,
		)
	}

	// Validate project config
	if err := project.Validate(); err != nil {
		return nil, errors.NewConfigError(
			filepath.Join(projectRoot, "dex.hcl"),
			0, 0,
			"invalid project config",
			err,
		)
	}

	// Get adapter for the platform
	adpt, err := adapter.Get(project.Project.AgenticPlatform)
	if err != nil {
		return nil, errors.NewConfigError(
			filepath.Join(projectRoot, "dex.hcl"),
			0, 0,
			fmt.Sprintf("unsupported platform: %s", project.Project.AgenticPlatform),
			err,
		)
	}

	// Load manifest
	mf, err := manifest.Load(projectRoot)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load manifest")
	}

	// Load lock file
	lf, err := lockfile.Load(projectRoot)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load lock file")
	}

	return &Installer{
		projectRoot: projectRoot,
		project:     project,
		adapter:     adpt,
		manifest:    mf,
		lock:        lf,
	}, nil
}

// WithForce sets the force flag to overwrite non-managed files.
func (i *Installer) WithForce(force bool) *Installer {
	i.force = force
	return i
}

// WithNoLock disables lock file updates.
func (i *Installer) WithNoLock(noLock bool) *Installer {
	i.noLock = noLock
	return i
}

// WithNamespace enables namespacing for installed resources.
func (i *Installer) WithNamespace(namespace bool) *Installer {
	i.namespace = namespace
	return i
}

// shouldNamespacePackage determines if a package should be namespaced
// based on the install flag, global config, or package-specific config.
func (i *Installer) shouldNamespacePackage(packageName string) bool {
	// Flag takes precedence
	if i.namespace {
		return true
	}

	// Check global namespace_all config
	if i.project.Project.NamespaceAll {
		return true
	}

	// Check package-specific namespace config
	for _, pkg := range i.project.Project.NamespacePackages {
		if pkg == packageName {
			return true
		}
	}

	return false
}

// Install installs the specified plugins.
// If specs is empty, installs all plugins from project config using lock file versions.
// Returns information about installed plugins for use with --save flag.
func (i *Installer) Install(specs []PluginSpec) ([]InstalledPlugin, error) {
	if len(specs) == 0 {
		return nil, i.InstallAll()
	}

	var installed []InstalledPlugin
	for _, spec := range specs {
		info, err := i.installPlugin(spec)
		if err != nil {
			return nil, err
		}
		installed = append(installed, *info)
	}

	// Save manifest and lock file
	if err := i.manifest.Save(); err != nil {
		return nil, errors.Wrap(err, "failed to save manifest")
	}

	if !i.noLock {
		// Set the agent platform in lock file
		i.lock.Agent = i.project.Project.AgenticPlatform
		if err := i.lock.Save(); err != nil {
			return nil, errors.Wrap(err, "failed to save lock file")
		}
	}

	return installed, nil
}

// InstallAll installs all plugins from the project config.
// Uses lock file versions if available, otherwise resolves latest.
func (i *Installer) InstallAll() error {
	if len(i.project.Plugins) == 0 && len(i.project.Resources) == 0 {
		fmt.Println("No plugins or resources defined in config")
		return nil
	}

	var specs []PluginSpec

	for _, plugin := range i.project.Plugins {
		spec := PluginSpec{
			Name:     plugin.Name,
			Version:  plugin.Version,
			Source:   plugin.Source,
			Registry: plugin.Registry,
			Config:   plugin.Config,
		}

		// If locked and no explicit version, use lock file version
		if locked := i.lock.Get(plugin.Name); locked != nil && plugin.Version == "" {
			spec.Version = locked.Version
		}

		specs = append(specs, spec)
	}

	// Call installPlugin directly to avoid recursion through Install
	for _, spec := range specs {
		if _, err := i.installPlugin(spec); err != nil {
			return err
		}
	}

	// Install resources defined directly in dex.hcl
	if err := i.installProjectResources(); err != nil {
		return err
	}

	// Save manifest and lock file
	if err := i.manifest.Save(); err != nil {
		return errors.Wrap(err, "failed to save manifest")
	}

	if !i.noLock {
		i.lock.Agent = i.project.Project.AgenticPlatform
		if err := i.lock.Save(); err != nil {
			return errors.Wrap(err, "failed to save lock file")
		}
	}

	return nil
}

// installPlugin installs a single plugin.
func (i *Installer) installPlugin(spec PluginSpec) (*InstalledPlugin, error) {
	// Resolve the registry to use
	reg, err := i.resolveRegistry(spec)
	if err != nil {
		return nil, errors.NewInstallError(spec.Name, "resolve", err)
	}

	// Resolve the version
	resolved, err := reg.ResolvePackage(spec.Name, spec.Version)
	if err != nil {
		return nil, errors.NewInstallError(spec.Name, "resolve", err)
	}

	// Use resolved package name (important when spec.Name is empty for direct sources)
	pluginName := resolved.Name
	if pluginName == "" {
		pluginName = spec.Name
	}

	// Create temp directory for fetching
	tempDir, err := os.MkdirTemp("", "dex-install-*")
	if err != nil {
		return nil, errors.NewInstallError(pluginName, "fetch", err)
	}
	defer os.RemoveAll(tempDir)

	// Fetch the package
	pluginDir, err := reg.FetchPackage(resolved, tempDir)
	if err != nil {
		return nil, errors.NewInstallError(pluginName, "fetch", err)
	}

	// Load and validate package config
	pkgConfig, err := config.LoadPackage(pluginDir)
	if err != nil {
		return nil, errors.NewInstallError(pluginName, "parse", err)
	}

	// Get the actual plugin name from the package
	pluginName = pkgConfig.Package.Name

	if err := pkgConfig.Validate(); err != nil {
		return nil, errors.NewInstallError(pluginName, "validate", err)
	}

	// Check platform compatibility
	if len(pkgConfig.Package.Platforms) > 0 {
		compatible := false
		for _, platform := range pkgConfig.Package.Platforms {
			if platform == i.project.Project.AgenticPlatform {
				compatible = true
				break
			}
		}
		if !compatible {
			return nil, errors.NewInstallError(pluginName, "validate",
				fmt.Errorf("plugin %q does not support platform %q",
					pluginName, i.project.Project.AgenticPlatform))
		}
	}

	// Install dependencies first
	if len(pkgConfig.Dependencies) > 0 {
		if err := i.installDependencies(pkgConfig.Dependencies, pluginName); err != nil {
			return nil, err
		}
	}

	// Resolve variable values
	vars, err := i.resolveVariables(pkgConfig, spec.Config)
	if err != nil {
		return nil, errors.NewInstallError(pluginName, "configure", err)
	}

	// Determine if namespacing should be enabled
	shouldNamespace := i.shouldNamespacePackage(pluginName)

	// Create install context
	ctx := &adapter.InstallContext{
		PackageName: pluginName,
		Namespace:   shouldNamespace,
	}

	// Create executor
	executor := NewExecutor(i.projectRoot, i.manifest, i.force)

	// Plan and execute installation for each resource
	// Filter resources to only include those matching the target platform
	targetPlatform := i.project.Project.AgenticPlatform
	var allPlans []*adapter.Plan
	for _, res := range pkgConfig.Resources {
		// Skip resources that don't match the target platform
		// Universal resources (like unified MCP servers) work on all platforms
		if res.Platform() != targetPlatform && res.Platform() != "universal" {
			continue
		}
		plan, err := i.adapter.PlanInstallation(res, pkgConfig, pluginDir, i.projectRoot, ctx)
		if err != nil {
			return nil, errors.NewInstallError(pluginName, "plan", err)
		}
		allPlans = append(allPlans, plan)
	}

	// Merge all plans
	mergedPlan := adapter.MergePlans(allPlans...)

	// Execute the merged plan
	if err := executor.Execute(mergedPlan, vars); err != nil {
		return nil, errors.NewInstallError(pluginName, "install", err)
	}

	// Update lock file with dependencies
	if !i.noLock {
		deps := make(map[string]string)
		for _, dep := range pkgConfig.Dependencies {
			deps[dep.Name] = dep.Version
		}
		i.lock.Set(pluginName, &lockfile.LockedPlugin{
			Version:      resolved.Version,
			Resolved:     resolved.URL,
			Integrity:    resolved.Integrity,
			Dependencies: deps,
		})
	}

	fmt.Printf("  ✓ Installed %s@%s\n", pluginName, resolved.Version)

	// Return installed plugin info
	return &InstalledPlugin{
		Name:    pluginName,
		Version: resolved.Version,
		Source:  spec.Source,
	}, nil
}

// installDependencies installs the dependencies of a package.
func (i *Installer) installDependencies(deps []config.DependencyBlock, parentName string) error {
	for _, dep := range deps {
		// Skip if already installed with a compatible version
		if locked := i.lock.Get(dep.Name); locked != nil {
			// Check if locked version satisfies the constraint
			lockedVer, err := version.Parse(locked.Version)
			if err == nil {
				constraint, err := version.ParseConstraint(dep.Version)
				if err == nil && constraint.Match(lockedVer) {
					// Already installed with compatible version
					continue
				}
			}
		}

		// Determine registry for this dependency
		registryName := dep.Registry
		source := dep.Source

		// If not specified, try to find from project config or use parent's registry
		if registryName == "" && source == "" {
			// Check if this dependency is declared in project plugins
			for _, p := range i.project.Plugins {
				if p.Name == dep.Name {
					registryName = p.Registry
					source = p.Source
					break
				}
			}

			// If still not found, use the first available registry
			if registryName == "" && source == "" && len(i.project.Registries) > 0 {
				registryName = i.project.Registries[0].Name
			}
		}

		fmt.Printf("  → Installing dependency %s@%s for %s\n", dep.Name, dep.Version, parentName)

		// Install the dependency
		_, err := i.installPlugin(PluginSpec{
			Name:     dep.Name,
			Version:  dep.Version,
			Source:   source,
			Registry: registryName,
		})
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to install dependency %s", dep.Name))
		}
	}
	return nil
}

// installProjectResources installs resources defined directly in dex.hcl.
func (i *Installer) installProjectResources() error {
	// Skip if no resources are defined in the project config
	if len(i.project.Resources) == 0 {
		return nil
	}

	// Create a synthetic package config for project-level resources
	// This is used by adapters that expect package metadata
	projectPkg := &config.PackageConfig{
		Package: config.PackageBlock{
			Name:    "project",
			Version: "0.0.0",
		},
	}
	if i.project.Project.Name != "" {
		projectPkg.Package.Name = i.project.Project.Name
	}

	// Determine if namespacing should be enabled for project resources
	shouldNamespace := i.shouldNamespacePackage(projectPkg.Package.Name)

	// Create install context for project resources
	ctx := &adapter.InstallContext{
		PackageName: projectPkg.Package.Name,
		Namespace:   shouldNamespace,
	}

	// Create executor
	executor := NewExecutor(i.projectRoot, i.manifest, i.force)

	// Filter and plan resources
	targetPlatform := i.project.Project.AgenticPlatform
	var allPlans []*adapter.Plan
	for _, res := range i.project.Resources {
		// Skip resources that don't match the target platform
		// Universal resources (like unified MCP servers) work on all platforms
		if res.Platform() != targetPlatform && res.Platform() != "universal" {
			continue
		}
		// Use projectRoot as the source directory for file references
		plan, err := i.adapter.PlanInstallation(res, projectPkg, i.projectRoot, i.projectRoot, ctx)
		if err != nil {
			return errors.Wrap(err, "failed to plan project resource installation")
		}
		allPlans = append(allPlans, plan)
	}

	// Skip if no resources match the platform
	if len(allPlans) == 0 {
		return nil
	}

	// Merge all plans
	mergedPlan := adapter.MergePlans(allPlans...)

	// Execute the merged plan with resolved project variables
	if err := executor.Execute(mergedPlan, i.project.ResolvedVars); err != nil {
		return errors.Wrap(err, "failed to install project resources")
	}

	fmt.Printf("  ✓ Installed project resources\n")
	return nil
}

// resolveRegistry determines which registry to use for fetching a plugin.
func (i *Installer) resolveRegistry(spec PluginSpec) (registry.Registry, error) {
	// If direct source is specified, use it
	if spec.Source != "" {
		return registry.NewRegistry(spec.Source, registry.ModePackage)
	}

	// If registry name is specified, look it up in project config
	if spec.Registry != "" {
		for _, reg := range i.project.Registries {
			if reg.Name == spec.Registry {
				if reg.Path != "" {
					return registry.NewRegistry("file:"+reg.Path, registry.ModeRegistry)
				}
				if reg.URL != "" {
					return registry.NewRegistry(reg.URL, registry.ModeRegistry)
				}
			}
		}
		return nil, fmt.Errorf("registry %q not found in project config", spec.Registry)
	}

	// Try to find the plugin in project config
	for _, plugin := range i.project.Plugins {
		if plugin.Name == spec.Name {
			if plugin.Source != "" {
				return registry.NewRegistry(plugin.Source, registry.ModePackage)
			}
			if plugin.Registry != "" {
				// Recursively resolve with registry name
				return i.resolveRegistry(PluginSpec{
					Name:     spec.Name,
					Registry: plugin.Registry,
				})
			}
		}
	}

	return nil, fmt.Errorf("no source or registry specified for plugin %q", spec.Name)
}

// resolveVariables resolves variable values from environment and config.
func (i *Installer) resolveVariables(pkg *config.PackageConfig, pluginConfig map[string]string) (map[string]string, error) {
	vars := make(map[string]string)

	for _, v := range pkg.Variables {
		value, err := v.ResolveValue(pluginConfig)
		if err != nil {
			return nil, err
		}
		vars[v.Name] = value
	}

	return vars, nil
}

// Uninstall removes installed plugins.
// If removeFromConfig is true, also removes the plugin from dex.hcl.
func (i *Installer) Uninstall(names []string, removeFromConfig bool) error {
	for _, name := range names {
		if err := i.uninstallPlugin(name); err != nil {
			return err
		}
	}

	// Save manifest
	if err := i.manifest.Save(); err != nil {
		return errors.Wrap(err, "failed to save manifest")
	}

	// Update lock file
	if !i.noLock {
		for _, name := range names {
			i.lock.Remove(name)
		}
		if err := i.lock.Save(); err != nil {
			return errors.Wrap(err, "failed to save lock file")
		}
	}

	return nil
}

// uninstallPlugin removes a single plugin.
func (i *Installer) uninstallPlugin(name string) error {
	// Get files to remove from manifest
	result := i.manifest.Untrack(name)

	// Delete tracked files
	for _, file := range result.Files {
		path := filepath.Join(i.projectRoot, file)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return errors.NewInstallError(name, "uninstall",
				fmt.Errorf("failed to remove file %s: %w", file, err))
		}
	}

	// Delete empty directories (in reverse order to handle nested dirs)
	for j := len(result.Directories) - 1; j >= 0; j-- {
		dir := result.Directories[j]
		path := filepath.Join(i.projectRoot, dir)
		// Only remove if empty
		entries, err := os.ReadDir(path)
		if err == nil && len(entries) == 0 {
			os.Remove(path)
		}
	}

	// Remove MCP servers from .mcp.json
	if len(result.MCPServers) > 0 {
		mcpPath := filepath.Join(i.projectRoot, ".mcp.json")
		mcpConfig, err := ReadJSONFile(mcpPath)
		if err == nil {
			mcpConfig = RemoveMCPServers(mcpConfig, result.MCPServers)
			if err := WriteJSONFile(mcpPath, mcpConfig); err != nil {
				return errors.NewInstallError(name, "uninstall",
					fmt.Errorf("failed to update .mcp.json: %w", err))
			}
		}
	}

	// Remove settings values from .claude/settings.json (only values not used by other plugins)
	if len(result.SettingsValues) > 0 {
		settingsPath := filepath.Join(i.projectRoot, ".claude", "settings.json")
		settingsConfig, err := ReadJSONFile(settingsPath)
		if err == nil {
			// For each key (allow, ask, deny, etc.), remove values not used by other plugins
			for key, values := range result.SettingsValues {
				if existing, ok := settingsConfig[key].([]any); ok {
					filtered := make([]any, 0)
					for _, v := range existing {
						vStr, ok := v.(string)
						if !ok {
							filtered = append(filtered, v)
							continue
						}
						// Check if this value was contributed by this plugin
						wasContributed := false
						for _, contributed := range values {
							if contributed == vStr {
								wasContributed = true
								break
							}
						}
						// Keep the value if it wasn't contributed by this plugin
						// OR if another plugin also uses it
						if !wasContributed || i.manifest.IsSettingsValueUsedByOthers(name, key, vStr) {
							filtered = append(filtered, v)
						}
					}
					// Remove the key entirely if no values remain
					if len(filtered) == 0 {
						delete(settingsConfig, key)
					} else {
						settingsConfig[key] = filtered
					}
				}
			}
			if err := WriteJSONFile(settingsPath, settingsConfig); err != nil {
				return errors.NewInstallError(name, "uninstall",
					fmt.Errorf("failed to update settings.json: %w", err))
			}
		}
	}

	// Remove agent content from CLAUDE.md
	if result.HasAgentContent {
		agentPath := filepath.Join(i.projectRoot, "CLAUDE.md")
		content, err := os.ReadFile(agentPath)
		if err == nil {
			newContent := RemoveAgentContent(string(content), name)
			if err := os.WriteFile(agentPath, []byte(newContent), 0644); err != nil {
				return errors.NewInstallError(name, "uninstall",
					fmt.Errorf("failed to update CLAUDE.md: %w", err))
			}
		}
	}

	return nil
}

// FindDependents returns packages that depend on the given package.
func (i *Installer) FindDependents(name string) []string {
	var dependents []string
	for pluginName, locked := range i.lock.Plugins {
		if pluginName == name {
			continue
		}
		for dep := range locked.Dependencies {
			if dep == name {
				dependents = append(dependents, pluginName)
				break
			}
		}
	}
	sort.Strings(dependents)
	return dependents
}

// FindOrphans returns dependencies that are no longer needed by any package.
// The excluding parameter specifies packages that should be considered as already removed.
func (i *Installer) FindOrphans(excluding []string) []string {
	// Build set of excluded packages
	excludeSet := make(map[string]bool)
	for _, name := range excluding {
		excludeSet[name] = true
	}

	// Build set of explicitly declared plugins (from project config)
	explicit := make(map[string]bool)
	for _, p := range i.project.Plugins {
		explicit[p.Name] = true
	}

	// Build set of all needed dependencies
	needed := make(map[string]bool)
	for pluginName, locked := range i.lock.Plugins {
		if excludeSet[pluginName] {
			continue
		}
		for dep := range locked.Dependencies {
			needed[dep] = true
		}
	}

	// Find installed packages that are not needed and not explicit
	var orphans []string
	for pluginName := range i.lock.Plugins {
		if excludeSet[pluginName] {
			continue
		}
		if !needed[pluginName] && !explicit[pluginName] {
			orphans = append(orphans, pluginName)
		}
	}
	sort.Strings(orphans)
	return orphans
}

// UpdateResult contains information about an update operation.
type UpdateResult struct {
	// Name is the plugin name
	Name string
	// OldVersion is the previously installed version
	OldVersion string
	// NewVersion is the version after update
	NewVersion string
	// Skipped indicates whether the update was skipped
	Skipped bool
	// Reason explains why the update was skipped or performed
	Reason string
}

// Update updates specified plugins to newer versions.
// If names is empty, updates all plugins.
// If dryRun is true, only reports what would be updated without making changes.
func (i *Installer) Update(names []string, dryRun bool) ([]UpdateResult, error) {
	// If no names specified, update all locked packages that are in project config
	if len(names) == 0 {
		for _, p := range i.project.Plugins {
			if i.lock.Has(p.Name) {
				names = append(names, p.Name)
			}
		}
	}

	var results []UpdateResult

	for _, name := range names {
		result, err := i.updatePlugin(name, dryRun)
		if err != nil {
			return nil, err
		}
		results = append(results, *result)
	}

	// Save files if not dry run
	if !dryRun {
		if err := i.manifest.Save(); err != nil {
			return nil, errors.Wrap(err, "failed to save manifest")
		}
		if !i.noLock {
			i.lock.Agent = i.project.Project.AgenticPlatform
			if err := i.lock.Save(); err != nil {
				return nil, errors.Wrap(err, "failed to save lock file")
			}
		}
	}

	return results, nil
}

// updatePlugin updates a single plugin.
func (i *Installer) updatePlugin(name string, dryRun bool) (*UpdateResult, error) {
	result := &UpdateResult{Name: name}

	// Get current locked version
	locked := i.lock.Get(name)
	if locked == nil {
		result.Skipped = true
		result.Reason = "not installed"
		return result, nil
	}
	result.OldVersion = locked.Version

	// Get constraint from project config
	var constraint string
	var spec PluginSpec
	for _, p := range i.project.Plugins {
		if p.Name == name {
			constraint = p.Version
			spec = PluginSpec{
				Name:     p.Name,
				Version:  p.Version,
				Source:   p.Source,
				Registry: p.Registry,
				Config:   p.Config,
			}
			break
		}
	}

	if constraint == "" {
		constraint = "latest"
	}

	// Resolve the registry
	reg, err := i.resolveRegistry(spec)
	if err != nil {
		return nil, errors.NewInstallError(name, "resolve", err)
	}

	// Get all available versions
	pkgInfo, err := reg.GetPackageInfo(name)
	if err != nil {
		return nil, errors.NewInstallError(name, "resolve", err)
	}

	// Parse constraint and find best matching version
	c, err := version.ParseConstraint(constraint)
	if err != nil {
		return nil, errors.NewInstallError(name, "resolve",
			fmt.Errorf("invalid version constraint %q: %w", constraint, err))
	}

	// Parse available versions
	var versions []*version.Version
	for _, v := range pkgInfo.Versions {
		if parsed, err := version.Parse(v); err == nil {
			versions = append(versions, parsed)
		}
	}

	// Find best matching version
	best := c.FindBest(versions)
	if best == nil {
		result.Skipped = true
		result.Reason = fmt.Sprintf("no version matches constraint %q", constraint)
		return result, nil
	}

	// Compare with current version
	current, err := version.Parse(locked.Version)
	if err != nil {
		current = nil
	}

	if current != nil && !best.GreaterThan(current) {
		result.Skipped = true
		result.NewVersion = locked.Version
		result.Reason = "already at latest compatible version"
		return result, nil
	}

	result.NewVersion = best.String()

	if dryRun {
		result.Reason = fmt.Sprintf("would update from %s to %s", locked.Version, best.String())
		return result, nil
	}

	// Perform the update by reinstalling with the new version
	spec.Version = best.String()
	_, err = i.installPlugin(spec)
	if err != nil {
		return nil, err
	}

	result.Reason = fmt.Sprintf("updated from %s to %s", locked.Version, best.String())
	return result, nil
}

// GetResolver returns a new resolver instance for dependency operations.
func (i *Installer) GetResolver() *resolver.Resolver {
	return resolver.NewResolver(i.project, i.lock)
}
