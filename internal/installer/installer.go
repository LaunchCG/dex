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

	"github.com/dex-tools/dex/internal/adapter"
	"github.com/dex-tools/dex/internal/config"
	"github.com/dex-tools/dex/internal/errors"
	"github.com/dex-tools/dex/internal/lockfile"
	"github.com/dex-tools/dex/internal/manifest"
	"github.com/dex-tools/dex/internal/registry"
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

// Install installs the specified plugins.
// If specs is empty, installs all plugins from project config using lock file versions.
func (i *Installer) Install(specs []PluginSpec) error {
	if len(specs) == 0 {
		return i.InstallAll()
	}

	for _, spec := range specs {
		if err := i.installPlugin(spec); err != nil {
			return err
		}
	}

	// Save manifest and lock file
	if err := i.manifest.Save(); err != nil {
		return errors.Wrap(err, "failed to save manifest")
	}

	if !i.noLock {
		// Set the agent platform in lock file
		i.lock.Agent = i.project.Project.AgenticPlatform
		if err := i.lock.Save(); err != nil {
			return errors.Wrap(err, "failed to save lock file")
		}
	}

	return nil
}

// InstallAll installs all plugins from the project config.
// Uses lock file versions if available, otherwise resolves latest.
func (i *Installer) InstallAll() error {
	if len(i.project.Plugins) == 0 {
		fmt.Println("No plugins defined in config")
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
		if err := i.installPlugin(spec); err != nil {
			return err
		}
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
func (i *Installer) installPlugin(spec PluginSpec) error {
	// Resolve the registry to use
	reg, err := i.resolveRegistry(spec)
	if err != nil {
		return errors.NewInstallError(spec.Name, "resolve", err)
	}

	// Resolve the version
	resolved, err := reg.ResolvePackage(spec.Name, spec.Version)
	if err != nil {
		return errors.NewInstallError(spec.Name, "resolve", err)
	}

	// Use resolved package name (important when spec.Name is empty for direct sources)
	pluginName := resolved.Name
	if pluginName == "" {
		pluginName = spec.Name
	}

	// Create temp directory for fetching
	tempDir, err := os.MkdirTemp("", "dex-install-*")
	if err != nil {
		return errors.NewInstallError(pluginName, "fetch", err)
	}
	defer os.RemoveAll(tempDir)

	// Fetch the package
	pluginDir, err := reg.FetchPackage(resolved, tempDir)
	if err != nil {
		return errors.NewInstallError(pluginName, "fetch", err)
	}

	// Load and validate package config
	pkgConfig, err := config.LoadPackage(pluginDir)
	if err != nil {
		return errors.NewInstallError(pluginName, "parse", err)
	}

	// Get the actual plugin name from the package
	pluginName = pkgConfig.Package.Name

	if err := pkgConfig.Validate(); err != nil {
		return errors.NewInstallError(pluginName, "validate", err)
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
			return errors.NewInstallError(pluginName, "validate",
				fmt.Errorf("plugin %q does not support platform %q",
					pluginName, i.project.Project.AgenticPlatform))
		}
	}

	// Resolve variable values
	vars, err := i.resolveVariables(pkgConfig, spec.Config)
	if err != nil {
		return errors.NewInstallError(pluginName, "configure", err)
	}

	// Create executor
	executor := NewExecutor(i.projectRoot, i.manifest, i.force)

	// Plan and execute installation for each resource
	// Filter resources to only include those matching the target platform
	targetPlatform := i.project.Project.AgenticPlatform
	var allPlans []*adapter.Plan
	for _, res := range pkgConfig.Resources {
		// Skip resources that don't match the target platform
		if res.Platform() != targetPlatform {
			continue
		}
		plan, err := i.adapter.PlanInstallation(res, pkgConfig, pluginDir, i.projectRoot)
		if err != nil {
			return errors.NewInstallError(pluginName, "plan", err)
		}
		allPlans = append(allPlans, plan)
	}

	// Merge all plans
	mergedPlan := adapter.MergePlans(allPlans...)

	// Execute the merged plan
	if err := executor.Execute(mergedPlan, vars); err != nil {
		return errors.NewInstallError(pluginName, "install", err)
	}

	// Update lock file
	if !i.noLock {
		i.lock.Set(pluginName, &lockfile.LockedPlugin{
			Version:   resolved.Version,
			Resolved:  resolved.URL,
			Integrity: resolved.Integrity,
		})
	}

	fmt.Printf("  ✓ Installed %s@%s\n", pluginName, resolved.Version)
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
