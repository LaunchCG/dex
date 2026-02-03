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

// getDirPaths extracts directory paths from a plan for test assertions.
func getDirPaths(plan *Plan) []string {
	paths := make([]string, len(plan.Directories))
	for i, d := range plan.Directories {
		paths[i] = d.Path
	}
	return paths
}

func TestGet_ClaudeCode(t *testing.T) {
	adapter, err := Get("claude-code")
	require.NoError(t, err)
	assert.NotNil(t, adapter)
	assert.Equal(t, "claude-code", adapter.Name())
}

func TestGet_Unknown(t *testing.T) {
	adapter, err := Get("unknown-platform")
	assert.Nil(t, adapter)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown platform adapter")
	assert.Contains(t, err.Error(), "unknown-platform")
}

func TestRegisteredAdapters(t *testing.T) {
	adapters := RegisteredAdapters()
	require.NotEmpty(t, adapters)
	assert.Contains(t, adapters, "claude-code")

	// Verify sorted order
	for i := 1; i < len(adapters); i++ {
		assert.True(t, adapters[i-1] < adapters[i], "adapters should be sorted")
	}
}

func TestClaudeAdapter_Name(t *testing.T) {
	adapter := &ClaudeAdapter{}
	assert.Equal(t, "claude-code", adapter.Name())
}

func TestClaudeAdapter_Directories(t *testing.T) {
	adapter := &ClaudeAdapter{}
	root := "/project"

	assert.Equal(t, "/project/.claude", adapter.BaseDir(root))
	assert.Equal(t, "/project/.claude/skills", adapter.SkillsDir(root))
	assert.Equal(t, "/project/.claude/commands", adapter.CommandsDir(root))
	assert.Equal(t, "/project/.claude/agents", adapter.SubagentsDir(root))
	assert.Equal(t, "/project/.claude/rules", adapter.RulesDir(root))
}

func TestClaudeAdapter_PlanSkill(t *testing.T) {
	adapter := &ClaudeAdapter{}

	skill := &resource.ClaudeSkill{
		Name:        "test-skill",
		Description: "A test skill",
		Content:     "Skill content here",
	}

	pkg := &config.PackageConfig{
		Package: config.PackageBlock{
			Name:    "my-plugin",
			Version: "1.0.0",
		},
	}

	plan, err := adapter.PlanInstallation(skill, pkg, "/plugin", "/project", &InstallContext{PackageName: "my-plugin", Namespace: true})
	require.NoError(t, err)
	assert.NotNil(t, plan)
	assert.Equal(t, "my-plugin", plan.PluginName)

	// Should create skill directory
	assert.Contains(t, getDirPaths(plan), ".claude/skills/my-plugin-test-skill")

	// Should have SKILL.md file
	require.Len(t, plan.Files, 1)
	assert.Equal(t, ".claude/skills/my-plugin-test-skill/SKILL.md", plan.Files[0].Path)
	assert.Contains(t, plan.Files[0].Content, "name: test-skill")
	assert.Contains(t, plan.Files[0].Content, "description: A test skill")
	assert.Contains(t, plan.Files[0].Content, "Skill content here")
}

func TestClaudeAdapter_PlanCommand(t *testing.T) {
	adapter := &ClaudeAdapter{}

	cmd := &resource.ClaudeCommand{
		Name:         "deploy",
		Description:  "Deploy the application",
		Content:      "Deployment instructions",
		ArgumentHint: "[environment]",
		Model:        "sonnet",
	}

	pkg := &config.PackageConfig{
		Package: config.PackageBlock{
			Name:    "my-plugin",
			Version: "1.0.0",
		},
	}

	plan, err := adapter.PlanInstallation(cmd, pkg, "/plugin", "/project", &InstallContext{PackageName: "my-plugin", Namespace: true})
	require.NoError(t, err)
	assert.NotNil(t, plan)

	// Should create commands directory
	assert.Contains(t, getDirPaths(plan), ".claude/commands")

	// Should have command file
	require.Len(t, plan.Files, 1)
	assert.Equal(t, ".claude/commands/my-plugin-deploy.md", plan.Files[0].Path)
	assert.Contains(t, plan.Files[0].Content, "name: deploy")
	assert.Contains(t, plan.Files[0].Content, "argument_hint: [environment]")
	assert.Contains(t, plan.Files[0].Content, "model: sonnet")
}

func TestClaudeAdapter_PlanSubagent(t *testing.T) {
	adapter := &ClaudeAdapter{}

	agent := &resource.ClaudeSubagent{
		Name:        "researcher",
		Description: "Research agent",
		Content:     "Research instructions",
		Model:       "opus",
		Color:       "blue",
		Tools:       []string{"Read", "WebSearch"},
	}

	pkg := &config.PackageConfig{
		Package: config.PackageBlock{
			Name:    "my-plugin",
			Version: "1.0.0",
		},
	}

	plan, err := adapter.PlanInstallation(agent, pkg, "/plugin", "/project", &InstallContext{PackageName: "my-plugin", Namespace: true})
	require.NoError(t, err)
	assert.NotNil(t, plan)

	// Should create agents directory
	assert.Contains(t, getDirPaths(plan), ".claude/agents")

	// Should have agent file
	require.Len(t, plan.Files, 1)
	assert.Equal(t, ".claude/agents/my-plugin-researcher.md", plan.Files[0].Path)
	assert.Contains(t, plan.Files[0].Content, "name: researcher")
	assert.Contains(t, plan.Files[0].Content, "model: opus")
	assert.Contains(t, plan.Files[0].Content, "color: blue")
	assert.Contains(t, plan.Files[0].Content, "- Read")
	assert.Contains(t, plan.Files[0].Content, "- WebSearch")
}

