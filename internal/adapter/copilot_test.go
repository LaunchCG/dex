package adapter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/launchcg/dex/internal/config"
	"github.com/launchcg/dex/internal/resource"
)

func TestGet_GithubCopilot(t *testing.T) {
	adapter, err := Get("github-copilot")
	require.NoError(t, err)
	assert.NotNil(t, adapter)
	assert.Equal(t, "github-copilot", adapter.Name())
}

func TestCopilotAdapter_Name(t *testing.T) {
	adapter := &CopilotAdapter{}
	assert.Equal(t, "github-copilot", adapter.Name())
}

func TestCopilotAdapter_Directories(t *testing.T) {
	adapter := &CopilotAdapter{}
	root := "/project"

	assert.Equal(t, "/project/.github", adapter.BaseDir(root))
	assert.Equal(t, "/project/.github/skills", adapter.SkillsDir(root))
	assert.Equal(t, "/project/.github/prompts", adapter.CommandsDir(root))
	assert.Equal(t, "/project/.github/agents", adapter.SubagentsDir(root))
	assert.Equal(t, "/project/.github/instructions", adapter.RulesDir(root))
}

func TestCopilotAdapter_PlanInstruction(t *testing.T) {
	adapter := &CopilotAdapter{}

	inst := &resource.CopilotInstruction{
		Name:        "coding-standards",
		Description: "Project coding standards",
		Content:     "Always use TypeScript strict mode.",
	}

	pkg := &config.PackageConfig{
		Package: config.PackageBlock{
			Name:    "my-plugin",
			Version: "1.0.0",
		},
	}

	plan, err := adapter.PlanInstallation(inst, pkg, "/plugin", "/project", &InstallContext{PackageName: "my-plugin", Namespace: false})
	require.NoError(t, err)
	require.NotNil(t, plan)

	assert.Equal(t, "my-plugin", plan.PluginName)
	assert.Equal(t, "Always use TypeScript strict mode.", plan.AgentFileContent)
	assert.Equal(t, filepath.Join(".github", "copilot-instructions.md"), plan.AgentFilePath)
	assert.Empty(t, plan.Files)
	assert.Empty(t, plan.Directories)
}

func TestCopilotAdapter_PlanMCPServer_Stdio(t *testing.T) {
	adapter := &CopilotAdapter{}

	server := &resource.CopilotMCPServer{
		Name:    "filesystem",
		Type:    "stdio",
		Command: "npx",
		Args:    []string{"-y", "@anthropic/mcp-filesystem"},
		Env:     map[string]string{"HOME": "/home/user"},
	}

	pkg := &config.PackageConfig{
		Package: config.PackageBlock{
			Name:    "my-plugin",
			Version: "1.0.0",
		},
	}

	plan, err := adapter.PlanInstallation(server, pkg, "/plugin", "/project", &InstallContext{PackageName: "my-plugin", Namespace: false})
	require.NoError(t, err)
	require.NotNil(t, plan)

	assert.Equal(t, filepath.Join(".vscode", "mcp.json"), plan.MCPPath)
	assert.Equal(t, "servers", plan.MCPKey)

	expectedMCPEntries := map[string]any{
		"servers": map[string]any{
			"filesystem": map[string]any{
				"type":    "stdio",
				"command": "npx",
				"args":    []string{"-y", "@anthropic/mcp-filesystem"},
				"env":     map[string]string{"HOME": "/home/user"},
			},
		},
	}
	assert.Equal(t, expectedMCPEntries, plan.MCPEntries)
}

func TestCopilotAdapter_PlanMCPServer_HTTP(t *testing.T) {
	adapter := &CopilotAdapter{}

	server := &resource.CopilotMCPServer{
		Name:    "context7",
		Type:    "http",
		URL:     "https://mcp.context7.com/mcp",
		Headers: map[string]string{"Authorization": "Bearer token"},
	}

	pkg := &config.PackageConfig{
		Package: config.PackageBlock{
			Name:    "my-plugin",
			Version: "1.0.0",
		},
	}

	plan, err := adapter.PlanInstallation(server, pkg, "/plugin", "/project", &InstallContext{PackageName: "my-plugin", Namespace: false})
	require.NoError(t, err)

	expectedMCPEntries := map[string]any{
		"servers": map[string]any{
			"context7": map[string]any{
				"type":    "http",
				"url":     "https://mcp.context7.com/mcp",
				"headers": map[string]string{"Authorization": "Bearer token"},
			},
		},
	}
	assert.Equal(t, expectedMCPEntries, plan.MCPEntries)
}

