package installer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Test Helpers
// =============================================================================

// createTestProject creates a minimal dex project for testing
func createTestProject(t *testing.T, dir string, plugins string) {
	t.Helper()
	content := `project {
  name = "test-project"
  agentic_platform = "claude-code"
}

` + plugins
	err := os.WriteFile(filepath.Join(dir, "dex.hcl"), []byte(content), 0644)
	require.NoError(t, err)
}

// createTestPlugin creates a minimal package.hcl for a test plugin
func createTestPlugin(t *testing.T, dir, name, version, description string) {
	t.Helper()
	content := `package {
  name = "` + name + `"
  version = "` + version + `"
  description = "` + description + `"
}

claude_rule "` + name + `-rule" {
  description = "Rule from ` + name + `"
  content = "Follow this rule from ` + name + `"
}
`
	err := os.WriteFile(filepath.Join(dir, "package.hcl"), []byte(content), 0644)
	require.NoError(t, err)
}

// createLocalRegistryIndex creates a registry.json for a local registry
func createLocalRegistryIndex(t *testing.T, dir string, packages map[string][]string) {
	t.Helper()
	entries := make(map[string]map[string]any)
	for name, versions := range packages {
		latest := versions[len(versions)-1]
		entries[name] = map[string]any{
			"versions": versions,
			"latest":   latest,
		}
	}
	index := map[string]any{
		"name":     "test-registry",
		"version":  "1.0",
		"packages": entries,
	}
	data, err := json.MarshalIndent(index, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, "registry.json"), data, 0644)
	require.NoError(t, err)
}

// =============================================================================
// Local Registry End-to-End Tests
// =============================================================================

func TestInstaller_LocalRegistry_DirectSource(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Set up a local plugin
	pluginDir := t.TempDir()
	createTestPlugin(t, pluginDir, "my-test-plugin", "1.0.0", "Test plugin")

	// Create project config with direct source
	createTestProject(t, projectDir, `
plugin "my-test-plugin" {
  source = "file:`+pluginDir+`"
}
`)

	// Create installer
	installer, err := NewInstaller(projectDir)
	require.NoError(t, err)

	// Install all plugins
	err = installer.InstallAll()
	require.NoError(t, err)

	// Verify CLAUDE.md was created with the rule content
	claudeContent, err := os.ReadFile(filepath.Join(projectDir, "CLAUDE.md"))
	require.NoError(t, err)
	assert.Contains(t, string(claudeContent), "dex:my-test-plugin")
	assert.Contains(t, string(claudeContent), "Follow this rule from my-test-plugin")

	// Verify lock file was updated (JSON format)
	lockContent, err := os.ReadFile(filepath.Join(projectDir, "dex.lock"))
	require.NoError(t, err)
	assert.Contains(t, string(lockContent), "my-test-plugin")
	assert.Contains(t, string(lockContent), `"version": "1.0.0"`)

	// Verify manifest tracks the plugin
	manifestPath := filepath.Join(projectDir, ".dex", "manifest.json")
	_, err = os.Stat(manifestPath)
	require.NoError(t, err)
}

func TestInstaller_LocalRegistry_RegistryMode(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Set up a local registry with multiple packages
	registryDir := t.TempDir()
	createLocalRegistryIndex(t, registryDir, map[string][]string{
		"plugin-a": {"1.0.0", "1.1.0", "2.0.0"},
		"plugin-b": {"0.5.0", "1.0.0"},
	})

	// Create the package directories
	pluginADir := filepath.Join(registryDir, "plugin-a")
	err := os.MkdirAll(pluginADir, 0755)
	require.NoError(t, err)
	createTestPlugin(t, pluginADir, "plugin-a", "2.0.0", "Plugin A")

	pluginBDir := filepath.Join(registryDir, "plugin-b")
	err = os.MkdirAll(pluginBDir, 0755)
	require.NoError(t, err)
	createTestPlugin(t, pluginBDir, "plugin-b", "1.0.0", "Plugin B")

	// Create project config with registry
	createTestProject(t, projectDir, `
registry "local" {
  path = "`+registryDir+`"
}

plugin "plugin-a" {
  registry = "local"
  version = "^2.0.0"
}

plugin "plugin-b" {
  registry = "local"
}
`)

	// Create installer
	installer, err := NewInstaller(projectDir)
	require.NoError(t, err)

	// Install all plugins
	err = installer.InstallAll()
	require.NoError(t, err)

	// Verify both plugins' rules were installed
	claudeContent, err := os.ReadFile(filepath.Join(projectDir, "CLAUDE.md"))
	require.NoError(t, err)
	assert.Contains(t, string(claudeContent), "dex:plugin-a")
	assert.Contains(t, string(claudeContent), "dex:plugin-b")
	assert.Contains(t, string(claudeContent), "Follow this rule from plugin-a")
	assert.Contains(t, string(claudeContent), "Follow this rule from plugin-b")

	// Verify lock file has both plugins
	lockContent, err := os.ReadFile(filepath.Join(projectDir, "dex.lock"))
	require.NoError(t, err)
	assert.Contains(t, string(lockContent), "plugin-a")
	assert.Contains(t, string(lockContent), "plugin-b")
}