func TestClaudeAdapter_PlanRule(t *testing.T) {
	adapter := &ClaudeAdapter{}

	rule := &resource.ClaudeRule{
		Name:        "coding-style",
		Description: "Coding style rules",
		Content:     "Always use tabs for indentation",
	}

	pkg := &config.PackageConfig{
		Package: config.PackageBlock{
			Name:    "my-plugin",
			Version: "1.0.0",
		},
	}

	plan, err := adapter.PlanInstallation(rule, pkg, "/plugin", "/project", &InstallContext{PackageName: "my-plugin", Namespace: true})
	require.NoError(t, err)
	assert.NotNil(t, plan)

	// Rules are merged into CLAUDE.md
	assert.Equal(t, "Always use tabs for indentation", plan.AgentFileContent)
	assert.Empty(t, plan.Files)
}

func TestClaudeAdapter_PlanRules(t *testing.T) {
	adapter := &ClaudeAdapter{}

	rules := &resource.ClaudeRules{
		Name:        "typescript-rules",
		Description: "TypeScript rules",
		Content:     "TypeScript specific rules content",
		Paths:       []string{"*.ts", "*.tsx"},
	}

	pkg := &config.PackageConfig{
		Package: config.PackageBlock{
			Name:    "my-plugin",
			Version: "1.0.0",
		},
	}

	plan, err := adapter.PlanInstallation(rules, pkg, "/plugin", "/project", &InstallContext{PackageName: "my-plugin", Namespace: true})
	require.NoError(t, err)
	assert.NotNil(t, plan)

	// Should create rules subdirectory (similar to skills)
	assert.Contains(t, getDirPaths(plan), ".claude/rules/my-plugin-typescript-rules")

	// Should have main rules file in the subdirectory
	require.Len(t, plan.Files, 1)
	assert.Equal(t, ".claude/rules/my-plugin-typescript-rules/typescript-rules.md", plan.Files[0].Path)
	assert.Contains(t, plan.Files[0].Content, "name: typescript-rules")
	assert.Contains(t, plan.Files[0].Content, "- *.ts")
	assert.Contains(t, plan.Files[0].Content, "- *.tsx")
}

func TestClaudeAdapter_PlanRules_WithFiles(t *testing.T) {
	adapter := &ClaudeAdapter{}

	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	helperFile := filepath.Join(tmpDir, "helper.md")
	err := os.WriteFile(helperFile, []byte("Helper content"), 0644)
	require.NoError(t, err)

	rules := &resource.ClaudeRules{
		Name:        "tailwind",
		Description: "Tailwind CSS standards",
		Content:     "# Tailwind Rules\n\nMain rules content here",
		Files: []resource.FileBlock{
			{
				Src:  "helper.md",
				Dest: "tailwind-classes.md",
			},
			{
				Src:  "helper.md",
				Dest: "tailwind-components.md",
			},
		},
	}

	pkg := &config.PackageConfig{
		Package: config.PackageBlock{
			Name:    "my-plugin",
			Version: "1.0.0",
		},
	}

	plan, err := adapter.PlanInstallation(rules, pkg, tmpDir, "/project", &InstallContext{PackageName: "my-plugin", Namespace: false})
	require.NoError(t, err)
	assert.NotNil(t, plan)

	// Should create rules subdirectory
	assert.Contains(t, getDirPaths(plan), ".claude/rules/tailwind")

	// Should have main rules file + 2 additional files
	require.Len(t, plan.Files, 3)

	// Check main file
	assert.Equal(t, ".claude/rules/tailwind/tailwind.md", plan.Files[0].Path)
	assert.Contains(t, plan.Files[0].Content, "Main rules content here")

	// Check additional files
	assert.Equal(t, ".claude/rules/tailwind/tailwind-classes.md", plan.Files[1].Path)
	assert.Equal(t, "Helper content", plan.Files[1].Content)

	assert.Equal(t, ".claude/rules/tailwind/tailwind-components.md", plan.Files[2].Path)
	assert.Equal(t, "Helper content", plan.Files[2].Content)
}

func TestClaudeAdapter_PlanSettings(t *testing.T) {
	adapter := &ClaudeAdapter{}

	settings := &resource.ClaudeSettings{
		Name:  "plugin-settings",
		Allow: []string{"Bash(*)"},
		Deny:  []string{"Write(/etc/*)"},
		Env:   map[string]string{"DEBUG": "true"},
	}

	pkg := &config.PackageConfig{
		Package: config.PackageBlock{
			Name:    "my-plugin",
			Version: "1.0.0",
		},
	}

	plan, err := adapter.PlanInstallation(settings, pkg, "/plugin", "/project", &InstallContext{PackageName: "my-plugin", Namespace: true})
	require.NoError(t, err)
	assert.NotNil(t, plan)

	// Settings are merged via SettingsEntries
	assert.NotEmpty(t, plan.SettingsEntries)
	assert.Empty(t, plan.Files)
}

