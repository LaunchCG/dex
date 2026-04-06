package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/launchcg/dex/internal/resource"
)

// ResourceSet holds all platform resource slices. It is embedded in ProjectConfig,
// LocalConfig, and ProfileBlock so that adding a new resource type requires editing
// only this struct and its methods. When adding a field here, also update copyFrom,
// mergeFrom, appendFrom, and buildResources.
type ResourceSet struct {
	// Claude resources
	Skills     []resource.ClaudeSkill     `hcl:"claude_skill,block"`
	Commands   []resource.ClaudeCommand   `hcl:"claude_command,block"`
	Subagents  []resource.ClaudeSubagent  `hcl:"claude_subagent,block"`
	Rules      []resource.ClaudeRule      `hcl:"claude_rule,block"`
	RulesFiles []resource.ClaudeRules     `hcl:"claude_rules,block"`
	Settings   []resource.ClaudeSettings  `hcl:"claude_settings,block"`
	MCPServers []resource.ClaudeMCPServer `hcl:"claude_mcp_server,block"`

	// Universal MCP servers
	UniversalMCPServers []resource.MCPServer `hcl:"mcp_server,block"`

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
}

// copyFrom replaces all resource fields with those from src.
func (r *ResourceSet) copyFrom(src *ResourceSet) {
	*r = *src
}

// appendFrom appends all resource slices from src into r.
func (r *ResourceSet) appendFrom(src *ResourceSet) {
	r.Skills = append(r.Skills, src.Skills...)
	r.Commands = append(r.Commands, src.Commands...)
	r.Subagents = append(r.Subagents, src.Subagents...)
	r.Rules = append(r.Rules, src.Rules...)
	r.RulesFiles = append(r.RulesFiles, src.RulesFiles...)
	r.Settings = append(r.Settings, src.Settings...)
	r.MCPServers = append(r.MCPServers, src.MCPServers...)
	r.UniversalMCPServers = append(r.UniversalMCPServers, src.UniversalMCPServers...)
	r.CopilotInstruction = append(r.CopilotInstruction, src.CopilotInstruction...)
	r.CopilotMCPServers = append(r.CopilotMCPServers, src.CopilotMCPServers...)
	r.CopilotInstructions = append(r.CopilotInstructions, src.CopilotInstructions...)
	r.CopilotPrompts = append(r.CopilotPrompts, src.CopilotPrompts...)
	r.CopilotAgents = append(r.CopilotAgents, src.CopilotAgents...)
	r.CopilotSkills = append(r.CopilotSkills, src.CopilotSkills...)
	r.CursorRules_ = append(r.CursorRules_, src.CursorRules_...)
	r.CursorMCPServers = append(r.CursorMCPServers, src.CursorMCPServers...)
	r.CursorRules = append(r.CursorRules, src.CursorRules...)
	r.CursorCommands = append(r.CursorCommands, src.CursorCommands...)
}