func TestCopilotAdapter_PlanMCPServer_SSE(t *testing.T) {
	adapter := &CopilotAdapter{}

	server := &resource.CopilotMCPServer{
		Name:    "sse-server",
		Type:    "sse",
		URL:     "https://api.example.com/sse",
		Headers: map[string]string{"X-API-Key": "secret"},
	}

	pkg := &config.PackageConfig{
		Package: config.PackageBlock{
			Name:    "my-plugin",
			Version: "1.0.0",
		},
	}

	plan, err := adapter.PlanInstallation(server, pkg, "/plugin", "/project", &InstallContext{PackageName: "my-plugin", Namespace: false})
	require.NoError(t, err)

	expectedMCPEntries := map[string]any{
		"servers": map[string]any{
			"sse-server": map[string]any{
				"type":    "sse",
				"url":     "https://api.example.com/sse",
				"headers": map[string]string{"X-API-Key": "secret"},
			},
		},
	}
	assert.Equal(t, expectedMCPEntries, plan.MCPEntries)
}

func TestCopilotAdapter_PlanMCPServer_EnvFile(t *testing.T) {
	adapter := &CopilotAdapter{}

	server := &resource.CopilotMCPServer{
		Name:    "env-server",
		Type:    "stdio",
		Command: "node",
		Args:    []string{"server.js"},
		EnvFile: ".env.local",
	}

	pkg := &config.PackageConfig{
		Package: config.PackageBlock{
			Name:    "my-plugin",
			Version: "1.0.0",
		},
	}

	plan, err := adapter.PlanInstallation(server, pkg, "/plugin", "/project", &InstallContext{PackageName: "my-plugin", Namespace: false})
	require.NoError(t, err)

	expectedMCPEntries := map[string]any{
		"servers": map[string]any{
			"env-server": map[string]any{
				"type":    "stdio",
				"command": "node",
				"args":    []string{"server.js"},
				"envFile": ".env.local",
			},
		},
	}
	assert.Equal(t, expectedMCPEntries, plan.MCPEntries)
}

func TestCopilotAdapter_PlanInstructions(t *testing.T) {
	adapter := &CopilotAdapter{}

	inst := &resource.CopilotInstructions{
		Name:        "typescript",
		Description: "TypeScript best practices",
		Content:     "Use interfaces over types.",
		ApplyTo:     "**/*.ts",
	}

	pkg := &config.PackageConfig{
		Package: config.PackageBlock{
			Name:    "my-plugin",
			Version: "1.0.0",
		},
	}

	plan, err := adapter.PlanInstallation(inst, pkg, "/plugin", "/project", &InstallContext{PackageName: "my-plugin", Namespace: true})
	require.NoError(t, err)
	require.NotNil(t, plan)

	assert.Equal(t, []string{filepath.Join(".github", "instructions")}, getDirPaths(plan))
	require.Len(t, plan.Files, 1)
	assert.Equal(t, filepath.Join(".github", "instructions", "my-plugin-typescript.instructions.md"), plan.Files[0].Path)
	assert.Equal(t, "", plan.Files[0].Chmod)

	expectedContent := `---
description: TypeScript best practices
applyTo: **/*.ts
---
Use interfaces over types.`
	assert.Equal(t, expectedContent, plan.Files[0].Content)
}