func TestClaudeAdapter_PlanMCPServer(t *testing.T) {
	adapter := &ClaudeAdapter{}

	server := &resource.ClaudeMCPServer{
		Name:    "github",
		Type:    "command",
		Command: "npx",
		Args:    []string{"-y", "@modelcontextprotocol/server-github"},
		Env:     map[string]string{"GITHUB_TOKEN": "${GITHUB_TOKEN}"},
	}

	pkg := &config.PackageConfig{
		Package: config.PackageBlock{
			Name:    "my-plugin",
			Version: "1.0.0",
		},
	}

	plan, err := adapter.PlanInstallation(server, pkg, "/plugin", "/project", &InstallContext{PackageName: "my-plugin", Namespace: false})
	require.NoError(t, err)
	assert.NotNil(t, plan)

	// MCP servers are merged via MCPEntries
	assert.NotEmpty(t, plan.MCPEntries)
	mcpServers, ok := plan.MCPEntries["mcpServers"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, mcpServers, "github")
}

// mockUnknownResource implements resource.Resource but is not a known type
type mockUnknownResource struct {
	name string
}

func (m *mockUnknownResource) ResourceType() string                           { return "unknown_type" }
func (m *mockUnknownResource) ResourceName() string                           { return m.name }
func (m *mockUnknownResource) Platform() string                               { return "unknown" }
func (m *mockUnknownResource) GetContent() string                             { return "" }
func (m *mockUnknownResource) GetFiles() []resource.FileBlock                 { return nil }
func (m *mockUnknownResource) GetTemplateFiles() []resource.TemplateFileBlock { return nil }
func (m *mockUnknownResource) Validate() error                                { return nil }

func TestClaudeAdapter_PlanInstallation_UnsupportedType(t *testing.T) {
	adapter := &ClaudeAdapter{}

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
	assert.Contains(t, err.Error(), "unsupported resource type")
}

func TestClaudeAdapter_GenerateFrontmatter_Skill(t *testing.T) {
	adapter := &ClaudeAdapter{}

	skill := &resource.ClaudeSkill{
		Name:        "test-skill",
		Description: "A test skill",
		Metadata: map[string]string{
			"category": "testing",
		},
	}

	pkg := &config.PackageConfig{
		Package: config.PackageBlock{
			Name:    "my-plugin",
			Version: "1.0.0",
		},
	}

	frontmatter := adapter.GenerateFrontmatter(skill, pkg)
	expected := `---
name: test-skill
description: A test skill
category: testing
---
`
	assert.Equal(t, expected, frontmatter)
}

func TestClaudeAdapter_GenerateFrontmatter_Skill_NoMetadata(t *testing.T) {
	adapter := &ClaudeAdapter{}

	skill := &resource.ClaudeSkill{
		Name:        "simple-skill",
		Description: "Simple skill",
	}

	pkg := &config.PackageConfig{}

	frontmatter := adapter.GenerateFrontmatter(skill, pkg)
	expected := `---
name: simple-skill
description: Simple skill
---
`
	assert.Equal(t, expected, frontmatter)
}

func TestClaudeAdapter_GenerateFrontmatter_Command(t *testing.T) {
	adapter := &ClaudeAdapter{}

	cmd := &resource.ClaudeCommand{
		Name:         "deploy",
		Description:  "Deploy app",
		ArgumentHint: "[env]",
		AllowedTools: []string{"Bash", "Read"},
		Model:        "opus",
	}

	pkg := &config.PackageConfig{}

	frontmatter := adapter.GenerateFrontmatter(cmd, pkg)
	expected := `---
name: deploy
description: Deploy app
argument_hint: [env]
allowed_tools:
- Bash
- Read
model: opus
---
`
	assert.Equal(t, expected, frontmatter)
}

func TestClaudeAdapter_GenerateFrontmatter_Command_Minimal(t *testing.T) {
	adapter := &ClaudeAdapter{}

	cmd := &resource.ClaudeCommand{
		Name:        "simple",
		Description: "Simple command",
	}

	pkg := &config.PackageConfig{}

	frontmatter := adapter.GenerateFrontmatter(cmd, pkg)
	expected := `---
name: simple
description: Simple command
---
`
	assert.Equal(t, expected, frontmatter)
}

func TestClaudeAdapter_MergeMCPConfig(t *testing.T) {
	adapter := &ClaudeAdapter{}

	existing := map[string]any{
		"mcpServers": map[string]any{
			"existing-server": map[string]any{
				"command": "node",
				"args":    []string{"server.js"},
			},
		},
	}

	servers := []*resource.ClaudeMCPServer{
		{
			Name:    "new-server",
			Type:    "command",
			Command: "npx",
			Args:    []string{"-y", "@mcp/server"},
			Env:     map[string]string{"KEY": "value"},
		},
	}

	result := adapter.MergeMCPConfig(existing, "my-plugin", servers)

	mcpServers := result["mcpServers"].(map[string]any)
	assert.Contains(t, mcpServers, "existing-server")
	assert.Contains(t, mcpServers, "new-server")

	newServer := mcpServers["new-server"].(map[string]any)
	assert.Equal(t, "npx", newServer["command"])
	assert.Equal(t, []string{"-y", "@mcp/server"}, newServer["args"])
	assert.Equal(t, map[string]string{"KEY": "value"}, newServer["env"])
}

func TestClaudeAdapter_MergeMCPConfig_HTTP(t *testing.T) {
	adapter := &ClaudeAdapter{}

	servers := []*resource.ClaudeMCPServer{
		{
			Name: "http-server",
			Type: "http",
			URL:  "https://api.example.com/mcp",
		},
	}

	result := adapter.MergeMCPConfig(nil, "my-plugin", servers)

	mcpServers := result["mcpServers"].(map[string]any)
	httpServer := mcpServers["http-server"].(map[string]any)
	assert.Equal(t, "https://api.example.com/mcp", httpServer["url"])
}

