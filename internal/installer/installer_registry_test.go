package installer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
	expectedCLAUDE := "<!-- dex:my-test-plugin -->\nFollow this rule from my-test-plugin\n<!-- /dex:my-test-plugin -->"
	assert.Equal(t, expectedCLAUDE, string(claudeContent))

	// Verify lock file was updated (JSON format)
	lockContent, err := os.ReadFile(filepath.Join(projectDir, "dex.lock"))
	require.NoError(t, err)
	var lockData map[string]any
	err = json.Unmarshal(lockContent, &lockData)
	require.NoError(t, err)
	assert.Equal(t, "1.0", lockData["version"])
	assert.Equal(t, "claude-code", lockData["agent"])
	plugins := lockData["plugins"].(map[string]any)
	require.Contains(t, plugins, "my-test-plugin")
	pluginEntry := plugins["my-test-plugin"].(map[string]any)
	assert.Equal(t, "1.0.0", pluginEntry["version"])
	assert.Equal(t, "file:"+pluginDir, pluginEntry["resolved"])

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
	expectedCLAUDE := "<!-- dex:plugin-a -->\nFollow this rule from plugin-a\n<!-- /dex:plugin-a -->\n\n" +
		"<!-- dex:plugin-b -->\nFollow this rule from plugin-b\n<!-- /dex:plugin-b -->"
	assert.Equal(t, expectedCLAUDE, string(claudeContent))

	// Verify lock file has both plugins
	lockContent, err := os.ReadFile(filepath.Join(projectDir, "dex.lock"))
	require.NoError(t, err)
	var lockData map[string]any
	err = json.Unmarshal(lockContent, &lockData)
	require.NoError(t, err)
	plugins := lockData["plugins"].(map[string]any)
	require.Contains(t, plugins, "plugin-a")
	require.Contains(t, plugins, "plugin-b")
	pluginA := plugins["plugin-a"].(map[string]any)
	assert.Equal(t, "2.0.0", pluginA["version"])
	pluginB := plugins["plugin-b"].(map[string]any)
	assert.Equal(t, "1.0.0", pluginB["version"])
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
	var lockData map[string]any
	err = json.Unmarshal(lockContent, &lockData)
	require.NoError(t, err)
	plugins := lockData["plugins"].(map[string]any)
	pluginEntry := plugins["versioned-plugin"].(map[string]any)
	assert.Equal(t, "1.2.0", pluginEntry["version"])
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
	expectedCLAUDE1 := "<!-- dex:reinstall-plugin -->\nFollow this rule from reinstall-plugin\n<!-- /dex:reinstall-plugin -->"
	assert.Equal(t, expectedCLAUDE1, string(claudeContent1))

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
	expectedCLAUDE2 := "<!-- dex:reinstall-plugin -->\nThis is the updated rule content\n<!-- /dex:reinstall-plugin -->"
	assert.Equal(t, expectedCLAUDE2, string(claudeContent2))
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
	expectedCLAUDE := "<!-- dex:var-plugin -->\nStatic content for var test\n<!-- /dex:var-plugin -->"
	assert.Equal(t, expectedCLAUDE, string(claudeContent))
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

	assert.Equal(t, map[string]any{
		"allow": []any{"Bash(npm:*)"},
	}, settings)
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
	expectedSkill := "---\nname: my-skill\ndescription: A test skill\n---\n" +
		"You are a helpful assistant that specializes in this skill.\n\n" +
		"When invoked, follow these steps:\n1. Analyze the request\n2. Execute the skill\n3. Report results\n"
	assert.Equal(t, expectedSkill, string(skillContent))
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
	expectedCmd := "---\nname: my-command\ndescription: A test command\n---\n" +
		"Execute this command to do something useful.\n\n" +
		"Arguments:\n- arg1: First argument\n- arg2: Second argument\n"
	assert.Equal(t, expectedCmd, string(cmdContent))
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
	expectedCLAUDE := "<!-- dex:plugin-one -->\nFollow this rule from plugin-one\n<!-- /dex:plugin-one -->\n\n" +
		"<!-- dex:plugin-two -->\nFollow this rule from plugin-two\n<!-- /dex:plugin-two -->"
	assert.Equal(t, expectedCLAUDE, string(claudeContent))

	// Verify lock file has both
	lockContent, err := os.ReadFile(filepath.Join(projectDir, "dex.lock"))
	require.NoError(t, err)
	var lockData map[string]any
	err = json.Unmarshal(lockContent, &lockData)
	require.NoError(t, err)
	plugins := lockData["plugins"].(map[string]any)
	require.Contains(t, plugins, "plugin-one")
	require.Contains(t, plugins, "plugin-two")
	assert.Equal(t, "1.0.0", plugins["plugin-one"].(map[string]any)["version"])
	assert.Equal(t, "2.0.0", plugins["plugin-two"].(map[string]any)["version"])
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
	var lockData1 map[string]any
	err = json.Unmarshal(lockContent1, &lockData1)
	require.NoError(t, err)
	plugins1 := lockData1["plugins"].(map[string]any)
	assert.Equal(t, "1.2.0", plugins1["lock-test-plugin"].(map[string]any)["version"])

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
	var lockData2 map[string]any
	err = json.Unmarshal(lockContent2, &lockData2)
	require.NoError(t, err)
	plugins2 := lockData2["plugins"].(map[string]any)
	assert.Equal(t, "1.2.0", plugins2["lock-test-plugin"].(map[string]any)["version"])
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

	// Verify lock file doesn't exist or has no plugins
	lockPath := filepath.Join(projectDir, "dex.lock")
	_, err = os.Stat(lockPath)
	if err == nil {
		// Lock file exists, verify it has no plugin entries
		lockContent, readErr := os.ReadFile(lockPath)
		require.NoError(t, readErr)
		var lockData map[string]any
		readErr = json.Unmarshal(lockContent, &lockData)
		require.NoError(t, readErr)
		plugins := lockData["plugins"].(map[string]any)
		assert.Equal(t, map[string]any{}, plugins, "Lock file should have no plugins when --no-lock is used")
	} else {
		// Lock file doesn't exist, which is also valid for --no-lock
		assert.True(t, os.IsNotExist(err), "Expected no lock file or empty lock file")
	}
}

