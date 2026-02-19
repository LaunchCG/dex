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
// Sync Tests
// =============================================================================

func TestSync_InstallsMissingPlugins(t *testing.T) {
	projectDir := t.TempDir()

	// Set up a local plugin
	pluginDir := t.TempDir()
	createTestPlugin(t, pluginDir, "new-plugin", "1.0.0", "A new plugin")

	// Create project config referencing the plugin
	createTestProject(t, projectDir, `
plugin "new-plugin" {
  source = "file:`+pluginDir+`"
}
`)

	inst, err := NewInstaller(projectDir)
	require.NoError(t, err)

	results, err := inst.Sync(false)
	require.NoError(t, err)
	require.Len(t, results, 1)

	assert.Equal(t, "new-plugin", results[0].Name)
	assert.Equal(t, SyncInstalled, results[0].Action)
	assert.Equal(t, "1.0.0", results[0].NewVersion)
	assert.Empty(t, results[0].OldVersion)

	// Verify it was actually installed
	claudeContent, err := os.ReadFile(filepath.Join(projectDir, "CLAUDE.md"))
	require.NoError(t, err)
	expected := "Follow this rule from new-plugin"
	assert.Equal(t, expected, string(claudeContent))

	// Verify lock file
	lockContent, err := os.ReadFile(filepath.Join(projectDir, "dex.lock"))
	require.NoError(t, err)
	var lockData map[string]any
	err = json.Unmarshal(lockContent, &lockData)
	require.NoError(t, err)
	plugins := lockData["plugins"].(map[string]any)
	require.Contains(t, plugins, "new-plugin")
}

func TestSync_UpdatesOutdatedPlugins(t *testing.T) {
	projectDir := t.TempDir()

	// Set up a local registry with multiple versions
	registryDir := t.TempDir()
	createLocalRegistryIndex(t, registryDir, map[string][]string{
		"updatable-plugin": {"1.0.0", "1.1.0"},
	})

	pluginDir := filepath.Join(registryDir, "updatable-plugin")
	err := os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)
	createTestPlugin(t, pluginDir, "updatable-plugin", "1.1.0", "Updatable plugin")

	// Create project config
	createTestProject(t, projectDir, `
registry "local" {
  path = "`+registryDir+`"
}

plugin "updatable-plugin" {
  registry = "local"
}
`)

	// First install to get v1.1.0 locked
	inst1, err := NewInstaller(projectDir)
	require.NoError(t, err)
	err = inst1.InstallAll()
	require.NoError(t, err)

	// Now add a newer version to the registry
	createLocalRegistryIndex(t, registryDir, map[string][]string{
		"updatable-plugin": {"1.0.0", "1.1.0", "1.2.0"},
	})
	// Update the package.hcl to reflect the new version
	createTestPlugin(t, pluginDir, "updatable-plugin", "1.2.0", "Updatable plugin v1.2.0")

	// Sync should detect the update
	inst2, err := NewInstaller(projectDir)
	require.NoError(t, err)

	results, err := inst2.Sync(false)
	require.NoError(t, err)
	require.Len(t, results, 1)

	assert.Equal(t, "updatable-plugin", results[0].Name)
	assert.Equal(t, SyncUpdated, results[0].Action)
	assert.Equal(t, "1.1.0", results[0].OldVersion)
	assert.Equal(t, "1.2.0", results[0].NewVersion)
}

func TestSync_SkipsUpToDatePlugins(t *testing.T) {
	projectDir := t.TempDir()

	// Set up a local registry
	registryDir := t.TempDir()
	createLocalRegistryIndex(t, registryDir, map[string][]string{
		"stable-plugin": {"1.0.0"},
	})

	pluginDir := filepath.Join(registryDir, "stable-plugin")
	err := os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)
	createTestPlugin(t, pluginDir, "stable-plugin", "1.0.0", "Stable plugin")

	// Create project config
	createTestProject(t, projectDir, `
registry "local" {
  path = "`+registryDir+`"
}

plugin "stable-plugin" {
  registry = "local"
}
`)

	// First install
	inst1, err := NewInstaller(projectDir)
	require.NoError(t, err)
	err = inst1.InstallAll()
	require.NoError(t, err)

	// Sync should show up-to-date
	inst2, err := NewInstaller(projectDir)
	require.NoError(t, err)

	results, err := inst2.Sync(false)
	require.NoError(t, err)
	require.Len(t, results, 1)

	assert.Equal(t, "stable-plugin", results[0].Name)
	assert.Equal(t, SyncUpToDate, results[0].Action)
	assert.Equal(t, "1.0.0", results[0].OldVersion)
	assert.Equal(t, "1.0.0", results[0].NewVersion)
}