func TestClaudeAdapter_MergeMCPConfig_Nil(t *testing.T) {
	adapter := &ClaudeAdapter{}

	servers := []*resource.ClaudeMCPServer{
		{
			Name:    "server",
			Type:    "command",
			Command: "node",
		},
	}

	result := adapter.MergeMCPConfig(nil, "my-plugin", servers)

	assert.NotNil(t, result)
	mcpServers := result["mcpServers"].(map[string]any)
	assert.Contains(t, mcpServers, "server")
}

func TestClaudeAdapter_MergeSettingsConfig(t *testing.T) {
	adapter := &ClaudeAdapter{}

	existing := map[string]any{
		"allow": []any{"Read(*)"},
		"env": map[string]any{
			"EXISTING": "value",
		},
	}

	settings := &resource.ClaudeSettings{
		Name:             "test",
		Allow:            []string{"Write(*)"},
		Deny:             []string{"Delete(*)"},
		Env:              map[string]string{"NEW": "new_value"},
		RespectGitignore: true,
		Model:            "opus",
		OutputStyle:      "concise",
		PlansDirectory:   ".plans",
	}

	result := adapter.MergeSettingsConfig(existing, settings)

	// Check merged arrays
	allow := result["allow"].([]any)
	assert.Contains(t, allow, "Read(*)")
	assert.Contains(t, allow, "Write(*)")

	deny := result["deny"].([]any)
	assert.Contains(t, deny, "Delete(*)")

	// Check merged env
	env := result["env"].(map[string]any)
	assert.Equal(t, "value", env["EXISTING"])
	assert.Equal(t, "new_value", env["NEW"])

	// Check scalar values
	assert.Equal(t, true, result["respectGitignore"])
	assert.Equal(t, "opus", result["model"])
	assert.Equal(t, "concise", result["outputStyle"])
	assert.Equal(t, ".plans", result["plansDirectory"])
}

func TestClaudeAdapter_MergeSettingsConfig_Nil(t *testing.T) {
	adapter := &ClaudeAdapter{}

	settings := &resource.ClaudeSettings{
		Name:  "test",
		Allow: []string{"Bash(*)"},
	}

	result := adapter.MergeSettingsConfig(nil, settings)

	assert.NotNil(t, result)
	allow := result["allow"].([]any)
	assert.Contains(t, allow, "Bash(*)")
}

func TestClaudeAdapter_MergeAgentFile(t *testing.T) {
	adapter := &ClaudeAdapter{}

	existing := "# Existing Content\n\nSome existing content."
	content := "New rule content"

	result := adapter.MergeAgentFile(existing, "my-plugin", content)

	assert.Contains(t, result, "# Existing Content")
	assert.Contains(t, result, "<!-- dex:my-plugin -->")
	assert.Contains(t, result, "New rule content")
	assert.Contains(t, result, "<!-- /dex:my-plugin -->")
}

func TestClaudeAdapter_MergeAgentFile_Update(t *testing.T) {
	adapter := &ClaudeAdapter{}

	existing := `# Existing Content

<!-- dex:my-plugin -->
Old content
<!-- /dex:my-plugin -->

Other content`

	content := "Updated content"

	result := adapter.MergeAgentFile(existing, "my-plugin", content)

	assert.Contains(t, result, "# Existing Content")
	assert.Contains(t, result, "Updated content")
	assert.NotContains(t, result, "Old content")
	assert.Contains(t, result, "Other content")

	// Should only have one pair of markers
	assert.Equal(t, 1, countOccurrences(result, "<!-- dex:my-plugin -->"))
	assert.Equal(t, 1, countOccurrences(result, "<!-- /dex:my-plugin -->"))
}

func TestClaudeAdapter_MergeAgentFile_Empty(t *testing.T) {
	adapter := &ClaudeAdapter{}

	result := adapter.MergeAgentFile("", "my-plugin", "New content")

	expected := `<!-- dex:my-plugin -->
New content
<!-- /dex:my-plugin -->`
	assert.Equal(t, expected, result)
}

func TestMergePlans(t *testing.T) {
	plan1 := &Plan{
		PluginName: "plugin1",
		Directories: []DirectoryCreate{
			{Path: "dir1", Parents: true},
			{Path: "dir2", Parents: true},
		},
		Files: []FileWrite{
			{Path: "file1.md", Content: "content1"},
		},
		MCPEntries: map[string]any{
			"server1": "config1",
		},
		SettingsEntries: map[string]any{
			"allow": []string{"Bash(*)"},
		},
		AgentFileContent: "Rule 1",
	}

	plan2 := &Plan{
		PluginName: "plugin1",
		Directories: []DirectoryCreate{
			{Path: "dir2", Parents: true}, // dir2 is duplicate
			{Path: "dir3", Parents: true},
		},
		Files: []FileWrite{
			{Path: "file2.md", Content: "content2"},
		},
		MCPEntries: map[string]any{
			"server2": "config2",
		},
		SettingsEntries: map[string]any{
			"deny": []string{"Write(*)"},
		},
		AgentFileContent: "Rule 2",
	}

	merged := MergePlans(plan1, plan2)

	assert.Equal(t, "plugin1", merged.PluginName)

	// Directories should be deduplicated by path
	assert.Len(t, merged.Directories, 3)
	dirPaths := make([]string, len(merged.Directories))
	for i, d := range merged.Directories {
		dirPaths[i] = d.Path
	}
	assert.Contains(t, dirPaths, "dir1")
	assert.Contains(t, dirPaths, "dir2")
	assert.Contains(t, dirPaths, "dir3")

	// Files should be concatenated
	assert.Len(t, merged.Files, 2)

	// MCP entries should be merged
	assert.Contains(t, merged.MCPEntries, "server1")
	assert.Contains(t, merged.MCPEntries, "server2")

	// Settings entries should be merged
	assert.Contains(t, merged.SettingsEntries, "allow")
	assert.Contains(t, merged.SettingsEntries, "deny")

	// Agent file content should be concatenated
	assert.Contains(t, merged.AgentFileContent, "Rule 1")
	assert.Contains(t, merged.AgentFileContent, "Rule 2")
}

