package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/launchcg/dex/internal/resource"
)

// PackageConfig represents a plugin's package.hcl file.
// This file defines the metadata and resources provided by a plugin.
type PackageConfig struct {
	// Package contains package metadata
	Package PackageBlock `hcl:"package,block"`

	// Variables defines user-configurable variables
	Variables []VariableBlock `hcl:"variable,block"`

	// Dependencies defines package dependencies
	Dependencies []DependencyBlock `hcl:"dependency,block"`

	// Unified MCP servers (work across all platforms)
	UnifiedMCPServers []resource.MCPServer `hcl:"mcp_server,block"`

	// Claude resources - each type is a separate field for HCL parsing
	Skills     []resource.ClaudeSkill     `hcl:"claude_skill,block"`
	Commands   []resource.ClaudeCommand   `hcl:"claude_command,block"`
	Subagents  []resource.ClaudeSubagent  `hcl:"claude_subagent,block"`
	Rules      []resource.ClaudeRule      `hcl:"claude_rule,block"`
	RulesFiles []resource.ClaudeRules     `hcl:"claude_rules,block"`
	Settings   []resource.ClaudeSettings  `hcl:"claude_settings,block"`
	MCPServers []resource.ClaudeMCPServer `hcl:"claude_mcp_server,block"`

	// GitHub Copilot resources - merged (multiple plugins contribute to same file)
	CopilotInstruction []resource.CopilotInstruction `hcl:"copilot_instruction,block"`
	CopilotMCPServers  []resource.CopilotMCPServer   `hcl:"copilot_mcp_server,block"`

	// GitHub Copilot resources - standalone (one file per resource)
	CopilotInstructions []resource.CopilotInstructions `hcl:"copilot_instructions,block"`
	CopilotPrompts      []resource.CopilotPrompt       `hcl:"copilot_prompt,block"`
	CopilotAgents       []resource.CopilotAgent        `hcl:"copilot_agent,block"`
	CopilotSkills       []resource.CopilotSkill        `hcl:"copilot_skill,block"`

	// Cursor resources - merged (multiple plugins contribute to same file)
	CursorRules_     []resource.CursorRule      `hcl:"cursor_rule,block"`
	CursorMCPServers []resource.CursorMCPServer `hcl:"cursor_mcp_server,block"`

	// Cursor resources - standalone (one file per resource)
	CursorRules    []resource.CursorRules   `hcl:"cursor_rules,block"`
	CursorCommands []resource.CursorCommand `hcl:"cursor_command,block"`

	// Resources is a unified view of all resources (populated after parsing)
	// This field has no hcl tag because it's populated programmatically, not from HCL.
	Resources []resource.Resource
}

// PackageBlock contains package metadata defined in the package {} block.
type PackageBlock struct {
	// Name is the package name
	Name string `hcl:"name,attr"`

	// Version is the package version (semver recommended)
	Version string `hcl:"version,attr"`

	// Description explains what this package provides
	Description string `hcl:"description,optional"`

	// Author is the package author's name or handle
	Author string `hcl:"author,optional"`

	// License is the package license identifier (e.g., "MIT", "Apache-2.0")
	License string `hcl:"license,optional"`

	// Repository is the URL to the package's source repository
	Repository string `hcl:"repository,optional"`

	// Platforms lists the supported AI agent platforms (e.g., ["claude-code", "cursor"])
	// If empty, the package is assumed to support all platforms
	Platforms []string `hcl:"platforms,optional"`
}

// VariableBlock defines a user-configurable variable.
// Variables allow users to customize plugin behavior at installation time.
type VariableBlock struct {
	// Name is the variable identifier used in templates
	Name string `hcl:"name,label"`

	// Description explains what this variable controls
	Description string `hcl:"description,optional"`

	// Default is the default value if not provided by the user
	Default string `hcl:"default,optional"`

	// Required indicates whether the user must provide a value
	Required bool `hcl:"required,optional"`

	// Env specifies an environment variable to read the value from
	Env string `hcl:"env,optional"`
}

// DependencyBlock defines a package dependency.
// Dependencies are other packages that must be installed before this package.
//
// Syntax in package.hcl:
//
//	dependency "other-plugin" {
//	  version = "^2.0.0"
//	}
//
//	dependency "another" {
//	  version  = ">=1.0.0"
//	  registry = "custom"
//	}
type DependencyBlock struct {
	// Name is the dependency package name (from label)
	Name string `hcl:"name,label"`

	// Version is the version constraint (e.g., "^1.0.0", ">=2.0.0")
	Version string `hcl:"version,attr"`

	// Registry is optional registry name to use for this dependency
	Registry string `hcl:"registry,optional"`

	// Source is optional direct source URL
	Source string `hcl:"source,optional"`
}