func TestSync_PrunesOrphanedPlugins(t *testing.T) {
	projectDir := t.TempDir()

	// Set up two local plugins
	pluginADir := t.TempDir()
	createTestPlugin(t, pluginADir, "keep-plugin", "1.0.0", "Keep this plugin")

	pluginBDir := t.TempDir()
	createTestPlugin(t, pluginBDir, "remove-plugin", "1.0.0", "Remove this plugin")

	// Create project config with both plugins
	createTestProject(t, projectDir, `
plugin "keep-plugin" {
  source = "file:`+pluginADir+`"
}

plugin "remove-plugin" {
  source = "file:`+pluginBDir+`"
}
`)

	// Install both
	inst1, err := NewInstaller(projectDir)
	require.NoError(t, err)
	err = inst1.InstallAll()
	require.NoError(t, err)

	// Now update config to only have keep-plugin
	createTestProject(t, projectDir, `
plugin "keep-plugin" {
  source = "file:`+pluginADir+`"
}
`)

	// Sync should prune remove-plugin
	inst2, err := NewInstaller(projectDir)
	require.NoError(t, err)

	results, err := inst2.Sync(false)
	require.NoError(t, err)
	require.Len(t, results, 2)

	// Results are sorted: up_to_date first, then pruned
	var upToDate, pruned *SyncResult
	for idx := range results {
		switch results[idx].Action {
		case SyncUpToDate:
			upToDate = &results[idx]
		case SyncPruned:
			pruned = &results[idx]
		}
	}

	require.NotNil(t, upToDate)
	assert.Equal(t, "keep-plugin", upToDate.Name)

	require.NotNil(t, pruned)
	assert.Equal(t, "remove-plugin", pruned.Name)
	assert.Equal(t, "1.0.0", pruned.OldVersion)

	// Verify remove-plugin is no longer in lock file
	lockContent, err := os.ReadFile(filepath.Join(projectDir, "dex.lock"))
	require.NoError(t, err)
	var lockData map[string]any
	err = json.Unmarshal(lockContent, &lockData)
	require.NoError(t, err)
	plugins := lockData["plugins"].(map[string]any)
	assert.NotContains(t, plugins, "remove-plugin")
	assert.Contains(t, plugins, "keep-plugin")
}

func TestSync_DryRunReportsWithoutChanges(t *testing.T) {
	projectDir := t.TempDir()

	// Set up a local plugin that isn't installed yet
	pluginDir := t.TempDir()
	createTestPlugin(t, pluginDir, "dry-run-plugin", "1.0.0", "Dry run plugin")

	// Create project config
	createTestProject(t, projectDir, `
plugin "dry-run-plugin" {
  source = "file:`+pluginDir+`"
}
`)

	inst, err := NewInstaller(projectDir)
	require.NoError(t, err)

	results, err := inst.Sync(true)
	require.NoError(t, err)
	require.Len(t, results, 1)

	assert.Equal(t, "dry-run-plugin", results[0].Name)
	assert.Equal(t, SyncInstalled, results[0].Action)
	assert.Equal(t, "1.0.0", results[0].NewVersion)
	assert.Equal(t, "would install", results[0].Reason)

	// Verify nothing was actually installed
	_, err = os.Stat(filepath.Join(projectDir, "CLAUDE.md"))
	assert.True(t, os.IsNotExist(err), "CLAUDE.md should not exist after dry-run")

	_, err = os.Stat(filepath.Join(projectDir, "dex.lock"))
	assert.True(t, os.IsNotExist(err), "dex.lock should not exist after dry-run")
}

