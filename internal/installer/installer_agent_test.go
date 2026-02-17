package installer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===========================================================================
// Project Agent Instructions Integration Tests
// ===========================================================================

func TestInstaller_ProjectAgentInstructions_ClaudeCodeOnly(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Create project config with agent instructions but no plugins
	projectContent := `project {
  name = "test-project"
  agentic_platform = "claude-code"
  agent_instructions = <<EOF
# My Project Guidelines

Always use TypeScript for all code.
Follow the project coding standards.
EOF
}
`
	err := os.WriteFile(filepath.Join(projectDir, "dex.hcl"), []byte(projectContent), 0644)
	require.NoError(t, err)

	// Create installer
	installer, err := NewInstaller(projectDir)
	require.NoError(t, err)

	// Install
	err = installer.InstallAll()
	require.NoError(t, err)

	// Verify CLAUDE.md was created with project instructions
	claudePath := filepath.Join(projectDir, "CLAUDE.md")
	require.FileExists(t, claudePath)

	content, err := os.ReadFile(claudePath)
	require.NoError(t, err)

	expected := `# My Project Guidelines

Always use TypeScript for all code.
Follow the project coding standards.`

	assert.Equal(t, expected, string(content))

	// Verify tracked in manifest
	plugin := installer.manifest.GetPlugin("__project__")
	assert.NotNil(t, plugin)
	assert.True(t, plugin.HasAgentContent)
}

func TestInstaller_ProjectAgentInstructions_WithPlugin(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Set up a local plugin with a rule
	pluginDir := t.TempDir()
	pluginContent := `package {
  name = "linting-rules"
  version = "1.0.0"
  description = "Linting rules plugin"
}

claude_rule "eslint" {
  description = "ESLint rules"
  content = "Always run ESLint before committing."
}
`
	err := os.WriteFile(filepath.Join(pluginDir, "package.hcl"), []byte(pluginContent), 0644)
	require.NoError(t, err)

	// Create project config with agent instructions AND a plugin
	projectContent := `project {
  name = "test-project"
  agentic_platform = "claude-code"
  agent_instructions = <<EOF
# My Project

This is my project's main context.
EOF
}

plugin "linting-rules" {
  source = "file:` + pluginDir + `"
}
`
	err = os.WriteFile(filepath.Join(projectDir, "dex.hcl"), []byte(projectContent), 0644)
	require.NoError(t, err)

	// Create installer and install
	installer, err := NewInstaller(projectDir)
	require.NoError(t, err)

	err = installer.InstallAll()
	require.NoError(t, err)

	// Verify CLAUDE.md has project instructions BEFORE plugin content
	claudePath := filepath.Join(projectDir, "CLAUDE.md")
	content, err := os.ReadFile(claudePath)
	require.NoError(t, err)

	expected := "# My Project\n\nThis is my project's main context.\n\nAlways run ESLint before committing."
	assert.Equal(t, expected, string(content))
}

func TestInstaller_ProjectAgentInstructions_UpdateInstructions(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Set up a plugin
	pluginDir := t.TempDir()
	createTestPlugin(t, pluginDir, "my-plugin", "1.0.0", "Test plugin")

	// Create project config v1
	projectContent := `project {
  name = "test-project"
  agentic_platform = "claude-code"
  agent_instructions = "# V1 Instructions"
}

plugin "my-plugin" {
  source = "file:` + pluginDir + `"
}
`
	err := os.WriteFile(filepath.Join(projectDir, "dex.hcl"), []byte(projectContent), 0644)
	require.NoError(t, err)

	// Install v1
	installer1, err := NewInstaller(projectDir)
	require.NoError(t, err)
	err = installer1.InstallAll()
	require.NoError(t, err)

	// Verify v1 content
	content1, err := os.ReadFile(filepath.Join(projectDir, "CLAUDE.md"))
	require.NoError(t, err)
	expected1 := "# V1 Instructions\n\nFollow this rule from my-plugin"
	assert.Equal(t, expected1, string(content1))

	// Update project config to v2
	projectContent = `project {
  name = "test-project"
  agentic_platform = "claude-code"
  agent_instructions = "# V2 Updated Instructions\n\nThis is the new version."
}

plugin "my-plugin" {
  source = "file:` + pluginDir + `"
}
`
	err = os.WriteFile(filepath.Join(projectDir, "dex.hcl"), []byte(projectContent), 0644)
	require.NoError(t, err)

	// Reinstall
	installer2, err := NewInstaller(projectDir)
	require.NoError(t, err)
	err = installer2.InstallAll()
	require.NoError(t, err)

	// Verify v2 content: V2 instructions replace V1, plugin content preserved
	content2, err := os.ReadFile(filepath.Join(projectDir, "CLAUDE.md"))
	require.NoError(t, err)
	expected2 := "# V2 Updated Instructions\n\nThis is the new version.\n\nFollow this rule from my-plugin"
	assert.Equal(t, expected2, string(content2))
}

