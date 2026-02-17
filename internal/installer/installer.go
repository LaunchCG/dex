// Package installer handles the installation and uninstallation of dex plugins.
//
// The installer orchestrates the complete installation flow:
//   - Loading project configuration and lock file
//   - Resolving plugin versions from registries
//   - Fetching plugin packages
//   - Planning and executing installations via adapters
//   - Tracking installed files in the manifest
//   - Updating the lock file with resolved versions
//
// All shared files (CLAUDE.md, .mcp.json, settings.json) are regenerated from
// scratch after all plugins are processed, using hash comparison to avoid
// unnecessary writes.
package installer

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/launchcg/dex/internal/adapter"
	"github.com/launchcg/dex/internal/config"
	"github.com/launchcg/dex/internal/errors"
	"github.com/launchcg/dex/internal/lockfile"
	"github.com/launchcg/dex/internal/manifest"
	"github.com/launchcg/dex/internal/registry"
	"github.com/launchcg/dex/internal/resolver"
	"github.com/launchcg/dex/pkg/version"
)

// pluginContribution holds a plugin's shared file contributions for deferred generation.
type pluginContribution struct {
	pluginName      string
	mcpEntries      map[string]any
	mcpPath         string
	mcpKey          string
	settingsEntries map[string]any
	settingsPath    string
	agentContent    string
	agentFilePath   string
}

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

	// contributions collects shared file contributions from all plugins
	// for deferred generation by generateSharedFiles().
	contributions []pluginContribution

	// removedServers tracks MCP server names from uninstalled plugins,
	// so generateMCPConfig knows to remove them from the config file.
	removedServers map[string]bool
	// removedSettings tracks settings values from uninstalled plugins.
	removedSettings map[string]map[string]bool
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

	// Registry is the registry name used (for registry-based installs)
	Registry string
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

	i.contributions = nil

	var installed []InstalledPlugin
	for _, spec := range specs {
		info, err := i.installPlugin(spec)
		if err != nil {
			return nil, err
		}
		installed = append(installed, *info)
	}

	// Collect contributions from remaining locked plugins
	if err := i.collectAllContributions(); err != nil {
		return nil, err
	}

	// Install project-level resources (dedicated files only)
	if err := i.installProjectResources(); err != nil {
		return nil, err
	}

	// Generate all shared files from scratch
	if err := i.generateSharedFiles(); err != nil {
		return nil, err
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
	if len(i.project.Plugins) == 0 && len(i.project.Resources) == 0 && i.project.Project.AgentInstructions == "" {
		fmt.Println("No plugins, resources, or agent instructions defined in config")
		return nil
	}

	i.contributions = nil

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

	// Install project-level resources (dedicated files only)
	if err := i.installProjectResources(); err != nil {
		return err
	}

	// Generate all shared files from scratch
	if err := i.generateSharedFiles(); err != nil {
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
// It creates directories and writes dedicated files via the executor,
// and collects shared file contributions for later generation.
func (i *Installer) installPlugin(spec PluginSpec) (*InstalledPlugin, error) {
	// Resolve the registry to use
	reg, err := i.resolveRegistry(&spec)
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

	// Execute the merged plan (creates dirs, writes dedicated files, tracks in manifest)
	if err := executor.Execute(mergedPlan, vars); err != nil {
		return nil, errors.NewInstallError(pluginName, "install", err)
	}

	// Collect shared file contributions for deferred generation
	i.contributions = append(i.contributions, pluginContribution{
		pluginName:      pluginName,
		mcpEntries:      mergedPlan.MCPEntries,
		mcpPath:         mergedPlan.MCPPath,
		mcpKey:          mergedPlan.MCPKey,
		settingsEntries: mergedPlan.SettingsEntries,
		settingsPath:    mergedPlan.SettingsPath,
		agentContent:    mergedPlan.AgentFileContent,
		agentFilePath:   mergedPlan.AgentFilePath,
	})

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
		Name:     pluginName,
		Version:  resolved.Version,
		Source:   spec.Source,
		Registry: spec.Registry,
	}, nil
}

// collectAllContributions reinstalls all locked plugins that haven't already
// been installed in the current session to collect their shared file contributions.
// This ensures generateSharedFiles() has complete data from all plugins.
func (i *Installer) collectAllContributions() error {
	// Build set of plugins already collected
	collected := make(map[string]bool)
	for _, c := range i.contributions {
		collected[c.pluginName] = true
	}

	// Re-install remaining locked plugins to collect their contributions
	for _, plugin := range i.project.Plugins {
		if collected[plugin.Name] {
			continue
		}
		locked := i.lock.Get(plugin.Name)
		if locked == nil {
			continue
		}
		spec := PluginSpec{
			Name:     plugin.Name,
			Version:  locked.Version,
			Source:   plugin.Source,
			Registry: plugin.Registry,
			Config:   plugin.Config,
		}
		if _, err := i.installPlugin(spec); err != nil {
			return err
		}
	}

	return nil
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
// This creates directories and writes dedicated files. Shared file contributions
// are collected for deferred generation.
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

	// Collect shared file contributions
	i.contributions = append(i.contributions, pluginContribution{
		pluginName:      mergedPlan.PluginName,
		mcpEntries:      mergedPlan.MCPEntries,
		mcpPath:         mergedPlan.MCPPath,
		mcpKey:          mergedPlan.MCPKey,
		settingsEntries: mergedPlan.SettingsEntries,
		settingsPath:    mergedPlan.SettingsPath,
		agentContent:    mergedPlan.AgentFileContent,
		agentFilePath:   mergedPlan.AgentFilePath,
	})

	fmt.Printf("  ✓ Installed project resources\n")
	return nil
}

// generateSharedFiles regenerates all shared files (agent file, MCP config, settings)
// from scratch using collected plugin contributions. Uses hash comparison to avoid
// unnecessary writes. Non-dex entries in MCP and settings files are preserved.
func (i *Installer) generateSharedFiles() error {
	// Determine default paths from platform
	agentPath := "CLAUDE.md"
	mcpPath := ".mcp.json"
	mcpKey := "mcpServers"
	settingsPath := filepath.Join(".claude", "settings.json")

	switch i.project.Project.AgenticPlatform {
	case "cursor":
		agentPath = "AGENTS.md"
	case "github-copilot":
		agentPath = filepath.Join(".github", "copilot-instructions.md")
	}

	// Override from contributions if specified
	for _, c := range i.contributions {
		if c.agentFilePath != "" {
			agentPath = c.agentFilePath
			break
		}
	}
	for _, c := range i.contributions {
		if c.mcpPath != "" {
			mcpPath = c.mcpPath
			break
		}
	}
	for _, c := range i.contributions {
		if c.mcpKey != "" {
			mcpKey = c.mcpKey
			break
		}
	}
	for _, c := range i.contributions {
		if c.settingsPath != "" {
			settingsPath = c.settingsPath
			break
		}
	}

	// 1. Generate agent file (project instructions + plugin content, no markers)
	if err := i.generateAgentFile(agentPath); err != nil {
		return err
	}

	// 2. Generate MCP config from scratch (preserving non-dex entries)
	if err := i.generateMCPConfig(mcpPath, mcpKey); err != nil {
		return err
	}

	// 3. Generate settings config from scratch (preserving non-dex entries)
	if err := i.generateSettingsConfig(settingsPath); err != nil {
		return err
	}

	return nil
}

// generateAgentFile regenerates the agent file (e.g., CLAUDE.md) from scratch.
// Content is: project instructions + all plugin agent content (no markers).
func (i *Installer) generateAgentFile(agentPath string) error {
	var content strings.Builder

	// Project instructions first
	if i.project.Project.AgentInstructions != "" {
		content.WriteString(strings.TrimSpace(i.project.Project.AgentInstructions))
	}

	// Plugin contributions
	for _, c := range i.contributions {
		if c.agentContent != "" {
			if content.Len() > 0 {
				content.WriteString("\n\n")
			}
			content.WriteString(c.agentContent)
		}
	}

	fullPath := filepath.Join(i.projectRoot, agentPath)

	if content.Len() > 0 {
		newContent := []byte(content.String())
		if contentChanged(fullPath, newContent) {
			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				return fmt.Errorf("creating directory for agent file: %w", err)
			}
			if err := os.WriteFile(fullPath, newContent, 0644); err != nil {
				return fmt.Errorf("writing agent file: %w", err)
			}
		}
	} else {
		// No content — delete the file if it exists
		if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing empty agent file: %w", err)
		}
	}

	// Track project agent content in manifest
	if i.project.Project.AgentInstructions != "" {
		i.manifest.TrackAgentContent("__project__")
		i.manifest.TrackMergedFile("__project__", agentPath)
	} else {
		plugin := i.manifest.GetPlugin("__project__")
		if plugin != nil {
			plugin.HasAgentContent = false
			plugin.MergedFiles = removeString(plugin.MergedFiles, agentPath)
		}
	}

	return nil
}