func TestMergePlans_Empty(t *testing.T) {
	merged := MergePlans()

	assert.NotNil(t, merged)
	assert.Empty(t, merged.PluginName)
	assert.Empty(t, merged.Directories)
	assert.Empty(t, merged.Files)
	assert.Empty(t, merged.MCPEntries)
	assert.Empty(t, merged.SettingsEntries)
	assert.Empty(t, merged.AgentFileContent)
}

func TestMergePlans_WithNil(t *testing.T) {
	plan1 := &Plan{
		PluginName: "plugin1",
		Files: []FileWrite{
			{Path: "file.md", Content: "content"},
		},
	}

	merged := MergePlans(plan1, nil)

	assert.Equal(t, "plugin1", merged.PluginName)
	assert.Len(t, merged.Files, 1)
}

func TestMergePlans_MCPDeepMerge(t *testing.T) {
	// Test that mcpServers are deep merged, not overwritten
	plan1 := &Plan{
		PluginName: "plugin1",
		MCPEntries: map[string]any{
			"mcpServers": map[string]any{
				"server1": map[string]any{
					"command": "cmd1",
					"args":    []string{"arg1"},
				},
			},
		},
	}

	plan2 := &Plan{
		PluginName: "plugin1",
		MCPEntries: map[string]any{
			"mcpServers": map[string]any{
				"server2": map[string]any{
					"command": "cmd2",
					"args":    []string{"arg2"},
				},
			},
		},
	}

	plan3 := &Plan{
		PluginName: "plugin1",
		MCPEntries: map[string]any{
			"mcpServers": map[string]any{
				"server3": map[string]any{
					"command": "cmd3",
					"args":    []string{"arg3"},
				},
			},
		},
	}

	merged := MergePlans(plan1, plan2, plan3)

	// All three servers should be present
	mcpServers, ok := merged.MCPEntries["mcpServers"].(map[string]any)
	require.True(t, ok, "mcpServers should be a map")
	assert.Len(t, mcpServers, 3, "should have 3 MCP servers after merge")
	assert.Contains(t, mcpServers, "server1")
	assert.Contains(t, mcpServers, "server2")
	assert.Contains(t, mcpServers, "server3")

	// Verify each server's config is preserved
	server1 := mcpServers["server1"].(map[string]any)
	assert.Equal(t, "cmd1", server1["command"])

	server2 := mcpServers["server2"].(map[string]any)
	assert.Equal(t, "cmd2", server2["command"])

	server3 := mcpServers["server3"].(map[string]any)
	assert.Equal(t, "cmd3", server3["command"])
}

func TestMergePlans_SettingsArrayMerge(t *testing.T) {
	// Test that settings arrays are appended, not overwritten
	plan1 := &Plan{
		PluginName: "plugin1",
		SettingsEntries: map[string]any{
			"allow": []string{"Bash(*)", "Read(*)"},
		},
	}

	plan2 := &Plan{
		PluginName: "plugin1",
		SettingsEntries: map[string]any{
			"allow": []string{"Write(*)"},
			"deny":  []string{"Delete(*)"},
		},
	}

	merged := MergePlans(plan1, plan2)

	// allow should contain all entries from both plans
	allow, ok := merged.SettingsEntries["allow"].([]string)
	require.True(t, ok, "allow should be []string")
	assert.Len(t, allow, 3)
	assert.Contains(t, allow, "Bash(*)")
	assert.Contains(t, allow, "Read(*)")
	assert.Contains(t, allow, "Write(*)")

	// deny should be present from plan2
	deny, ok := merged.SettingsEntries["deny"].([]string)
	require.True(t, ok, "deny should be []string")
	assert.Contains(t, deny, "Delete(*)")
}

func TestMergePlans_MCPWithNilInitialServers(t *testing.T) {
	// Test merging when first plan has no mcpServers key
	plan1 := &Plan{
		PluginName: "plugin1",
		MCPEntries: map[string]any{}, // No mcpServers key
	}

	plan2 := &Plan{
		PluginName: "plugin1",
		MCPEntries: map[string]any{
			"mcpServers": map[string]any{
				"server1": map[string]any{
					"command": "cmd1",
				},
			},
		},
	}

	merged := MergePlans(plan1, plan2)

	mcpServers, ok := merged.MCPEntries["mcpServers"].(map[string]any)
	require.True(t, ok, "mcpServers should be a map")
	assert.Contains(t, mcpServers, "server1")
}

func TestNewPlan(t *testing.T) {
	plan := NewPlan("my-plugin")

	assert.Equal(t, "my-plugin", plan.PluginName)
	assert.NotNil(t, plan.MCPEntries)
	assert.NotNil(t, plan.SettingsEntries)
	assert.Empty(t, plan.Directories)
	assert.Empty(t, plan.Files)
	assert.Empty(t, plan.AgentFileContent)
}

