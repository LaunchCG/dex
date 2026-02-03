package adapter

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/launchcg/dex/internal/config"
	"github.com/launchcg/dex/internal/resource"
	"github.com/launchcg/dex/internal/template"
)

// ClaudeAdapter implements the Adapter interface for Claude Code.
// It handles installation of resources into the .claude directory structure.
type ClaudeAdapter struct{}

func init() {
	Register("claude-code", &ClaudeAdapter{})
}

// Name returns "claude-code".
func (a *ClaudeAdapter) Name() string {
	return "claude-code"
}

// BaseDir returns ".claude" joined with the root path.
func (a *ClaudeAdapter) BaseDir(root string) string {
	return filepath.Join(root, ".claude")
}

// SkillsDir returns ".claude/skills" joined with the root path.
func (a *ClaudeAdapter) SkillsDir(root string) string {
	return filepath.Join(root, ".claude", "skills")
}

// CommandsDir returns ".claude/commands" joined with the root path.
func (a *ClaudeAdapter) CommandsDir(root string) string {
	return filepath.Join(root, ".claude", "commands")
}

// SubagentsDir returns ".claude/agents" joined with the root path.
func (a *ClaudeAdapter) SubagentsDir(root string) string {
	return filepath.Join(root, ".claude", "agents")
}

// RulesDir returns ".claude/rules" joined with the root path.
func (a *ClaudeAdapter) RulesDir(root string) string {
	return filepath.Join(root, ".claude", "rules")
}

// PlanInstallation dispatches to the appropriate planner based on resource type.
func (a *ClaudeAdapter) PlanInstallation(res resource.Resource, pkg *config.PackageConfig, pluginDir, projectRoot string, ctx *InstallContext) (*Plan, error) {
	switch r := res.(type) {
	// Unified MCP server (translate to Claude-specific)
	case *resource.MCPServer:
		claudeServer := resource.TranslateToClaudeMCPServer(r)
		if claudeServer == nil {
			// Server is disabled for Claude platform
			return &Plan{}, nil
		}
		return a.planMCPServer(claudeServer, pkg, pluginDir, projectRoot, ctx)

	case *resource.ClaudeSkill:
		return a.planSkill(r, pkg, pluginDir, projectRoot, ctx)
	case *resource.ClaudeCommand:
		return a.planCommand(r, pkg, pluginDir, projectRoot, ctx)
	case *resource.ClaudeSubagent:
		return a.planSubagent(r, pkg, pluginDir, projectRoot, ctx)
	case *resource.ClaudeRule:
		return a.planRule(r, pkg, pluginDir, projectRoot, ctx)
	case *resource.ClaudeRules:
		return a.planRules(r, pkg, pluginDir, projectRoot, ctx)
	case *resource.ClaudeSettings:
		return a.planSettings(r, pkg, pluginDir, projectRoot, ctx)
	case *resource.ClaudeMCPServer:
		return a.planMCPServer(r, pkg, pluginDir, projectRoot, ctx)

	// Universal resources
	case *resource.File:
		return PlanUniversalFile(r, pkg, pluginDir, projectRoot, "claude-code", ctx)
	case *resource.Directory:
		return PlanUniversalDirectory(r, pkg, ctx)

	default:
		return nil, fmt.Errorf("unsupported resource type for claude-code adapter: %T", res)
	}
}

// GenerateFrontmatter generates YAML frontmatter for a resource.
func (a *ClaudeAdapter) GenerateFrontmatter(res resource.Resource, pkg *config.PackageConfig) string {
	switch r := res.(type) {
	case *resource.ClaudeSkill:
		return a.generateSkillFrontmatter(r, pkg)
	case *resource.ClaudeCommand:
		return a.generateCommandFrontmatter(r, pkg)
	case *resource.ClaudeSubagent:
		return a.generateSubagentFrontmatter(r, pkg)
	case *resource.ClaudeRules:
		return a.generateRulesFrontmatter(r, pkg)
	default:
		return ""
	}
}