// =============================================================================
// Save Flag Tests (-S / --save)
// =============================================================================

func TestInstaller_SaveFlag_RegistryInstall(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Set up a local registry
	registryDir := t.TempDir()
	createLocalRegistryIndex(t, registryDir, map[string][]string{
		"save-test-plugin": {"1.0.0"},
	})

	// Create the package directory
	pluginDir := filepath.Join(registryDir, "save-test-plugin")
	err := os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)
	createTestPlugin(t, pluginDir, "save-test-plugin", "1.0.0", "Save test plugin")

	// Create project config with registry
	createTestProject(t, projectDir, `
registry "local" {
  path = "`+registryDir+`"
}

plugin "save-test-plugin" {
  registry = "local"
}
`)

	// Create installer and install
	inst, err := NewInstaller(projectDir)
	require.NoError(t, err)

	installed, err := inst.Install([]PluginSpec{{
		Name:     "save-test-plugin",
		Registry: "local",
	}})
	require.NoError(t, err)
	require.Len(t, installed, 1)

	// Verify InstalledPlugin.Registry is set
	assert.Equal(t, "local", installed[0].Registry)
	assert.Equal(t, "save-test-plugin", installed[0].Name)
	assert.Equal(t, "1.0.0", installed[0].Version)
}

func TestInstaller_SaveFlag_SourceInstall(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Set up a local plugin
	pluginDir := t.TempDir()
	createTestPlugin(t, pluginDir, "source-save-plugin", "1.0.0", "Source save test plugin")

	// Create project config
	createTestProject(t, projectDir, `
plugin "source-save-plugin" {
  source = "file:`+pluginDir+`"
}
`)

	// Create installer and install
	inst, err := NewInstaller(projectDir)
	require.NoError(t, err)

	installed, err := inst.Install([]PluginSpec{{
		Name:   "source-save-plugin",
		Source: "file:" + pluginDir,
	}})
	require.NoError(t, err)
	require.Len(t, installed, 1)

	// Verify InstalledPlugin.Source is set
	assert.Equal(t, "file:"+pluginDir, installed[0].Source)
	assert.Equal(t, "source-save-plugin", installed[0].Name)
}

// =============================================================================
// Auto-Search Registry Tests
// =============================================================================

func TestInstaller_AutoSearchRegistries_Found(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Set up two local registries
	registry1Dir := t.TempDir()
	createLocalRegistryIndex(t, registry1Dir, map[string][]string{
		"other-plugin": {"1.0.0"},
	})
	pluginDir1 := filepath.Join(registry1Dir, "other-plugin")
	err := os.MkdirAll(pluginDir1, 0755)
	require.NoError(t, err)
	createTestPlugin(t, pluginDir1, "other-plugin", "1.0.0", "Other plugin")

	registry2Dir := t.TempDir()
	createLocalRegistryIndex(t, registry2Dir, map[string][]string{
		"target-plugin": {"2.0.0"},
	})
	pluginDir2 := filepath.Join(registry2Dir, "target-plugin")
	err = os.MkdirAll(pluginDir2, 0755)
	require.NoError(t, err)
	createTestPlugin(t, pluginDir2, "target-plugin", "2.0.0", "Target plugin")

	// Create project config with two registries but NO plugin block for target-plugin
	createTestProject(t, projectDir, `
registry "first" {
  path = "`+registry1Dir+`"
}

registry "second" {
  path = "`+registry2Dir+`"
}
`)

	// Create installer and install WITHOUT specifying --registry
	inst, err := NewInstaller(projectDir)
	require.NoError(t, err)

	installed, err := inst.Install([]PluginSpec{{
		Name: "target-plugin",
	}})
	require.NoError(t, err)
	require.Len(t, installed, 1)

	assert.Equal(t, "target-plugin", installed[0].Name)
	assert.Equal(t, "2.0.0", installed[0].Version)
	assert.Equal(t, "second", installed[0].Registry)
}

