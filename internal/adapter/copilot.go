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

// CopilotAdapter implements the Adapter interface for GitHub Copilot.
// It handles installation of resources into the .github and .vscode directory structures.
type CopilotAdapter struct{}

func init() {
	Register("github-copilot", &CopilotAdapter{})
}

// Name returns "github-copilot".
func (a *CopilotAdapter) Name() string {
	return "github-copilot"
}

// BaseDir returns ".github" joined with the root path.
func (a *CopilotAdapter) BaseDir(root string) string {
	return filepath.Join(root, ".github")
}

// SkillsDir returns ".github/skills" joined with the root path.
func (a *CopilotAdapter) SkillsDir(root string) string {
	return filepath.Join(root, ".github", "skills")
}

// CommandsDir returns ".github/prompts" joined with the root path.
// In Copilot, prompts serve the role of commands.
func (a *CopilotAdapter) CommandsDir(root string) string {
	return filepath.Join(root, ".github", "prompts")
}

// SubagentsDir returns ".github/agents" joined with the root path.
func (a *CopilotAdapter) SubagentsDir(root string) string {
	return filepath.Join(root, ".github", "agents")
}

// RulesDir returns ".github/instructions" joined with the root path.
// In Copilot, instructions serve the role of rules.
func (a *CopilotAdapter) RulesDir(root string) string {
	return filepath.Join(root, ".github", "instructions")
}

// PlanInstallation dispatches to the appropriate planner based on resource type.
func (a *CopilotAdapter) PlanInstallation(res resource.Resource, pkg *config.PackageConfig, pluginDir, projectRoot string, ctx *InstallContext) (*Plan, error) {
	switch r := res.(type) {
	// Unified MCP server (translate to Copilot-specific)
	case *resource.MCPServer:
		copilotServer := resource.TranslateToCopilotMCPServer(r)
		if copilotServer == nil {
			// Server is disabled for Copilot platform
			return &Plan{}, nil
		}
		return a.planMCPServer(copilotServer, pkg, pluginDir, projectRoot, ctx)

	// Merged resources
	case *resource.CopilotInstruction:
		return a.planInstruction(r, pkg, pluginDir, projectRoot, ctx)
	case *resource.CopilotMCPServer:
		return a.planMCPServer(r, pkg, pluginDir, projectRoot, ctx)

	// Standalone resources
	case *resource.CopilotInstructions:
		return a.planInstructions(r, pkg, pluginDir, projectRoot, ctx)
	case *resource.CopilotPrompt:
		return a.planPrompt(r, pkg, pluginDir, projectRoot, ctx)
	case *resource.CopilotAgent:
		return a.planAgent(r, pkg, pluginDir, projectRoot, ctx)
	case *resource.CopilotSkill:
		return a.planSkill(r, pkg, pluginDir, projectRoot, ctx)

	// Universal resources
	case *resource.File:
		return PlanUniversalFile(r, pkg, pluginDir, projectRoot, "github-copilot", ctx)
	case *resource.Directory:
		return PlanUniversalDirectory(r, pkg, ctx)

	default:
		return nil, fmt.Errorf("unsupported resource type for github-copilot adapter: %T", res)
	}
}

// GenerateFrontmatter generates YAML frontmatter for a resource.
func (a *CopilotAdapter) GenerateFrontmatter(res resource.Resource, pkg *config.PackageConfig) string {
	switch r := res.(type) {
	case *resource.CopilotInstructions:
		return a.generateInstructionsFrontmatter(r, pkg)
	case *resource.CopilotPrompt:
		return a.generatePromptFrontmatter(r, pkg)
	case *resource.CopilotAgent:
		return a.generateAgentFrontmatter(r, pkg)
	case *resource.CopilotSkill:
		return a.generateSkillFrontmatter(r, pkg)
	default:
		return ""
	}
}

// MergeMCPConfig merges MCP servers into .vscode/mcp.json format.
// Format: {"servers": {"name": {"command": "...", "args": [...], "env": {...}}}}
// Note: This method signature accepts ClaudeMCPServer for interface compatibility,
// but Copilot resources should use planMCPServer directly.
func (a *CopilotAdapter) MergeMCPConfig(existing map[string]any, pluginName string, servers []*resource.ClaudeMCPServer) map[string]any {
	// This method is kept for interface compatibility but Copilot uses its own server type
	// See MergeCopilotMCPConfig for the actual implementation
	return existing
}

// MergeCopilotMCPConfig merges Copilot MCP servers into .vscode/mcp.json format.
// Format: {"servers": {"name": {"type": "...", "command": "...", "args": [...], "env": {...}}}}
func (a *CopilotAdapter) MergeCopilotMCPConfig(existing map[string]any, pluginName string, servers []*resource.CopilotMCPServer) map[string]any {
	if existing == nil {
		existing = make(map[string]any)
	}

	// Get or create the servers map
	serversMap, ok := existing["servers"].(map[string]any)
	if !ok {
		serversMap = make(map[string]any)
	}

	// Add each server
	for _, server := range servers {
		serverConfig := make(map[string]any)

		if server.Type == "stdio" {
			serverConfig["type"] = "stdio"
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
			serverConfig["type"] = server.Type
			serverConfig["url"] = server.URL
			if len(server.Headers) > 0 {
				serverConfig["headers"] = server.Headers
			}
		}

		serversMap[server.Name] = serverConfig
	}

	existing["servers"] = serversMap
	return existing
}