func TestInstaller_ProjectAgentInstructions_RemoveInstructions(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Set up a plugin
	pluginDir := t.TempDir()
	createTestPlugin(t, pluginDir, "my-plugin", "1.0.0", "Test plugin")

	// Create project config WITH instructions
	projectContent := `project {
  name = "test-project"
  agentic_platform = "claude-code"
  agent_instructions = "# Project Instructions"
}

plugin "my-plugin" {
  source = "file:` + pluginDir + `"
}
`
	err := os.WriteFile(filepath.Join(projectDir, "dex.hcl"), []byte(projectContent), 0644)
	require.NoError(t, err)

	// Install
	installer1, err := NewInstaller(projectDir)
	require.NoError(t, err)
	err = installer1.InstallAll()
	require.NoError(t, err)

	// Verify instructions are there
	content1, err := os.ReadFile(filepath.Join(projectDir, "CLAUDE.md"))
	require.NoError(t, err)
	expected1 := "# Project Instructions\n\nFollow this rule from my-plugin"
	assert.Equal(t, expected1, string(content1))

	// Remove agent_instructions from config
	projectContent = `project {
  name = "test-project"
  agentic_platform = "claude-code"
}

plugin "my-plugin" {
  source = "file:` + pluginDir + `"
}
`
	err = os.WriteFile(filepath.Join(projectDir, "dex.hcl"), []byte(projectContent), 0644)
	require.NoError(t, err)

	// Reinstall
	installer2, err := NewInstaller(projectDir)
	require.NoError(t, err)
	err = installer2.InstallAll()
	require.NoError(t, err)

	// Verify project instructions are gone, plugin content remains
	content2, err := os.ReadFile(filepath.Join(projectDir, "CLAUDE.md"))
	require.NoError(t, err)
	expected2 := "Follow this rule from my-plugin"
	assert.Equal(t, expected2, string(content2))
}

func TestInstaller_ProjectAgentInstructions_Cursor(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Create project config for Cursor
	projectContent := `project {
  name = "test-project"
  agentic_platform = "cursor"
  agent_instructions = "# Cursor Project Guidelines\n\nUse Cursor-specific instructions."
}
`
	err := os.WriteFile(filepath.Join(projectDir, "dex.hcl"), []byte(projectContent), 0644)
	require.NoError(t, err)

	// Create installer and install
	installer, err := NewInstaller(projectDir)
	require.NoError(t, err)

	err = installer.InstallAll()
	require.NoError(t, err)

	// Verify AGENTS.md was created (Cursor uses AGENTS.md)
	agentsPath := filepath.Join(projectDir, "AGENTS.md")
	require.FileExists(t, agentsPath)

	content, err := os.ReadFile(agentsPath)
	require.NoError(t, err)

	expected := "# Cursor Project Guidelines\n\nUse Cursor-specific instructions."
	assert.Equal(t, expected, string(content))
}

func TestInstaller_ProjectAgentInstructions_Copilot(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Create project config for GitHub Copilot
	projectContent := `project {
  name = "test-project"
  agentic_platform = "github-copilot"
  agent_instructions = "# Copilot Project Guidelines\n\nUse GitHub Copilot best practices."
}
`
	err := os.WriteFile(filepath.Join(projectDir, "dex.hcl"), []byte(projectContent), 0644)
	require.NoError(t, err)

	// Create installer and install
	installer, err := NewInstaller(projectDir)
	require.NoError(t, err)

	err = installer.InstallAll()
	require.NoError(t, err)

	// Verify .github/copilot-instructions.md was created
	copilotPath := filepath.Join(projectDir, ".github", "copilot-instructions.md")
	require.FileExists(t, copilotPath)

	content, err := os.ReadFile(copilotPath)
	require.NoError(t, err)

	expected := "# Copilot Project Guidelines\n\nUse GitHub Copilot best practices."
	assert.Equal(t, expected, string(content))
}