// generateMCPConfig regenerates the MCP config file from scratch.
// Non-dex entries are preserved by reading the existing file, removing
// all dex-managed servers, and adding back current contributions.
func (i *Installer) generateMCPConfig(mcpPath, mcpKey string) error {
	fullPath := filepath.Join(i.projectRoot, mcpPath)

	// Build the set of servers that current contributions will provide
	contributedServers := make(map[string]bool)
	for _, c := range i.contributions {
		for name := range c.mcpEntries {
			contributedServers[name] = true
		}
	}

	// Read existing config (preserves non-dex entries)
	existing, err := ReadJSONFile(fullPath)
	if err != nil {
		return fmt.Errorf("reading MCP config: %w", err)
	}

	// Remove all previously dex-managed servers from existing config.
	// A server is considered dex-managed if it was tracked in the manifest,
	// being contributed by current plugins, or was recently uninstalled.
	dexServers := make(map[string]bool)
	for _, plugin := range i.manifest.Plugins {
		for _, server := range plugin.MCPServers {
			dexServers[server] = true
		}
	}
	for name := range contributedServers {
		dexServers[name] = true
	}
	for name := range i.removedServers {
		dexServers[name] = true
	}
	if servers, ok := existing[mcpKey].(map[string]any); ok {
		for name := range servers {
			if dexServers[name] {
				delete(servers, name)
			}
		}
		existing[mcpKey] = servers
	}

	// Add all current contributions
	for _, c := range i.contributions {
		if len(c.mcpEntries) > 0 {
			existing = MergeMCPServersWithKey(existing, c.mcpEntries, mcpKey)
		}
	}

	// Check if there are any servers or other content
	hasContent := len(existing) > 0
	if hasContent {
		// Check if the only key is an empty servers map
		if len(existing) == 1 {
			if servers, ok := existing[mcpKey].(map[string]any); ok && len(servers) == 0 {
				hasContent = false
			}
		}
	}

	if hasContent {
		content, marshalErr := json.MarshalIndent(existing, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("marshaling MCP config: %w", marshalErr)
		}
		content = append(content, '\n')
		if contentChanged(fullPath, content) {
			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				return fmt.Errorf("creating directory for MCP config: %w", err)
			}
			if err := os.WriteFile(fullPath, content, 0644); err != nil {
				return fmt.Errorf("writing MCP config: %w", err)
			}
		}
	} else {
		// No content — delete the file if it exists
		if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing empty MCP config: %w", err)
		}
	}

	return nil
}