func TestCopilotAdapter_PlanInstructions_NoApplyTo(t *testing.T) {
	adapter := &CopilotAdapter{}

	inst := &resource.CopilotInstructions{
		Name:        "general",
		Description: "General guidelines",
		Content:     "Follow best practices.",
	}

	pkg := &config.PackageConfig{
		Package: config.PackageBlock{
			Name:    "my-plugin",
			Version: "1.0.0",
		},
	}

	plan, err := adapter.PlanInstallation(inst, pkg, "/plugin", "/project", &InstallContext{PackageName: "my-plugin", Namespace: false})
	require.NoError(t, err)
	require.Len(t, plan.Files, 1)

	expectedContent := `---
description: General guidelines
---
Follow best practices.`
	assert.Equal(t, expectedContent, plan.Files[0].Content)
}

func TestCopilotAdapter_PlanPrompt(t *testing.T) {
	adapter := &CopilotAdapter{}

	prompt := &resource.CopilotPrompt{
		Name:         "review",
		Description:  "Code review prompt",
		Content:      "Review this code for bugs.",
		ArgumentHint: "[filename]",
		Agent:        "ask",
		Model:        "gpt-4o",
		Tools:        []string{"fetch", "search"},
	}

	pkg := &config.PackageConfig{
		Package: config.PackageBlock{
			Name:    "my-plugin",
			Version: "1.0.0",
		},
	}

	plan, err := adapter.PlanInstallation(prompt, pkg, "/plugin", "/project", &InstallContext{PackageName: "my-plugin", Namespace: true})
	require.NoError(t, err)
	require.NotNil(t, plan)

	assert.Equal(t, []string{filepath.Join(".github", "prompts")}, getDirPaths(plan))
	require.Len(t, plan.Files, 1)
	assert.Equal(t, filepath.Join(".github", "prompts", "my-plugin-review.prompt.md"), plan.Files[0].Path)

	expectedContent := `---
description: Code review prompt
argument-hint: [filename]
agent: ask
model: gpt-4o
tools:
- fetch
- search
---
Review this code for bugs.`
	assert.Equal(t, expectedContent, plan.Files[0].Content)
}

func TestCopilotAdapter_PlanPrompt_Minimal(t *testing.T) {
	adapter := &CopilotAdapter{}

	prompt := &resource.CopilotPrompt{
		Name:        "simple",
		Description: "Simple prompt",
		Content:     "Do the thing.",
	}

	pkg := &config.PackageConfig{
		Package: config.PackageBlock{
			Name:    "my-plugin",
			Version: "1.0.0",
		},
	}

	plan, err := adapter.PlanInstallation(prompt, pkg, "/plugin", "/project", &InstallContext{PackageName: "my-plugin", Namespace: false})
	require.NoError(t, err)
	require.Len(t, plan.Files, 1)

	expectedContent := `---
description: Simple prompt
---
Do the thing.`
	assert.Equal(t, expectedContent, plan.Files[0].Content)
}

func TestCopilotAdapter_PlanAgent(t *testing.T) {
	adapter := &CopilotAdapter{}

	boolFalse := false
	agent := &resource.CopilotAgent{
		Name:        "planner",
		Description: "Planning agent",
		Content:     "Create implementation plans.",
		Model:       "gpt-4o",
		Tools:       []string{"fetch", "search"},
		Handoffs:    []string{"implementer", "reviewer"},
		Infer:       &boolFalse,
		Target:      "vscode",
	}

	pkg := &config.PackageConfig{
		Package: config.PackageBlock{
			Name:    "my-plugin",
			Version: "1.0.0",
		},
	}

	plan, err := adapter.PlanInstallation(agent, pkg, "/plugin", "/project", &InstallContext{PackageName: "my-plugin", Namespace: true})
	require.NoError(t, err)
	require.NotNil(t, plan)

	assert.Equal(t, []string{filepath.Join(".github", "agents")}, getDirPaths(plan))
	require.Len(t, plan.Files, 1)
	assert.Equal(t, filepath.Join(".github", "agents", "my-plugin-planner.agent.md"), plan.Files[0].Path)

	expectedContent := `---
description: Planning agent
model: gpt-4o
tools:
- fetch
- search
handoffs:
- implementer
- reviewer
infer: false
target: vscode
---
Create implementation plans.`
	assert.Equal(t, expectedContent, plan.Files[0].Content)
}