func TestPlan_AddDirectory(t *testing.T) {
	plan := NewPlan("plugin")
	plan.AddDirectory("dir1", true)
	plan.AddDirectory("dir2", false)

	require.Len(t, plan.Directories, 2)
	assert.Equal(t, "dir1", plan.Directories[0].Path)
	assert.True(t, plan.Directories[0].Parents)
	assert.Equal(t, "dir2", plan.Directories[1].Path)
	assert.False(t, plan.Directories[1].Parents)
}

func TestPlan_AddFile(t *testing.T) {
	plan := NewPlan("plugin")
	plan.AddFile("path/to/file.md", "content", "644")

	require.Len(t, plan.Files, 1)
	assert.Equal(t, "path/to/file.md", plan.Files[0].Path)
	assert.Equal(t, "content", plan.Files[0].Content)
	assert.Equal(t, "644", plan.Files[0].Chmod)
}

func TestPlan_IsEmpty(t *testing.T) {
	plan := NewPlan("plugin")
	assert.True(t, plan.IsEmpty())

	plan.AddDirectory("dir", true)
	assert.False(t, plan.IsEmpty())

	plan2 := NewPlan("plugin")
	plan2.AddFile("file", "content", "")
	assert.False(t, plan2.IsEmpty())

	plan3 := NewPlan("plugin")
	plan3.MCPEntries["server"] = "config"
	assert.False(t, plan3.IsEmpty())

	plan4 := NewPlan("plugin")
	plan4.SettingsEntries["allow"] = []string{"*"}
	assert.False(t, plan4.IsEmpty())

	plan5 := NewPlan("plugin")
	plan5.AgentFileContent = "content"
	assert.False(t, plan5.IsEmpty())
}

func TestPlan_FilePaths(t *testing.T) {
	plan := NewPlan("plugin")
	plan.AddFile("file1.md", "content1", "")
	plan.AddFile("file2.md", "content2", "")
	plan.AddFile("dir/file3.md", "content3", "")

	paths := plan.FilePaths()
	assert.Equal(t, []string{"file1.md", "file2.md", "dir/file3.md"}, paths)
}

func TestClaudeAdapter_PlanSkill_WithFiles(t *testing.T) {
	// Create a temp directory with a file to copy
	tmpDir := t.TempDir()
	err := os.MkdirAll(filepath.Join(tmpDir, "assets"), 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "assets", "helper.sh"), []byte("#!/bin/bash\necho hello"), 0644)
	require.NoError(t, err)

	adapter := &ClaudeAdapter{}

	skill := &resource.ClaudeSkill{
		Name:        "test-skill",
		Description: "A test skill",
		Content:     "Skill content",
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

	// Should have SKILL.md and helper.sh
	require.Len(t, plan.Files, 2)

	// Find the helper.sh file
	var helperFile *FileWrite
	for i := range plan.Files {
		if filepath.Base(plan.Files[i].Path) == "helper.sh" {
			helperFile = &plan.Files[i]
			break
		}
	}
	require.NotNil(t, helperFile)
	assert.Contains(t, helperFile.Content, "#!/bin/bash")
	assert.Equal(t, "755", helperFile.Chmod)
}

func TestClaudeAdapter_PlanSkill_ContentNotTemplated(t *testing.T) {
	adapter := &ClaudeAdapter{}

	// Content with template-like syntax should NOT be rendered
	// Users should use templatefile() in HCL for templating
	skill := &resource.ClaudeSkill{
		Name:        "dynamic",
		Description: "A dynamic skill",
		Content:     "Plugin: {{ .PluginName }}, JSX: {{ borderColor: dynamic }}",
	}

	pkg := &config.PackageConfig{
		Package: config.PackageBlock{
			Name:    "my-plugin",
			Version: "2.0.0",
		},
	}

	plan, err := adapter.PlanInstallation(skill, pkg, "/plugin", "/project", &InstallContext{PackageName: "my-plugin", Namespace: true})
	require.NoError(t, err)
	require.Len(t, plan.Files, 1)

	// Content should be passed through as-is, not templated
	assert.Contains(t, plan.Files[0].Content, "Plugin: {{ .PluginName }}")
	assert.Contains(t, plan.Files[0].Content, "JSX: {{ borderColor: dynamic }}")
}