// generateSettingsConfig regenerates the settings config file from scratch.
// Non-dex entries are preserved by reading the existing file, removing
// all dex-managed settings values, and adding back current contributions.
func (i *Installer) generateSettingsConfig(settingsPath string) error {
	fullPath := filepath.Join(i.projectRoot, settingsPath)

	// Collect all dex-managed settings values from manifest and recently removed plugins
	dexValues := make(map[string]map[string]bool)
	for _, plugin := range i.manifest.Plugins {
		for key, vals := range plugin.SettingsValues {
			if dexValues[key] == nil {
				dexValues[key] = make(map[string]bool)
			}
			for _, v := range vals {
				dexValues[key][v] = true
			}
		}
	}
	for key, vals := range i.removedSettings {
		if dexValues[key] == nil {
			dexValues[key] = make(map[string]bool)
		}
		for v := range vals {
			dexValues[key][v] = true
		}
	}

	// Read existing config (preserves non-dex entries)
	existing, err := ReadJSONFile(fullPath)
	if err != nil {
		return fmt.Errorf("reading settings config: %w", err)
	}

	// Remove all dex-managed values from existing config
	for key, managed := range dexValues {
		if arr, ok := existing[key].([]any); ok {
			filtered := make([]any, 0, len(arr))
			for _, v := range arr {
				if s, ok := v.(string); ok {
					if !managed[s] {
						filtered = append(filtered, v)
					}
				} else {
					filtered = append(filtered, v)
				}
			}
			if len(filtered) == 0 {
				delete(existing, key)
			} else {
				existing[key] = filtered
			}
		}
	}

	// Add all current contributions
	for _, c := range i.contributions {
		if len(c.settingsEntries) > 0 {
			existing = MergeJSON(existing, c.settingsEntries)
		}
	}

	// Write if there's content, delete if empty
	if len(existing) > 0 {
		content, marshalErr := json.MarshalIndent(existing, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("marshaling settings config: %w", marshalErr)
		}
		content = append(content, '\n')
		if contentChanged(fullPath, content) {
			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				return fmt.Errorf("creating directory for settings config: %w", err)
			}
			if err := os.WriteFile(fullPath, content, 0644); err != nil {
				return fmt.Errorf("writing settings config: %w", err)
			}
		}
	} else {
		if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing empty settings config: %w", err)
		}
	}

	return nil
}