// PackageResourcesConfig is used to parse *.pkg.hcl files that contain only
// resource definitions without a package {} block.
type PackageResourcesConfig struct {
	// Variables defines user-configurable variables
	Variables []VariableBlock `hcl:"variable,block"`

	// Dependencies defines package dependencies
	Dependencies []DependencyBlock `hcl:"dependency,block"`

	// Unified MCP servers (work across all platforms)
	UnifiedMCPServers []resource.MCPServer `hcl:"mcp_server,block"`

	// Claude resources
	Skills     []resource.ClaudeSkill     `hcl:"claude_skill,block"`
	Commands   []resource.ClaudeCommand   `hcl:"claude_command,block"`
	Subagents  []resource.ClaudeSubagent  `hcl:"claude_subagent,block"`
	Rules      []resource.ClaudeRule      `hcl:"claude_rule,block"`
	RulesFiles []resource.ClaudeRules     `hcl:"claude_rules,block"`
	Settings   []resource.ClaudeSettings  `hcl:"claude_settings,block"`
	MCPServers []resource.ClaudeMCPServer `hcl:"claude_mcp_server,block"`

	// GitHub Copilot resources - merged
	CopilotInstruction []resource.CopilotInstruction `hcl:"copilot_instruction,block"`
	CopilotMCPServers  []resource.CopilotMCPServer   `hcl:"copilot_mcp_server,block"`

	// GitHub Copilot resources - standalone
	CopilotInstructions []resource.CopilotInstructions `hcl:"copilot_instructions,block"`
	CopilotPrompts      []resource.CopilotPrompt       `hcl:"copilot_prompt,block"`
	CopilotAgents       []resource.CopilotAgent        `hcl:"copilot_agent,block"`
	CopilotSkills       []resource.CopilotSkill        `hcl:"copilot_skill,block"`

	// Cursor resources - merged
	CursorRules_     []resource.CursorRule      `hcl:"cursor_rule,block"`
	CursorMCPServers []resource.CursorMCPServer `hcl:"cursor_mcp_server,block"`

	// Cursor resources - standalone
	CursorRules    []resource.CursorRules   `hcl:"cursor_rules,block"`
	CursorCommands []resource.CursorCommand `hcl:"cursor_command,block"`
}

// LoadPackage loads package.hcl and all *.pkg.hcl files from the given directory.
// The main package.hcl is required and contains the package {} block with metadata.
// Additional *.pkg.hcl files are optional and contain only resource definitions.
// All resources from all files are merged into the final configuration.
func LoadPackage(dir string) (*PackageConfig, error) {
	mainFile := filepath.Join(dir, "package.hcl")

	parser := NewParser()
	file, diags := parser.ParseFile(mainFile)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse %s: %s", mainFile, diags.Error())
	}

	// Create evaluation context with file() function for this directory
	ctx := NewPackageEvalContext(dir)

	var config PackageConfig
	diags = DecodeBody(file.Body, ctx, &config)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to decode %s: %s", mainFile, diags.Error())
	}

	// Load additional *.pkg.hcl files and merge resources
	matches, err := filepath.Glob(filepath.Join(dir, "*.pkg.hcl"))
	if err != nil {
		return nil, fmt.Errorf("failed to glob *.pkg.hcl files: %w", err)
	}

	for _, match := range matches {
		additionalFile, diags := parser.ParseFile(match)
		if diags.HasErrors() {
			return nil, fmt.Errorf("failed to parse %s: %s", match, diags.Error())
		}

		// Use PackageResourcesConfig for *.pkg.hcl files (no package block required)
		var additionalConfig PackageResourcesConfig
		diags = DecodeBody(additionalFile.Body, ctx, &additionalConfig)
		if diags.HasErrors() {
			return nil, fmt.Errorf("failed to decode %s: %s", match, diags.Error())
		}

		// Merge resources from additional file into main config
		config.mergeResourcesFrom(&additionalConfig)
	}

	// Build unified Resources slice from typed fields
	config.buildResources()

	return &config, nil
}