func TestSync_DryRunPrune(t *testing.T) {
	projectDir := t.TempDir()

	// Install a plugin first
	pluginDir := t.TempDir()
	createTestPlugin(t, pluginDir, "prune-me", "1.0.0", "Will be pruned")

	createTestProject(t, projectDir, `
plugin "prune-me" {
  source = "file:`+pluginDir+`"
}
`)

	inst1, err := NewInstaller(projectDir)
	require.NoError(t, err)
	err = inst1.InstallAll()
	require.NoError(t, err)

	// Remove plugin from config
	createTestProject(t, projectDir, "")

	// Dry-run sync
	inst2, err := NewInstaller(projectDir)
	require.NoError(t, err)

	results, err := inst2.Sync(true)
	require.NoError(t, err)
	require.Len(t, results, 1)

	assert.Equal(t, "prune-me", results[0].Name)
	assert.Equal(t, SyncPruned, results[0].Action)
	assert.Equal(t, "would prune", results[0].Reason)

	// Verify the plugin is still installed (dry-run didn't remove it)
	lockContent, err := os.ReadFile(filepath.Join(projectDir, "dex.lock"))
	require.NoError(t, err)
	var lockData map[string]any
	err = json.Unmarshal(lockContent, &lockData)
	require.NoError(t, err)
	plugins := lockData["plugins"].(map[string]any)
	assert.Contains(t, plugins, "prune-me")
}

func TestSync_MixedInstallUpdatePrune(t *testing.T) {
	projectDir := t.TempDir()

	// Set up a registry with two plugins
	registryDir := t.TempDir()
	createLocalRegistryIndex(t, registryDir, map[string][]string{
		"existing-plugin": {"1.0.0"},
		"orphan-plugin":   {"1.0.0"},
	})

	existingDir := filepath.Join(registryDir, "existing-plugin")
	err := os.MkdirAll(existingDir, 0755)
	require.NoError(t, err)
	createTestPlugin(t, existingDir, "existing-plugin", "1.0.0", "Existing plugin")

	orphanDir := filepath.Join(registryDir, "orphan-plugin")
	err = os.MkdirAll(orphanDir, 0755)
	require.NoError(t, err)
	createTestPlugin(t, orphanDir, "orphan-plugin", "1.0.0", "Orphan plugin")

	// Install both plugins
	createTestProject(t, projectDir, `
registry "local" {
  path = "`+registryDir+`"
}

plugin "existing-plugin" {
  registry = "local"
}

plugin "orphan-plugin" {
  registry = "local"
}
`)

	inst1, err := NewInstaller(projectDir)
	require.NoError(t, err)
	err = inst1.InstallAll()
	require.NoError(t, err)

	// Now: add a new version for existing-plugin, add a new plugin, remove orphan-plugin
	createLocalRegistryIndex(t, registryDir, map[string][]string{
		"existing-plugin": {"1.0.0", "1.1.0"},
		"orphan-plugin":   {"1.0.0"},
		"new-plugin":      {"2.0.0"},
	})
	createTestPlugin(t, existingDir, "existing-plugin", "1.1.0", "Existing plugin v1.1.0")

	newDir := filepath.Join(registryDir, "new-plugin")
	err = os.MkdirAll(newDir, 0755)
	require.NoError(t, err)
	createTestPlugin(t, newDir, "new-plugin", "2.0.0", "Brand new plugin")

	// Update config: keep existing-plugin, add new-plugin, remove orphan-plugin
	createTestProject(t, projectDir, `
registry "local" {
  path = "`+registryDir+`"
}

plugin "existing-plugin" {
  registry = "local"
}

plugin "new-plugin" {
  registry = "local"
}
`)

	// Sync
	inst2, err := NewInstaller(projectDir)
	require.NoError(t, err)

	results, err := inst2.Sync(false)
	require.NoError(t, err)
	require.Len(t, results, 3)

	// Build a map for easier assertion
	resultMap := make(map[string]SyncResult)
	for _, r := range results {
		resultMap[r.Name] = r
	}

	// new-plugin should be installed
	assert.Equal(t, SyncInstalled, resultMap["new-plugin"].Action)
	assert.Equal(t, "2.0.0", resultMap["new-plugin"].NewVersion)

	// existing-plugin should be updated
	assert.Equal(t, SyncUpdated, resultMap["existing-plugin"].Action)
	assert.Equal(t, "1.0.0", resultMap["existing-plugin"].OldVersion)
	assert.Equal(t, "1.1.0", resultMap["existing-plugin"].NewVersion)

	// orphan-plugin should be pruned
	assert.Equal(t, SyncPruned, resultMap["orphan-plugin"].Action)
	assert.Equal(t, "1.0.0", resultMap["orphan-plugin"].OldVersion)

	// Verify lock file reflects the sync
	lockContent, err := os.ReadFile(filepath.Join(projectDir, "dex.lock"))
	require.NoError(t, err)
	var lockData map[string]any
	err = json.Unmarshal(lockContent, &lockData)
	require.NoError(t, err)
	plugins := lockData["plugins"].(map[string]any)
	assert.Contains(t, plugins, "existing-plugin")
	assert.Contains(t, plugins, "new-plugin")
	assert.NotContains(t, plugins, "orphan-plugin")
	assert.Equal(t, "1.1.0", plugins["existing-plugin"].(map[string]any)["version"])
	assert.Equal(t, "2.0.0", plugins["new-plugin"].(map[string]any)["version"])
}

