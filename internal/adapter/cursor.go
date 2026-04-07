package adapter

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"strings"

	"github.com/launchcg/dex/internal/config"
	"github.com/launchcg/dex/internal/resource"
)

// CursorAdapter implements the Adapter interface for Cursor.
// It handles installation of resources into the .cursor directory structure.
type CursorAdapter struct{}

func init() {
	Register("cursor", &CursorAdapter{})
}

// Name returns "cursor".
func (a *CursorAdapter) Name() string {
	return "cursor"
}

// BaseDir returns ".cursor" joined with the root path.
func (a *CursorAdapter) BaseDir(root string) string {
	return filepath.Join(root, ".cursor")
}

// SkillsDir returns ".cursor/skills" joined with the root path.
// Note: Cursor doesn't use skills, but this is required by the interface.
func (a *CursorAdapter) SkillsDir(root string) string {
	return filepath.Join(root, ".cursor", "skills")
}

// CommandsDir returns ".cursor/commands" joined with the root path.
func (a *CursorAdapter) CommandsDir(root string) string {
	return filepath.Join(root, ".cursor", "commands")
}

// SubagentsDir returns ".cursor/agents" joined with the root path.
// Note: Cursor doesn't use agents, but this is required by the interface.
func (a *CursorAdapter) SubagentsDir(root string) string {
	return filepath.Join(root, ".cursor", "agents")
}

// RulesDir returns ".cursor/rules" joined with the root path.
func (a *CursorAdapter) RulesDir(root string) string {
	return filepath.Join(root, ".cursor", "rules")
}

// PlanInstallation dispatches to the appropriate planner based on resource type.
func (a *CursorAdapter) PlanInstallation(res resource.Resource, pkg *config.PackageConfig, pkgDir, projectRoot string, ctx *InstallContext) (*Plan, error) {
	switch r := res.(type) {
	// Universal resource types (translate to Cursor-specific)
	case *resource.MCPServer:
		translated := resource.TranslateToCursorMCPServer(r)
		if translated == nil {
			slog.Warn("resource skipped: disabled for platform", "resource", r.Name, "type", "mcp_server", "platform", "cursor")
			return &Plan{}, nil
		}
		return a.planMCPServer(translated, pkg, pkgDir, projectRoot, ctx)
	case *resource.Command:
		translated := resource.TranslateToCursorCommand(r)
		if translated == nil {
			slog.Warn("resource skipped: disabled for platform", "resource", r.Name, "type", "command", "platform", "cursor")
			return &Plan{}, nil
		}
		return a.planCommand(translated, pkg, pkgDir, projectRoot, ctx)
	case *resource.Rule:
		translated := resource.TranslateToCursorRule(r)
		if translated == nil {
			slog.Warn("resource skipped: disabled for platform", "resource", r.Name, "type", "rule", "platform", "cursor")
			return &Plan{}, nil
		}
		return a.planRule(translated, pkg, pkgDir, projectRoot, ctx)
	case *resource.Rules:
		translated := resource.TranslateToCursorRules(r)
		if translated == nil {
			slog.Warn("resource skipped: disabled for platform", "resource", r.Name, "type", "rules", "platform", "cursor")
			return &Plan{}, nil
		}
		return a.planRules(translated, pkg, pkgDir, projectRoot, ctx)
	case *resource.Skill:
		slog.Warn("resource skipped: not supported by platform", "resource", r.Name, "type", "skill", "platform", "cursor")
		return &Plan{}, nil
	case *resource.Agent:
		slog.Warn("resource skipped: not supported by platform", "resource", r.Name, "type", "agent", "platform", "cursor")
		return &Plan{}, nil
	case *resource.Settings:
		slog.Warn("resource skipped: not supported by platform", "resource", r.Name, "type", "settings", "platform", "cursor")
		return &Plan{}, nil

	// Platform-specific types (used internally by translators)
	case *resource.CursorRule:
		return a.planRule(r, pkg, pkgDir, projectRoot, ctx)
	case *resource.CursorMCPServer:
		return a.planMCPServer(r, pkg, pkgDir, projectRoot, ctx)
	case *resource.CursorRules:
		return a.planRules(r, pkg, pkgDir, projectRoot, ctx)
	case *resource.CursorCommand:
		return a.planCommand(r, pkg, pkgDir, projectRoot, ctx)

	// Universal file/directory resources
	case *resource.File:
		return PlanUniversalFile(r, pkg, pkgDir, projectRoot, "cursor", ctx)
	case *resource.Directory:
		return PlanUniversalDirectory(r, pkg, ctx)

	default:
		return nil, fmt.Errorf("unsupported resource type for cursor adapter: %T", res)
	}
}