// MergeMCPConfig merges MCP servers into .mcp.json format.
// Format: {"mcpServers": {"name": {"command": "...", "args": [...], "env": {...}}}}
func (a *ClaudeAdapter) MergeMCPConfig(existing map[string]any, pluginName string, servers []*resource.ClaudeMCPServer) map[string]any {
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

		if server.Type == "command" {
			if server.Command != "" {
				serverConfig["command"] = server.Command
			}
			if len(server.Args) > 0 {
				serverConfig["args"] = server.Args
			}
			if len(server.Env) > 0 {
				serverConfig["env"] = server.Env
			}
		} else if server.Type == "http" {
			serverConfig["url"] = server.URL
		}

		mcpServers[server.Name] = serverConfig
	}

	existing["mcpServers"] = mcpServers
	return existing
}

// MergeSettingsConfig merges settings into .claude/settings.json format.
func (a *ClaudeAdapter) MergeSettingsConfig(existing map[string]any, settings *resource.ClaudeSettings) map[string]any {
	if existing == nil {
		existing = make(map[string]any)
	}

	// Helper to merge string slices
	mergeSlice := func(key string, values []string) {
		if len(values) == 0 {
			return
		}
		existingSlice, _ := existing[key].([]any)
		// Convert to string set for deduplication
		seen := make(map[string]bool)
		for _, v := range existingSlice {
			if s, ok := v.(string); ok {
				seen[s] = true
			}
		}
		for _, v := range values {
			if !seen[v] {
				seen[v] = true
				existingSlice = append(existingSlice, v)
			}
		}
		if len(existingSlice) > 0 {
			existing[key] = existingSlice
		}
	}

	// Merge permission arrays
	mergeSlice("allow", settings.Allow)
	mergeSlice("ask", settings.Ask)
	mergeSlice("deny", settings.Deny)
	mergeSlice("enabledMcpServers", settings.EnabledMCPServers)
	mergeSlice("disabledMcpServers", settings.DisabledMCPServers)

	// Merge env map
	if len(settings.Env) > 0 {
		envMap, ok := existing["env"].(map[string]any)
		if !ok {
			envMap = make(map[string]any)
		}
		for k, v := range settings.Env {
			envMap[k] = v
		}
		existing["env"] = envMap
	}

	// Set boolean/string settings only if explicitly set
	if settings.EnableAllProjectMCPServers {
		existing["enableAllProjectMcpServers"] = true
	}
	if settings.RespectGitignore {
		existing["respectGitignore"] = true
	}
	if settings.IncludeCoAuthoredBy {
		existing["includeCoAuthoredBy"] = true
	}
	if settings.AlwaysThinkingEnabled {
		existing["alwaysThinkingEnabled"] = true
	}
	if settings.Model != "" {
		existing["model"] = settings.Model
	}
	if settings.OutputStyle != "" {
		existing["outputStyle"] = settings.OutputStyle
	}
	if settings.PlansDirectory != "" {
		existing["plansDirectory"] = settings.PlansDirectory
	}

	return existing
}

// MergeAgentFile merges rule content into CLAUDE.md with markers.
// Format:
// <!-- dex:{plugin-name} -->
// {content}
// <!-- /dex:{plugin-name} -->
func (a *ClaudeAdapter) MergeAgentFile(existing, pluginName, content string) string {
	startMarker := fmt.Sprintf("<!-- dex:%s -->", pluginName)
	endMarker := fmt.Sprintf("<!-- /dex:%s -->", pluginName)
	markedContent := fmt.Sprintf("%s\n%s\n%s", startMarker, content, endMarker)

	// Check if markers already exist
	pattern := regexp.MustCompile(fmt.Sprintf(`(?s)<!-- dex:%s -->.*?<!-- /dex:%s -->`, regexp.QuoteMeta(pluginName), regexp.QuoteMeta(pluginName)))

	if pattern.MatchString(existing) {
		// Replace existing marked section
		return pattern.ReplaceAllString(existing, markedContent)
	}

	// Append new section
	if existing == "" {
		return markedContent
	}
	return existing + "\n\n" + markedContent
}

// hasFrontmatter checks if content already starts with YAML frontmatter.
func hasFrontmatter(content string) bool {
	return strings.HasPrefix(strings.TrimSpace(content), "---")
}