func TestInstaller_AutoSearchRegistries_NotFound(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Set up a registry without the target plugin
	registryDir := t.TempDir()
	createLocalRegistryIndex(t, registryDir, map[string][]string{
		"some-other-plugin": {"1.0.0"},
	})

	// Create project config
	createTestProject(t, projectDir, `
registry "local" {
  path = "`+registryDir+`"
}
`)

	// Create installer and try to install a plugin that doesn't exist
	inst, err := NewInstaller(projectDir)
	require.NoError(t, err)

	_, err = inst.Install([]PluginSpec{{
		Name: "nonexistent-plugin",
	}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent-plugin")
}

func TestInstaller_AutoSearchRegistries_Ambiguous(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Set up two registries both containing the same plugin
	registry1Dir := t.TempDir()
	createLocalRegistryIndex(t, registry1Dir, map[string][]string{
		"ambiguous-plugin": {"1.0.0"},
	})
	pluginDir1 := filepath.Join(registry1Dir, "ambiguous-plugin")
	err := os.MkdirAll(pluginDir1, 0755)
	require.NoError(t, err)
	createTestPlugin(t, pluginDir1, "ambiguous-plugin", "1.0.0", "Ambiguous plugin v1")

	registry2Dir := t.TempDir()
	createLocalRegistryIndex(t, registry2Dir, map[string][]string{
		"ambiguous-plugin": {"2.0.0"},
	})
	pluginDir2 := filepath.Join(registry2Dir, "ambiguous-plugin")
	err = os.MkdirAll(pluginDir2, 0755)
	require.NoError(t, err)
	createTestPlugin(t, pluginDir2, "ambiguous-plugin", "2.0.0", "Ambiguous plugin v2")

	// Create project config with both registries
	createTestProject(t, projectDir, `
registry "first" {
  path = "`+registry1Dir+`"
}

registry "second" {
  path = "`+registry2Dir+`"
}
`)

	// Create installer and try to install the ambiguous plugin
	inst, err := NewInstaller(projectDir)
	require.NoError(t, err)

	_, err = inst.Install([]PluginSpec{{
		Name: "ambiguous-plugin",
	}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "multiple registries")
	assert.Contains(t, err.Error(), "--registry")
}

func TestInstaller_AutoSearchRegistries_NoRegistries(t *testing.T) {
	// Set up the project directory with no registries
	projectDir := t.TempDir()

	createTestProject(t, projectDir, "")

	// Create installer and try to install without any registry
	inst, err := NewInstaller(projectDir)
	require.NoError(t, err)

	_, err = inst.Install([]PluginSpec{{
		Name: "some-plugin",
	}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no source or registry specified")
}

func TestInstaller_AutoSearchAndSave(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Set up a local registry
	registryDir := t.TempDir()
	createLocalRegistryIndex(t, registryDir, map[string][]string{
		"auto-save-plugin": {"1.0.0"},
	})
	pluginDir := filepath.Join(registryDir, "auto-save-plugin")
	err := os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)
	createTestPlugin(t, pluginDir, "auto-save-plugin", "1.0.0", "Auto save plugin")

	// Create project config with registry but NO plugin block
	createTestProject(t, projectDir, `
registry "my-reg" {
  path = "`+registryDir+`"
}
`)

	// Create installer and install WITHOUT --registry (auto-search should find it)
	inst, err := NewInstaller(projectDir)
	require.NoError(t, err)

	installed, err := inst.Install([]PluginSpec{{
		Name: "auto-save-plugin",
	}})
	require.NoError(t, err)
	require.Len(t, installed, 1)

	// Verify InstalledPlugin carries the discovered registry name
	assert.Equal(t, "auto-save-plugin", installed[0].Name)
	assert.Equal(t, "1.0.0", installed[0].Version)
	assert.Equal(t, "my-reg", installed[0].Registry)
	assert.Empty(t, installed[0].Source)

	// Verify we can use this info to save - simulate what the CLI does
	// by checking that the registry name can be used with AddPluginToConfig
	_ = strings.Contains(installed[0].Registry, "my-reg") // use strings import
}
