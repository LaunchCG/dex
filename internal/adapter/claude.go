package adapter

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"strings"

	"github.com/launchcg/dex/internal/config"
	"github.com/launchcg/dex/internal/resource"
	"github.com/launchcg/dex/internal/template"
	"gopkg.in/yaml.v3"
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
func (a *ClaudeAdapter) PlanInstallation(res resource.Resource, pkg *config.PackageConfig, pkgDir, projectRoot string, ctx *InstallContext) (*Plan, error) {
	switch r := res.(type) {
	// Universal resource types (translate to Claude-specific)
	case *resource.MCPServer:
		translated := resource.TranslateToClaudeMCPServer(r)
		if translated == nil {
			slog.Warn("resource skipped: disabled for platform", "resource", r.Name, "type", "mcp_server", "platform", "claude-code")
			return &Plan{}, nil
		}
		return a.planMCPServer(translated, pkg, pkgDir, projectRoot, ctx)
	case *resource.Skill:
		translated := resource.TranslateToClaudeSkill(r)
		if translated == nil {
			slog.Warn("resource skipped: disabled for platform", "resource", r.Name, "type", "skill", "platform", "claude-code")
			return &Plan{}, nil
		}
		return a.planSkill(translated, pkg, pkgDir, projectRoot, ctx)
	case *resource.Command:
		translated := resource.TranslateToClaudeCommand(r)
		if translated == nil {
			slog.Warn("resource skipped: disabled for platform", "resource", r.Name, "type", "command", "platform", "claude-code")
			return &Plan{}, nil
		}
		return a.planCommand(translated, pkg, pkgDir, projectRoot, ctx)
	case *resource.Agent:
		translated := resource.TranslateToClaudeSubagent(r)
		if translated == nil {
			slog.Warn("resource skipped: disabled for platform", "resource", r.Name, "type", "agent", "platform", "claude-code")
			return &Plan{}, nil
		}
		return a.planSubagent(translated, pkg, pkgDir, projectRoot, ctx)
	case *resource.Rule:
		translated := resource.TranslateToClaudeRule(r)
		if translated == nil {
			slog.Warn("resource skipped: disabled for platform", "resource", r.Name, "type", "rule", "platform", "claude-code")
			return &Plan{}, nil
		}
		return a.planRule(translated, pkg, pkgDir, projectRoot, ctx)
	case *resource.Rules:
		translated := resource.TranslateToClaudeRules(r)
		if translated == nil {
			slog.Warn("resource skipped: disabled for platform", "resource", r.Name, "type", "rules", "platform", "claude-code")
			return &Plan{}, nil
		}
		return a.planRules(translated, pkg, pkgDir, projectRoot, ctx)
	case *resource.Settings:
		translated := resource.TranslateToClaudeSettings(r)
		if translated == nil {
			slog.Warn("resource skipped: disabled for platform", "resource", r.Name, "type", "settings", "platform", "claude-code")
			return &Plan{}, nil
		}
		return a.planSettings(translated, pkg, pkgDir, projectRoot, ctx)

	// Platform-specific types (used internally by translators, kept for direct use)
	case *resource.ClaudeSkill:
		return a.planSkill(r, pkg, pkgDir, projectRoot, ctx)
	case *resource.ClaudeCommand:
		return a.planCommand(r, pkg, pkgDir, projectRoot, ctx)
	case *resource.ClaudeSubagent:
		return a.planSubagent(r, pkg, pkgDir, projectRoot, ctx)
	case *resource.ClaudeRule:
		return a.planRule(r, pkg, pkgDir, projectRoot, ctx)
	case *resource.ClaudeRules:
		return a.planRules(r, pkg, pkgDir, projectRoot, ctx)
	case *resource.ClaudeSettings:
		return a.planSettings(r, pkg, pkgDir, projectRoot, ctx)
	case *resource.ClaudeMCPServer:
		return a.planMCPServer(r, pkg, pkgDir, projectRoot, ctx)

	// Universal file/directory resources
	case *resource.File:
		return PlanUniversalFile(r, pkg, pkgDir, projectRoot, "claude-code", ctx)
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
		return a.generateSkillFrontmatter((*resource.ClaudeSkill)(r), pkg)
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
func (a *ClaudeAdapter) MergeMCPConfig(existing map[string]any, pkgName string, servers []*resource.ClaudeMCPServer) map[string]any {
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
	mergeSlice("additionalDirectories", settings.AdditionalDirectories)

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
	if settings.AutoMemoryDirectory != "" {
		existing["autoMemoryDirectory"] = settings.AutoMemoryDirectory
	}
	if settings.IncludeGitInstructions != nil {
		existing["includeGitInstructions"] = *settings.IncludeGitInstructions
	}
	if settings.Agent != "" {
		existing["agent"] = settings.Agent
	}

	return existing
}

// hasFrontmatter checks if content already starts with YAML frontmatter.
func hasFrontmatter(content string) bool {
	return strings.HasPrefix(strings.TrimSpace(content), "---")
}

// planSkill creates an installation plan for a Claude skill.
// Skills are installed to .claude/skills/{pkg}-{skill-name}/SKILL.md
func (a *ClaudeAdapter) planSkill(skill *resource.ClaudeSkill, pkg *config.PackageConfig, pkgDir, root string, ctx *InstallContext) (*Plan, error) {
	plan := NewPlan(pkg.Meta.Name)

	// Create skill directory with optional namespacing
	var skillDirName string
	if ctx != nil && ctx.Namespace {
		skillDirName = fmt.Sprintf("%s-%s", pkg.Meta.Name, skill.Name)
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
	if err := a.planFiles(plan, skill.GetFiles(), pkgDir, skillDir); err != nil {
		return nil, fmt.Errorf("planning skill files: %w", err)
	}

	// Handle template files
	if err := a.planTemplateFiles(plan, skill.GetTemplateFiles(), pkg, pkgDir, skillDir, root); err != nil {
		return nil, fmt.Errorf("planning skill template files: %w", err)
	}

	return plan, nil
}

// planCommand creates an installation plan for a Claude command.
// Commands are installed to .claude/commands/{{pkg}-}{command}.md (namespaced or not)
func (a *ClaudeAdapter) planCommand(cmd *resource.ClaudeCommand, pkg *config.PackageConfig, pkgDir, root string, ctx *InstallContext) (*Plan, error) {
	plan := NewPlan(pkg.Meta.Name)

	// Create commands directory
	commandsDir := filepath.Join(".claude", "commands")
	plan.AddDirectory(commandsDir, true)

	// Use content as-is if it already has frontmatter, otherwise add ours
	var content string
	if hasFrontmatter(cmd.Content) {
		content = cmd.Content
	} else {
		frontmatter := a.generateSkillFrontmatter((*resource.ClaudeSkill)(cmd), pkg)
		content = frontmatter + cmd.Content
	}

	// Add command file with optional namespacing
	var fileName string
	if ctx != nil && ctx.Namespace {
		fileName = fmt.Sprintf("%s-%s.md", pkg.Meta.Name, cmd.Name)
	} else {
		fileName = fmt.Sprintf("%s.md", cmd.Name)
	}
	cmdFile := filepath.Join(commandsDir, fileName)
	plan.AddFile(cmdFile, content, "")

	return plan, nil
}

// planSubagent creates an installation plan for a Claude subagent.
// Subagents are installed to .claude/agents/{{pkg}-}{agent}.md (namespaced or not)
func (a *ClaudeAdapter) planSubagent(agent *resource.ClaudeSubagent, pkg *config.PackageConfig, pkgDir, root string, ctx *InstallContext) (*Plan, error) {
	plan := NewPlan(pkg.Meta.Name)

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
		fileName = fmt.Sprintf("%s-%s.md", pkg.Meta.Name, agent.Name)
	} else {
		fileName = fmt.Sprintf("%s.md", agent.Name)
	}
	agentFile := filepath.Join(agentsDir, fileName)
	plan.AddFile(agentFile, content, "")

	return plan, nil
}

// planRule creates an installation plan for a Claude rule (singular).
// Rules are merged into CLAUDE.md with markers.
func (a *ClaudeAdapter) planRule(rule *resource.ClaudeRule, pkg *config.PackageConfig, pkgDir, root string, ctx *InstallContext) (*Plan, error) {
	plan := NewPlan(pkg.Meta.Name)

	// Rules are merged into CLAUDE.md via AgentFileContent
	// Content is used as-is; use templatefile() in HCL for templating
	plan.AgentFileContent = rule.Content

	return plan, nil
}

// planRules creates an installation plan for Claude rules (plural).
// Rules are installed to .claude/rules/{name}/ as a directory structure,
// similar to skills, allowing for multiple related rule files.
func (a *ClaudeAdapter) planRules(rules *resource.ClaudeRules, pkg *config.PackageConfig, pkgDir, root string, ctx *InstallContext) (*Plan, error) {
	plan := NewPlan(pkg.Meta.Name)

	// Create rules subdirectory with optional namespacing
	var rulesDirName string
	if ctx != nil && ctx.Namespace {
		rulesDirName = fmt.Sprintf("%s-%s", pkg.Meta.Name, rules.Name)
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
	if err := a.planFiles(plan, rules.GetFiles(), pkgDir, rulesDir); err != nil {
		return nil, fmt.Errorf("planning rules files: %w", err)
	}

	// Handle template files
	if err := a.planTemplateFiles(plan, rules.GetTemplateFiles(), pkg, pkgDir, rulesDir, root); err != nil {
		return nil, fmt.Errorf("planning rules template files: %w", err)
	}

	return plan, nil
}

// planSettings creates an installation plan for Claude settings.
// Settings are merged into .claude/settings.json
func (a *ClaudeAdapter) planSettings(settings *resource.ClaudeSettings, pkg *config.PackageConfig, pkgDir, root string, ctx *InstallContext) (*Plan, error) {
	plan := NewPlan(pkg.Meta.Name)

	// Settings are merged via SettingsEntries
	plan.SettingsEntries = a.MergeSettingsConfig(nil, settings)

	return plan, nil
}

// planMCPServer creates an installation plan for a Claude MCP server.
// MCP servers are merged into .mcp.json with optional namespacing
func (a *ClaudeAdapter) planMCPServer(server *resource.ClaudeMCPServer, pkg *config.PackageConfig, pkgDir, root string, ctx *InstallContext) (*Plan, error) {
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
	plan.MCPEntries = a.MergeMCPConfig(nil, pkg.Meta.Name, []*resource.ClaudeMCPServer{&namespacedServer})

	// Set Claude-specific MCP config path and key
	plan.MCPPath = ".mcp.json"
	plan.MCPKey = "mcpServers"

	return plan, nil
}

// planFiles adds file copy operations to the plan.
func (a *ClaudeAdapter) planFiles(plan *Plan, files []resource.FileBlock, pkgDir, destDir string) error {
	for _, file := range files {
		// Read source file
		srcPath := filepath.Join(pkgDir, file.Src)
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
func (a *ClaudeAdapter) planTemplateFiles(plan *Plan, files []resource.TemplateFileBlock, pkg *config.PackageConfig, pkgDir, destDir, projectRoot string) error {
	// Create template context
	ctx := template.NewContext(pkg.Meta.Name, pkg.Meta.Version, projectRoot, "claude-code")
	ctx.WithComponentDir(filepath.Join(projectRoot, destDir))
	engine := template.NewEngine(pkgDir, ctx)

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

	if skill.WhenToUse != "" {
		b.WriteString(fmt.Sprintf("when_to_use: %s\n", skill.WhenToUse))
	}
	if skill.ArgumentHint != "" {
		b.WriteString(fmt.Sprintf("argument-hint: %s\n", skill.ArgumentHint))
	}
	if len(skill.Arguments) > 0 {
		b.WriteString("arguments:\n")
		for _, arg := range skill.Arguments {
			b.WriteString(fmt.Sprintf("- %s\n", arg))
		}
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
	if skill.Effort != "" {
		b.WriteString(fmt.Sprintf("effort: %s\n", skill.Effort))
	}
	if skill.Context != "" {
		b.WriteString(fmt.Sprintf("context: %s\n", skill.Context))
	}
	if skill.Agent != "" {
		b.WriteString(fmt.Sprintf("agent: %s\n", skill.Agent))
	}
	if len(skill.Paths) > 0 {
		b.WriteString("paths:\n")
		for _, p := range skill.Paths {
			b.WriteString(fmt.Sprintf("- %s\n", p))
		}
	}
	if skill.Shell != "" {
		b.WriteString(fmt.Sprintf("shell: %s\n", skill.Shell))
	}

	// Add any additional metadata
	for k, v := range skill.Metadata {
		b.WriteString(fmt.Sprintf("%s: %s\n", k, v))
	}

	// Add hooks if present
	if len(skill.Hooks) > 0 {
		hooksYAML, err := yaml.Marshal(skill.Hooks)
		if err == nil {
			b.WriteString("hooks:\n")
			// Indent each line of marshaled YAML
			for _, line := range strings.Split(string(hooksYAML), "\n") {
				if line != "" {
					b.WriteString("  " + line + "\n")
				}
			}
		}
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

	if len(agent.DisallowedTools) > 0 {
		b.WriteString("disallowedTools:\n")
		for _, tool := range agent.DisallowedTools {
			b.WriteString(fmt.Sprintf("- %s\n", tool))
		}
	}

	if agent.PermissionMode != "" {
		b.WriteString(fmt.Sprintf("permissionMode: %s\n", agent.PermissionMode))
	}

	if agent.MaxTurns != nil {
		b.WriteString(fmt.Sprintf("maxTurns: %d\n", *agent.MaxTurns))
	}

	if len(agent.Skills) > 0 {
		b.WriteString("skills:\n")
		for _, s := range agent.Skills {
			b.WriteString(fmt.Sprintf("- %s\n", s))
		}
	}

	if len(agent.MCPServers) > 0 {
		b.WriteString("mcpServers:\n")
		for _, s := range agent.MCPServers {
			b.WriteString(fmt.Sprintf("- %s\n", s))
		}
	}

	if agent.Memory != "" {
		b.WriteString(fmt.Sprintf("memory: %s\n", agent.Memory))
	}

	if agent.Background {
		b.WriteString("background: true\n")
	}

	if agent.Effort != "" {
		b.WriteString(fmt.Sprintf("effort: %s\n", agent.Effort))
	}

	if agent.Isolation != "" {
		b.WriteString(fmt.Sprintf("isolation: %s\n", agent.Isolation))
	}

	if agent.InitialPrompt != "" {
		b.WriteString(fmt.Sprintf("initialPrompt: %s\n", agent.InitialPrompt))
	}

	// Add hooks if present
	if len(agent.Hooks) > 0 {
		hooksYAML, err := yaml.Marshal(agent.Hooks)
		if err == nil {
			b.WriteString("hooks:\n")
			// Indent each line of marshaled YAML
			for _, line := range strings.Split(string(hooksYAML), "\n") {
				if line != "" {
					b.WriteString("  " + line + "\n")
				}
			}
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