// planSkill creates an installation plan for a Claude skill.
// Skills are installed to .claude/skills/{plugin}-{skill-name}/SKILL.md
func (a *ClaudeAdapter) planSkill(skill *resource.ClaudeSkill, pkg *config.PackageConfig, pluginDir, root string, ctx *InstallContext) (*Plan, error) {
	plan := NewPlan(pkg.Package.Name)

	// Create skill directory with optional namespacing
	var skillDirName string
	if ctx != nil && ctx.Namespace {
		skillDirName = fmt.Sprintf("%s-%s", pkg.Package.Name, skill.Name)
	} else {
		skillDirName = skill.Name
	}
	skillDir := filepath.Join(".claude", "skills", skillDirName)
	plan.AddDirectory(skillDir, true)

	// Use content as-is if it already has frontmatter, otherwise add ours
	var content string
	if hasFrontmatter(skill.Content) {
		content = skill.Content
	} else {
		frontmatter := a.generateSkillFrontmatter(skill, pkg)
		content = frontmatter + skill.Content
	}

	// Add main SKILL.md file
	skillFile := filepath.Join(skillDir, "SKILL.md")
	plan.AddFile(skillFile, content, "")

	// Copy nested files
	if err := a.planFiles(plan, skill.GetFiles(), pluginDir, skillDir); err != nil {
		return nil, fmt.Errorf("planning skill files: %w", err)
	}

	// Handle template files
	if err := a.planTemplateFiles(plan, skill.GetTemplateFiles(), pkg, pluginDir, skillDir, root); err != nil {
		return nil, fmt.Errorf("planning skill template files: %w", err)
	}

	return plan, nil
}

// planCommand creates an installation plan for a Claude command.
// Commands are installed to .claude/commands/{{plugin}-}{command}.md (namespaced or not)
func (a *ClaudeAdapter) planCommand(cmd *resource.ClaudeCommand, pkg *config.PackageConfig, pluginDir, root string, ctx *InstallContext) (*Plan, error) {
	plan := NewPlan(pkg.Package.Name)

	// Create commands directory
	commandsDir := filepath.Join(".claude", "commands")
	plan.AddDirectory(commandsDir, true)

	// Use content as-is if it already has frontmatter, otherwise add ours
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
		fileName = fmt.Sprintf("%s-%s.md", pkg.Package.Name, cmd.Name)
	} else {
		fileName = fmt.Sprintf("%s.md", cmd.Name)
	}
	cmdFile := filepath.Join(commandsDir, fileName)
	plan.AddFile(cmdFile, content, "")

	return plan, nil
}

// planSubagent creates an installation plan for a Claude subagent.
// Subagents are installed to .claude/agents/{{plugin}-}{agent}.md (namespaced or not)
func (a *ClaudeAdapter) planSubagent(agent *resource.ClaudeSubagent, pkg *config.PackageConfig, pluginDir, root string, ctx *InstallContext) (*Plan, error) {
	plan := NewPlan(pkg.Package.Name)

	// Create agents directory
	agentsDir := filepath.Join(".claude", "agents")
	plan.AddDirectory(agentsDir, true)

	// Use content as-is if it already has frontmatter, otherwise add ours
	var content string
	if hasFrontmatter(agent.Content) {
		content = agent.Content
	} else {
		frontmatter := a.generateSubagentFrontmatter(agent, pkg)
		content = frontmatter + agent.Content
	}

	// Add agent file with optional namespacing
	var fileName string
	if ctx != nil && ctx.Namespace {
		fileName = fmt.Sprintf("%s-%s.md", pkg.Package.Name, agent.Name)
	} else {
		fileName = fmt.Sprintf("%s.md", agent.Name)
	}
	agentFile := filepath.Join(agentsDir, fileName)
	plan.AddFile(agentFile, content, "")

	return plan, nil
}

// planRule creates an installation plan for a Claude rule (singular).
// Rules are merged into CLAUDE.md with markers.
func (a *ClaudeAdapter) planRule(rule *resource.ClaudeRule, pkg *config.PackageConfig, pluginDir, root string, ctx *InstallContext) (*Plan, error) {
	plan := NewPlan(pkg.Package.Name)

	// Rules are merged into CLAUDE.md via AgentFileContent
	// Content is used as-is; use templatefile() in HCL for templating
	plan.AgentFileContent = rule.Content

	return plan, nil
}