// mergeFrom performs additive merge: same-name resources are replaced, new ones appended.
func (r *ResourceSet) mergeFrom(src *ResourceSet) {
	r.Skills = mergeByName(r.Skills, src.Skills, func(s resource.ClaudeSkill) string { return s.Name })
	r.Commands = mergeByName(r.Commands, src.Commands, func(c resource.ClaudeCommand) string { return c.Name })
	r.Subagents = mergeByName(r.Subagents, src.Subagents, func(s resource.ClaudeSubagent) string { return s.Name })
	r.Rules = mergeByName(r.Rules, src.Rules, func(v resource.ClaudeRule) string { return v.Name })
	r.RulesFiles = mergeByName(r.RulesFiles, src.RulesFiles, func(v resource.ClaudeRules) string { return v.Name })
	r.Settings = mergeByName(r.Settings, src.Settings, func(s resource.ClaudeSettings) string { return s.Name })
	r.MCPServers = mergeByName(r.MCPServers, src.MCPServers, func(m resource.ClaudeMCPServer) string { return m.Name })
	r.UniversalMCPServers = mergeByName(r.UniversalMCPServers, src.UniversalMCPServers, func(m resource.MCPServer) string { return m.Name })
	r.CopilotInstruction = mergeByName(r.CopilotInstruction, src.CopilotInstruction, func(c resource.CopilotInstruction) string { return c.Name })
	r.CopilotMCPServers = mergeByName(r.CopilotMCPServers, src.CopilotMCPServers, func(m resource.CopilotMCPServer) string { return m.Name })
	r.CopilotInstructions = mergeByName(r.CopilotInstructions, src.CopilotInstructions, func(c resource.CopilotInstructions) string { return c.Name })
	r.CopilotPrompts = mergeByName(r.CopilotPrompts, src.CopilotPrompts, func(c resource.CopilotPrompt) string { return c.Name })
	r.CopilotAgents = mergeByName(r.CopilotAgents, src.CopilotAgents, func(c resource.CopilotAgent) string { return c.Name })
	r.CopilotSkills = mergeByName(r.CopilotSkills, src.CopilotSkills, func(c resource.CopilotSkill) string { return c.Name })
	r.CursorRules_ = mergeByName(r.CursorRules_, src.CursorRules_, func(v resource.CursorRule) string { return v.Name })
	r.CursorMCPServers = mergeByName(r.CursorMCPServers, src.CursorMCPServers, func(m resource.CursorMCPServer) string { return m.Name })
	r.CursorRules = mergeByName(r.CursorRules, src.CursorRules, func(v resource.CursorRules) string { return v.Name })
	r.CursorCommands = mergeByName(r.CursorCommands, src.CursorCommands, func(c resource.CursorCommand) string { return c.Name })
}

// buildResources returns a unified Resource slice from the typed fields.
func (r *ResourceSet) buildResources() []resource.Resource {
	var res []resource.Resource
	for i := range r.Skills {
		res = append(res, &r.Skills[i])
	}
	for i := range r.Commands {
		res = append(res, &r.Commands[i])
	}
	for i := range r.Subagents {
		res = append(res, &r.Subagents[i])
	}
	for i := range r.Rules {
		res = append(res, &r.Rules[i])
	}
	for i := range r.RulesFiles {
		res = append(res, &r.RulesFiles[i])
	}
	for i := range r.Settings {
		res = append(res, &r.Settings[i])
	}
	for i := range r.MCPServers {
		res = append(res, &r.MCPServers[i])
	}
	for i := range r.UniversalMCPServers {
		res = append(res, &r.UniversalMCPServers[i])
	}
	for i := range r.CopilotInstruction {
		res = append(res, &r.CopilotInstruction[i])
	}
	for i := range r.CopilotMCPServers {
		res = append(res, &r.CopilotMCPServers[i])
	}
	for i := range r.CopilotInstructions {
		res = append(res, &r.CopilotInstructions[i])
	}
	for i := range r.CopilotPrompts {
		res = append(res, &r.CopilotPrompts[i])
	}
	for i := range r.CopilotAgents {
		res = append(res, &r.CopilotAgents[i])
	}
	for i := range r.CopilotSkills {
		res = append(res, &r.CopilotSkills[i])
	}
	for i := range r.CursorRules_ {
		res = append(res, &r.CursorRules_[i])
	}
	for i := range r.CursorMCPServers {
		res = append(res, &r.CursorMCPServers[i])
	}
	for i := range r.CursorRules {
		res = append(res, &r.CursorRules[i])
	}
	for i := range r.CursorCommands {
		res = append(res, &r.CursorCommands[i])
	}
	return res
}