func TestSync_AlwaysReinstallsUpToDatePlugins(t *testing.T) {
	projectDir := t.TempDir()

	// Set up a local plugin with an MCP server
	pluginDir := t.TempDir()
	pluginContent := `package {
  name = "mcp-plugin"
  version = "1.0.0"
  description = "Plugin with MCP server"
}

mcp_server "test-server" {
  description = "Test MCP server"
  command = "npx"
  args = ["-y", "test-server"]
}

claude_rule "mcp-plugin-rule" {
  description = "Rule from mcp-plugin"
  content = "Follow this rule from mcp-plugin"
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

	// First install
	inst1, err := NewInstaller(projectDir)
	require.NoError(t, err)
	err = inst1.InstallAll()
	require.NoError(t, err)

	// Verify .mcp.json was created
	mcpPath := filepath.Join(projectDir, ".mcp.json")
	require.FileExists(t, mcpPath)

	mcpData, err := os.ReadFile(mcpPath)
	require.NoError(t, err)
	var mcpConfig map[string]any
	err = json.Unmarshal(mcpData, &mcpConfig)
	require.NoError(t, err)
	servers := mcpConfig["mcpServers"].(map[string]any)
	require.Contains(t, servers, "test-server")

	// Delete .mcp.json to simulate user deletion or corruption
	err = os.Remove(mcpPath)
	require.NoError(t, err)

	// Sync should reinstall the plugin and recreate .mcp.json
	inst2, err := NewInstaller(projectDir)
	require.NoError(t, err)

	results, err := inst2.Sync(false)
	require.NoError(t, err)
	require.Len(t, results, 1)

	// The plugin should be reinstalled, not just marked "up to date"
	assert.Equal(t, "mcp-plugin", results[0].Name)

	// Verify .mcp.json was recreated
	require.FileExists(t, mcpPath, ".mcp.json should be recreated by sync")

	mcpData, err = os.ReadFile(mcpPath)
	require.NoError(t, err)
	err = json.Unmarshal(mcpData, &mcpConfig)
	require.NoError(t, err)
	servers = mcpConfig["mcpServers"].(map[string]any)
	assert.Contains(t, servers, "test-server", "MCP server should be present after sync recreates .mcp.json")
}

func TestSync_ReinstallsWhenRegularFilesDeleted(t *testing.T) {
	projectDir := t.TempDir()

	// Set up a local plugin
	pluginDir := t.TempDir()
	createTestPlugin(t, pluginDir, "file-plugin", "1.0.0", "Plugin with files")

	// Create project config
	createTestProject(t, projectDir, `
plugin "file-plugin" {
  source = "file:`+pluginDir+`"
}
`)

	// First install
	inst1, err := NewInstaller(projectDir)
	require.NoError(t, err)
	err = inst1.InstallAll()
	require.NoError(t, err)

	// Verify CLAUDE.md was created with plugin content
	claudePath := filepath.Join(projectDir, "CLAUDE.md")
	claudeContent, err := os.ReadFile(claudePath)
	require.NoError(t, err)
	assert.Contains(t, string(claudeContent), "Follow this rule from file-plugin")

	// Delete CLAUDE.md
	err = os.Remove(claudePath)
	require.NoError(t, err)

	// Sync should reinstall and recreate CLAUDE.md
	inst2, err := NewInstaller(projectDir)
	require.NoError(t, err)

	_, err = inst2.Sync(false)
	require.NoError(t, err)

	// Verify CLAUDE.md was recreated
	require.FileExists(t, claudePath, "CLAUDE.md should be recreated by sync")
	claudeContent, err = os.ReadFile(claudePath)
	require.NoError(t, err)
	assert.Contains(t, string(claudeContent), "Follow this rule from file-plugin",
		"CLAUDE.md should contain plugin content after sync")
}

func TestSync_EmptyConfig(t *testing.T) {
	projectDir := t.TempDir()
	createTestProject(t, projectDir, "")

	inst, err := NewInstaller(projectDir)
	require.NoError(t, err)

	results, err := inst.Sync(false)
	require.NoError(t, err)
	assert.Empty(t, results)
}

// =============================================================================
// Platform Switch Tests
// =============================================================================

// createClaudePlugin creates a package.hcl with claude-code-specific resources.
func createClaudePlugin(t *testing.T, dir, name, version string) {
	t.Helper()
	content := `package {
  name = "` + name + `"
  version = "` + version + `"
  description = "Claude-code plugin"
}

claude_rule "` + name + `-rule" {
  description = "Rule from ` + name + `"
  content = "Follow this rule from ` + name + `"
}

claude_skill "` + name + `-skill" {
  description = "Skill from ` + name + `"
  content = "This is a skill from ` + name + `"
}
`
	err := os.WriteFile(filepath.Join(dir, "package.hcl"), []byte(content), 0644)
	require.NoError(t, err)
}

// createMCPUniversalPlugin creates a package.hcl with a universal mcp_server.
func createMCPUniversalPlugin(t *testing.T, dir, name, version string) {
	t.Helper()
	content := `package {
  name = "` + name + `"
  version = "` + version + `"
  description = "MCP plugin"
}

mcp_server "` + name + `-server" {
  description = "MCP server from ` + name + `"
  command = "npx"
  args = ["-y", "` + name + `"]
}
`
	err := os.WriteFile(filepath.Join(dir, "package.hcl"), []byte(content), 0644)
	require.NoError(t, err)
}

func TestSync_PlatformSwitch_ClaudeToGithubCopilot_SharedFiles(t *testing.T) {
	projectDir := t.TempDir()
	pluginDir := t.TempDir()
	createMCPUniversalPlugin(t, pluginDir, "mcp-plugin", "1.0.0")

	writeConfig := func(platform string) {
		t.Helper()
		content := `project {
  name = "test-project"
  agentic_platform = "` + platform + `"
}

plugin "mcp-plugin" {
  source = "file:` + pluginDir + `"
}
`
		err := os.WriteFile(filepath.Join(projectDir, "dex.hcl"), []byte(content), 0644)
		require.NoError(t, err)
	}

	// Install as claude-code
	writeConfig("claude-code")
	inst1, err := NewInstaller(projectDir)
	require.NoError(t, err)
	_, err = inst1.Sync(false)
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(projectDir, ".mcp.json"))

	// Switch to github-copilot
	writeConfig("github-copilot")
	inst2, err := NewInstaller(projectDir)
	require.NoError(t, err)
	_, err = inst2.Sync(false)
	require.NoError(t, err)

	// claude-code files must be gone
	assert.NoFileExists(t, filepath.Join(projectDir, "CLAUDE.md"), "CLAUDE.md must be removed after platform switch")
	assert.NoFileExists(t, filepath.Join(projectDir, ".mcp.json"), ".mcp.json must be removed after platform switch")
	assert.NoFileExists(t, filepath.Join(projectDir, ".claude", "settings.json"), ".claude/settings.json must be removed after platform switch")

	// copilot MCP must exist with correct key
	vscodeMCP := filepath.Join(projectDir, ".vscode", "mcp.json")
	assert.FileExists(t, vscodeMCP, ".vscode/mcp.json must be created for github-copilot")

	data, err := os.ReadFile(vscodeMCP)
	require.NoError(t, err)
	var mcpConfig map[string]any
	require.NoError(t, json.Unmarshal(data, &mcpConfig))
	assert.Contains(t, mcpConfig, "servers", ".vscode/mcp.json must use 'servers' key")
	assert.NotContains(t, mcpConfig, "mcpServers", ".vscode/mcp.json must not use 'mcpServers' key")
}

func TestSync_PlatformSwitch_ClaudeToGithubCopilot_DedicatedFiles(t *testing.T) {
	projectDir := t.TempDir()
	pluginDir := t.TempDir()
	createClaudePlugin(t, pluginDir, "my-plugin", "1.0.0")

	writeConfig := func(platform string) {
		t.Helper()
		content := `project {
  name = "test-project"
  agentic_platform = "` + platform + `"
}

plugin "my-plugin" {
  source = "file:` + pluginDir + `"
}
`
		err := os.WriteFile(filepath.Join(projectDir, "dex.hcl"), []byte(content), 0644)
		require.NoError(t, err)
	}

	// Install as claude-code
	writeConfig("claude-code")
	inst1, err := NewInstaller(projectDir)
	require.NoError(t, err)
	_, err = inst1.Sync(false)
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(projectDir, "CLAUDE.md"))
	skillDir := filepath.Join(projectDir, ".claude", "skills", "my-plugin-skill")
	assert.DirExists(t, skillDir, ".claude/skills/ must be created by claude-code install")

	// Switch to github-copilot — plugin has no copilot resources, plan will be empty
	writeConfig("github-copilot")
	inst2, err := NewInstaller(projectDir)
	require.NoError(t, err)
	_, err = inst2.Sync(false)
	require.NoError(t, err)

	// All claude-code dedicated files and dirs must be gone
	assert.NoFileExists(t, filepath.Join(projectDir, "CLAUDE.md"), "CLAUDE.md must be removed")
	assert.NoDirExists(t, skillDir, ".claude/skills/my-plugin-skill/ must be removed")
	assert.NoDirExists(t, filepath.Join(projectDir, ".claude", "skills"), ".claude/skills/ must be removed when empty")
	assert.NoDirExists(t, filepath.Join(projectDir, ".claude", "rules"), ".claude/rules/ must be removed when empty")
}

func TestSync_PlatformSwitch_ClaudeToGithubCopilot_AgentInstructions(t *testing.T) {
	projectDir := t.TempDir()

	writeConfig := func(platform string) {
		t.Helper()
		content := `project {
  name = "test-project"
  agentic_platform = "` + platform + `"
  agent_instructions = "# Project Rules\n\nAlways follow these rules."
}
`
		err := os.WriteFile(filepath.Join(projectDir, "dex.hcl"), []byte(content), 0644)
		require.NoError(t, err)
	}

	// Install as claude-code
	writeConfig("claude-code")
	inst1, err := NewInstaller(projectDir)
	require.NoError(t, err)
	_, err = inst1.Sync(false)
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(projectDir, "CLAUDE.md"))

	// Switch to github-copilot
	writeConfig("github-copilot")
	inst2, err := NewInstaller(projectDir)
	require.NoError(t, err)
	_, err = inst2.Sync(false)
	require.NoError(t, err)

	assert.NoFileExists(t, filepath.Join(projectDir, "CLAUDE.md"), "CLAUDE.md must be removed after platform switch")

	copilotInstructions := filepath.Join(projectDir, ".github", "copilot-instructions.md")
	assert.FileExists(t, copilotInstructions, ".github/copilot-instructions.md must be created")

	content, err := os.ReadFile(copilotInstructions)
	require.NoError(t, err)
	assert.Contains(t, string(content), "Always follow these rules.")
}

func TestSync_PlatformSwitch_IdempotentAfterSwitch(t *testing.T) {
	projectDir := t.TempDir()
	pluginDir := t.TempDir()
	createMCPUniversalPlugin(t, pluginDir, "mcp-plugin", "1.0.0")

	writeConfig := func(platform string) {
		t.Helper()
		content := `project {
  name = "test-project"
  agentic_platform = "` + platform + `"
}

plugin "mcp-plugin" {
  source = "file:` + pluginDir + `"
}
`
		err := os.WriteFile(filepath.Join(projectDir, "dex.hcl"), []byte(content), 0644)
		require.NoError(t, err)
	}

	// Install on claude-code
	writeConfig("claude-code")
	inst1, err := NewInstaller(projectDir)
	require.NoError(t, err)
	_, err = inst1.Sync(false)
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(projectDir, ".mcp.json"))

	// Switch to github-copilot
	writeConfig("github-copilot")
	inst2, err := NewInstaller(projectDir)
	require.NoError(t, err)
	_, err = inst2.Sync(false)
	require.NoError(t, err)
	assert.NoFileExists(t, filepath.Join(projectDir, ".mcp.json"))
	assert.FileExists(t, filepath.Join(projectDir, ".vscode", "mcp.json"))

	// Sync again — must be idempotent
	inst3, err := NewInstaller(projectDir)
	require.NoError(t, err)
	_, err = inst3.Sync(false)
	require.NoError(t, err)
	assert.NoFileExists(t, filepath.Join(projectDir, ".mcp.json"), "second sync must not recreate old platform files")
	assert.FileExists(t, filepath.Join(projectDir, ".vscode", "mcp.json"), "second sync must keep new platform files")
}