// mergeResourcesFrom merges resources from a PackageResourcesConfig into this config.
// Used to merge *.pkg.hcl files that contain only resources.
func (p *PackageConfig) mergeResourcesFrom(other *PackageResourcesConfig) {
	// Unified MCP servers
	p.UnifiedMCPServers = append(p.UnifiedMCPServers, other.UnifiedMCPServers...)

	// Claude resources
	p.Skills = append(p.Skills, other.Skills...)
	p.Commands = append(p.Commands, other.Commands...)
	p.Subagents = append(p.Subagents, other.Subagents...)
	p.Rules = append(p.Rules, other.Rules...)
	p.RulesFiles = append(p.RulesFiles, other.RulesFiles...)
	p.Settings = append(p.Settings, other.Settings...)
	p.MCPServers = append(p.MCPServers, other.MCPServers...)

	// GitHub Copilot resources - merged
	p.CopilotInstruction = append(p.CopilotInstruction, other.CopilotInstruction...)
	p.CopilotMCPServers = append(p.CopilotMCPServers, other.CopilotMCPServers...)

	// GitHub Copilot resources - standalone
	p.CopilotInstructions = append(p.CopilotInstructions, other.CopilotInstructions...)
	p.CopilotPrompts = append(p.CopilotPrompts, other.CopilotPrompts...)
	p.CopilotAgents = append(p.CopilotAgents, other.CopilotAgents...)
	p.CopilotSkills = append(p.CopilotSkills, other.CopilotSkills...)

	// Cursor resources - merged
	p.CursorRules_ = append(p.CursorRules_, other.CursorRules_...)
	p.CursorMCPServers = append(p.CursorMCPServers, other.CursorMCPServers...)

	// Cursor resources - standalone
	p.CursorRules = append(p.CursorRules, other.CursorRules...)
	p.CursorCommands = append(p.CursorCommands, other.CursorCommands...)

	// Variables can also be merged from additional files
	p.Variables = append(p.Variables, other.Variables...)

	// Dependencies can also be merged from additional files
	p.Dependencies = append(p.Dependencies, other.Dependencies...)
}

// buildResources populates the Resources slice from the typed resource fields.
func (p *PackageConfig) buildResources() {
	p.Resources = nil

	// Unified MCP servers
	for i := range p.UnifiedMCPServers {
		p.Resources = append(p.Resources, &p.UnifiedMCPServers[i])
	}

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

	// GitHub Copilot resources - merged
	for i := range p.CopilotInstruction {
		p.Resources = append(p.Resources, &p.CopilotInstruction[i])
	}
	for i := range p.CopilotMCPServers {
		p.Resources = append(p.Resources, &p.CopilotMCPServers[i])
	}

	// GitHub Copilot resources - standalone
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

	// Cursor resources - merged
	for i := range p.CursorRules_ {
		p.Resources = append(p.Resources, &p.CursorRules_[i])
	}
	for i := range p.CursorMCPServers {
		p.Resources = append(p.Resources, &p.CursorMCPServers[i])
	}

	// Cursor resources - standalone
	for i := range p.CursorRules {
		p.Resources = append(p.Resources, &p.CursorRules[i])
	}
	for i := range p.CursorCommands {
		p.Resources = append(p.Resources, &p.CursorCommands[i])
	}
}

// Validate checks the package config for errors.
// It ensures required fields are present and values are valid.
func (p *PackageConfig) Validate() error {
	// Validate package block
	if p.Package.Name == "" {
		return fmt.Errorf("package.name is required")
	}
	if p.Package.Version == "" {
		return fmt.Errorf("package.version is required")
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

		// Required variables should not have defaults (optional validation)
		if v.Required && v.Default != "" {
			return fmt.Errorf("variable %q is marked required but has a default value", v.Name)
		}
	}

	// Validate resources
	for _, res := range p.Resources {
		if err := res.Validate(); err != nil {
			return fmt.Errorf("invalid resource %s.%s: %w", res.ResourceType(), res.ResourceName(), err)
		}
	}

	return nil
}

// GetVariable returns the variable with the given name, or nil if not found.
func (p *PackageConfig) GetVariable(name string) *VariableBlock {
	for i := range p.Variables {
		if p.Variables[i].Name == name {
			return &p.Variables[i]
		}
	}
	return nil
}

// ResolveVariableValue resolves the value for a variable.
// It checks (in order): environment variable, provided config, default value.
// Returns an error if the variable is required and no value is available.
func (v *VariableBlock) ResolveValue(config map[string]string) (string, error) {
	// Check environment variable first
	if v.Env != "" {
		if val, ok := lookupEnv(v.Env); ok {
			return val, nil
		}
	}

	// Check provided config
	if config != nil {
		if val, ok := config[v.Name]; ok {
			return val, nil
		}
	}

	// Use default if available
	if v.Default != "" {
		return v.Default, nil
	}

	// If required and no value found, return error
	if v.Required {
		return "", fmt.Errorf("required variable %q has no value", v.Name)
	}

	return "", nil
}

// lookupEnv is a wrapper around os.LookupEnv for testability.
// Can be replaced in tests to mock environment variables.
var lookupEnv = os.LookupEnv