// MergeSettingsConfig is not used for Copilot (no settings.json equivalent).
// This method is kept for interface compatibility.
func (a *CopilotAdapter) MergeSettingsConfig(existing map[string]any, settings *resource.ClaudeSettings) map[string]any {
	return existing
}

// MergeAgentFile merges instruction content into .github/copilot-instructions.md with markers.
// Format:
// <!-- dex:{plugin-name} -->
// {content}
// <!-- /dex:{plugin-name} -->
func (a *CopilotAdapter) MergeAgentFile(existing, pluginName, content string) string {
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

// planInstruction creates an installation plan for a Copilot instruction (singular).
// Instructions are merged into .github/copilot-instructions.md with markers.
func (a *CopilotAdapter) planInstruction(inst *resource.CopilotInstruction, pkg *config.PackageConfig, pluginDir, root string, ctx *InstallContext) (*Plan, error) {
	plan := NewPlan(pkg.Package.Name)

	// Instructions are merged into .github/copilot-instructions.md via AgentFileContent
	plan.AgentFileContent = inst.Content
	plan.AgentFilePath = filepath.Join(".github", "copilot-instructions.md")

	return plan, nil
}

// planMCPServer creates an installation plan for a Copilot MCP server.
// MCP servers are merged into .vscode/mcp.json with optional namespacing
func (a *CopilotAdapter) planMCPServer(server *resource.CopilotMCPServer, pkg *config.PackageConfig, pluginDir, root string, ctx *InstallContext) (*Plan, error) {
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
	plan.MCPEntries = a.MergeCopilotMCPConfig(nil, pkg.Package.Name, []*resource.CopilotMCPServer{&namespacedServer})

	// Set Copilot-specific MCP config path and key
	plan.MCPPath = filepath.Join(".vscode", "mcp.json")
	plan.MCPKey = "servers"

	return plan, nil
}

// planInstructions creates an installation plan for Copilot instructions (plural).
// Instructions files are installed to .github/instructions/{{plugin}-}{name}.instructions.md (namespaced or not)
func (a *CopilotAdapter) planInstructions(inst *resource.CopilotInstructions, pkg *config.PackageConfig, pluginDir, root string, ctx *InstallContext) (*Plan, error) {
	plan := NewPlan(pkg.Package.Name)

	// Create instructions directory
	instructionsDir := filepath.Join(".github", "instructions")
	plan.AddDirectory(instructionsDir, true)

	// Generate frontmatter and content
	var content string
	if hasFrontmatter(inst.Content) {
		content = inst.Content
	} else {
		frontmatter := a.generateInstructionsFrontmatter(inst, pkg)
		content = frontmatter + inst.Content
	}

	// Add instructions file with optional namespacing
	var fileName string
	if ctx != nil && ctx.Namespace {
		fileName = fmt.Sprintf("%s-%s.instructions.md", pkg.Package.Name, inst.Name)
	} else {
		fileName = fmt.Sprintf("%s.instructions.md", inst.Name)
	}
	instFile := filepath.Join(instructionsDir, fileName)
	plan.AddFile(instFile, content, "")

	return plan, nil
}

// planPrompt creates an installation plan for a Copilot prompt.
// Prompts are installed to .github/prompts/{{plugin}-}{name}.prompt.md (namespaced or not)
func (a *CopilotAdapter) planPrompt(prompt *resource.CopilotPrompt, pkg *config.PackageConfig, pluginDir, root string, ctx *InstallContext) (*Plan, error) {
	plan := NewPlan(pkg.Package.Name)

	// Create prompts directory
	promptsDir := filepath.Join(".github", "prompts")
	plan.AddDirectory(promptsDir, true)

	// Generate frontmatter and content
	var content string
	if hasFrontmatter(prompt.Content) {
		content = prompt.Content
	} else {
		frontmatter := a.generatePromptFrontmatter(prompt, pkg)
		content = frontmatter + prompt.Content
	}

	// Add prompt file with optional namespacing
	var fileName string
	if ctx != nil && ctx.Namespace {
		fileName = fmt.Sprintf("%s-%s.prompt.md", pkg.Package.Name, prompt.Name)
	} else {
		fileName = fmt.Sprintf("%s.prompt.md", prompt.Name)
	}
	promptFile := filepath.Join(promptsDir, fileName)
	plan.AddFile(promptFile, content, "")

	return plan, nil
}

// planAgent creates an installation plan for a Copilot agent.
// Agents are installed to .github/agents/{{plugin}-}{name}.agent.md (namespaced or not)
func (a *CopilotAdapter) planAgent(agent *resource.CopilotAgent, pkg *config.PackageConfig, pluginDir, root string, ctx *InstallContext) (*Plan, error) {
	plan := NewPlan(pkg.Package.Name)

	// Create agents directory
	agentsDir := filepath.Join(".github", "agents")
	plan.AddDirectory(agentsDir, true)

	// Generate frontmatter and content
	var content string
	if hasFrontmatter(agent.Content) {
		content = agent.Content
	} else {
		frontmatter := a.generateAgentFrontmatter(agent, pkg)
		content = frontmatter + agent.Content
	}

	// Add agent file with optional namespacing
	var fileName string
	if ctx != nil && ctx.Namespace {
		fileName = fmt.Sprintf("%s-%s.agent.md", pkg.Package.Name, agent.Name)
	} else {
		fileName = fmt.Sprintf("%s.agent.md", agent.Name)
	}
	agentFile := filepath.Join(agentsDir, fileName)
	plan.AddFile(agentFile, content, "")

	return plan, nil
}

// planSkill creates an installation plan for a Copilot skill.
// Skills are installed to .github/skills/{{plugin}-}{name}/SKILL.md (namespaced or not)
func (a *CopilotAdapter) planSkill(skill *resource.CopilotSkill, pkg *config.PackageConfig, pluginDir, root string, ctx *InstallContext) (*Plan, error) {
	plan := NewPlan(pkg.Package.Name)

	// Create skill directory with optional namespacing
	var skillDirName string
	if ctx != nil && ctx.Namespace {
		skillDirName = fmt.Sprintf("%s-%s", pkg.Package.Name, skill.Name)
	} else {
		skillDirName = skill.Name
	}
	skillDir := filepath.Join(".github", "skills", skillDirName)
	plan.AddDirectory(skillDir, true)

	// Generate frontmatter and content
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

// planFiles adds file copy operations to the plan.
func (a *CopilotAdapter) planFiles(plan *Plan, files []resource.FileBlock, pluginDir, destDir string) error {
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
func (a *CopilotAdapter) planTemplateFiles(plan *Plan, files []resource.TemplateFileBlock, pkg *config.PackageConfig, pluginDir, destDir, projectRoot string) error {
	// Create template context
	ctx := template.NewContext(pkg.Package.Name, pkg.Package.Version, projectRoot, "github-copilot")
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

// generateInstructionsFrontmatter generates YAML frontmatter for standalone instructions.
func (a *CopilotAdapter) generateInstructionsFrontmatter(inst *resource.CopilotInstructions, pkg *config.PackageConfig) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("description: %s\n", inst.Description))

	if inst.ApplyTo != "" {
		b.WriteString(fmt.Sprintf("applyTo: %s\n", inst.ApplyTo))
	}

	b.WriteString("---\n")
	return b.String()
}

// generatePromptFrontmatter generates YAML frontmatter for a prompt.
func (a *CopilotAdapter) generatePromptFrontmatter(prompt *resource.CopilotPrompt, pkg *config.PackageConfig) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("description: %s\n", prompt.Description))

	if prompt.ArgumentHint != "" {
		b.WriteString(fmt.Sprintf("argument-hint: %s\n", prompt.ArgumentHint))
	}
	if prompt.Agent != "" {
		b.WriteString(fmt.Sprintf("agent: %s\n", prompt.Agent))
	}
	if prompt.Model != "" {
		b.WriteString(fmt.Sprintf("model: %s\n", prompt.Model))
	}
	if len(prompt.Tools) > 0 {
		b.WriteString("tools:\n")
		for _, tool := range prompt.Tools {
			b.WriteString(fmt.Sprintf("- %s\n", tool))
		}
	}

	b.WriteString("---\n")
	return b.String()
}

