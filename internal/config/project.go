package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/launchcg/dex/internal/resource"
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

	// Claude resources - can be defined directly in dex.hcl
	Skills     []resource.ClaudeSkill     `hcl:"claude_skill,block"`
	Commands   []resource.ClaudeCommand   `hcl:"claude_command,block"`
	Subagents  []resource.ClaudeSubagent  `hcl:"claude_subagent,block"`
	Rules      []resource.ClaudeRule      `hcl:"claude_rule,block"`
	RulesFiles []resource.ClaudeRules     `hcl:"claude_rules,block"`
	Settings   []resource.ClaudeSettings  `hcl:"claude_settings,block"`
	MCPServers []resource.ClaudeMCPServer `hcl:"claude_mcp_server,block"`

	// GitHub Copilot resources
	CopilotInstruction  []resource.CopilotInstruction  `hcl:"copilot_instruction,block"`
	CopilotMCPServers   []resource.CopilotMCPServer    `hcl:"copilot_mcp_server,block"`
	CopilotInstructions []resource.CopilotInstructions `hcl:"copilot_instructions,block"`
	CopilotPrompts      []resource.CopilotPrompt       `hcl:"copilot_prompt,block"`
	CopilotAgents       []resource.CopilotAgent        `hcl:"copilot_agent,block"`
	CopilotSkills       []resource.CopilotSkill        `hcl:"copilot_skill,block"`

	// Cursor resources
	CursorRules_     []resource.CursorRule      `hcl:"cursor_rule,block"`
	CursorMCPServers []resource.CursorMCPServer `hcl:"cursor_mcp_server,block"`
	CursorRules      []resource.CursorRules     `hcl:"cursor_rules,block"`
	CursorCommands   []resource.CursorCommand   `hcl:"cursor_command,block"`

	// Resources is a unified view of all resources (populated after parsing)
	Resources []resource.Resource

	// Variables defines user-configurable variables for the project
	// Populated from first-pass extraction, not from HCL decode
	Variables []ProjectVariableBlock

	// ResolvedVars contains the resolved variable values (populated after parsing, not from HCL)
	ResolvedVars map[string]string
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

// ProjectVariableBlock defines a user-configurable variable for the project.
// Variables can source values from environment variables with fallback defaults.
type ProjectVariableBlock struct {
	// Name is the variable identifier used in var.NAME references
	Name string `hcl:"name,label"`

	// Description explains what this variable controls
	Description string `hcl:"description,optional"`

	// Default is the default value if not provided by environment
	Default string `hcl:"default,optional"`

	// Env specifies an environment variable to read the value from
	Env string `hcl:"env,optional"`

	// Required indicates whether a value must be available
	Required bool `hcl:"required,optional"`
}

// LoadProject loads a dex.hcl file from the given directory.
// It parses the HCL file, evaluates expressions, and returns the configuration.
// Uses two-pass parsing to first extract and resolve variables, then decode the full config.
func LoadProject(dir string) (*ProjectConfig, error) {
	filename := filepath.Join(dir, "dex.hcl")

	parser := NewParser()
	file, diags := parser.ParseFile(filename)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse %s: %s", filename, diags.Error())
	}

	// Pass 1: Extract and resolve variable blocks
	// remain contains the body with variable blocks removed
	variables, resolvedVars, remain, err := extractAndResolveProjectVariables(file.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve variables in %s: %w", filename, err)
	}

	// Pass 2: Decode the remaining body with resolved vars in the eval context
	ctx := NewProjectEvalContext(resolvedVars)
	var config ProjectConfig
	diags = DecodeBody(remain, ctx, &config)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to decode %s: %s", filename, diags.Error())
	}

	// Store resolved variables for later use (populated from pass 1)
	config.Variables = variables
	config.ResolvedVars = resolvedVars

	// Build unified Resources slice from typed fields
	config.buildResources()

	return &config, nil
}