func TestCopilotAdapter_PlanAgent_Minimal(t *testing.T) {
	adapter := &CopilotAdapter{}

	agent := &resource.CopilotAgent{
		Name:        "simple",
		Description: "Simple agent",
		Content:     "Do simple things.",
	}

	pkg := &config.PackageConfig{
		Package: config.PackageBlock{
			Name:    "my-plugin",
			Version: "1.0.0",
		},
	}

	plan, err := adapter.PlanInstallation(agent, pkg, "/plugin", "/project", &InstallContext{PackageName: "my-plugin", Namespace: false})
	require.NoError(t, err)
	require.Len(t, plan.Files, 1)

	expectedContent := `---
description: Simple agent
---
Do simple things.`
	assert.Equal(t, expectedContent, plan.Files[0].Content)
}

func TestCopilotAdapter_PlanSkill(t *testing.T) {
	adapter := &CopilotAdapter{}

	skill := &resource.CopilotSkill{
		Name:        "testing",
		Description: "Testing best practices",
		Content:     "Write comprehensive tests.",
	}

	pkg := &config.PackageConfig{
		Package: config.PackageBlock{
			Name:    "my-plugin",
			Version: "1.0.0",
		},
	}

	plan, err := adapter.PlanInstallation(skill, pkg, "/plugin", "/project", &InstallContext{PackageName: "my-plugin", Namespace: true})
	require.NoError(t, err)
	require.NotNil(t, plan)
	assert.Equal(t, "my-plugin", plan.PluginName)

	assert.Equal(t, []string{filepath.Join(".github", "skills", "my-plugin-testing")}, getDirPaths(plan))
	require.Len(t, plan.Files, 1)
	assert.Equal(t, filepath.Join(".github", "skills", "my-plugin-testing", "SKILL.md"), plan.Files[0].Path)

	expectedContent := `---
name: testing
description: Testing best practices
---
Write comprehensive tests.`
	assert.Equal(t, expectedContent, plan.Files[0].Content)
}

func TestCopilotAdapter_PlanSkill_WithFiles(t *testing.T) {
	// Create a temp directory with a file to copy
	tmpDir := t.TempDir()
	err := os.MkdirAll(filepath.Join(tmpDir, "assets"), 0755)
	require.NoError(t, err)

	helperContent := "#!/bin/bash\necho hello"
	err = os.WriteFile(filepath.Join(tmpDir, "assets", "helper.sh"), []byte(helperContent), 0644)
	require.NoError(t, err)

	adapter := &CopilotAdapter{}

	skill := &resource.CopilotSkill{
		Name:        "test-skill",
		Description: "A test skill",
		Content:     "Skill content here.",
		Files: []resource.FileBlock{
			{Src: "assets/helper.sh", Chmod: "755"},
		},
	}

	pkg := &config.PackageConfig{
		Package: config.PackageBlock{
			Name:    "my-plugin",
			Version: "1.0.0",
		},
	}

	plan, err := adapter.PlanInstallation(skill, pkg, tmpDir, "/project", &InstallContext{PackageName: "my-plugin", Namespace: true})
	require.NoError(t, err)

	assert.Equal(t, []string{filepath.Join(".github", "skills", "my-plugin-test-skill")}, getDirPaths(plan))
	require.Len(t, plan.Files, 2)

	// Find SKILL.md and helper.sh
	var skillFile, helperFile *FileWrite
	for i := range plan.Files {
		if filepath.Base(plan.Files[i].Path) == "SKILL.md" {
			skillFile = &plan.Files[i]
		} else if filepath.Base(plan.Files[i].Path) == "helper.sh" {
			helperFile = &plan.Files[i]
		}
	}

	require.NotNil(t, skillFile)
	expectedSkillContent := `---
name: test-skill
description: A test skill
---
Skill content here.`
	assert.Equal(t, expectedSkillContent, skillFile.Content)
	assert.Equal(t, filepath.Join(".github", "skills", "my-plugin-test-skill", "SKILL.md"), skillFile.Path)
	assert.Equal(t, "", skillFile.Chmod)

	require.NotNil(t, helperFile)
	assert.Equal(t, helperContent, helperFile.Content)
	assert.Equal(t, filepath.Join(".github", "skills", "my-plugin-test-skill", "helper.sh"), helperFile.Path)
	assert.Equal(t, "755", helperFile.Chmod)
}