func TestClaudeAdapter_PlanSkill_WithTemplateFiles(t *testing.T) {
	// Create a temp directory with a template file
	tmpDir := t.TempDir()
	templateContent := "Config for {{ .PluginName }} v{{ .PluginVersion }}\nEnv: {{ .environment }}"
	err := os.WriteFile(filepath.Join(tmpDir, "config.yaml.tmpl"), []byte(templateContent), 0644)
	require.NoError(t, err)

	adapter := &ClaudeAdapter{}

	skill := &resource.ClaudeSkill{
		Name:        "test-skill",
		Description: "A test skill",
		Content:     "Skill content",
		TemplateFiles: []resource.TemplateFileBlock{
			{
				Src:  "config.yaml.tmpl",
				Dest: "config.yaml",
				Vars: map[string]string{"environment": "production"},
			},
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

	// Should have SKILL.md and config.yaml
	require.Len(t, plan.Files, 2)

	// Find the config.yaml file
	var configFile *FileWrite
	for i := range plan.Files {
		if filepath.Base(plan.Files[i].Path) == "config.yaml" {
			configFile = &plan.Files[i]
			break
		}
	}
	require.NotNil(t, configFile)

	// Template should be rendered with context and extra vars
	assert.Contains(t, configFile.Content, "Config for my-plugin v1.0.0")
	assert.Contains(t, configFile.Content, "Env: production")
}

func TestClaudeAdapter_PlanRule_ContentNotTemplated(t *testing.T) {
	adapter := &ClaudeAdapter{}

	// Content with template-like syntax should NOT be rendered
	rule := &resource.ClaudeRule{
		Name:        "dynamic-rule",
		Description: "A dynamic rule",
		Content:     "This rule is from {{ .PluginName }} version {{ .PluginVersion }}",
	}

	pkg := &config.PackageConfig{
		Package: config.PackageBlock{
			Name:    "rules-plugin",
			Version: "3.0.0",
		},
	}

	plan, err := adapter.PlanInstallation(rule, pkg, "/plugin", "/project", &InstallContext{PackageName: "my-plugin", Namespace: true})
	require.NoError(t, err)

	// Content should be passed through as-is, not templated
	assert.Contains(t, plan.AgentFileContent, "{{ .PluginName }}")
	assert.Contains(t, plan.AgentFileContent, "{{ .PluginVersion }}")
}

func TestClaudeAdapter_GenerateFrontmatter_Subagent(t *testing.T) {
	adapter := &ClaudeAdapter{}

	agent := &resource.ClaudeSubagent{
		Name:        "researcher",
		Description: "Research assistant",
		Model:       "opus",
		Color:       "green",
		Tools:       []string{"Read", "WebSearch"},
	}

	pkg := &config.PackageConfig{}

	frontmatter := adapter.GenerateFrontmatter(agent, pkg)
	expected := `---
name: researcher
description: Research assistant
model: opus
color: green
tools:
- Read
- WebSearch
---
`
	assert.Equal(t, expected, frontmatter)
}

func TestClaudeAdapter_GenerateFrontmatter_Rules(t *testing.T) {
	adapter := &ClaudeAdapter{}

	rules := &resource.ClaudeRules{
		Name:        "go-rules",
		Description: "Go language rules",
		Paths:       []string{"*.go", "go.mod"},
	}

	pkg := &config.PackageConfig{}

	frontmatter := adapter.GenerateFrontmatter(rules, pkg)
	expected := `---
name: go-rules
description: Go language rules
paths:
- *.go
- go.mod
---
`
	assert.Equal(t, expected, frontmatter)
}

func TestClaudeAdapter_GenerateFrontmatter_UnknownType(t *testing.T) {
	adapter := &ClaudeAdapter{}

	// Settings doesn't generate frontmatter
	settings := &resource.ClaudeSettings{Name: "test"}
	pkg := &config.PackageConfig{}

	frontmatter := adapter.GenerateFrontmatter(settings, pkg)
	assert.Equal(t, "", frontmatter)
}

// Helper function to count occurrences of a substring
func countOccurrences(s, substr string) int {
	count := 0
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			count++
		}
	}
	return count
}

func TestClaudeAdapter_GenerateFrontmatter_Skill_WithHooks(t *testing.T) {
	adapter := &ClaudeAdapter{}

	skill := &resource.ClaudeSkill{
		Name:        "test-skill",
		Description: "A test skill with hooks",
		Hooks: map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "Bash",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "./scripts/check.sh",
							"timeout": 30,
						},
					},
				},
			},
		},
	}

	pkg := &config.PackageConfig{
		Package: config.PackageBlock{
			Name:    "my-plugin",
			Version: "1.0.0",
		},
	}

	frontmatter := adapter.GenerateFrontmatter(skill, pkg)

	// Verify basic frontmatter fields
	assert.Contains(t, frontmatter, "name: test-skill")
	assert.Contains(t, frontmatter, "description: A test skill with hooks")

	// Verify hooks section is present
	assert.Contains(t, frontmatter, "hooks:")
	assert.Contains(t, frontmatter, "PreToolUse:")
	assert.Contains(t, frontmatter, "matcher: Bash")
	assert.Contains(t, frontmatter, "type: command")
	assert.Contains(t, frontmatter, "command: ./scripts/check.sh")
	assert.Contains(t, frontmatter, "timeout: 30")
}

func TestClaudeAdapter_GenerateFrontmatter_Skill_WithMultipleHooks(t *testing.T) {
	adapter := &ClaudeAdapter{}

	skill := &resource.ClaudeSkill{
		Name:        "security-skill",
		Description: "Skill with multiple hook types",
		Hooks: map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "Bash",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "./check.sh",
						},
					},
				},
			},
			"SessionStart": []interface{}{
				map[string]interface{}{
					"hooks": []interface{}{
						map[string]interface{}{
							"type":   "prompt",
							"prompt": "Initialize security context",
						},
					},
				},
			},
		},
	}

	pkg := &config.PackageConfig{}

	frontmatter := adapter.GenerateFrontmatter(skill, pkg)

	// Verify multiple hook events
	assert.Contains(t, frontmatter, "hooks:")
	assert.Contains(t, frontmatter, "PreToolUse:")
	assert.Contains(t, frontmatter, "SessionStart:")
	assert.Contains(t, frontmatter, "type: command")
	assert.Contains(t, frontmatter, "type: prompt")
}