// GenerateFrontmatter generates YAML frontmatter for a resource.
func (a *CursorAdapter) GenerateFrontmatter(res resource.Resource, pkg *config.PackageConfig) string {
	switch r := res.(type) {
	case *resource.CursorRules:
		return a.generateRulesFrontmatter(r, pkg)
	case *resource.CursorCommand:
		return a.generateCommandFrontmatter(r, pkg)
	default:
		return ""
	}
}

// MergeMCPConfig merges MCP servers into .cursor/mcp.json format.
// Format: {"mcpServers": {"name": {"command": "...", "args": [...], "env": {...}}}}
// Note: This method signature accepts ClaudeMCPServer for interface compatibility,
// but Cursor resources should use planMCPServer directly.
func (a *CursorAdapter) MergeMCPConfig(existing map[string]any, pkgName string, servers []*resource.ClaudeMCPServer) map[string]any {
	// This method is kept for interface compatibility but Cursor uses its own server type
	// See MergeCursorMCPConfig for the actual implementation
	return existing
}

// MergeCursorMCPConfig merges Cursor MCP servers into .cursor/mcp.json format.
// Format: {"mcpServers": {"name": {"command": "...", "args": [...], "env": {...}}}}
func (a *CursorAdapter) MergeCursorMCPConfig(existing map[string]any, pkgName string, servers []*resource.CursorMCPServer) map[string]any {
	if existing == nil {
		existing = make(map[string]any)
	}

	// Get or create the mcpServers map
	mcpServers, ok := existing["mcpServers"].(map[string]any)
	if !ok {
		mcpServers = make(map[string]any)
	}

	// Add each server
	for _, server := range servers {
		serverConfig := make(map[string]any)

		if server.Type == "stdio" {
			if server.Command != "" {
				serverConfig["command"] = server.Command
			}
			if len(server.Args) > 0 {
				serverConfig["args"] = server.Args
			}
			if len(server.Env) > 0 {
				serverConfig["env"] = server.Env
			}
			if server.EnvFile != "" {
				serverConfig["envFile"] = server.EnvFile
			}
		} else if server.Type == "http" || server.Type == "sse" {
			serverConfig["url"] = server.URL
			if len(server.Headers) > 0 {
				serverConfig["headers"] = server.Headers
			}
		}

		mcpServers[server.Name] = serverConfig
	}

	existing["mcpServers"] = mcpServers
	return existing
}

// MergeSettingsConfig is not used for Cursor (no settings.json equivalent).
// This method is kept for interface compatibility.
func (a *CursorAdapter) MergeSettingsConfig(existing map[string]any, settings *resource.ClaudeSettings) map[string]any {
	return existing
}

// planRule creates an installation plan for a Cursor rule (singular).
// Rules are merged into AGENTS.md with markers.
func (a *CursorAdapter) planRule(rule *resource.CursorRule, pkg *config.PackageConfig, pkgDir, root string, ctx *InstallContext) (*Plan, error) {
	plan := NewPlan(pkg.Meta.Name)

	// Rules are merged into AGENTS.md via AgentFileContent
	plan.AgentFileContent = rule.Content
	plan.AgentFilePath = "AGENTS.md"

	return plan, nil
}