func TestCopilotAdapter_PlanInstallation_UnsupportedType(t *testing.T) {
	adapter := &CopilotAdapter{}

	unknown := &mockUnknownResource{name: "unknown"}
	pkg := &config.PackageConfig{
		Package: config.PackageBlock{
			Name:    "my-plugin",
			Version: "1.0.0",
		},
	}

	plan, err := adapter.PlanInstallation(unknown, pkg, "/plugin", "/project", nil)
	assert.Nil(t, plan)
	require.Error(t, err)
	assert.Equal(t, "unsupported resource type for github-copilot adapter: *adapter.mockUnknownResource", err.Error())
}

func TestCopilotAdapter_GenerateFrontmatter_Skill(t *testing.T) {
	adapter := &CopilotAdapter{}

	skill := &resource.CopilotSkill{
		Name:        "test-skill",
		Description: "A test skill",
	}

	pkg := &config.PackageConfig{}

	frontmatter := adapter.GenerateFrontmatter(skill, pkg)
	expected := `---
name: test-skill
description: A test skill
---
`
	assert.Equal(t, expected, frontmatter)
}

func TestCopilotAdapter_GenerateFrontmatter_Prompt_Full(t *testing.T) {
	adapter := &CopilotAdapter{}

	prompt := &resource.CopilotPrompt{
		Name:         "deploy",
		Description:  "Deploy app",
		ArgumentHint: "[env]",
		Agent:        "edit",
		Model:        "gpt-4o",
		Tools:        []string{"fetch", "search"},
	}

	pkg := &config.PackageConfig{}

	frontmatter := adapter.GenerateFrontmatter(prompt, pkg)
	expected := `---
description: Deploy app
argument-hint: [env]
agent: edit
model: gpt-4o
tools:
- fetch
- search
---
`
	assert.Equal(t, expected, frontmatter)
}

func TestCopilotAdapter_GenerateFrontmatter_Prompt_Minimal(t *testing.T) {
	adapter := &CopilotAdapter{}

	prompt := &resource.CopilotPrompt{
		Name:        "simple",
		Description: "Simple prompt",
	}

	pkg := &config.PackageConfig{}

	frontmatter := adapter.GenerateFrontmatter(prompt, pkg)
	expected := `---
description: Simple prompt
---
`
	assert.Equal(t, expected, frontmatter)
}

func TestCopilotAdapter_GenerateFrontmatter_Agent_Full(t *testing.T) {
	adapter := &CopilotAdapter{}

	boolFalse := false
	agent := &resource.CopilotAgent{
		Name:        "researcher",
		Description: "Research assistant",
		Model:       "gpt-4o",
		Tools:       []string{"fetch", "search"},
		Handoffs:    []string{"writer"},
		Infer:       &boolFalse,
		Target:      "github-copilot",
	}

	pkg := &config.PackageConfig{}

	frontmatter := adapter.GenerateFrontmatter(agent, pkg)
	expected := `---
description: Research assistant
model: gpt-4o
tools:
- fetch
- search
handoffs:
- writer
infer: false
target: github-copilot
---
`
	assert.Equal(t, expected, frontmatter)
}

func TestCopilotAdapter_GenerateFrontmatter_Agent_Minimal(t *testing.T) {
	adapter := &CopilotAdapter{}

	agent := &resource.CopilotAgent{
		Name:        "simple",
		Description: "Simple agent",
	}

	pkg := &config.PackageConfig{}

	frontmatter := adapter.GenerateFrontmatter(agent, pkg)
	expected := `---
description: Simple agent
---
`
	assert.Equal(t, expected, frontmatter)
}

func TestCopilotAdapter_GenerateFrontmatter_Instructions_Full(t *testing.T) {
	adapter := &CopilotAdapter{}

	inst := &resource.CopilotInstructions{
		Name:        "go-rules",
		Description: "Go language rules",
		ApplyTo:     "**/*.go",
	}

	pkg := &config.PackageConfig{}

	frontmatter := adapter.GenerateFrontmatter(inst, pkg)
	expected := `---
description: Go language rules
applyTo: **/*.go
---
`
	assert.Equal(t, expected, frontmatter)
}