// ProjectConfig represents the dex.hcl file structure.
// This is the main configuration file for a dex-managed project.
//
// Note: gohcl does not decode into embedded struct fields, so resource fields must be
// declared directly here (and in LocalConfig and ProfileBlock). The ResourceSet type
// and its methods (copyFrom, mergeFrom, appendFrom, buildResources) centralize the
// field-iteration logic so that adding a new resource type only requires updating
// ResourceSet and the toResourceSet/applyResourceSet methods.
type ProjectConfig struct {
	// Project contains project metadata
	Project ProjectBlock `hcl:"project,block"`

	// Profiles defines named configuration variants
	Profiles []ProfileBlock `hcl:"profile,block"`

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

	// Universal MCP servers - work across all platforms
	UniversalMCPServers []resource.MCPServer `hcl:"mcp_server,block"`

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
	AgenticPlatform string `hcl:"default_platform,attr"`

	// NamespaceAll enables namespacing for all installed packages
	NamespaceAll bool `hcl:"namespace_all,optional"`

	// NamespacePackages lists specific packages to namespace
	NamespacePackages []string `hcl:"namespace_packages,optional"`

	// AgentInstructions contains project-level instructions that appear at the top
	// of agent files (CLAUDE.md, AGENTS.md, copilot-instructions.md) before any
	// plugin-contributed content. This content is owned by the project, not plugins.
	AgentInstructions string `hcl:"agent_instructions,optional"`

	// GitExclude controls whether dex sync automatically updates .git/info/exclude
	// to locally hide dex-managed files from git without modifying .gitignore
	GitExclude bool `hcl:"git_exclude,optional"`
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

// ProfileBlock defines a named configuration variant.
// Profiles allow switching between different sets of plugins, registries, and resources
// using `dex sync --profile <name>`. By default, profile contents are merged additively
// with the default config. Set exclude_defaults = true to start clean.
type ProfileBlock struct {
	// Name is the profile identifier used in --profile flag
	Name string `hcl:"name,label"`

	// ExcludeDefaults when true means only profile-defined items are used
	ExcludeDefaults bool `hcl:"exclude_defaults,optional"`

	// AgentInstructions overrides the project-level agent instructions
	AgentInstructions string `hcl:"agent_instructions,optional"`

	// Registries defines profile-specific plugin registry sources
	Registries []RegistryBlock `hcl:"registry,block"`

	// Plugins defines profile-specific plugin dependencies
	Plugins []PluginBlock `hcl:"plugin,block"`

	// Claude resources
	Skills     []resource.ClaudeSkill     `hcl:"claude_skill,block"`
	Commands   []resource.ClaudeCommand   `hcl:"claude_command,block"`
	Subagents  []resource.ClaudeSubagent  `hcl:"claude_subagent,block"`
	Rules      []resource.ClaudeRule      `hcl:"claude_rule,block"`
	RulesFiles []resource.ClaudeRules     `hcl:"claude_rules,block"`
	Settings   []resource.ClaudeSettings  `hcl:"claude_settings,block"`
	MCPServers []resource.ClaudeMCPServer `hcl:"claude_mcp_server,block"`

	// Universal MCP servers
	UniversalMCPServers []resource.MCPServer `hcl:"mcp_server,block"`

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
}

// toResourceSet extracts the resource fields into a ResourceSet.
func (pb *ProfileBlock) toResourceSet() ResourceSet {
	return ResourceSet{
		Skills: pb.Skills, Commands: pb.Commands, Subagents: pb.Subagents,
		Rules: pb.Rules, RulesFiles: pb.RulesFiles, Settings: pb.Settings,
		MCPServers: pb.MCPServers, UniversalMCPServers: pb.UniversalMCPServers,
		CopilotInstruction: pb.CopilotInstruction, CopilotMCPServers: pb.CopilotMCPServers,
		CopilotInstructions: pb.CopilotInstructions, CopilotPrompts: pb.CopilotPrompts,
		CopilotAgents: pb.CopilotAgents, CopilotSkills: pb.CopilotSkills,
		CursorRules_: pb.CursorRules_, CursorMCPServers: pb.CursorMCPServers,
		CursorRules: pb.CursorRules, CursorCommands: pb.CursorCommands,
	}
}

// mergeByName merges overrides into defaults. If an override has the same name
// as a default (determined by getName), the override replaces that default.
// New overrides are appended.
func mergeByName[T any](defaults, overrides []T, getName func(T) string) []T {
	if len(overrides) == 0 {
		return defaults
	}

	// Build index of override names for quick lookup
	overrideNames := make(map[string]int, len(overrides))
	for i, o := range overrides {
		overrideNames[getName(o)] = i
	}

	// Replace matching defaults, track which overrides were used
	used := make(map[int]bool)
	result := make([]T, 0, len(defaults)+len(overrides))
	for _, d := range defaults {
		if idx, ok := overrideNames[getName(d)]; ok {
			result = append(result, overrides[idx])
			used[idx] = true
		} else {
			result = append(result, d)
		}
	}

	// Append overrides that didn't replace a default
	for i, o := range overrides {
		if !used[i] {
			result = append(result, o)
		}
	}

	return result
}

// ApplyProfile applies the named profile to this ProjectConfig.
// By default, profile contents are merged additively with defaults (same-name items
// are replaced). With exclude_defaults = true, defaults are cleared first.
// Returns an error if the profile is not found or if duplicate profile names exist.
func (p *ProjectConfig) ApplyProfile(name string) error {
	var profile *ProfileBlock
	seen := make(map[string]bool, len(p.Profiles))
	for i := range p.Profiles {
		if seen[p.Profiles[i].Name] {
			return fmt.Errorf("duplicate profile name: %s", p.Profiles[i].Name)
		}
		seen[p.Profiles[i].Name] = true
		if p.Profiles[i].Name == name {
			profile = &p.Profiles[i]
		}
	}

	if profile == nil {
		available := make([]string, 0, len(p.Profiles))
		for i := range p.Profiles {
			available = append(available, p.Profiles[i].Name)
		}
		return fmt.Errorf("profile %q not found; available profiles: %s", name, strings.Join(available, ", "))
	}

	profResources := profile.toResourceSet()

	if profile.ExcludeDefaults {
		p.Registries = profile.Registries
		p.Plugins = profile.Plugins
		p.Project.AgentInstructions = profile.AgentInstructions
		p.applyResourceSet(&profResources)
	} else {
		p.Registries = mergeByName(p.Registries, profile.Registries, func(r RegistryBlock) string { return r.Name })
		p.Plugins = mergeByName(p.Plugins, profile.Plugins, func(pl PluginBlock) string { return pl.Name })
		if profile.AgentInstructions != "" {
			p.Project.AgentInstructions = profile.AgentInstructions
		}
		rs := p.toResourceSet()
		rs.mergeFrom(&profResources)
		p.applyResourceSet(&rs)
	}

	// Clear profiles to prevent double-apply
	p.Profiles = nil

	return nil
}

// LoadProjectWithProfile loads a dex.hcl file and optionally applies a named profile.
// If profile is empty, it behaves identically to LoadProject.
func LoadProjectWithProfile(dir string, profile string) (*ProjectConfig, error) {
	cfg, err := LoadProject(dir)
	if err != nil {
		return nil, err
	}

	if profile != "" {
		if err := cfg.ApplyProfile(profile); err != nil {
			return nil, err
		}
		cfg.buildResources()
	}

	return cfg, nil
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
	ctx := NewProjectEvalContext(dir, resolvedVars)
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

// toResourceSet extracts the resource fields into a ResourceSet.
func (p *ProjectConfig) toResourceSet() ResourceSet {
	return ResourceSet{
		Skills: p.Skills, Commands: p.Commands, Subagents: p.Subagents,
		Rules: p.Rules, RulesFiles: p.RulesFiles, Settings: p.Settings,
		MCPServers: p.MCPServers, UniversalMCPServers: p.UniversalMCPServers,
		CopilotInstruction: p.CopilotInstruction, CopilotMCPServers: p.CopilotMCPServers,
		CopilotInstructions: p.CopilotInstructions, CopilotPrompts: p.CopilotPrompts,
		CopilotAgents: p.CopilotAgents, CopilotSkills: p.CopilotSkills,
		CursorRules_: p.CursorRules_, CursorMCPServers: p.CursorMCPServers,
		CursorRules: p.CursorRules, CursorCommands: p.CursorCommands,
	}
}

// applyResourceSet writes the ResourceSet fields back into ProjectConfig.
func (p *ProjectConfig) applyResourceSet(r *ResourceSet) {
	p.Skills = r.Skills
	p.Commands = r.Commands
	p.Subagents = r.Subagents
	p.Rules = r.Rules
	p.RulesFiles = r.RulesFiles
	p.Settings = r.Settings
	p.MCPServers = r.MCPServers
	p.UniversalMCPServers = r.UniversalMCPServers
	p.CopilotInstruction = r.CopilotInstruction
	p.CopilotMCPServers = r.CopilotMCPServers
	p.CopilotInstructions = r.CopilotInstructions
	p.CopilotPrompts = r.CopilotPrompts
	p.CopilotAgents = r.CopilotAgents
	p.CopilotSkills = r.CopilotSkills
	p.CursorRules_ = r.CursorRules_
	p.CursorMCPServers = r.CursorMCPServers
	p.CursorRules = r.CursorRules
	p.CursorCommands = r.CursorCommands
}

// buildResources populates the Resources slice from the typed resource fields.
func (p *ProjectConfig) buildResources() {
	rs := p.toResourceSet()
	p.Resources = rs.buildResources()
}

// toLocalConfig extracts the resource slices from this ProjectConfig into a LocalConfig
// so that MergeLocal can delegate to LocalConfig.merge (the single merge source of truth).
// ResolvedVars is intentionally omitted: MergeLocal handles var precedence separately.
func (p *ProjectConfig) toLocalConfig() *LocalConfig {
	rs := p.toResourceSet()
	lc := &LocalConfig{
		Registries: p.Registries,
		Plugins:    p.Plugins,
		Variables:  p.Variables,
	}
	lc.applyResourceSet(&rs)
	return lc
}

// applyLocalConfig writes the merged resource slices from l back into this ProjectConfig.
// Used by MergeLocal after delegating to LocalConfig.merge.
// ResolvedVars is intentionally omitted: MergeLocal handles var precedence separately.
// IMPORTANT: this leaves p.Resources stale. Callers must call p.buildResources() afterward.
// applyLocalConfig is unexported to enforce this — MergeLocal is the only call site.
func (p *ProjectConfig) applyLocalConfig(l *LocalConfig) {
	p.Registries = l.Registries
	p.Plugins = l.Plugins
	rs := l.toResourceSet()
	p.applyResourceSet(&rs)
	p.Variables = l.Variables
}

// MergeLocal appends all resources from a LocalConfig into this ProjectConfig and
// rebuilds the unified Resources slice. Resource slice merging is delegated to
// LocalConfig.merge so the merge logic lives in one place. Project-defined resolved
// vars take precedence over local config vars.
func (p *ProjectConfig) MergeLocal(local *LocalConfig) {
	base := p.toLocalConfig()
	base.merge(local)
	p.applyLocalConfig(base)

	// Var precedence at the project level: skip-if-exists (project wins).
	// This is asymmetric with LocalConfig.merge, which uses last-writer-wins within
	// local configs. The full chain is intentional:
	//   project vars > per-project local vars > global local vars
	// merge() above has already collapsed global+per-project into local using
	// last-writer-wins, so here we only copy vars not already set by the project.
	if p.ResolvedVars == nil {
		p.ResolvedVars = make(map[string]string)
	}
	for k, v := range local.ResolvedVars {
		if _, exists := p.ResolvedVars[k]; !exists {
			p.ResolvedVars[k] = v
		}
	}

	p.buildResources()
}

// Validate checks the project config for errors.
// It ensures required fields are present and values are valid.
func (p *ProjectConfig) Validate() error {
	// Validate project block
	// Name is optional - will default to directory name if not specified
	if p.Project.AgenticPlatform == "" {
		return fmt.Errorf("project.default_platform is required")
	}

	// Validate profiles
	profileNames := make(map[string]bool)
	for _, prof := range p.Profiles {
		if prof.Name == "" {
			return fmt.Errorf("profile name is required")
		}
		if profileNames[prof.Name] {
			return fmt.Errorf("duplicate profile name: %s", prof.Name)
		}
		profileNames[prof.Name] = true
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

		// Cannot have both source and registry
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

// AddPlugin adds a plugin block with a source to the dex.hcl file.
// It appends the plugin block to the end of the file.
func AddPlugin(dir string, name string, source string, version string) error {
	return AddPluginToConfig(dir, name, source, "", version)
}

// AddPluginToConfig adds a plugin block to the dex.hcl file.
// Supports both source-based and registry-based plugins.
// Exactly one of source or registryName must be non-empty.
func AddPluginToConfig(dir, name, source, registryName, version string) error {
	if source == "" && registryName == "" {
		return fmt.Errorf("either source or registry must be specified for plugin %q", name)
	}

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
	if source != "" {
		if version != "" {
			pluginBlock = fmt.Sprintf("\nplugin %q {\n  source  = %q\n  version = %q\n}\n", name, source, version)
		} else {
			pluginBlock = fmt.Sprintf("\nplugin %q {\n  source = %q\n}\n", name, source)
		}
	} else {
		if version != "" {
			pluginBlock = fmt.Sprintf("\nplugin %q {\n  registry = %q\n  version  = %q\n}\n", name, registryName, version)
		} else {
			pluginBlock = fmt.Sprintf("\nplugin %q {\n  registry = %q\n}\n", name, registryName)
		}
	}

	// Append to file
	newContent := string(content) + pluginBlock
	if err := os.WriteFile(filename, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", filename, err)
	}

	return nil
}

// AddRegistry adds a registry block to the dex.hcl file.
// It appends the registry block to the end of the file.
// Exactly one of url or path must be provided.
func AddRegistry(dir string, name string, url string, path string, force bool) error {
	// Validate exactly one of url or path is provided
	if url == "" && path == "" {
		return fmt.Errorf("exactly one of --url or --local must be provided")
	}
	if url != "" && path != "" {
		return fmt.Errorf("cannot specify both --url and --local")
	}

	filename := filepath.Join(dir, "dex.hcl")

	// Read existing content
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", filename, err)
	}

	// Check if registry already exists
	existingConfig, err := LoadProject(dir)
	if err == nil {
		for _, r := range existingConfig.Registries {
			if r.Name == name {
				if !force {
					return fmt.Errorf("registry %q already exists; use --force to overwrite", name)
				}
				// Remove the existing registry block using regex
				re := regexp.MustCompile(`(?m)\n?registry\s+"` + regexp.QuoteMeta(name) + `"\s*\{[^}]*\}\n?`)
				content = re.ReplaceAll(content, []byte(""))
				break
			}
		}
	}

	// Build the registry block
	var registryBlock string
	if url != "" {
		registryBlock = fmt.Sprintf("\nregistry %q {\n  url = %q\n}\n", name, url)
	} else {
		registryBlock = fmt.Sprintf("\nregistry %q {\n  path = %q\n}\n", name, path)
	}

	// Append to file
	newContent := string(content) + registryBlock
	if err := os.WriteFile(filename, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", filename, err)
	}

	return nil
}
