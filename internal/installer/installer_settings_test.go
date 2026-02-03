package installer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInstaller_ClaudeSettingsIntegration tests that a plugin with claude_settings
// creates .claude/settings.json during installation.
// This is a full integration test that simulates real plugin installation.
func TestInstaller_ClaudeSettingsIntegration(t *testing.T) {
	// Create a temporary directory for the test project
	projectDir := t.TempDir()

	// Create a temporary directory for the test plugin
	pluginDir := t.TempDir()

	// Write package.hcl with claude_settings
	packageHCL := `package {
  name        = "test-settings-plugin"
  version     = "1.0.0"
  description = "Test plugin with settings"
  platforms   = ["claude-code"]
}

claude_settings "mcp-permissions" {
  allow = [
    "mcp__test-server",
    "Bash(docker:*)"
  ]
  deny = [
    "Bash(rm -rf /)"
  ]
  env = {
    TEST_VAR = "test_value"
  }
}
`
	err := os.WriteFile(filepath.Join(pluginDir, "package.hcl"), []byte(packageHCL), 0644)
	require.NoError(t, err)

	// Write dex.hcl for the project
	projectHCL := `project {
  name = "test-project"
  agentic_platform = "claude-code"
}

plugin "test-settings-plugin" {
  source = "file://` + pluginDir + `"
}
`
	err = os.WriteFile(filepath.Join(projectDir, "dex.hcl"), []byte(projectHCL), 0644)
	require.NoError(t, err)

	// Create installer and run installation
	installer, err := NewInstaller(projectDir)
	require.NoError(t, err)

	_, err = installer.Install(nil)
	require.NoError(t, err)

	// Verify .claude directory was created
	claudeDir := filepath.Join(projectDir, ".claude")
	_, err = os.Stat(claudeDir)
	require.NoError(t, err, ".claude directory should exist")

	// Verify .claude/settings.json was created
	settingsPath := filepath.Join(claudeDir, "settings.json")
	_, err = os.Stat(settingsPath)
	require.NoError(t, err, ".claude/settings.json should exist")

	// Read and verify settings content
	settingsContent, err := os.ReadFile(settingsPath)
	require.NoError(t, err)

	var settings map[string]any
	err = json.Unmarshal(settingsContent, &settings)
	require.NoError(t, err, "settings.json should be valid JSON")

	// Verify allow array
	allow, ok := settings["allow"].([]any)
	require.True(t, ok, "allow should be an array")
	assert.Len(t, allow, 2)
	assert.Contains(t, allow, "mcp__test-server")
	assert.Contains(t, allow, "Bash(docker:*)")

	// Verify deny array
	deny, ok := settings["deny"].([]any)
	require.True(t, ok, "deny should be an array")
	assert.Contains(t, deny, "Bash(rm -rf /)")

	// Verify env map
	env, ok := settings["env"].(map[string]any)
	require.True(t, ok, "env should be a map")
	assert.Equal(t, "test_value", env["TEST_VAR"])
}

// TestInstaller_ClaudeSettingsWithOtherResources tests that claude_settings
// works correctly when a plugin has multiple resource types.
func TestInstaller_ClaudeSettingsWithOtherResources(t *testing.T) {
	projectDir := t.TempDir()
	pluginDir := t.TempDir()

	// Create a skill file
	skillDir := filepath.Join(pluginDir, "skills")
	err := os.MkdirAll(skillDir, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(skillDir, "test.md"), []byte("Test skill content"), 0644)
	require.NoError(t, err)

	// Write package.hcl with multiple resources including claude_settings
	packageHCL := `package {
  name        = "multi-resource-plugin"
  version     = "1.0.0"
  description = "Plugin with multiple resource types"
  platforms   = ["claude-code"]
}

claude_skill "test-skill" {
  description = "A test skill"
  content     = file("skills/test.md")
}

claude_mcp_server "test-server" {
  type    = "command"
  command = "npx"
  args    = ["-y", "test-mcp"]
}

claude_settings "permissions" {
  allow = [
    "mcp__test-server"
  ]
}
`
	err = os.WriteFile(filepath.Join(pluginDir, "package.hcl"), []byte(packageHCL), 0644)
	require.NoError(t, err)

	// Write dex.hcl
	projectHCL := `project {
  name = "test-project"
  agentic_platform = "claude-code"
}

plugin "multi-resource-plugin" {
  source = "file://` + pluginDir + `"
}
`
	err = os.WriteFile(filepath.Join(projectDir, "dex.hcl"), []byte(projectHCL), 0644)
	require.NoError(t, err)

	// Install
	installer, err := NewInstaller(projectDir)
	require.NoError(t, err)
	_, err = installer.Install(nil)
	require.NoError(t, err)

	// Verify skill was created
	skillPath := filepath.Join(projectDir, ".claude/skills/test-skill/SKILL.md")
	_, err = os.Stat(skillPath)
	assert.NoError(t, err, "skill should be created")

	// Verify MCP server was created
	mcpPath := filepath.Join(projectDir, ".mcp.json")
	mcpContent, err := os.ReadFile(mcpPath)
	require.NoError(t, err)
	var mcpConfig map[string]any
	err = json.Unmarshal(mcpContent, &mcpConfig)
	require.NoError(t, err)
	servers := mcpConfig["mcpServers"].(map[string]any)
	assert.Contains(t, servers, "test-server")

	// Verify settings file was created
	settingsPath := filepath.Join(projectDir, ".claude/settings.json")
	settingsContent, err := os.ReadFile(settingsPath)
	require.NoError(t, err)
	var settings map[string]any
	err = json.Unmarshal(settingsContent, &settings)
	require.NoError(t, err)
	allow := settings["allow"].([]any)
	assert.Contains(t, allow, "mcp__test-server")
}