func TestInstaller_LocalRegistry_VersionConstraint(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Set up a local registry with multiple versions
	registryDir := t.TempDir()
	createLocalRegistryIndex(t, registryDir, map[string][]string{
		"versioned-plugin": {"1.0.0", "1.1.0", "1.2.0", "2.0.0"},
	})

	// Create the package directory
	pluginDir := filepath.Join(registryDir, "versioned-plugin")
	err := os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)
	createTestPlugin(t, pluginDir, "versioned-plugin", "1.2.0", "Versioned Plugin")

	// Create project config with caret constraint
	createTestProject(t, projectDir, `
registry "local" {
  path = "`+registryDir+`"
}

plugin "versioned-plugin" {
  registry = "local"
  version = "^1.0.0"
}
`)

	// Create installer
	installer, err := NewInstaller(projectDir)
	require.NoError(t, err)

	// Install with version constraint
	err = installer.InstallAll()
	require.NoError(t, err)

	// Verify lock file has version 1.2.0 (highest matching ^1.0.0)
	lockContent, err := os.ReadFile(filepath.Join(projectDir, "dex.lock"))
	require.NoError(t, err)
	assert.Contains(t, string(lockContent), `"version": "1.2.0"`)
}

func TestInstaller_LocalRegistry_Reinstall(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Set up a local plugin
	pluginDir := t.TempDir()
	createTestPlugin(t, pluginDir, "reinstall-plugin", "1.0.0", "Reinstall test plugin")

	// Create project config
	createTestProject(t, projectDir, `
plugin "reinstall-plugin" {
  source = "file:`+pluginDir+`"
}
`)

	// First install
	installer1, err := NewInstaller(projectDir)
	require.NoError(t, err)
	err = installer1.InstallAll()
	require.NoError(t, err)

	// Verify first install
	claudeContent1, err := os.ReadFile(filepath.Join(projectDir, "CLAUDE.md"))
	require.NoError(t, err)
	assert.Contains(t, string(claudeContent1), "dex:reinstall-plugin")
	assert.Contains(t, string(claudeContent1), "Follow this rule from reinstall-plugin")

	// Update the plugin content
	updatedContent := `package {
  name = "reinstall-plugin"
  version = "1.0.0"
  description = "Updated reinstall test plugin"
}

claude_rule "reinstall-plugin-rule" {
  description = "Updated rule"
  content = "This is the updated rule content"
}
`
	err = os.WriteFile(filepath.Join(pluginDir, "package.hcl"), []byte(updatedContent), 0644)
	require.NoError(t, err)

	// Reinstall
	installer2, err := NewInstaller(projectDir)
	require.NoError(t, err)
	err = installer2.InstallAll()
	require.NoError(t, err)

	// Verify updated content
	claudeContent2, err := os.ReadFile(filepath.Join(projectDir, "CLAUDE.md"))
	require.NoError(t, err)
	assert.Contains(t, string(claudeContent2), "This is the updated rule content")
}