func TestCopilotAdapter_GenerateFrontmatter_Instructions_NoApplyTo(t *testing.T) {
	adapter := &CopilotAdapter{}

	inst := &resource.CopilotInstructions{
		Name:        "general",
		Description: "General rules",
	}

	pkg := &config.PackageConfig{}

	frontmatter := adapter.GenerateFrontmatter(inst, pkg)
	expected := `---
description: General rules
---
`
	assert.Equal(t, expected, frontmatter)
}

func TestCopilotAdapter_GenerateFrontmatter_UnknownType(t *testing.T) {
	adapter := &CopilotAdapter{}

	// CopilotInstruction doesn't generate frontmatter (it's merged content)
	inst := &resource.CopilotInstruction{Name: "test", Description: "test"}
	pkg := &config.PackageConfig{}

	frontmatter := adapter.GenerateFrontmatter(inst, pkg)
	assert.Equal(t, "", frontmatter)

	// CopilotMCPServer doesn't generate frontmatter
	server := &resource.CopilotMCPServer{Name: "test", Type: "stdio", Command: "cmd"}
	frontmatter = adapter.GenerateFrontmatter(server, pkg)
	assert.Equal(t, "", frontmatter)
}

func TestCopilotAdapter_MergeCopilotMCPConfig_Merge(t *testing.T) {
	adapter := &CopilotAdapter{}

	existing := map[string]any{
		"servers": map[string]any{
			"existing-server": map[string]any{
				"type":    "stdio",
				"command": "node",
				"args":    []string{"server.js"},
			},
		},
	}

	servers := []*resource.CopilotMCPServer{
		{
			Name:    "new-server",
			Type:    "stdio",
			Command: "npx",
			Args:    []string{"-y", "@mcp/server"},
			Env:     map[string]string{"KEY": "value"},
		},
	}

	result := adapter.MergeCopilotMCPConfig(existing, "my-plugin", servers)

	expected := map[string]any{
		"servers": map[string]any{
			"existing-server": map[string]any{
				"type":    "stdio",
				"command": "node",
				"args":    []string{"server.js"},
			},
			"new-server": map[string]any{
				"type":    "stdio",
				"command": "npx",
				"args":    []string{"-y", "@mcp/server"},
				"env":     map[string]string{"KEY": "value"},
			},
		},
	}
	assert.Equal(t, expected, result)
}

func TestCopilotAdapter_MergeCopilotMCPConfig_Nil(t *testing.T) {
	adapter := &CopilotAdapter{}

	servers := []*resource.CopilotMCPServer{
		{
			Name:    "server",
			Type:    "stdio",
			Command: "node",
		},
	}

	result := adapter.MergeCopilotMCPConfig(nil, "my-plugin", servers)

	expected := map[string]any{
		"servers": map[string]any{
			"server": map[string]any{
				"type":    "stdio",
				"command": "node",
			},
		},
	}
	assert.Equal(t, expected, result)
}

func TestRegisteredAdapters_IncludesCopilot(t *testing.T) {
	adapters := RegisteredAdapters()
	assert.Equal(t, []string{"claude-code", "cursor", "github-copilot"}, adapters)
}

func TestCopilotAdapter_PlanMCPServer_WithInputs(t *testing.T) {
	adapter := &CopilotAdapter{}

	server := &resource.CopilotMCPServer{
		Name:    "ado",
		Type:    "stdio",
		Command: "npx",
		Args:    []string{"-y", "@azure-devops/mcp", "${input:ado_org}"},
		Inputs: []resource.MCPInput{
			{
				ID:          "ado_org",
				Type:        "promptString",
				Description: "Azure DevOps organization name",
			},
		},
	}

	pkg := &config.PackageConfig{
		Package: config.PackageBlock{
			Name:    "my-plugin",
			Version: "1.0.0",
		},
	}

	plan, err := adapter.PlanInstallation(server, pkg, "/plugin", "/project", &InstallContext{PackageName: "my-plugin", Namespace: false})
	require.NoError(t, err)
	require.NotNil(t, plan)

	assert.Equal(t, filepath.Join(".vscode", "mcp.json"), plan.MCPPath)
	assert.Equal(t, "servers", plan.MCPKey)

	expectedMCPEntries := map[string]any{
		"servers": map[string]any{
			"ado": map[string]any{
				"type":    "stdio",
				"command": "npx",
				"args":    []string{"-y", "@azure-devops/mcp", "${input:ado_org}"},
			},
		},
		"inputs": []any{
			map[string]any{
				"id":          "ado_org",
				"type":        "promptString",
				"description": "Azure DevOps organization name",
			},
		},
	}
	assert.Equal(t, expectedMCPEntries, plan.MCPEntries)
}