// planRules creates an installation plan for Claude rules (plural).
// Rules are installed to .claude/rules/{name}/ as a directory structure,
// similar to skills, allowing for multiple related rule files.
func (a *ClaudeAdapter) planRules(rules *resource.ClaudeRules, pkg *config.PackageConfig, pluginDir, root string, ctx *InstallContext) (*Plan, error) {
	plan := NewPlan(pkg.Package.Name)

	// Create rules subdirectory with optional namespacing
	var rulesDirName string
	if ctx != nil && ctx.Namespace {
		rulesDirName = fmt.Sprintf("%s-%s", pkg.Package.Name, rules.Name)
	} else {
		rulesDirName = rules.Name
	}
	rulesDir := filepath.Join(".claude", "rules", rulesDirName)
	plan.AddDirectory(rulesDir, true)

	// Only create main rules file if content is provided
	if rules.Content != "" {
		// Generate frontmatter and content for the main rules file
		// Content is used as-is; use templatefile() in HCL for templating
		// Skip frontmatter if content already has it
		var content string
		if hasFrontmatter(rules.Content) {
			content = rules.Content
		} else {
			frontmatter := a.generateRulesFrontmatter(rules, pkg)
			content = frontmatter + rules.Content
		}

		// Add main rules file (named after the resource)
		mainFile := filepath.Join(rulesDir, rules.Name+".md")
		plan.AddFile(mainFile, content, "")
	}

	// Copy nested files
	if err := a.planFiles(plan, rules.GetFiles(), pluginDir, rulesDir); err != nil {
		return nil, fmt.Errorf("planning rules files: %w", err)
	}

	// Handle template files
	if err := a.planTemplateFiles(plan, rules.GetTemplateFiles(), pkg, pluginDir, rulesDir, root); err != nil {
		return nil, fmt.Errorf("planning rules template files: %w", err)
	}

	return plan, nil
}

// planSettings creates an installation plan for Claude settings.
// Settings are merged into .claude/settings.json
func (a *ClaudeAdapter) planSettings(settings *resource.ClaudeSettings, pkg *config.PackageConfig, pluginDir, root string, ctx *InstallContext) (*Plan, error) {
	plan := NewPlan(pkg.Package.Name)

	// Settings are merged via SettingsEntries
	plan.SettingsEntries = a.MergeSettingsConfig(nil, settings)

	return plan, nil
}

// planMCPServer creates an installation plan for a Claude MCP server.
// MCP servers are merged into .mcp.json with optional namespacing
func (a *ClaudeAdapter) planMCPServer(server *resource.ClaudeMCPServer, pkg *config.PackageConfig, pluginDir, root string, ctx *InstallContext) (*Plan, error) {
	plan := NewPlan(pkg.Package.Name)

	// Apply namespacing to server name if requested
	serverName := server.Name
	if ctx != nil && ctx.Namespace {
		serverName = fmt.Sprintf("%s-%s", pkg.Package.Name, server.Name)
	}

	// Create a copy of the server with the potentially namespaced name
	namespacedServer := *server
	namespacedServer.Name = serverName

	// MCP servers are merged via MCPEntries
	plan.MCPEntries = a.MergeMCPConfig(nil, pkg.Package.Name, []*resource.ClaudeMCPServer{&namespacedServer})

	// Set Claude-specific MCP config path and key
	plan.MCPPath = ".mcp.json"
	plan.MCPKey = "mcpServers"

	return plan, nil
}

// planFiles adds file copy operations to the plan.
func (a *ClaudeAdapter) planFiles(plan *Plan, files []resource.FileBlock, pluginDir, destDir string) error {
	for _, file := range files {
		// Read source file
		srcPath := filepath.Join(pluginDir, file.Src)
		content, err := os.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("reading file %s: %w", file.Src, err)
		}

		// Determine destination filename
		destName := file.Dest
		if destName == "" {
			destName = filepath.Base(file.Src)
		}

		destPath := filepath.Join(destDir, destName)
		plan.AddFile(destPath, string(content), file.Chmod)
	}
	return nil
}

// planTemplateFiles adds template file operations to the plan.
// Template files are rendered using the template engine with context variables.
func (a *ClaudeAdapter) planTemplateFiles(plan *Plan, files []resource.TemplateFileBlock, pkg *config.PackageConfig, pluginDir, destDir, projectRoot string) error {
	// Create template context
	ctx := template.NewContext(pkg.Package.Name, pkg.Package.Version, projectRoot, "claude-code")
	ctx.WithComponentDir(filepath.Join(projectRoot, destDir))
	engine := template.NewEngine(pluginDir, ctx)

	for _, file := range files {
		// Convert file.Vars to map[string]any
		vars := make(map[string]any)
		for k, v := range file.Vars {
			vars[k] = v
		}

		// Render the template file with the additional vars
		content, err := engine.RenderFileWithVars(file.Src, vars)
		if err != nil {
			return fmt.Errorf("rendering template %s: %w", file.Src, err)
		}

		// Determine destination filename (remove .tmpl suffix if present)
		destName := file.Dest
		if destName == "" {
			destName = filepath.Base(file.Src)
			destName = strings.TrimSuffix(destName, ".tmpl")
		}

		destPath := filepath.Join(destDir, destName)
		plan.AddFile(destPath, content, file.Chmod)
	}
	return nil
}