func TestInstaller_LocalRegistry_WithVariables(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Set up a local plugin with variables
	pluginDir := t.TempDir()
	pluginContent := `package {
  name = "var-plugin"
  version = "1.0.0"
  description = "Plugin with variables"
}

variable "custom_value" {
  description = "A custom value"
  default = "default-value"
}

claude_rule "var-rule" {
  description = "Rule with variable"
  content = "Static content for var test"
}
`
	err := os.WriteFile(filepath.Join(pluginDir, "package.hcl"), []byte(pluginContent), 0644)
	require.NoError(t, err)

	// Create project config with variable override
	createTestProject(t, projectDir, `
plugin "var-plugin" {
  source = "file:`+pluginDir+`"
  config = {
    custom_value = "overridden-value"
  }
}
`)

	// Create installer
	installer, err := NewInstaller(projectDir)
	require.NoError(t, err)

	// Install
	err = installer.InstallAll()
	require.NoError(t, err)

	// Verify the plugin was installed (variable resolution tested separately)
	claudeContent, err := os.ReadFile(filepath.Join(projectDir, "CLAUDE.md"))
	require.NoError(t, err)
	assert.Contains(t, string(claudeContent), "dex:var-plugin")
	assert.Contains(t, string(claudeContent), "Static content for var test")
}

func TestInstaller_LocalRegistry_WithMCPServer(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Set up a local plugin with MCP server
	pluginDir := t.TempDir()
	pluginContent := `package {
  name = "mcp-plugin"
  version = "1.0.0"
  description = "Plugin with MCP server"
}

claude_mcp_server "test-mcp" {
  type = "command"
  command = "npx"
  args = ["-y", "test-mcp-server"]
}
`
	err := os.WriteFile(filepath.Join(pluginDir, "package.hcl"), []byte(pluginContent), 0644)
	require.NoError(t, err)

	// Create project config
	createTestProject(t, projectDir, `
plugin "mcp-plugin" {
  source = "file:`+pluginDir+`"
}
`)

	// Create installer
	installer, err := NewInstaller(projectDir)
	require.NoError(t, err)

	// Install
	err = installer.InstallAll()
	require.NoError(t, err)

	// Verify .mcp.json was created
	mcpContent, err := os.ReadFile(filepath.Join(projectDir, ".mcp.json"))
	require.NoError(t, err)

	var mcpConfig map[string]any
	err = json.Unmarshal(mcpContent, &mcpConfig)
	require.NoError(t, err)

	servers, ok := mcpConfig["mcpServers"].(map[string]any)
	require.True(t, ok)
	server, ok := servers["test-mcp"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "npx", server["command"])
}

func TestInstaller_LocalRegistry_WithSettings(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Set up a local plugin with settings
	pluginDir := t.TempDir()
	pluginContent := `package {
  name = "settings-plugin"
  version = "1.0.0"
  description = "Plugin with settings"
}

claude_settings "test-settings" {
  allow = ["Bash(npm:*)"]
}
`
	err := os.WriteFile(filepath.Join(pluginDir, "package.hcl"), []byte(pluginContent), 0644)
	require.NoError(t, err)

	// Create project config
	createTestProject(t, projectDir, `
plugin "settings-plugin" {
  source = "file:`+pluginDir+`"
}
`)

	// Create installer
	installer, err := NewInstaller(projectDir)
	require.NoError(t, err)

	// Install
	err = installer.InstallAll()
	require.NoError(t, err)

	// Verify settings.json was created
	settingsContent, err := os.ReadFile(filepath.Join(projectDir, ".claude", "settings.json"))
	require.NoError(t, err)

	var settings map[string]any
	err = json.Unmarshal(settingsContent, &settings)
	require.NoError(t, err)

	allow, ok := settings["allow"].([]any)
	require.True(t, ok)
	assert.Contains(t, allow, "Bash(npm:*)")
}

func TestInstaller_LocalRegistry_WithSkill(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Set up a local plugin with a skill
	pluginDir := t.TempDir()
	pluginContent := `package {
  name = "skill-plugin"
  version = "1.0.0"
  description = "Plugin with skill"
}

claude_skill "my-skill" {
  description = "A test skill"
  content = <<EOF
You are a helpful assistant that specializes in this skill.

When invoked, follow these steps:
1. Analyze the request
2. Execute the skill
3. Report results
EOF
}
`
	err := os.WriteFile(filepath.Join(pluginDir, "package.hcl"), []byte(pluginContent), 0644)
	require.NoError(t, err)

	// Create project config
	createTestProject(t, projectDir, `
plugin "skill-plugin" {
  source = "file:`+pluginDir+`"
}
`)

	// Create installer
	installer, err := NewInstaller(projectDir)
	require.NoError(t, err)

	// Install
	err = installer.InstallAll()
	require.NoError(t, err)

	// Verify skill file was created
	// Skills are installed to .claude/skills/{skill-name}/SKILL.md (non-namespaced by default)
	skillContent, err := os.ReadFile(filepath.Join(projectDir, ".claude", "skills", "my-skill", "SKILL.md"))
	require.NoError(t, err)
	assert.Contains(t, string(skillContent), "A test skill")
	assert.Contains(t, string(skillContent), "You are a helpful assistant")
}

