package config

import (
	"fmt"
	"path/filepath"
)

// ProjectConfig represents the dex.hcl file structure.
// This is the main configuration file for a dex-managed project.
type ProjectConfig struct {
	// Project contains project metadata
	Project ProjectBlock `hcl:"project,block"`

	// Registries defines plugin registry sources
	Registries []RegistryBlock `hcl:"registry,block"`

	// Plugins defines plugin dependencies
	Plugins []PluginBlock `hcl:"plugin,block"`
}

// ProjectBlock contains project metadata defined in the project {} block.
type ProjectBlock struct {
	// Name is the project name
	Name string `hcl:"name,attr"`

	// AgenticPlatform specifies the target AI agent platform (e.g., "claude-code", "cursor")
	AgenticPlatform string `hcl:"agentic_platform,attr"`
}

// RegistryBlock defines a plugin registry source.
// Registries can be local (file://) or remote (https://).
type RegistryBlock struct {
	// Name is the unique identifier for this registry
	Name string `hcl:"name,label"`

	// Path is the local filesystem path for file:// registries
	Path string `hcl:"path,optional"`

	// URL is the remote URL for https:// registries
	URL string `hcl:"url,optional"`
}

// PluginBlock defines a plugin dependency.
// Plugins can be sourced directly (git+https://, file://) or from a registry.
type PluginBlock struct {
	// Name is the unique identifier for this plugin dependency
	Name string `hcl:"name,label"`

	// Source is a direct source URL (git+https://, file://)
	Source string `hcl:"source,optional"`

	// Version is the version constraint for the plugin
	Version string `hcl:"version,optional"`

	// Registry is the name of the registry to fetch the plugin from
	Registry string `hcl:"registry,optional"`

	// Config provides plugin-specific configuration values
	Config map[string]string `hcl:"config,optional"`
}

// LoadProject loads a dex.hcl file from the given directory.
// It parses the HCL file, evaluates expressions, and returns the configuration.
func LoadProject(dir string) (*ProjectConfig, error) {
	filename := filepath.Join(dir, "dex.hcl")

	parser := NewParser()
	file, diags := parser.ParseFile(filename)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse %s: %s", filename, diags.Error())
	}

	ctx := NewEvalContext()
	var config ProjectConfig
	diags = DecodeBody(file.Body, ctx, &config)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to decode %s: %s", filename, diags.Error())
	}

	return &config, nil
}

// Validate checks the project config for errors.
// It ensures required fields are present and values are valid.
func (p *ProjectConfig) Validate() error {
	// Validate project block
	// Name is optional - will default to directory name if not specified
	if p.Project.AgenticPlatform == "" {
		return fmt.Errorf("project.agentic_platform is required")
	}

	// Validate registries
	registryNames := make(map[string]bool)
	for _, reg := range p.Registries {
		if reg.Name == "" {
			return fmt.Errorf("registry name is required")
		}
		if registryNames[reg.Name] {
			return fmt.Errorf("duplicate registry name: %s", reg.Name)
		}
		registryNames[reg.Name] = true

		// Must have either path or URL, but not both
		if reg.Path == "" && reg.URL == "" {
			return fmt.Errorf("registry %q must have either path or url", reg.Name)
		}
		if reg.Path != "" && reg.URL != "" {
			return fmt.Errorf("registry %q cannot have both path and url", reg.Name)
		}
	}

	// Validate plugins
	pluginNames := make(map[string]bool)
	for _, plugin := range p.Plugins {
		if plugin.Name == "" {
			return fmt.Errorf("plugin name is required")
		}
		if pluginNames[plugin.Name] {
			return fmt.Errorf("duplicate plugin name: %s", plugin.Name)
		}
		pluginNames[plugin.Name] = true

		// Must have either source or registry
		if plugin.Source == "" && plugin.Registry == "" {
			return fmt.Errorf("plugin %q must have either source or registry", plugin.Name)
		}
		if plugin.Source != "" && plugin.Registry != "" {
			return fmt.Errorf("plugin %q cannot have both source and registry", plugin.Name)
		}

		// If using registry, it must exist
		if plugin.Registry != "" && !registryNames[plugin.Registry] {
			return fmt.Errorf("plugin %q references unknown registry: %s", plugin.Name, plugin.Registry)
		}
	}

	return nil
}