// resolveRegistry determines which registry to use for fetching a plugin.
func (i *Installer) resolveRegistry(spec *PluginSpec) (registry.Registry, error) {
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
				spec.Registry = plugin.Registry
				return i.resolveRegistry(spec)
			}
		}
	}

	// Auto-search configured registries
	if spec.Name != "" && len(i.project.Registries) > 0 {
		registryName, reg, err := i.searchRegistries(spec.Name)
		if err != nil {
			return nil, err
		}
		spec.Registry = registryName
		return reg, nil
	}

	return nil, fmt.Errorf("no source or registry specified for plugin %q", spec.Name)
}

// searchRegistries searches all configured registries for a plugin by name.
// Returns the registry name and registry instance if found in exactly one registry.
// Returns an error if found in multiple registries (ambiguous) or not found in any.
func (i *Installer) searchRegistries(pluginName string) (string, registry.Registry, error) {
	type found struct {
		name string
		reg  registry.Registry
	}

	var matches []found

	for _, regConfig := range i.project.Registries {
		var regSource string
		if regConfig.Path != "" {
			regSource = "file:" + regConfig.Path
		} else if regConfig.URL != "" {
			regSource = regConfig.URL
		} else {
			continue
		}

		reg, err := registry.NewRegistry(regSource, registry.ModeRegistry)
		if err != nil {
			continue
		}

		_, err = reg.GetPackageInfo(pluginName)
		if err != nil {
			var notFound *errors.NotFoundError
			if stderrors.As(err, &notFound) {
				continue
			}
			// Real error (network, permission, etc.) - skip this registry
			continue
		}

		matches = append(matches, found{name: regConfig.Name, reg: reg})
	}

	switch len(matches) {
	case 0:
		return "", nil, fmt.Errorf("plugin %q not found in any configured registry", pluginName)
	case 1:
		return matches[0].name, matches[0].reg, nil
	default:
		var names []string
		for _, m := range matches {
			names = append(names, m.name)
		}
		return "", nil, fmt.Errorf("plugin %q found in multiple registries: %s (use --registry to specify which one)", pluginName, strings.Join(names, ", "))
	}
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
	// Track servers and settings from plugins being uninstalled so
	// generateSharedFiles knows to remove them from config files.
	i.removedServers = make(map[string]bool)
	i.removedSettings = make(map[string]map[string]bool)
	for _, name := range names {
		if pm := i.manifest.GetPlugin(name); pm != nil {
			for _, server := range pm.MCPServers {
				i.removedServers[server] = true
			}
			for key, vals := range pm.SettingsValues {
				if i.removedSettings[key] == nil {
					i.removedSettings[key] = make(map[string]bool)
				}
				for _, v := range vals {
					i.removedSettings[key][v] = true
				}
			}
		}
	}

	for _, name := range names {
		if err := i.uninstallPlugin(name); err != nil {
			return err
		}
		// Remove from lock immediately so collectAllContributions
		// won't re-install the uninstalled plugin.
		if !i.noLock {
			i.lock.Remove(name)
		}
	}

	// Regenerate shared files from remaining plugins
	i.contributions = nil
	if err := i.collectAllContributions(); err != nil {
		return err
	}
	if err := i.installProjectResources(); err != nil {
		return err
	}
	if err := i.generateSharedFiles(); err != nil {
		return err
	}

	// Save manifest
	if err := i.manifest.Save(); err != nil {
		return errors.Wrap(err, "failed to save manifest")
	}

	// Save lock file
	if !i.noLock {
		if err := i.lock.Save(); err != nil {
			return errors.Wrap(err, "failed to save lock file")
		}
	}

	return nil
}