// buildResources populates the Resources slice from the typed resource fields.
func (p *ProjectConfig) buildResources() {
	p.Resources = nil

	// Claude resources
	for i := range p.Skills {
		p.Resources = append(p.Resources, &p.Skills[i])
	}
	for i := range p.Commands {
		p.Resources = append(p.Resources, &p.Commands[i])
	}
	for i := range p.Subagents {
		p.Resources = append(p.Resources, &p.Subagents[i])
	}
	for i := range p.Rules {
		p.Resources = append(p.Resources, &p.Rules[i])
	}
	for i := range p.RulesFiles {
		p.Resources = append(p.Resources, &p.RulesFiles[i])
	}
	for i := range p.Settings {
		p.Resources = append(p.Resources, &p.Settings[i])
	}
	for i := range p.MCPServers {
		p.Resources = append(p.Resources, &p.MCPServers[i])
	}

	// GitHub Copilot resources
	for i := range p.CopilotInstruction {
		p.Resources = append(p.Resources, &p.CopilotInstruction[i])
	}
	for i := range p.CopilotMCPServers {
		p.Resources = append(p.Resources, &p.CopilotMCPServers[i])
	}
	for i := range p.CopilotInstructions {
		p.Resources = append(p.Resources, &p.CopilotInstructions[i])
	}
	for i := range p.CopilotPrompts {
		p.Resources = append(p.Resources, &p.CopilotPrompts[i])
	}
	for i := range p.CopilotAgents {
		p.Resources = append(p.Resources, &p.CopilotAgents[i])
	}
	for i := range p.CopilotSkills {
		p.Resources = append(p.Resources, &p.CopilotSkills[i])
	}

	// Cursor resources
	for i := range p.CursorRules_ {
		p.Resources = append(p.Resources, &p.CursorRules_[i])
	}
	for i := range p.CursorMCPServers {
		p.Resources = append(p.Resources, &p.CursorMCPServers[i])
	}
	for i := range p.CursorRules {
		p.Resources = append(p.Resources, &p.CursorRules[i])
	}
	for i := range p.CursorCommands {
		p.Resources = append(p.Resources, &p.CursorCommands[i])
	}
}

// Validate checks the project config for errors.
// It ensures required fields are present and values are valid.
func (p *ProjectConfig) Validate() error {
	// Validate project block
	// Name is optional - will default to directory name if not specified
	if p.Project.AgenticPlatform == "" {
		return fmt.Errorf("project.agentic_platform is required")
	}

	// Validate variables
	varNames := make(map[string]bool)
	for _, v := range p.Variables {
		if v.Name == "" {
			return fmt.Errorf("variable name is required")
		}
		if varNames[v.Name] {
			return fmt.Errorf("duplicate variable name: %s", v.Name)
		}
		varNames[v.Name] = true

		// Required variables should not have defaults
		if v.Required && v.Default != "" {
			return fmt.Errorf("variable %q is marked required but has a default value", v.Name)
		}
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

// AddPlugin adds a plugin block to the dex.hcl file.
// It appends the plugin block to the end of the file.
func AddPlugin(dir string, name string, source string, version string) error {
	filename := filepath.Join(dir, "dex.hcl")

	// Read existing content
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", filename, err)
	}

	// Check if plugin already exists
	existingConfig, err := LoadProject(dir)
	if err == nil {
		for _, p := range existingConfig.Plugins {
			if p.Name == name {
				// Plugin already exists, skip
				return nil
			}
		}
	}

	// Build the plugin block
	var pluginBlock string
	if version != "" {
		pluginBlock = fmt.Sprintf("\nplugin %q {\n  source  = %q\n  version = %q\n}\n", name, source, version)
	} else {
		pluginBlock = fmt.Sprintf("\nplugin %q {\n  source = %q\n}\n", name, source)
	}

	// Append to file
	newContent := string(content) + pluginBlock
	if err := os.WriteFile(filename, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", filename, err)
	}

	return nil
}
