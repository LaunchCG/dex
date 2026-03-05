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

// merge appends all slices from src into dst.
// This is the single source of truth for LocalConfig field merging.
// When adding a new field here, also update ProjectConfig.toLocalConfig and
// ProjectConfig.applyLocalConfig in project.go. Run TestMergeLocal_AllResourceFields
// to verify all fields are wired correctly.
func (dst *LocalConfig) merge(src *LocalConfig) {
	dst.Registries = append(dst.Registries, src.Registries...)
	dst.Plugins = append(dst.Plugins, src.Plugins...)
	dst.Skills = append(dst.Skills, src.Skills...)
	dst.Commands = append(dst.Commands, src.Commands...)
	dst.Subagents = append(dst.Subagents, src.Subagents...)
	dst.Rules = append(dst.Rules, src.Rules...)
	dst.RulesFiles = append(dst.RulesFiles, src.RulesFiles...)
	dst.Settings = append(dst.Settings, src.Settings...)
	dst.MCPServers = append(dst.MCPServers, src.MCPServers...)
	dst.UniversalMCPServers = append(dst.UniversalMCPServers, src.UniversalMCPServers...)
	dst.CopilotInstruction = append(dst.CopilotInstruction, src.CopilotInstruction...)
	dst.CopilotMCPServers = append(dst.CopilotMCPServers, src.CopilotMCPServers...)
	dst.CopilotInstructions = append(dst.CopilotInstructions, src.CopilotInstructions...)
	dst.CopilotPrompts = append(dst.CopilotPrompts, src.CopilotPrompts...)
	dst.CopilotAgents = append(dst.CopilotAgents, src.CopilotAgents...)
	dst.CopilotSkills = append(dst.CopilotSkills, src.CopilotSkills...)
	dst.CursorRules_ = append(dst.CursorRules_, src.CursorRules_...)
	dst.CursorMCPServers = append(dst.CursorMCPServers, src.CursorMCPServers...)
	dst.CursorRules = append(dst.CursorRules, src.CursorRules...)
	dst.CursorCommands = append(dst.CursorCommands, src.CursorCommands...)
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
