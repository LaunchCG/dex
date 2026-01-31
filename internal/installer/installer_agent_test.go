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

	contentStr := string(content)

	// Check project instructions are at the top
	assert.Contains(t, contentStr, "# My Project")
	assert.Contains(t, contentStr, "This is my project's main context.")

	// Check plugin content is after with markers
	assert.Contains(t, contentStr, "<!-- dex:linting-rules -->")
	assert.Contains(t, contentStr, "Always run ESLint before committing.")
	assert.Contains(t, contentStr, "<!-- /dex:linting-rules -->")

	// Verify order: project content comes before plugin marker
	projectIdx := 0
	markerIdx := len(contentStr)
	for i := range contentStr {
		if i+len("# My Project") <= len(contentStr) && contentStr[i:i+len("# My Project")] == "# My Project" {
			projectIdx = i
		}
		if i+len("<!-- dex:") <= len(contentStr) && contentStr[i:i+len("<!-- dex:")] == "<!-- dex:" {
			markerIdx = i
			break
		}
	}
	assert.Less(t, projectIdx, markerIdx, "Project content should appear before plugin markers")
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
	assert.Contains(t, string(content1), "# V1 Instructions")

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

	// Verify v2 content
	content2, err := os.ReadFile(filepath.Join(projectDir, "CLAUDE.md"))
	require.NoError(t, err)
	contentStr := string(content2)

	// V2 should be there, V1 should be gone
	assert.Contains(t, contentStr, "# V2 Updated Instructions")
	assert.Contains(t, contentStr, "This is the new version.")
	assert.NotContains(t, contentStr, "# V1 Instructions")

	// Plugin content should still be there
	assert.Contains(t, contentStr, "<!-- dex:my-plugin -->")
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
	assert.Contains(t, string(content1), "# Project Instructions")

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
	contentStr := string(content2)

	assert.NotContains(t, contentStr, "# Project Instructions")
	assert.Contains(t, contentStr, "<!-- dex:my-plugin -->")
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

	assert.Contains(t, string(content), "# Cursor Project Guidelines")
	assert.Contains(t, string(content), "Use Cursor-specific instructions.")
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

	assert.Contains(t, string(content), "# Copilot Project Guidelines")
	assert.Contains(t, string(content), "Use GitHub Copilot best practices.")
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

	// Project instructions at top
	assert.Contains(t, contentStr, "# Main Project Guidelines")

	// All three plugins present
	assert.Contains(t, contentStr, "<!-- dex:plugin-a -->")
	assert.Contains(t, contentStr, "<!-- dex:plugin-b -->")
	assert.Contains(t, contentStr, "<!-- dex:plugin-c -->")

	// Verify project content comes before any plugin markers
	projectIdx := 0
	for i := range contentStr {
		if i+len("# Main") <= len(contentStr) && contentStr[i:i+len("# Main")] == "# Main" {
			projectIdx = i
			break
		}
	}

	firstMarkerIdx := len(contentStr)
	for i := range contentStr {
		if i+len("<!-- dex:") <= len(contentStr) && contentStr[i:i+len("<!-- dex:")] == "<!-- dex:" {
			firstMarkerIdx = i
			break
		}
	}

	assert.Less(t, projectIdx, firstMarkerIdx, "Project content should come before plugin markers")
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