// generateSkillFrontmatter generates YAML frontmatter for a skill.
func (a *ClaudeAdapter) generateSkillFrontmatter(skill *resource.ClaudeSkill, pkg *config.PackageConfig) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("name: %s\n", skill.Name))
	b.WriteString(fmt.Sprintf("description: %s\n", skill.Description))

	if skill.ArgumentHint != "" {
		b.WriteString(fmt.Sprintf("argument-hint: %s\n", skill.ArgumentHint))
	}
	if skill.DisableModelInvocation {
		b.WriteString("disable-model-invocation: true\n")
	}
	if skill.UserInvocable != nil && !*skill.UserInvocable {
		b.WriteString("user-invocable: false\n")
	}
	if len(skill.AllowedTools) > 0 {
		b.WriteString("allowed-tools:\n")
		for _, tool := range skill.AllowedTools {
			b.WriteString(fmt.Sprintf("- %s\n", tool))
		}
	}
	if skill.Model != "" {
		b.WriteString(fmt.Sprintf("model: %s\n", skill.Model))
	}
	if skill.Context != "" {
		b.WriteString(fmt.Sprintf("context: %s\n", skill.Context))
	}
	if skill.Agent != "" {
		b.WriteString(fmt.Sprintf("agent: %s\n", skill.Agent))
	}

	// Add any additional metadata
	for k, v := range skill.Metadata {
		b.WriteString(fmt.Sprintf("%s: %s\n", k, v))
	}

	b.WriteString("---\n")
	return b.String()
}

// generateCommandFrontmatter generates YAML frontmatter for a command.
func (a *ClaudeAdapter) generateCommandFrontmatter(cmd *resource.ClaudeCommand, pkg *config.PackageConfig) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("name: %s\n", cmd.Name))
	b.WriteString(fmt.Sprintf("description: %s\n", cmd.Description))

	if cmd.ArgumentHint != "" {
		b.WriteString(fmt.Sprintf("argument_hint: %s\n", cmd.ArgumentHint))
	}

	if len(cmd.AllowedTools) > 0 {
		b.WriteString("allowed_tools:\n")
		for _, tool := range cmd.AllowedTools {
			b.WriteString(fmt.Sprintf("- %s\n", tool))
		}
	}

	if cmd.Model != "" {
		b.WriteString(fmt.Sprintf("model: %s\n", cmd.Model))
	}

	b.WriteString("---\n")
	return b.String()
}

// generateSubagentFrontmatter generates YAML frontmatter for a subagent.
func (a *ClaudeAdapter) generateSubagentFrontmatter(agent *resource.ClaudeSubagent, pkg *config.PackageConfig) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("name: %s\n", agent.Name))
	b.WriteString(fmt.Sprintf("description: %s\n", agent.Description))

	if agent.Model != "" {
		b.WriteString(fmt.Sprintf("model: %s\n", agent.Model))
	}

	if agent.Color != "" {
		b.WriteString(fmt.Sprintf("color: %s\n", agent.Color))
	}

	if len(agent.Tools) > 0 {
		b.WriteString("tools:\n")
		for _, tool := range agent.Tools {
			b.WriteString(fmt.Sprintf("- %s\n", tool))
		}
	}

	b.WriteString("---\n")
	return b.String()
}

// generateRulesFrontmatter generates YAML frontmatter for rules.
func (a *ClaudeAdapter) generateRulesFrontmatter(rules *resource.ClaudeRules, pkg *config.PackageConfig) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("name: %s\n", rules.Name))
	b.WriteString(fmt.Sprintf("description: %s\n", rules.Description))

	if len(rules.Paths) > 0 {
		b.WriteString("paths:\n")
		for _, path := range rules.Paths {
			b.WriteString(fmt.Sprintf("- %s\n", path))
		}
	}

	b.WriteString("---\n")
	return b.String()
}