// uninstallPlugin removes a single plugin's dedicated files and manifest entries.
// Shared files (MCP, settings, agent content) are NOT cleaned up here;
// they are regenerated from scratch by generateSharedFiles().
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

// SyncAction describes what action was taken for a plugin during sync.
type SyncAction string

const (
	// SyncInstalled means the plugin was freshly installed.
	SyncInstalled SyncAction = "installed"
	// SyncUpdated means the plugin was updated to a newer version.
	SyncUpdated SyncAction = "updated"
	// SyncUpToDate means the plugin was already at the latest compatible version.
	SyncUpToDate SyncAction = "up_to_date"
	// SyncPruned means the plugin was removed because it's no longer in config.
	SyncPruned SyncAction = "pruned"
)

// SyncResult contains information about a single plugin's sync outcome.
type SyncResult struct {
	// Name is the plugin name
	Name string
	// Action is what happened during sync
	Action SyncAction
	// OldVersion is the previously installed version (empty if newly installed)
	OldVersion string
	// NewVersion is the version after sync (empty if pruned)
	NewVersion string
	// Reason is a human-readable explanation
	Reason string
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

	i.contributions = nil

	var results []UpdateResult

	for _, name := range names {
		result, err := i.updatePlugin(name, dryRun)
		if err != nil {
			return nil, err
		}
		results = append(results, *result)
	}

	// Generate shared files if not dry run
	if !dryRun {
		// Collect contributions from non-updated plugins
		if err := i.collectAllContributions(); err != nil {
			return nil, err
		}

		// Install project-level resources
		if err := i.installProjectResources(); err != nil {
			return nil, err
		}

		// Generate all shared files from scratch
		if err := i.generateSharedFiles(); err != nil {
			return nil, err
		}
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

// checkForUpdate checks if a newer version is available for a plugin.
// Returns the best available version string, or empty if already up to date.
// Also returns the PluginSpec built from project config for use in installation.
func (i *Installer) checkForUpdate(name string) (bestVersion string, spec PluginSpec, err error) {
	locked := i.lock.Get(name)
	if locked == nil {
		return "", PluginSpec{}, fmt.Errorf("plugin %q not in lock file", name)
	}

	// Get constraint from project config
	var constraint string
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
	reg, regErr := i.resolveRegistry(&spec)
	if regErr != nil {
		return "", spec, errors.NewInstallError(name, "resolve", regErr)
	}

	// Get all available versions
	pkgInfo, regErr := reg.GetPackageInfo(name)
	if regErr != nil {
		return "", spec, errors.NewInstallError(name, "resolve", regErr)
	}

	// Parse constraint and find best matching version
	c, regErr := version.ParseConstraint(constraint)
	if regErr != nil {
		return "", spec, errors.NewInstallError(name, "resolve",
			fmt.Errorf("invalid version constraint %q: %w", constraint, regErr))
	}

	// Parse available versions
	var versions []*version.Version
	for _, v := range pkgInfo.Versions {
		if parsed, parseErr := version.Parse(v); parseErr == nil {
			versions = append(versions, parsed)
		}
	}

	// Find best matching version
	best := c.FindBest(versions)
	if best == nil {
		return "", spec, nil // No matching version
	}

	// Compare with current version
	current, parseErr := version.Parse(locked.Version)
	if parseErr != nil {
		current = nil
	}

	if current != nil && !best.GreaterThan(current) {
		return "", spec, nil // Already up to date
	}

	return best.String(), spec, nil
}

// Sync synchronizes the project to match dex.hcl.
// For each plugin in config: installs (always, even if up-to-date).
// For each plugin in lock file but not in config: prunes (uninstalls).
// The version check only affects the reported SyncAction.
// All shared files are regenerated from scratch after processing.
// If dryRun is true, only reports what would change without making modifications.
func (i *Installer) Sync(dryRun bool) ([]SyncResult, error) {
	var results []SyncResult

	i.contributions = nil

	// Build set of config plugin names
	configPlugins := make(map[string]bool)
	for _, p := range i.project.Plugins {
		configPlugins[p.Name] = true
	}

	// Process each plugin in config
	for _, plugin := range i.project.Plugins {
		locked := i.lock.Get(plugin.Name)

		if locked == nil {
			// Not installed → install
			if dryRun {
				// Resolve what version would be installed
				spec := PluginSpec{
					Name:     plugin.Name,
					Version:  plugin.Version,
					Source:   plugin.Source,
					Registry: plugin.Registry,
					Config:   plugin.Config,
				}
				reg, err := i.resolveRegistry(&spec)
				if err != nil {
					return nil, errors.NewInstallError(plugin.Name, "resolve", err)
				}
				resolved, err := reg.ResolvePackage(plugin.Name, plugin.Version)
				if err != nil {
					return nil, errors.NewInstallError(plugin.Name, "resolve", err)
				}
				results = append(results, SyncResult{
					Name:       plugin.Name,
					Action:     SyncInstalled,
					NewVersion: resolved.Version,
					Reason:     "would install",
				})
			} else {
				spec := PluginSpec{
					Name:     plugin.Name,
					Version:  plugin.Version,
					Source:   plugin.Source,
					Registry: plugin.Registry,
					Config:   plugin.Config,
				}
				// If locked version exists for version hint, use it
				if lockedEntry := i.lock.Get(plugin.Name); lockedEntry != nil && plugin.Version == "" {
					spec.Version = lockedEntry.Version
				}
				info, err := i.installPlugin(spec)
				if err != nil {
					return nil, err
				}
				results = append(results, SyncResult{
					Name:       plugin.Name,
					Action:     SyncInstalled,
					NewVersion: info.Version,
					Reason:     "installed",
				})
			}
		} else {
			// Already installed → check for update, but always reinstall
			bestVersion, spec, err := i.checkForUpdate(plugin.Name)
			if err != nil {
				return nil, err
			}

			if bestVersion == "" {
				// Up to date — still reinstall to regenerate files
				if !dryRun {
					spec = PluginSpec{
						Name:     plugin.Name,
						Version:  locked.Version,
						Source:   plugin.Source,
						Registry: plugin.Registry,
						Config:   plugin.Config,
					}
					if _, err := i.installPlugin(spec); err != nil {
						return nil, err
					}
				}
				results = append(results, SyncResult{
					Name:       plugin.Name,
					Action:     SyncUpToDate,
					OldVersion: locked.Version,
					NewVersion: locked.Version,
					Reason:     "up to date",
				})
			} else {
				// Update available
				if dryRun {
					results = append(results, SyncResult{
						Name:       plugin.Name,
						Action:     SyncUpdated,
						OldVersion: locked.Version,
						NewVersion: bestVersion,
						Reason:     "would update",
					})
				} else {
					spec.Version = bestVersion
					_, err := i.installPlugin(spec)
					if err != nil {
						return nil, err
					}
					results = append(results, SyncResult{
						Name:       plugin.Name,
						Action:     SyncUpdated,
						OldVersion: locked.Version,
						NewVersion: bestVersion,
						Reason:     "updated",
					})
				}
			}
		}
	}

	// Prune plugins in lock file but not in config
	for pluginName := range i.lock.Plugins {
		if !configPlugins[pluginName] {
			lockedVersion := i.lock.Plugins[pluginName].Version
			if dryRun {
				results = append(results, SyncResult{
					Name:       pluginName,
					Action:     SyncPruned,
					OldVersion: lockedVersion,
					Reason:     "would prune",
				})
			} else {
				if err := i.uninstallPlugin(pluginName); err != nil {
					return nil, err
				}
				i.lock.Remove(pluginName)
				results = append(results, SyncResult{
					Name:       pluginName,
					Action:     SyncPruned,
					OldVersion: lockedVersion,
					Reason:     "pruned",
				})
			}
		}
	}

	// Sort results for consistent output: installed, updated, up_to_date, pruned
	sort.Slice(results, func(a, b int) bool {
		order := map[SyncAction]int{SyncInstalled: 0, SyncUpdated: 1, SyncUpToDate: 2, SyncPruned: 3}
		if order[results[a].Action] != order[results[b].Action] {
			return order[results[a].Action] < order[results[b].Action]
		}
		return results[a].Name < results[b].Name
	})

	// Generate shared files and save if not dry-run
	if !dryRun {
		// Install project-level resources
		if err := i.installProjectResources(); err != nil {
			return nil, err
		}

		// Generate all shared files from scratch
		if err := i.generateSharedFiles(); err != nil {
			return nil, err
		}

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
	reg, err := i.resolveRegistry(&spec)
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

// removeString removes a string from a slice, returning a new slice without the string.
func removeString(slice []string, s string) []string {
	result := make([]string, 0, len(slice))
	for _, v := range slice {
		if v != s {
			result = append(result, v)
		}
	}
	return result
}
