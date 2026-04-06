package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/launchcg/dex/internal/resource"
)

// LocalConfig represents optional user-level configuration that augments a project config.
// It mirrors ProjectConfig but without the required project {} block.
// Files are loaded from ~/.dex/local.hcl and ~/.dex/projects/<name>/project.hcl.
type LocalConfig struct {
	// Registries defines additional plugin registry sources
	Registries []RegistryBlock `hcl:"registry,block"`

	// Plugins defines additional plugin dependencies
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

	// Variables defines user-configurable variables
	Variables    []ProjectVariableBlock
	ResolvedVars map[string]string
}

// toResourceSet extracts the resource fields into a ResourceSet.
func (l *LocalConfig) toResourceSet() ResourceSet {
	return ResourceSet{
		Skills: l.Skills, Commands: l.Commands, Subagents: l.Subagents,
		Rules: l.Rules, RulesFiles: l.RulesFiles, Settings: l.Settings,
		MCPServers: l.MCPServers, UniversalMCPServers: l.UniversalMCPServers,
		CopilotInstruction: l.CopilotInstruction, CopilotMCPServers: l.CopilotMCPServers,
		CopilotInstructions: l.CopilotInstructions, CopilotPrompts: l.CopilotPrompts,
		CopilotAgents: l.CopilotAgents, CopilotSkills: l.CopilotSkills,
		CursorRules_: l.CursorRules_, CursorMCPServers: l.CursorMCPServers,
		CursorRules: l.CursorRules, CursorCommands: l.CursorCommands,
	}
}

// applyResourceSet writes the ResourceSet fields back into LocalConfig.
func (l *LocalConfig) applyResourceSet(r *ResourceSet) {
	l.Skills = r.Skills
	l.Commands = r.Commands
	l.Subagents = r.Subagents
	l.Rules = r.Rules
	l.RulesFiles = r.RulesFiles
	l.Settings = r.Settings
	l.MCPServers = r.MCPServers
	l.UniversalMCPServers = r.UniversalMCPServers
	l.CopilotInstruction = r.CopilotInstruction
	l.CopilotMCPServers = r.CopilotMCPServers
	l.CopilotInstructions = r.CopilotInstructions
	l.CopilotPrompts = r.CopilotPrompts
	l.CopilotAgents = r.CopilotAgents
	l.CopilotSkills = r.CopilotSkills
	l.CursorRules_ = r.CursorRules_
	l.CursorMCPServers = r.CursorMCPServers
	l.CursorRules = r.CursorRules
	l.CursorCommands = r.CursorCommands
}

// merge appends all slices from src into dst.
// Resource fields are handled via ResourceSet.appendFrom. When adding a new resource
// type, update ResourceSet and its methods in project.go, then add toResourceSet/
// applyResourceSet mappings. Run TestMergeLocal_AllResourceFields to verify.
func (dst *LocalConfig) merge(src *LocalConfig) {
	dst.Registries = append(dst.Registries, src.Registries...)
	dst.Plugins = append(dst.Plugins, src.Plugins...)
	dstRS := dst.toResourceSet()
	srcRS := src.toResourceSet()
	dstRS.appendFrom(&srcRS)
	dst.applyResourceSet(&dstRS)
	dst.Variables = append(dst.Variables, src.Variables...)

	// Var precedence within a LocalConfig merge: last writer wins.
	// merge() is called as merge(projectCfg) where dst=global, src=per-project,
	// so per-project vars override global vars — intentionally.
	// Note: this differs from MergeLocal (in project.go), where project-defined
	// vars win over all local config vars (skip-if-exists semantics). The full
	// precedence chain is: project vars > per-project local vars > global local vars.
	if dst.ResolvedVars == nil {
		dst.ResolvedVars = make(map[string]string)
	}
	for k, v := range src.ResolvedVars {
		dst.ResolvedVars[k] = v
	}
}

// loadLocalConfigFile parses a single local HCL config file.
// configDir is the directory containing the file, used for file()/templatefile() resolution.
func loadLocalConfigFile(path, configDir string) (*LocalConfig, error) {
	parser := NewParser()
	file, diags := parser.ParseFile(path)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse %s: %s", path, diags.Error())
	}

	// Two-pass parsing: extract variables first, then decode remaining body
	variables, resolvedVars, remain, err := extractAndResolveProjectVariables(file.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve variables in %s: %w", path, err)
	}

	ctx := NewProjectEvalContext(configDir, resolvedVars)
	var cfg LocalConfig
	diags = DecodeBody(remain, ctx, &cfg)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to decode %s: %s", path, diags.Error())
	}

	cfg.Variables = variables
	cfg.ResolvedVars = resolvedVars
	return &cfg, nil
}

// fileExists returns true if path exists. Returns an error for non-ErrNotExist failures
// (e.g., permission denied) so callers don't silently skip files they can't access.
func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

// LoadLocalConfigs loads and merges user-level local configs:
//  1. ~/.dex/local.hcl (global, if exists)
//  2. ~/.dex/projects/<projectName>/project.hcl (per-project, if exists)
//
// Returns nil (not an error) if neither file exists.
//
// projectName must be non-empty and must not contain path separators. In
// practice it comes from ProjectBlock.Name which is validated before this is
// called, but callers should ensure this invariant holds.
func LoadLocalConfigs(projectName string) (*LocalConfig, error) {
	if projectName == "" {
		return nil, fmt.Errorf("project name must not be empty")
	}
	if strings.ContainsAny(projectName, `/\`) {
		return nil, fmt.Errorf("project name must not contain path separators, got %q", projectName)
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to determine home directory: %w", err)
	}

	dexDir := filepath.Join(homeDir, ".dex")
	globalPath := filepath.Join(dexDir, "local.hcl")
	projectPath := filepath.Join(dexDir, "projects", projectName, "project.hcl")

	var merged *LocalConfig

	// Load global config (~/.dex/local.hcl)
	exists, err := fileExists(globalPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat %s: %w", globalPath, err)
	}
	if exists {
		globalCfg, err := loadLocalConfigFile(globalPath, filepath.Dir(globalPath))
		if err != nil {
			return nil, fmt.Errorf("failed to load global local config: %w", err)
		}
		merged = globalCfg
	}

	// Load per-project config (~/.dex/projects/<name>/project.hcl)
	exists, err = fileExists(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat %s: %w", projectPath, err)
	}
	if exists {
		projectCfg, err := loadLocalConfigFile(projectPath, filepath.Dir(projectPath))
		if err != nil {
			return nil, fmt.Errorf("failed to load project local config: %w", err)
		}
		if merged == nil {
			merged = projectCfg
		} else {
			merged.merge(projectCfg)
		}
	}

	return merged, nil
}