func TestInstaller_LocalRegistry_WithCommand(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Set up a local plugin with a command
	pluginDir := t.TempDir()
	pluginContent := `package {
  name = "command-plugin"
  version = "1.0.0"
  description = "Plugin with command"
}

claude_command "my-command" {
  description = "A test command"
  content = <<EOF
Execute this command to do something useful.

Arguments:
- arg1: First argument
- arg2: Second argument
EOF
}
`
	err := os.WriteFile(filepath.Join(pluginDir, "package.hcl"), []byte(pluginContent), 0644)
	require.NoError(t, err)

	// Create project config
	createTestProject(t, projectDir, `
plugin "command-plugin" {
  source = "file:`+pluginDir+`"
}
`)

	// Create installer
	installer, err := NewInstaller(projectDir)
	require.NoError(t, err)

	// Install
	err = installer.InstallAll()
	require.NoError(t, err)

	// Verify command file was created
	// Commands are installed to .claude/commands/{name}.md (non-namespaced by default)
	cmdContent, err := os.ReadFile(filepath.Join(projectDir, ".claude", "commands", "my-command.md"))
	require.NoError(t, err)
	assert.Contains(t, string(cmdContent), "A test command")
	assert.Contains(t, string(cmdContent), "Execute this command")
}

func TestInstaller_LocalRegistry_MultiplePlugins(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Set up multiple local plugins
	plugin1Dir := t.TempDir()
	createTestPlugin(t, plugin1Dir, "plugin-one", "1.0.0", "First plugin")

	plugin2Dir := t.TempDir()
	createTestPlugin(t, plugin2Dir, "plugin-two", "2.0.0", "Second plugin")

	// Create project config with multiple plugins
	createTestProject(t, projectDir, `
plugin "plugin-one" {
  source = "file:`+plugin1Dir+`"
}

plugin "plugin-two" {
  source = "file:`+plugin2Dir+`"
}
`)

	// Create installer
	installer, err := NewInstaller(projectDir)
	require.NoError(t, err)

	// Install all
	err = installer.InstallAll()
	require.NoError(t, err)

	// Verify both plugins' content
	claudeContent, err := os.ReadFile(filepath.Join(projectDir, "CLAUDE.md"))
	require.NoError(t, err)
	assert.Contains(t, string(claudeContent), "dex:plugin-one")
	assert.Contains(t, string(claudeContent), "dex:plugin-two")
	assert.Contains(t, string(claudeContent), "Follow this rule from plugin-one")
	assert.Contains(t, string(claudeContent), "Follow this rule from plugin-two")

	// Verify lock file has both
	lockContent, err := os.ReadFile(filepath.Join(projectDir, "dex.lock"))
	require.NoError(t, err)
	assert.Contains(t, string(lockContent), "plugin-one")
	assert.Contains(t, string(lockContent), "plugin-two")
}

// =============================================================================
// Error Handling Tests
// =============================================================================

func TestInstaller_LocalRegistry_InvalidSource(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Create project config with invalid source
	createTestProject(t, projectDir, `
plugin "invalid-plugin" {
  source = "file:/nonexistent/path/that/does/not/exist"
}
`)

	// Create installer
	installer, err := NewInstaller(projectDir)
	require.NoError(t, err)

	// Install should fail
	err = installer.InstallAll()
	assert.Error(t, err)
}

func TestInstaller_LocalRegistry_MissingPackageHCL(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Create empty plugin directory (no package.hcl)
	emptyPluginDir := t.TempDir()

	// Create project config pointing to empty directory
	createTestProject(t, projectDir, `
plugin "empty-plugin" {
  source = "file:`+emptyPluginDir+`"
}
`)

	// Create installer
	installer, err := NewInstaller(projectDir)
	require.NoError(t, err)

	// Install should fail
	err = installer.InstallAll()
	assert.Error(t, err)
}