func TestCopilotAdapter_PlanMCPServer_WithInputs_DefaultAndPassword(t *testing.T) {
	adapter := &CopilotAdapter{}

	server := &resource.CopilotMCPServer{
		Name:    "db",
		Type:    "stdio",
		Command: "db-server",
		Inputs: []resource.MCPInput{
			{
				ID:          "db_host",
				Type:        "promptString",
				Description: "Database host",
				Default:     "localhost",
			},
			{
				ID:          "db_password",
				Type:        "promptString",
				Description: "Database password",
				Password:    true,
			},
		},
	}

	pkg := &config.PackageConfig{
		Package: config.PackageBlock{
			Name:    "my-plugin",
			Version: "1.0.0",
		},
	}

	plan, err := adapter.PlanInstallation(server, pkg, "/plugin", "/project", &InstallContext{PackageName: "my-plugin", Namespace: false})
	require.NoError(t, err)

	expectedMCPEntries := map[string]any{
		"servers": map[string]any{
			"db": map[string]any{
				"type":    "stdio",
				"command": "db-server",
			},
		},
		"inputs": []any{
			map[string]any{
				"id":          "db_host",
				"type":        "promptString",
				"description": "Database host",
				"default":     "localhost",
			},
			map[string]any{
				"id":          "db_password",
				"type":        "promptString",
				"description": "Database password",
				"password":    true,
			},
		},
	}
	assert.Equal(t, expectedMCPEntries, plan.MCPEntries)
}

func TestCopilotAdapter_MergeCopilotMCPConfig_WithInputs(t *testing.T) {
	adapter := &CopilotAdapter{}

	servers := []*resource.CopilotMCPServer{
		{
			Name:    "ado",
			Type:    "stdio",
			Command: "npx",
			Args:    []string{"-y", "@azure-devops/mcp", "${input:ado_org}"},
			Inputs: []resource.MCPInput{
				{
					ID:          "ado_org",
					Type:        "promptString",
					Description: "Azure DevOps organization name",
				},
			},
		},
	}

	result := adapter.MergeCopilotMCPConfig(nil, "my-plugin", servers)

	expected := map[string]any{
		"servers": map[string]any{
			"ado": map[string]any{
				"type":    "stdio",
				"command": "npx",
				"args":    []string{"-y", "@azure-devops/mcp", "${input:ado_org}"},
			},
		},
		"inputs": []any{
			map[string]any{
				"id":          "ado_org",
				"type":        "promptString",
				"description": "Azure DevOps organization name",
			},
		},
	}
	assert.Equal(t, expected, result)
}

func TestCopilotAdapter_MergeCopilotMCPConfig_DeduplicatesInputs(t *testing.T) {
	adapter := &CopilotAdapter{}

	existing := map[string]any{
		"servers": map[string]any{},
		"inputs": []any{
			map[string]any{
				"id":          "shared_token",
				"type":        "promptString",
				"description": "Old description",
			},
		},
	}

	servers := []*resource.CopilotMCPServer{
		{
			Name:    "server1",
			Type:    "stdio",
			Command: "cmd1",
			Inputs: []resource.MCPInput{
				{
					ID:          "shared_token",
					Type:        "promptString",
					Description: "Updated description",
				},
				{
					ID:          "new_input",
					Type:        "promptString",
					Description: "A new input",
				},
			},
		},
	}

	result := adapter.MergeCopilotMCPConfig(existing, "my-plugin", servers)

	expectedInputs := []any{
		map[string]any{
			"id":          "shared_token",
			"type":        "promptString",
			"description": "Updated description",
		},
		map[string]any{
			"id":          "new_input",
			"type":        "promptString",
			"description": "A new input",
		},
	}
	assert.Equal(t, expectedInputs, result["inputs"])
}