func TestClaudeAdapter_GenerateFrontmatter_Skill_NoHooks(t *testing.T) {
	adapter := &ClaudeAdapter{}

	skill := &resource.ClaudeSkill{
		Name:        "simple-skill",
		Description: "Simple skill without hooks",
	}

	pkg := &config.PackageConfig{}

	frontmatter := adapter.GenerateFrontmatter(skill, pkg)
	expected := `---
name: simple-skill
description: Simple skill without hooks
---
`
	assert.Equal(t, expected, frontmatter)

	// Ensure no hooks section when hooks map is nil or empty
	assert.NotContains(t, frontmatter, "hooks:")
}

func TestClaudeAdapter_GenerateFrontmatter_Subagent_WithHooks(t *testing.T) {
	adapter := &ClaudeAdapter{}

	agent := &resource.ClaudeSubagent{
		Name:        "security-agent",
		Description: "Security agent with hooks",
		Model:       "sonnet",
		Color:       "red",
		Hooks: map[string]interface{}{
			"SubagentStart": []interface{}{
				map[string]interface{}{
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "./init-security.sh",
							"async":   true,
						},
					},
				},
			},
		},
	}

	pkg := &config.PackageConfig{}

	frontmatter := adapter.GenerateFrontmatter(agent, pkg)

	// Verify basic frontmatter fields
	assert.Contains(t, frontmatter, "name: security-agent")
	assert.Contains(t, frontmatter, "description: Security agent with hooks")
	assert.Contains(t, frontmatter, "model: sonnet")
	assert.Contains(t, frontmatter, "color: red")

	// Verify hooks section
	assert.Contains(t, frontmatter, "hooks:")
	assert.Contains(t, frontmatter, "SubagentStart:")
	assert.Contains(t, frontmatter, "type: command")
	assert.Contains(t, frontmatter, "command: ./init-security.sh")
	assert.Contains(t, frontmatter, "async: true")
}

func TestClaudeAdapter_GenerateFrontmatter_Subagent_NoHooks(t *testing.T) {
	adapter := &ClaudeAdapter{}

	agent := &resource.ClaudeSubagent{
		Name:        "simple-agent",
		Description: "Simple agent without hooks",
		Model:       "haiku",
	}

	pkg := &config.PackageConfig{}

	frontmatter := adapter.GenerateFrontmatter(agent, pkg)
	expected := `---
name: simple-agent
description: Simple agent without hooks
model: haiku
---
`
	assert.Equal(t, expected, frontmatter)

	// Ensure no hooks section when hooks map is nil or empty
	assert.NotContains(t, frontmatter, "hooks:")
}

func TestClaudeAdapter_PlanSkill_WithHooks(t *testing.T) {
	adapter := &ClaudeAdapter{}

	skill := &resource.ClaudeSkill{
		Name:        "test-skill",
		Description: "A test skill with hooks",
		Content:     "Skill content here",
		Hooks: map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "Bash",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":          "command",
							"command":       "./check.sh",
							"statusMessage": "Running security check",
						},
					},
				},
			},
		},
	}

	pkg := &config.PackageConfig{
		Package: config.PackageBlock{
			Name:    "my-plugin",
			Version: "1.0.0",
		},
	}

	plan, err := adapter.PlanInstallation(skill, pkg, "/plugin", "/project", &InstallContext{PackageName: "my-plugin", Namespace: true})
	require.NoError(t, err)
	assert.NotNil(t, plan)

	// Should have SKILL.md file with hooks
	require.Len(t, plan.Files, 1)
	assert.Equal(t, ".claude/skills/my-plugin-test-skill/SKILL.md", plan.Files[0].Path)
	assert.Contains(t, plan.Files[0].Content, "name: test-skill")
	assert.Contains(t, plan.Files[0].Content, "hooks:")
	assert.Contains(t, plan.Files[0].Content, "PreToolUse:")
	assert.Contains(t, plan.Files[0].Content, "matcher: Bash")
	assert.Contains(t, plan.Files[0].Content, "statusMessage: Running security check")
	assert.Contains(t, plan.Files[0].Content, "Skill content here")
}

func TestClaudeAdapter_PlanSubagent_WithHooks(t *testing.T) {
	adapter := &ClaudeAdapter{}

	agent := &resource.ClaudeSubagent{
		Name:        "researcher",
		Description: "Research agent with hooks",
		Content:     "Research instructions",
		Model:       "opus",
		Tools:       []string{"Read", "WebSearch"},
		Hooks: map[string]interface{}{
			"SubagentStart": []interface{}{
				map[string]interface{}{
					"hooks": []interface{}{
						map[string]interface{}{
							"type":   "prompt",
							"prompt": "Initialize research context",
						},
					},
				},
			},
		},
	}

	pkg := &config.PackageConfig{
		Package: config.PackageBlock{
			Name:    "my-plugin",
			Version: "1.0.0",
		},
	}

	plan, err := adapter.PlanInstallation(agent, pkg, "/plugin", "/project", &InstallContext{PackageName: "my-plugin", Namespace: true})
	require.NoError(t, err)
	assert.NotNil(t, plan)

	// Should have agent file with hooks
	require.Len(t, plan.Files, 1)
	assert.Equal(t, ".claude/agents/my-plugin-researcher.md", plan.Files[0].Path)
	assert.Contains(t, plan.Files[0].Content, "name: researcher")
	assert.Contains(t, plan.Files[0].Content, "tools:")
	assert.Contains(t, plan.Files[0].Content, "hooks:")
	assert.Contains(t, plan.Files[0].Content, "SubagentStart:")
	assert.Contains(t, plan.Files[0].Content, "type: prompt")
}