// generateAgentFrontmatter generates YAML frontmatter for an agent.
func (a *CopilotAdapter) generateAgentFrontmatter(agent *resource.CopilotAgent, pkg *config.PackageConfig) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("description: %s\n", agent.Description))

	if agent.Model != "" {
		b.WriteString(fmt.Sprintf("model: %s\n", agent.Model))
	}
	if len(agent.Tools) > 0 {
		b.WriteString("tools:\n")
		for _, tool := range agent.Tools {
			b.WriteString(fmt.Sprintf("- %s\n", tool))
		}
	}
	if len(agent.Handoffs) > 0 {
		b.WriteString("handoffs:\n")
		for _, handoff := range agent.Handoffs {
			b.WriteString(fmt.Sprintf("- %s\n", handoff))
		}
	}
	if agent.Infer != nil && !*agent.Infer {
		b.WriteString("infer: false\n")
	}
	if agent.Target != "" {
		b.WriteString(fmt.Sprintf("target: %s\n", agent.Target))
	}

	b.WriteString("---\n")
	return b.String()
}

// generateSkillFrontmatter generates YAML frontmatter for a skill.
func (a *CopilotAdapter) generateSkillFrontmatter(skill *resource.CopilotSkill, pkg *config.PackageConfig) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("name: %s\n", skill.Name))
	b.WriteString(fmt.Sprintf("description: %s\n", skill.Description))
	b.WriteString("---\n")
	return b.String()
}

