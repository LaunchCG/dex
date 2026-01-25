package adapter

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/launchcg/dex/internal/config"
	"github.com/launchcg/dex/internal/resource"
)

func TestGet_Cursor(t *testing.T) {
	adapter, err := Get("cursor")
	require.NoError(t, err)
	assert.NotNil(t, adapter)
	assert.Equal(t, "cursor", adapter.Name())
}

func TestCursorAdapter_Name(t *testing.T) {
	adapter := &CursorAdapter{}
	assert.Equal(t, "cursor", adapter.Name())
}

func TestCursorAdapter_Directories(t *testing.T) {
	adapter := &CursorAdapter{}
	root := "/project"

	assert.Equal(t, "/project/.cursor", adapter.BaseDir(root))
	assert.Equal(t, "/project/.cursor/skills", adapter.SkillsDir(root))
	assert.Equal(t, "/project/.cursor/commands", adapter.CommandsDir(root))
	assert.Equal(t, "/project/.cursor/agents", adapter.SubagentsDir(root))
	assert.Equal(t, "/project/.cursor/rules", adapter.RulesDir(root))
}

// =============================================================================
// PlanInstallation Tests
// =============================================================================

func TestCursorAdapter_PlanRule(t *testing.T) {
	adapter := &CursorAdapter{}

	rule := &resource.CursorRule{
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

	plan, err := adapter.PlanInstallation(rule, pkg, "/plugin", "/project")
	require.NoError(t, err)
	require.NotNil(t, plan)

	assert.Equal(t, "my-plugin", plan.PluginName)
	assert.Equal(t, "Always use TypeScript strict mode.", plan.AgentFileContent)
	assert.Equal(t, "AGENTS.md", plan.AgentFilePath)
	assert.Empty(t, plan.Files)
	assert.Empty(t, plan.Directories)
}

func TestCursorAdapter_PlanMCPServer_Stdio(t *testing.T) {
	adapter := &CursorAdapter{}

	server := &resource.CursorMCPServer{
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

	plan, err := adapter.PlanInstallation(server, pkg, "/plugin", "/project")
	require.NoError(t, err)
	require.NotNil(t, plan)

	assert.Equal(t, filepath.Join(".cursor", "mcp.json"), plan.MCPPath)
	assert.Equal(t, "mcpServers", plan.MCPKey)

	expectedMCPEntries := map[string]any{
		"mcpServers": map[string]any{
			"filesystem": map[string]any{
				"command": "npx",
				"args":    []string{"-y", "@anthropic/mcp-filesystem"},
				"env":     map[string]string{"HOME": "/home/user"},
			},
		},
	}
	assert.Equal(t, expectedMCPEntries, plan.MCPEntries)
}

func TestCursorAdapter_PlanMCPServer_HTTP(t *testing.T) {
	adapter := &CursorAdapter{}

	server := &resource.CursorMCPServer{
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

	plan, err := adapter.PlanInstallation(server, pkg, "/plugin", "/project")
	require.NoError(t, err)

	expectedMCPEntries := map[string]any{
		"mcpServers": map[string]any{
			"context7": map[string]any{
				"url":     "https://mcp.context7.com/mcp",
				"headers": map[string]string{"Authorization": "Bearer token"},
			},
		},
	}
	assert.Equal(t, expectedMCPEntries, plan.MCPEntries)
}

func TestCursorAdapter_PlanMCPServer_SSE(t *testing.T) {
	adapter := &CursorAdapter{}

	server := &resource.CursorMCPServer{
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

	plan, err := adapter.PlanInstallation(server, pkg, "/plugin", "/project")
	require.NoError(t, err)

	expectedMCPEntries := map[string]any{
		"mcpServers": map[string]any{
			"sse-server": map[string]any{
				"url":     "https://api.example.com/sse",
				"headers": map[string]string{"X-API-Key": "secret"},
			},
		},
	}
	assert.Equal(t, expectedMCPEntries, plan.MCPEntries)
}

func TestCursorAdapter_PlanMCPServer_EnvFile(t *testing.T) {
	adapter := &CursorAdapter{}

	server := &resource.CursorMCPServer{
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

	plan, err := adapter.PlanInstallation(server, pkg, "/plugin", "/project")
	require.NoError(t, err)

	expectedMCPEntries := map[string]any{
		"mcpServers": map[string]any{
			"env-server": map[string]any{
				"command": "node",
				"args":    []string{"server.js"},
				"envFile": ".env.local",
			},
		},
	}
	assert.Equal(t, expectedMCPEntries, plan.MCPEntries)
}

func TestCursorAdapter_PlanRules(t *testing.T) {
	adapter := &CursorAdapter{}

	alwaysApply := false
	rules := &resource.CursorRules{
		Name:        "typescript",
		Description: "TypeScript best practices",
		Content:     "Use interfaces over types.",
		Globs:       []string{"**/*.ts", "**/*.tsx"},
		AlwaysApply: &alwaysApply,
	}

	pkg := &config.PackageConfig{
		Package: config.PackageBlock{
			Name:    "my-plugin",
			Version: "1.0.0",
		},
	}

	plan, err := adapter.PlanInstallation(rules, pkg, "/plugin", "/project")
	require.NoError(t, err)
	require.NotNil(t, plan)

	assert.Equal(t, []string{filepath.Join(".cursor", "rules")}, plan.Directories)
	require.Len(t, plan.Files, 1)
	assert.Equal(t, filepath.Join(".cursor", "rules", "my-plugin-typescript.mdc"), plan.Files[0].Path)
	assert.Equal(t, "", plan.Files[0].Chmod)

	expectedContent := `---
description: TypeScript best practices
globs:
- **/*.ts
- **/*.tsx
---
Use interfaces over types.`
	assert.Equal(t, expectedContent, plan.Files[0].Content)
}

func TestCursorAdapter_PlanRules_AlwaysApply(t *testing.T) {
	adapter := &CursorAdapter{}

	alwaysApply := true
	rules := &resource.CursorRules{
		Name:        "global",
		Description: "Global rules",
		Content:     "Always apply these.",
		AlwaysApply: &alwaysApply,
	}

	pkg := &config.PackageConfig{
		Package: config.PackageBlock{
			Name:    "my-plugin",
			Version: "1.0.0",
		},
	}

	plan, err := adapter.PlanInstallation(rules, pkg, "/plugin", "/project")
	require.NoError(t, err)
	require.Len(t, plan.Files, 1)

	expectedContent := `---
description: Global rules
alwaysApply: true
---
Always apply these.`
	assert.Equal(t, expectedContent, plan.Files[0].Content)
}

func TestCursorAdapter_PlanRules_Minimal(t *testing.T) {
	adapter := &CursorAdapter{}

	rules := &resource.CursorRules{
		Name:        "simple",
		Description: "Simple rules",
		Content:     "Follow best practices.",
	}

	pkg := &config.PackageConfig{
		Package: config.PackageBlock{
			Name:    "my-plugin",
			Version: "1.0.0",
		},
	}

	plan, err := adapter.PlanInstallation(rules, pkg, "/plugin", "/project")
	require.NoError(t, err)
	require.Len(t, plan.Files, 1)

	expectedContent := `---
description: Simple rules
---
Follow best practices.`
	assert.Equal(t, expectedContent, plan.Files[0].Content)
}

func TestCursorAdapter_PlanCommand(t *testing.T) {
	adapter := &CursorAdapter{}

	cmd := &resource.CursorCommand{
		Name:        "deploy",
		Description: "Deploy the application",
		Content:     "Deploy this app to production.",
	}

	pkg := &config.PackageConfig{
		Package: config.PackageBlock{
			Name:    "my-plugin",
			Version: "1.0.0",
		},
	}

	plan, err := adapter.PlanInstallation(cmd, pkg, "/plugin", "/project")
	require.NoError(t, err)
	require.NotNil(t, plan)

	assert.Equal(t, []string{filepath.Join(".cursor", "commands")}, plan.Directories)
	require.Len(t, plan.Files, 1)
	assert.Equal(t, filepath.Join(".cursor", "commands", "my-plugin-deploy.md"), plan.Files[0].Path)

	expectedContent := `---
description: Deploy the application
---
Deploy this app to production.`
	assert.Equal(t, expectedContent, plan.Files[0].Content)
}

func TestCursorAdapter_PlanInstallation_UnsupportedType(t *testing.T) {
	adapter := &CursorAdapter{}

	unknown := &mockUnknownResource{name: "unknown"}
	pkg := &config.PackageConfig{
		Package: config.PackageBlock{
			Name:    "my-plugin",
			Version: "1.0.0",
		},
	}

	plan, err := adapter.PlanInstallation(unknown, pkg, "/plugin", "/project")
	assert.Nil(t, plan)
	require.Error(t, err)
	assert.Equal(t, "unsupported resource type for cursor adapter: *adapter.mockUnknownResource", err.Error())
}

// =============================================================================
// GenerateFrontmatter Tests
// =============================================================================

func TestCursorAdapter_GenerateFrontmatter_Rules_Full(t *testing.T) {
	adapter := &CursorAdapter{}

	alwaysApply := true
	rules := &resource.CursorRules{
		Name:        "typescript",
		Description: "TypeScript guidelines",
		Globs:       []string{"**/*.ts", "src/**/*.tsx"},
		AlwaysApply: &alwaysApply,
	}

	pkg := &config.PackageConfig{}

	frontmatter := adapter.GenerateFrontmatter(rules, pkg)
	expected := `---
description: TypeScript guidelines
globs:
- **/*.ts
- src/**/*.tsx
alwaysApply: true
---
`
	assert.Equal(t, expected, frontmatter)
}

func TestCursorAdapter_GenerateFrontmatter_Rules_Minimal(t *testing.T) {
	adapter := &CursorAdapter{}

	rules := &resource.CursorRules{
		Name:        "simple",
		Description: "Simple rules",
	}

	pkg := &config.PackageConfig{}

	frontmatter := adapter.GenerateFrontmatter(rules, pkg)
	expected := `---
description: Simple rules
---
`
	assert.Equal(t, expected, frontmatter)
}

func TestCursorAdapter_GenerateFrontmatter_Command(t *testing.T) {
	adapter := &CursorAdapter{}

	cmd := &resource.CursorCommand{
		Name:        "deploy",
		Description: "Deploy the app",
	}

	pkg := &config.PackageConfig{}

	frontmatter := adapter.GenerateFrontmatter(cmd, pkg)
	expected := `---
description: Deploy the app
---
`
	assert.Equal(t, expected, frontmatter)
}

func TestCursorAdapter_GenerateFrontmatter_UnknownType(t *testing.T) {
	adapter := &CursorAdapter{}

	// CursorRule doesn't generate frontmatter (it's merged content)
	rule := &resource.CursorRule{Name: "test", Description: "test", Content: "content"}
	pkg := &config.PackageConfig{}

	frontmatter := adapter.GenerateFrontmatter(rule, pkg)
	assert.Equal(t, "", frontmatter)

	// CursorMCPServer doesn't generate frontmatter
	server := &resource.CursorMCPServer{Name: "test", Type: "stdio", Command: "cmd"}
	frontmatter = adapter.GenerateFrontmatter(server, pkg)
	assert.Equal(t, "", frontmatter)
}

// =============================================================================
// MergeCursorMCPConfig Tests
// =============================================================================

func TestCursorAdapter_MergeCursorMCPConfig_NewConfig(t *testing.T) {
	adapter := &CursorAdapter{}

	servers := []*resource.CursorMCPServer{
		{
			Name:    "server1",
			Type:    "stdio",
			Command: "npx",
			Args:    []string{"-y", "@mcp/server"},
		},
	}

	result := adapter.MergeCursorMCPConfig(nil, "my-plugin", servers)

	expected := map[string]any{
		"mcpServers": map[string]any{
			"server1": map[string]any{
				"command": "npx",
				"args":    []string{"-y", "@mcp/server"},
			},
		},
	}
	assert.Equal(t, expected, result)
}

func TestCursorAdapter_MergeCursorMCPConfig_MergeExisting(t *testing.T) {
	adapter := &CursorAdapter{}

	existing := map[string]any{
		"mcpServers": map[string]any{
			"existing-server": map[string]any{
				"command": "node",
				"args":    []string{"server.js"},
			},
		},
	}

	servers := []*resource.CursorMCPServer{
		{
			Name:    "new-server",
			Type:    "stdio",
			Command: "npx",
			Args:    []string{"-y", "@mcp/new"},
			Env:     map[string]string{"KEY": "value"},
		},
	}

	result := adapter.MergeCursorMCPConfig(existing, "my-plugin", servers)

	expected := map[string]any{
		"mcpServers": map[string]any{
			"existing-server": map[string]any{
				"command": "node",
				"args":    []string{"server.js"},
			},
			"new-server": map[string]any{
				"command": "npx",
				"args":    []string{"-y", "@mcp/new"},
				"env":     map[string]string{"KEY": "value"},
			},
		},
	}
	assert.Equal(t, expected, result)
}

func TestCursorAdapter_MergeCursorMCPConfig_OverwriteExisting(t *testing.T) {
	adapter := &CursorAdapter{}

	existing := map[string]any{
		"mcpServers": map[string]any{
			"server": map[string]any{
				"command": "old-command",
				"args":    []string{"old-arg"},
			},
		},
	}

	servers := []*resource.CursorMCPServer{
		{
			Name:    "server",
			Type:    "stdio",
			Command: "new-command",
			Args:    []string{"new-arg"},
		},
	}

	result := adapter.MergeCursorMCPConfig(existing, "my-plugin", servers)

	expected := map[string]any{
		"mcpServers": map[string]any{
			"server": map[string]any{
				"command": "new-command",
				"args":    []string{"new-arg"},
			},
		},
	}
	assert.Equal(t, expected, result)
}

func TestCursorAdapter_MergeCursorMCPConfig_MultipleServers(t *testing.T) {
	adapter := &CursorAdapter{}

	servers := []*resource.CursorMCPServer{
		{
			Name:    "stdio-server",
			Type:    "stdio",
			Command: "npx",
			Args:    []string{"-y", "@mcp/stdio"},
		},
		{
			Name: "http-server",
			Type: "http",
			URL:  "https://api.example.com/mcp",
		},
		{
			Name:    "sse-server",
			Type:    "sse",
			URL:     "https://api.example.com/sse",
			Headers: map[string]string{"Auth": "token"},
		},
	}

	result := adapter.MergeCursorMCPConfig(nil, "my-plugin", servers)

	expected := map[string]any{
		"mcpServers": map[string]any{
			"stdio-server": map[string]any{
				"command": "npx",
				"args":    []string{"-y", "@mcp/stdio"},
			},
			"http-server": map[string]any{
				"url": "https://api.example.com/mcp",
			},
			"sse-server": map[string]any{
				"url":     "https://api.example.com/sse",
				"headers": map[string]string{"Auth": "token"},
			},
		},
	}
	assert.Equal(t, expected, result)
}

func TestCursorAdapter_MergeCursorMCPConfig_EmptyExistingNoServers(t *testing.T) {
	adapter := &CursorAdapter{}

	existing := map[string]any{
		"otherKey": "otherValue",
	}

	servers := []*resource.CursorMCPServer{
		{
			Name:    "server",
			Type:    "stdio",
			Command: "cmd",
		},
	}

	result := adapter.MergeCursorMCPConfig(existing, "my-plugin", servers)

	expected := map[string]any{
		"otherKey": "otherValue",
		"mcpServers": map[string]any{
			"server": map[string]any{
				"command": "cmd",
			},
		},
	}
	assert.Equal(t, expected, result)
}

// =============================================================================
// MergeAgentFile Tests
// =============================================================================

func TestCursorAdapter_MergeAgentFile_New(t *testing.T) {
	adapter := &CursorAdapter{}

	result := adapter.MergeAgentFile("", "my-plugin", "New content")

	expected := `<!-- dex:my-plugin -->
New content
<!-- /dex:my-plugin -->`
	assert.Equal(t, expected, result)
}

func TestCursorAdapter_MergeAgentFile_Append(t *testing.T) {
	adapter := &CursorAdapter{}

	existing := "# Existing Content\n\nSome existing content."
	content := "New rule content"

	result := adapter.MergeAgentFile(existing, "my-plugin", content)

	expected := `# Existing Content

Some existing content.

<!-- dex:my-plugin -->
New rule content
<!-- /dex:my-plugin -->`
	assert.Equal(t, expected, result)
}

func TestCursorAdapter_MergeAgentFile_Update(t *testing.T) {
	adapter := &CursorAdapter{}

	existing := `# Existing Content

<!-- dex:my-plugin -->
Old content
<!-- /dex:my-plugin -->

Other content`

	content := "Updated content"

	result := adapter.MergeAgentFile(existing, "my-plugin", content)

	expected := `# Existing Content

<!-- dex:my-plugin -->
Updated content
<!-- /dex:my-plugin -->

Other content`
	assert.Equal(t, expected, result)
}

func TestCursorAdapter_MergeAgentFile_MultiplePlugins(t *testing.T) {
	adapter := &CursorAdapter{}

	// Start with one plugin
	result := adapter.MergeAgentFile("", "plugin-a", "Plugin A content")

	// Add another plugin
	result = adapter.MergeAgentFile(result, "plugin-b", "Plugin B content")

	expected := `<!-- dex:plugin-a -->
Plugin A content
<!-- /dex:plugin-a -->

<!-- dex:plugin-b -->
Plugin B content
<!-- /dex:plugin-b -->`
	assert.Equal(t, expected, result)

	// Update first plugin
	result = adapter.MergeAgentFile(result, "plugin-a", "Updated A content")

	expected = `<!-- dex:plugin-a -->
Updated A content
<!-- /dex:plugin-a -->

<!-- dex:plugin-b -->
Plugin B content
<!-- /dex:plugin-b -->`
	assert.Equal(t, expected, result)
}

func TestCursorAdapter_MergeAgentFile_MultilineContent(t *testing.T) {
	adapter := &CursorAdapter{}

	content := `This is a multiline rule.

It has:
- Multiple lines
- With formatting
- And structure`

	result := adapter.MergeAgentFile("", "my-plugin", content)

	expected := `<!-- dex:my-plugin -->
This is a multiline rule.

It has:
- Multiple lines
- With formatting
- And structure
<!-- /dex:my-plugin -->`
	assert.Equal(t, expected, result)
}

// =============================================================================
// MergeSettingsConfig Test (no-op for Cursor)
// =============================================================================

func TestCursorAdapter_MergeSettingsConfig_NoOp(t *testing.T) {
	adapter := &CursorAdapter{}

	existing := map[string]any{"key": "value"}
	settings := &resource.ClaudeSettings{Name: "test"}

	result := adapter.MergeSettingsConfig(existing, settings)

	// Should return existing unchanged
	assert.Equal(t, existing, result)
}

// =============================================================================
// MergeMCPConfig Test (no-op for Cursor - uses MergeCursorMCPConfig instead)
// =============================================================================

func TestCursorAdapter_MergeMCPConfig_NoOp(t *testing.T) {
	adapter := &CursorAdapter{}

	existing := map[string]any{"key": "value"}
	servers := []*resource.ClaudeMCPServer{}

	result := adapter.MergeMCPConfig(existing, "my-plugin", servers)

	// Should return existing unchanged
	assert.Equal(t, existing, result)
}

// =============================================================================
// MergePlans Tests with Cursor
// =============================================================================

func TestMergePlans_WithCursorPaths(t *testing.T) {
	plan1 := &Plan{
		PluginName:       "plugin1",
		Directories:      []string{".cursor/rules"},
		AgentFileContent: "Rule content",
		AgentFilePath:    "AGENTS.md",
		MCPEntries:       make(map[string]any),
		SettingsEntries:  make(map[string]any),
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
		MCPPath:         ".cursor/mcp.json",
		MCPKey:          "mcpServers",
		SettingsEntries: make(map[string]any),
	}

	merged := MergePlans(plan1, plan2)

	assert.Equal(t, "plugin1", merged.PluginName)
	assert.Equal(t, "AGENTS.md", merged.AgentFilePath)
	assert.Equal(t, ".cursor/mcp.json", merged.MCPPath)
	assert.Equal(t, "mcpServers", merged.MCPKey)
	assert.Equal(t, "Rule content", merged.AgentFileContent)
	assert.Equal(t, []string{".cursor/rules"}, merged.Directories)
}

// =============================================================================
// Frontmatter Preservation Tests
// =============================================================================

func TestCursorAdapter_PlanRules_PreservesExistingFrontmatter(t *testing.T) {
	adapter := &CursorAdapter{}

	rules := &resource.CursorRules{
		Name:        "custom",
		Description: "Custom rules",
		Content: `---
description: Custom description in content
globs:
- **/*.custom
alwaysApply: true
---
Rule body here.`,
	}

	pkg := &config.PackageConfig{
		Package: config.PackageBlock{
			Name:    "my-plugin",
			Version: "1.0.0",
		},
	}

	plan, err := adapter.PlanInstallation(rules, pkg, "/plugin", "/project")
	require.NoError(t, err)
	require.Len(t, plan.Files, 1)

	// Should preserve the original content with frontmatter intact
	assert.Equal(t, rules.Content, plan.Files[0].Content)
}

func TestCursorAdapter_PlanCommand_PreservesExistingFrontmatter(t *testing.T) {
	adapter := &CursorAdapter{}

	cmd := &resource.CursorCommand{
		Name:        "custom",
		Description: "Custom command",
		Content: `---
description: Custom description in content
---
Command body here.`,
	}

	pkg := &config.PackageConfig{
		Package: config.PackageBlock{
			Name:    "my-plugin",
			Version: "1.0.0",
		},
	}

	plan, err := adapter.PlanInstallation(cmd, pkg, "/plugin", "/project")
	require.NoError(t, err)
	require.Len(t, plan.Files, 1)

	// Should preserve the original content with frontmatter intact
	assert.Equal(t, cmd.Content, plan.Files[0].Content)
}