func TestCopilotAdapter_MergeCopilotMCPConfig_NoInputs(t *testing.T) {
	adapter := &CopilotAdapter{}

	servers := []*resource.CopilotMCPServer{
		{
			Name:    "server1",
			Type:    "stdio",
			Command: "cmd1",
		},
	}

	result := adapter.MergeCopilotMCPConfig(nil, "my-plugin", servers)

	// "inputs" key should not exist when no inputs are declared
	_, hasInputs := result["inputs"]
	assert.False(t, hasInputs, "inputs key should not be present when no inputs exist")
}

func TestMergePlans_WithCopilotInputs(t *testing.T) {
	plan1 := &Plan{
		PluginName: "plugin1",
		MCPEntries: map[string]any{
			"servers": map[string]any{
				"server1": map[string]any{"type": "stdio", "command": "cmd1"},
			},
			"inputs": []any{
				map[string]any{"id": "input1", "type": "promptString", "description": "Input 1"},
			},
		},
		MCPPath:         ".vscode/mcp.json",
		MCPKey:          "servers",
		SettingsEntries: make(map[string]any),
	}

	plan2 := &Plan{
		PluginName: "plugin1",
		MCPEntries: map[string]any{
			"servers": map[string]any{
				"server2": map[string]any{"type": "stdio", "command": "cmd2"},
			},
			"inputs": []any{
				map[string]any{"id": "input1", "type": "promptString", "description": "Updated Input 1"},
				map[string]any{"id": "input2", "type": "promptString", "description": "Input 2"},
			},
		},
		MCPPath:         ".vscode/mcp.json",
		MCPKey:          "servers",
		SettingsEntries: make(map[string]any),
	}

	merged := MergePlans(plan1, plan2)

	// Servers should be merged
	servers := merged.MCPEntries["servers"].(map[string]any)
	assert.Contains(t, servers, "server1")
	assert.Contains(t, servers, "server2")

	// Inputs should be merged with deduplication (plan2's "input1" replaces plan1's)
	inputs := merged.MCPEntries["inputs"].([]any)
	assert.Len(t, inputs, 2)
	assert.Equal(t, "input1", inputs[0].(map[string]any)["id"])
	assert.Equal(t, "Updated Input 1", inputs[0].(map[string]any)["description"])
	assert.Equal(t, "input2", inputs[1].(map[string]any)["id"])
}

func TestMergePlans_WithCopilotPaths(t *testing.T) {
	plan1 := &Plan{
		PluginName: "plugin1",
		Directories: []DirectoryCreate{
			{Path: ".github/instructions", Parents: true},
		},
		AgentFileContent: "Instruction 1",
		AgentFilePath:    ".github/copilot-instructions.md",
		MCPEntries:       make(map[string]any),
		SettingsEntries:  make(map[string]any),
	}

	plan2 := &Plan{
		PluginName: "plugin1",
		MCPEntries: map[string]any{
			"servers": map[string]any{
				"server1": map[string]any{
					"type":    "stdio",
					"command": "cmd1",
				},
			},
		},
		MCPPath:         ".vscode/mcp.json",
		MCPKey:          "servers",
		SettingsEntries: make(map[string]any),
	}

	merged := MergePlans(plan1, plan2)

	assert.Equal(t, "plugin1", merged.PluginName)
	assert.Equal(t, ".github/copilot-instructions.md", merged.AgentFilePath)
	assert.Equal(t, ".vscode/mcp.json", merged.MCPPath)
	assert.Equal(t, "servers", merged.MCPKey)
	assert.Equal(t, "Instruction 1", merged.AgentFileContent)
	assert.Equal(t, []string{".github/instructions"}, getDirPaths(merged))
}