// planMCPServer creates an installation plan for a Cursor MCP server.
// MCP servers are merged into .cursor/mcp.json with optional namespacing
func (a *CursorAdapter) planMCPServer(server *resource.CursorMCPServer, pkg *config.PackageConfig, pkgDir, root string, ctx *InstallContext) (*Plan, error) {
	plan := NewPlan(pkg.Meta.Name)

	// Apply namespacing to server name if requested
	serverName := server.Name
	if ctx != nil && ctx.Namespace {
		serverName = fmt.Sprintf("%s-%s", pkg.Meta.Name, server.Name)
	}

	// Create a copy of the server with the potentially namespaced name
	namespacedServer := *server
	namespacedServer.Name = serverName

	// MCP servers are merged via MCPEntries
	plan.MCPEntries = a.MergeCursorMCPConfig(nil, pkg.Meta.Name, []*resource.CursorMCPServer{&namespacedServer})

	// Set Cursor-specific MCP config path and key
	plan.MCPPath = filepath.Join(".cursor", "mcp.json")
	plan.MCPKey = "mcpServers"

	return plan, nil
}

// planRules creates an installation plan for Cursor rules (plural).
// Rules files are installed to .cursor/rules/{{pkg}-}{name}.mdc (namespaced or not)
func (a *CursorAdapter) planRules(rules *resource.CursorRules, pkg *config.PackageConfig, pkgDir, root string, ctx *InstallContext) (*Plan, error) {
	plan := NewPlan(pkg.Meta.Name)

	// Create rules directory
	rulesDir := filepath.Join(".cursor", "rules")
	plan.AddDirectory(rulesDir, true)

	// Generate frontmatter and content
	var content string
	if hasFrontmatter(rules.Content) {
		content = rules.Content
	} else {
		frontmatter := a.generateRulesFrontmatter(rules, pkg)
		content = frontmatter + rules.Content
	}

	// Add rules file with optional namespacing
	var fileName string
	if ctx != nil && ctx.Namespace {
		fileName = fmt.Sprintf("%s-%s.mdc", pkg.Meta.Name, rules.Name)
	} else {
		fileName = fmt.Sprintf("%s.mdc", rules.Name)
	}
	rulesFile := filepath.Join(rulesDir, fileName)
	plan.AddFile(rulesFile, content, "")

	return plan, nil
}

// planCommand creates an installation plan for a Cursor command.
// Commands are installed to .cursor/commands/{{pkg}-}{name}.md (namespaced or not)
func (a *CursorAdapter) planCommand(cmd *resource.CursorCommand, pkg *config.PackageConfig, pkgDir, root string, ctx *InstallContext) (*Plan, error) {
	plan := NewPlan(pkg.Meta.Name)

	// Create commands directory
	commandsDir := filepath.Join(".cursor", "commands")
	plan.AddDirectory(commandsDir, true)

	// Generate frontmatter and content
	var content string
	if hasFrontmatter(cmd.Content) {
		content = cmd.Content
	} else {
		frontmatter := a.generateCommandFrontmatter(cmd, pkg)
		content = frontmatter + cmd.Content
	}

	// Add command file with optional namespacing
	var fileName string
	if ctx != nil && ctx.Namespace {
		fileName = fmt.Sprintf("%s-%s.md", pkg.Meta.Name, cmd.Name)
	} else {
		fileName = fmt.Sprintf("%s.md", cmd.Name)
	}
	commandFile := filepath.Join(commandsDir, fileName)
	plan.AddFile(commandFile, content, "")

	return plan, nil
}

// generateRulesFrontmatter generates YAML frontmatter for standalone rules.
// Uses .mdc format which supports frontmatter.
func (a *CursorAdapter) generateRulesFrontmatter(rules *resource.CursorRules, pkg *config.PackageConfig) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("description: %s\n", rules.Description))

	if len(rules.Globs) > 0 {
		b.WriteString("globs:\n")
		for _, glob := range rules.Globs {
			b.WriteString(fmt.Sprintf("- %s\n", glob))
		}
	}

	if rules.AlwaysApply != nil && *rules.AlwaysApply {
		b.WriteString("alwaysApply: true\n")
	}

	b.WriteString("---\n")
	return b.String()
}

// generateCommandFrontmatter generates YAML frontmatter for a command.
// Commands in Cursor use plain markdown, but we add description in frontmatter.
func (a *CursorAdapter) generateCommandFrontmatter(cmd *resource.CursorCommand, pkg *config.PackageConfig) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("description: %s\n", cmd.Description))
	b.WriteString("---\n")
	return b.String()
}