func TestInstaller_ProjectAgentInstructions_MultiplePlugins(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Set up three plugins
	plugin1Dir := t.TempDir()
	createTestPlugin(t, plugin1Dir, "plugin-a", "1.0.0", "Plugin A")

	plugin2Dir := t.TempDir()
	createTestPlugin(t, plugin2Dir, "plugin-b", "1.0.0", "Plugin B")

	plugin3Dir := t.TempDir()
	createTestPlugin(t, plugin3Dir, "plugin-c", "1.0.0", "Plugin C")

	// Create project config
	projectContent := `project {
  name = "test-project"
  agentic_platform = "claude-code"
  agent_instructions = <<EOF
# Main Project Guidelines

These are the overarching project rules.
All plugins must follow these.
EOF
}

plugin "plugin-a" {
  source = "file:` + plugin1Dir + `"
}

plugin "plugin-b" {
  source = "file:` + plugin2Dir + `"
}

plugin "plugin-c" {
  source = "file:` + plugin3Dir + `"
}
`
	err := os.WriteFile(filepath.Join(projectDir, "dex.hcl"), []byte(projectContent), 0644)
	require.NoError(t, err)

	// Install
	installer, err := NewInstaller(projectDir)
	require.NoError(t, err)
	err = installer.InstallAll()
	require.NoError(t, err)

	// Verify CLAUDE.md structure
	content, err := os.ReadFile(filepath.Join(projectDir, "CLAUDE.md"))
	require.NoError(t, err)
	contentStr := string(content)

	// Verify full content: project instructions at top, then all plugin content
	expected := "# Main Project Guidelines\n\nThese are the overarching project rules.\nAll plugins must follow these.\n\n" +
		"Follow this rule from plugin-a\n\n" +
		"Follow this rule from plugin-b\n\n" +
		"Follow this rule from plugin-c"
	assert.Equal(t, expected, contentStr)
}

func TestInstaller_ProjectAgentInstructions_NoInstructionsNoPlugins(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Create minimal project config
	projectContent := `project {
  name = "test-project"
  agentic_platform = "claude-code"
}
`
	err := os.WriteFile(filepath.Join(projectDir, "dex.hcl"), []byte(projectContent), 0644)
	require.NoError(t, err)

	// Install
	installer, err := NewInstaller(projectDir)
	require.NoError(t, err)
	err = installer.InstallAll()
	require.NoError(t, err)

	// Verify CLAUDE.md was NOT created
	claudePath := filepath.Join(projectDir, "CLAUDE.md")
	_, err = os.Stat(claudePath)
	assert.True(t, os.IsNotExist(err), "CLAUDE.md should not exist when there are no instructions or plugins")
}

func TestInstall_SpecificPlugins_TracksProjectResources(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Set up a local plugin
	pluginDir := t.TempDir()
	createTestPlugin(t, pluginDir, "my-plugin", "1.0.0", "Test plugin")

	// Create project config with agent instructions AND a plugin
	projectContent := `project {
  name = "test-project"
  agentic_platform = "claude-code"
  agent_instructions = "# Project Rules\n\nAlways follow these rules."
}

plugin "my-plugin" {
  source = "file:` + pluginDir + `"
}
`
	err := os.WriteFile(filepath.Join(projectDir, "dex.hcl"), []byte(projectContent), 0644)
	require.NoError(t, err)

	// Install specific plugin (not bare install)
	installer, err := NewInstaller(projectDir)
	require.NoError(t, err)

	_, err = installer.Install([]PluginSpec{
		{
			Name:   "my-plugin",
			Source: "file:" + pluginDir,
		},
	})
	require.NoError(t, err)

	// Assert __project__ appears in manifest with exact expected tracking
	projectPlugin := installer.manifest.GetPlugin("__project__")
	require.NotNil(t, projectPlugin, "__project__ should be tracked in manifest after Install(specs)")
	assert.Equal(t, true, projectPlugin.HasAgentContent)
	assert.Equal(t, []string{"CLAUDE.md"}, projectPlugin.MergedFiles)

	// Assert the plugin is also tracked
	myPlugin := installer.manifest.GetPlugin("my-plugin")
	require.NotNil(t, myPlugin, "my-plugin should be tracked in manifest")
	assert.Equal(t, true, myPlugin.HasAgentContent)
	assert.Equal(t, []string{"CLAUDE.md"}, myPlugin.MergedFiles)

	// Assert exact CLAUDE.md content
	claudePath := filepath.Join(projectDir, "CLAUDE.md")
	content, err := os.ReadFile(claudePath)
	require.NoError(t, err)

	expectedCLAUDE := "# Project Rules\n\nAlways follow these rules.\n\nFollow this rule from my-plugin"
	assert.Equal(t, expectedCLAUDE, string(content))
}