func TestInstaller_LocalRegistry_NoVersionsMatch(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Set up a local registry
	registryDir := t.TempDir()
	createLocalRegistryIndex(t, registryDir, map[string][]string{
		"strict-plugin": {"1.0.0", "1.1.0"},
	})

	// Create the package directory
	pluginDir := filepath.Join(registryDir, "strict-plugin")
	err := os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)
	createTestPlugin(t, pluginDir, "strict-plugin", "1.1.0", "Strict Plugin")

	// Create project config with unsatisfiable constraint
	createTestProject(t, projectDir, `
registry "local" {
  path = "`+registryDir+`"
}

plugin "strict-plugin" {
  registry = "local"
  version = "^3.0.0"
}
`)

	// Create installer
	installer, err := NewInstaller(projectDir)
	require.NoError(t, err)

	// Install should fail due to version constraint
	err = installer.InstallAll()
	assert.Error(t, err)
}

// =============================================================================
// Lock File Tests
// =============================================================================

func TestInstaller_LocalRegistry_LockFilePreservation(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Set up a local registry with multiple versions
	registryDir := t.TempDir()
	createLocalRegistryIndex(t, registryDir, map[string][]string{
		"lock-test-plugin": {"1.0.0", "1.1.0", "1.2.0"},
	})

	// Create the package directory
	pluginDir := filepath.Join(registryDir, "lock-test-plugin")
	err := os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)
	createTestPlugin(t, pluginDir, "lock-test-plugin", "1.1.0", "Lock Test Plugin")

	// Create project config without version (will resolve to latest)
	createTestProject(t, projectDir, `
registry "local" {
  path = "`+registryDir+`"
}

plugin "lock-test-plugin" {
  registry = "local"
}
`)

	// First install - should get 1.2.0 (latest)
	installer1, err := NewInstaller(projectDir)
	require.NoError(t, err)
	err = installer1.InstallAll()
	require.NoError(t, err)

	// Verify lock file has 1.2.0 (JSON format)
	lockContent1, err := os.ReadFile(filepath.Join(projectDir, "dex.lock"))
	require.NoError(t, err)
	assert.Contains(t, string(lockContent1), `"version": "1.2.0"`)

	// Simulate adding a new version to registry
	createLocalRegistryIndex(t, registryDir, map[string][]string{
		"lock-test-plugin": {"1.0.0", "1.1.0", "1.2.0", "1.3.0"},
	})

	// Second install should use locked version, not latest
	installer2, err := NewInstaller(projectDir)
	require.NoError(t, err)
	err = installer2.InstallAll()
	require.NoError(t, err)

	// Verify lock file still has 1.2.0 (JSON format)
	lockContent2, err := os.ReadFile(filepath.Join(projectDir, "dex.lock"))
	require.NoError(t, err)
	assert.Contains(t, string(lockContent2), `"version": "1.2.0"`)
}

func TestInstaller_LocalRegistry_NoLockOption(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Set up a local plugin
	pluginDir := t.TempDir()
	createTestPlugin(t, pluginDir, "no-lock-plugin", "1.0.0", "No lock test plugin")

	// Create project config
	createTestProject(t, projectDir, `
plugin "no-lock-plugin" {
  source = "file:`+pluginDir+`"
}
`)

	// Create installer with --no-lock
	installer, err := NewInstaller(projectDir)
	require.NoError(t, err)
	installer = installer.WithNoLock(true)

	// Install
	err = installer.InstallAll()
	require.NoError(t, err)

	// Verify lock file doesn't exist or is empty/minimal
	lockPath := filepath.Join(projectDir, "dex.lock")
	_, err = os.Stat(lockPath)
	if err == nil {
		// Lock file exists, verify it doesn't have the plugin entry
		lockContent, readErr := os.ReadFile(lockPath)
		require.NoError(t, readErr)
		assert.NotContains(t, string(lockContent), `"no-lock-plugin"`)
	} else {
		// Lock file doesn't exist, which is also valid for --no-lock
		assert.True(t, os.IsNotExist(err), "Expected no lock file or empty lock file")
	}
}
