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
package "new-plugin" {
  source = "file:`+pluginDir+`"
}
`)

	inst, err := NewInstaller(projectDir, "")
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
	plugins := lockData["packages"].(map[string]any)
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

package "updatable-plugin" {
  registry = "local"
}
`)

	// First install to get v1.1.0 locked
	inst1, err := NewInstaller(projectDir, "")
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
	inst2, err := NewInstaller(projectDir, "")
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

package "stable-plugin" {
  registry = "local"
}
`)

	// First install
	inst1, err := NewInstaller(projectDir, "")
	require.NoError(t, err)
	err = inst1.InstallAll()
	require.NoError(t, err)

	// Sync should show up-to-date
	inst2, err := NewInstaller(projectDir, "")
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
package "keep-plugin" {
  source = "file:`+pluginADir+`"
}

package "remove-plugin" {
  source = "file:`+pluginBDir+`"
}
`)

	// Install both
	inst1, err := NewInstaller(projectDir, "")
	require.NoError(t, err)
	err = inst1.InstallAll()
	require.NoError(t, err)

	// Now update config to only have keep-plugin
	createTestProject(t, projectDir, `
package "keep-plugin" {
  source = "file:`+pluginADir+`"
}
`)

	// Sync should prune remove-plugin
	inst2, err := NewInstaller(projectDir, "")
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
	plugins := lockData["packages"].(map[string]any)
	_, hasRemoved := plugins["remove-plugin"]
	assert.False(t, hasRemoved, "remove-plugin should not be in lock file")
	_, hasKept := plugins["keep-plugin"]
	assert.True(t, hasKept, "keep-plugin should be in lock file")
}

func TestSync_DryRunReportsWithoutChanges(t *testing.T) {
	projectDir := t.TempDir()

	// Set up a local plugin that isn't installed yet
	pluginDir := t.TempDir()
	createTestPlugin(t, pluginDir, "dry-run-plugin", "1.0.0", "Dry run plugin")

	// Create project config
	createTestProject(t, projectDir, `
package "dry-run-plugin" {
  source = "file:`+pluginDir+`"
}
`)

	inst, err := NewInstaller(projectDir, "")
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
package "prune-me" {
  source = "file:`+pluginDir+`"
}
`)

	inst1, err := NewInstaller(projectDir, "")
	require.NoError(t, err)
	err = inst1.InstallAll()
	require.NoError(t, err)

	// Remove plugin from config
	createTestProject(t, projectDir, "")

	// Dry-run sync
	inst2, err := NewInstaller(projectDir, "")
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
	plugins := lockData["packages"].(map[string]any)
	_, hasPruneMe := plugins["prune-me"]
	assert.True(t, hasPruneMe, "prune-me should still be in lock file after dry-run")
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

package "existing-plugin" {
  registry = "local"
}

package "orphan-plugin" {
  registry = "local"
}
`)

	inst1, err := NewInstaller(projectDir, "")
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

package "existing-plugin" {
  registry = "local"
}

package "new-plugin" {
  registry = "local"
}
`)

	// Sync
	inst2, err := NewInstaller(projectDir, "")
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
	plugins := lockData["packages"].(map[string]any)
	_, hasExisting := plugins["existing-plugin"]
	assert.True(t, hasExisting, "existing-plugin should be in lock file")
	_, hasNew := plugins["new-plugin"]
	assert.True(t, hasNew, "new-plugin should be in lock file")
	_, hasOrphan := plugins["orphan-plugin"]
	assert.False(t, hasOrphan, "orphan-plugin should not be in lock file")
	assert.Equal(t, "1.1.0", plugins["existing-plugin"].(map[string]any)["version"])
	assert.Equal(t, "2.0.0", plugins["new-plugin"].(map[string]any)["version"])
}

func TestSync_AlwaysReinstallsUpToDatePlugins(t *testing.T) {
	projectDir := t.TempDir()

	// Set up a local plugin with an MCP server
	pluginDir := t.TempDir()
	pluginContent := `meta {
  name = "mcp-plugin"
  version = "1.0.0"
  description = "Plugin with MCP server"
}

mcp_server "test-server" {
  description = "Test MCP server"
  command = "npx"
  args = ["-y", "test-server"]
}

rule "mcp-plugin-rule" {
  description = "Rule from mcp-plugin"
  content = "Follow this rule from mcp-plugin"
}
`
	err := os.WriteFile(filepath.Join(pluginDir, "package.hcl"), []byte(pluginContent), 0644)
	require.NoError(t, err)

	// Create project config
	createTestProject(t, projectDir, `
package "mcp-plugin" {
  source = "file:`+pluginDir+`"
}
`)

	// First install
	inst1, err := NewInstaller(projectDir, "")
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
	inst2, err := NewInstaller(projectDir, "")
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
	_, hasServer := servers["test-server"]
	assert.True(t, hasServer, "MCP server should be present after sync recreates .mcp.json")
}

func TestSync_ReinstallsWhenRegularFilesDeleted(t *testing.T) {
	projectDir := t.TempDir()

	// Set up a local plugin
	pluginDir := t.TempDir()
	createTestPlugin(t, pluginDir, "file-plugin", "1.0.0", "Plugin with files")

	// Create project config
	createTestProject(t, projectDir, `
package "file-plugin" {
  source = "file:`+pluginDir+`"
}
`)

	// First install
	inst1, err := NewInstaller(projectDir, "")
	require.NoError(t, err)
	err = inst1.InstallAll()
	require.NoError(t, err)

	// Verify CLAUDE.md was created with plugin content
	claudePath := filepath.Join(projectDir, "CLAUDE.md")
	claudeContent, err := os.ReadFile(claudePath)
	require.NoError(t, err)
	assert.Equal(t, "Follow this rule from file-plugin", string(claudeContent))

	// Delete CLAUDE.md
	err = os.Remove(claudePath)
	require.NoError(t, err)

	// Sync should reinstall and recreate CLAUDE.md
	inst2, err := NewInstaller(projectDir, "")
	require.NoError(t, err)

	_, err = inst2.Sync(false)
	require.NoError(t, err)

	// Verify CLAUDE.md was recreated
	require.FileExists(t, claudePath, "CLAUDE.md should be recreated by sync")
	claudeContent, err = os.ReadFile(claudePath)
	require.NoError(t, err)
	assert.Equal(t, "Follow this rule from file-plugin", string(claudeContent),
		"CLAUDE.md should contain plugin content after sync")
}

func TestSync_EmptyConfig(t *testing.T) {
	projectDir := t.TempDir()
	createTestProject(t, projectDir, "")

	inst, err := NewInstaller(projectDir, "")
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
	content := `meta {
  name = "` + name + `"
  version = "` + version + `"
  description = "Claude-code plugin"
}

rule "` + name + `-rule" {
  description = "Rule from ` + name + `"
  content = "Follow this rule from ` + name + `"
}

skill "` + name + `-skill" {
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
	content := `meta {
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
  default_platform = "` + platform + `"
}

package "mcp-plugin" {
  source = "file:` + pluginDir + `"
}
`
		err := os.WriteFile(filepath.Join(projectDir, "dex.hcl"), []byte(content), 0644)
		require.NoError(t, err)
	}

	// Install as claude-code
	writeConfig("claude-code")
	inst1, err := NewInstaller(projectDir, "")
	require.NoError(t, err)
	_, err = inst1.Sync(false)
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(projectDir, ".mcp.json"))

	// Switch to github-copilot
	writeConfig("github-copilot")
	inst2, err := NewInstaller(projectDir, "")
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
	_, hasServers := mcpConfig["servers"]
	assert.True(t, hasServers, ".vscode/mcp.json must use 'servers' key")
	_, hasMcpServers := mcpConfig["mcpServers"]
	assert.False(t, hasMcpServers, ".vscode/mcp.json must not use 'mcpServers' key")
}

func TestSync_PlatformSwitch_ClaudeToGithubCopilot_DedicatedFiles(t *testing.T) {
	projectDir := t.TempDir()
	pluginDir := t.TempDir()
	createClaudePlugin(t, pluginDir, "my-plugin", "1.0.0")

	writeConfig := func(platform string) {
		t.Helper()
		content := `project {
  name = "test-project"
  default_platform = "` + platform + `"
}

package "my-plugin" {
  source = "file:` + pluginDir + `"
}
`
		err := os.WriteFile(filepath.Join(projectDir, "dex.hcl"), []byte(content), 0644)
		require.NoError(t, err)
	}

	// Install as claude-code
	writeConfig("claude-code")
	inst1, err := NewInstaller(projectDir, "")
	require.NoError(t, err)
	_, err = inst1.Sync(false)
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(projectDir, "CLAUDE.md"))
	skillDir := filepath.Join(projectDir, ".claude", "skills", "my-plugin-skill")
	assert.DirExists(t, skillDir, ".claude/skills/ must be created by claude-code install")

	// Switch to github-copilot — plugin has no copilot resources, plan will be empty
	writeConfig("github-copilot")
	inst2, err := NewInstaller(projectDir, "")
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
  default_platform = "` + platform + `"
  agent_instructions = "# Project Rules\n\nAlways follow these rules."
}
`
		err := os.WriteFile(filepath.Join(projectDir, "dex.hcl"), []byte(content), 0644)
		require.NoError(t, err)
	}

	// Install as claude-code
	writeConfig("claude-code")
	inst1, err := NewInstaller(projectDir, "")
	require.NoError(t, err)
	_, err = inst1.Sync(false)
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(projectDir, "CLAUDE.md"))

	// Switch to github-copilot
	writeConfig("github-copilot")
	inst2, err := NewInstaller(projectDir, "")
	require.NoError(t, err)
	_, err = inst2.Sync(false)
	require.NoError(t, err)

	assert.NoFileExists(t, filepath.Join(projectDir, "CLAUDE.md"), "CLAUDE.md must be removed after platform switch")

	copilotInstructions := filepath.Join(projectDir, ".github", "copilot-instructions.md")
	assert.FileExists(t, copilotInstructions, ".github/copilot-instructions.md must be created")

	content, err := os.ReadFile(copilotInstructions)
	require.NoError(t, err)
	assert.Equal(t, "# Project Rules\n\nAlways follow these rules.", string(content))
}

func TestSync_PlatformSwitch_IdempotentAfterSwitch(t *testing.T) {
	projectDir := t.TempDir()
	pluginDir := t.TempDir()
	createMCPUniversalPlugin(t, pluginDir, "mcp-plugin", "1.0.0")

	writeConfig := func(platform string) {
		t.Helper()
		content := `project {
  name = "test-project"
  default_platform = "` + platform + `"
}

package "mcp-plugin" {
  source = "file:` + pluginDir + `"
}
`
		err := os.WriteFile(filepath.Join(projectDir, "dex.hcl"), []byte(content), 0644)
		require.NoError(t, err)
	}

	// Install on claude-code
	writeConfig("claude-code")
	inst1, err := NewInstaller(projectDir, "")
	require.NoError(t, err)
	_, err = inst1.Sync(false)
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(projectDir, ".mcp.json"))

	// Switch to github-copilot
	writeConfig("github-copilot")
	inst2, err := NewInstaller(projectDir, "")
	require.NoError(t, err)
	_, err = inst2.Sync(false)
	require.NoError(t, err)
	assert.NoFileExists(t, filepath.Join(projectDir, ".mcp.json"))
	assert.FileExists(t, filepath.Join(projectDir, ".vscode", "mcp.json"))

	// Sync again — must be idempotent
	inst3, err := NewInstaller(projectDir, "")
	require.NoError(t, err)
	_, err = inst3.Sync(false)
	require.NoError(t, err)
	assert.NoFileExists(t, filepath.Join(projectDir, ".mcp.json"), "second sync must not recreate old platform files")
	assert.FileExists(t, filepath.Join(projectDir, ".vscode", "mcp.json"), "second sync must keep new platform files")
}

// createPluginWithFile creates a package.hcl with a universal file resource.
func createPluginWithFile(t *testing.T, dir, name, version, dest, content string) {
	t.Helper()
	hcl := `meta {
  name = "` + name + `"
  version = "` + version + `"
  description = "Plugin with file resource"
}

file "my-file" {
  dest    = "` + dest + `"
  content = "` + content + `"
}
`
	err := os.WriteFile(filepath.Join(dir, "package.hcl"), []byte(hcl), 0644)
	require.NoError(t, err)
}

func TestSync_ClearsManifestWhenFileResourceRemoved(t *testing.T) {
	projectDir := t.TempDir()
	pluginDir := t.TempDir()

	// v1: plugin has a file resource
	createPluginWithFile(t, pluginDir, "file-plugin", "1.0.0", "config/my-file.txt", "hello world")

	createTestProject(t, projectDir, `
package "file-plugin" {
  source = "file:`+pluginDir+`"
}
`)

	// Install v1
	inst1, err := NewInstaller(projectDir, "")
	require.NoError(t, err)
	err = inst1.InstallAll()
	require.NoError(t, err)

	// Verify file exists and is in manifest
	filePath := filepath.Join(projectDir, "config", "my-file.txt")
	require.FileExists(t, filePath)

	mf1, err := loadManifestForTest(projectDir)
	require.NoError(t, err)
	require.NotNil(t, mf1["file-plugin"])
	assert.Equal(t, []string{"config/my-file.txt"}, mf1["file-plugin"], "file must be tracked in manifest after install")

	// Update plugin: remove the file resource entirely
	noResourcePlugin := `meta {
  name = "file-plugin"
  version = "1.0.0"
  description = "Plugin with no resources"
}
`
	err = os.WriteFile(filepath.Join(pluginDir, "package.hcl"), []byte(noResourcePlugin), 0644)
	require.NoError(t, err)

	// Sync
	inst2, err := NewInstaller(projectDir, "")
	require.NoError(t, err)
	_, err = inst2.Sync(false)
	require.NoError(t, err)

	// File must be gone from disk
	assert.NoFileExists(t, filePath, "old file must be deleted when resource is removed")

	// Manifest must NOT contain the old path
	mf2, err := loadManifestForTest(projectDir)
	require.NoError(t, err)
	plugin2 := mf2["file-plugin"]
	assert.Empty(t, plugin2, "manifest should have no files for plugin after resource removal")
}

func TestSync_ClearsManifestWhenFileDestinationChanges(t *testing.T) {
	projectDir := t.TempDir()
	pluginDir := t.TempDir()

	// v1: dest = "config/v1.txt"
	createPluginWithFile(t, pluginDir, "dest-plugin", "1.0.0", "config/v1.txt", "version one")

	createTestProject(t, projectDir, `
package "dest-plugin" {
  source = "file:`+pluginDir+`"
}
`)

	// Install v1
	inst1, err := NewInstaller(projectDir, "")
	require.NoError(t, err)
	err = inst1.InstallAll()
	require.NoError(t, err)

	v1Path := filepath.Join(projectDir, "config", "v1.txt")
	require.FileExists(t, v1Path)

	mf1, err := loadManifestForTest(projectDir)
	require.NoError(t, err)
	assert.Equal(t, []string{"config/v1.txt"}, mf1["dest-plugin"])

	// Update plugin: change dest to "config/v2.txt"
	createPluginWithFile(t, pluginDir, "dest-plugin", "1.0.0", "config/v2.txt", "version two")

	// Sync
	inst2, err := NewInstaller(projectDir, "")
	require.NoError(t, err)
	_, err = inst2.Sync(false)
	require.NoError(t, err)

	v2Path := filepath.Join(projectDir, "config", "v2.txt")

	// Old file must be gone
	assert.NoFileExists(t, v1Path, "old destination file must be deleted after dest change")
	// New file must exist
	assert.FileExists(t, v2Path, "new destination file must exist after dest change")

	// Manifest must have v2.txt and NOT v1.txt
	mf2, err := loadManifestForTest(projectDir)
	require.NoError(t, err)
	plugin2 := mf2["dest-plugin"]
	assert.Equal(t, []string{"config/v2.txt"}, plugin2, "manifest must have only new file path")
}

// loadManifestForTest reads the manifest and returns a map of plugin -> []files.
func loadManifestForTest(projectDir string) (map[string][]string, error) {
	data, err := os.ReadFile(filepath.Join(projectDir, ".dex", "manifest.json"))
	if err != nil {
		return nil, err
	}
	var raw struct {
		Packages map[string]struct {
			Files []string `json:"files"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	result := make(map[string][]string, len(raw.Packages))
	for name, pm := range raw.Packages {
		result[name] = pm.Files
	}
	return result, nil
}

// TestSync_TransitiveDependenciesNotPruned reproduces the bug where transitive
// dependencies installed during Sync are immediately pruned in the same run.
//
// Repro: package "parent" declares dependency "child". Sync installs parent,
// which pulls in child as a transitive dep and writes it to the lock file.
// The prune pass then sees child in the lock but not in configPlugins (which
// only contains direct project plugins) and prunes it.
func TestSync_TransitiveDependenciesNotPruned(t *testing.T) {
	projectDir := t.TempDir()
	registryDir := t.TempDir()

	// Set up registry index with both plugins
	createLocalRegistryIndex(t, registryDir, map[string][]string{
		"parent-plugin": {"1.0.0"},
		"child-plugin":  {"1.0.0"},
	})

	// child-plugin: a simple plugin with no dependencies
	childDir := filepath.Join(registryDir, "child-plugin")
	require.NoError(t, os.MkdirAll(childDir, 0755))
	createTestPlugin(t, childDir, "child-plugin", "1.0.0", "Child plugin (transitive dep)")

	// parent-plugin: declares child-plugin as a dependency
	parentDir := filepath.Join(registryDir, "parent-plugin")
	require.NoError(t, os.MkdirAll(parentDir, 0755))
	err := os.WriteFile(filepath.Join(parentDir, "package.hcl"), []byte(`
meta {
  name        = "parent-plugin"
  version     = "1.0.0"
  description = "Parent plugin"
}

dependency "child-plugin" {
  version = ">=1.0.0"
}

rule "parent-rule" {
  description = "Rule from parent"
  content     = "Follow this rule from parent-plugin"
}
`), 0644)
	require.NoError(t, err)

	// Project only references parent-plugin directly
	createTestProject(t, projectDir, `
registry "local" {
  path = "`+registryDir+`"
}

package "parent-plugin" {
  registry = "local"
}
`)

	inst, err := NewInstaller(projectDir, "")
	require.NoError(t, err)

	results, err := inst.Sync(false)
	require.NoError(t, err)

	// Collect pruned plugin names for a readable failure message
	var pruned []string
	for _, r := range results {
		if r.Action == SyncPruned {
			pruned = append(pruned, r.Name)
		}
	}

	assert.Empty(t, pruned,
		"transitive dependencies should not be pruned in the same sync that installs them; got pruned: %v", pruned)

	// Also verify child-plugin is still in the lock file after sync
	lockContent, err := os.ReadFile(filepath.Join(projectDir, "dex.lock"))
	require.NoError(t, err)
	var lockData map[string]any
	require.NoError(t, json.Unmarshal(lockContent, &lockData))
	plugins := lockData["packages"].(map[string]any)
	_, hasChild := plugins["child-plugin"]
	assert.True(t, hasChild, "child-plugin (transitive dep) must remain in lock file after sync")
}

// =============================================================================
// Profile Sync Tests
// =============================================================================

func TestSync_ProfileAddsPlugins(t *testing.T) {
	projectDir := t.TempDir()

	// Create two test plugins
	defaultPluginDir := t.TempDir()
	createTestPlugin(t, defaultPluginDir, "default-plugin", "1.0.0", "Default plugin")

	profilePluginDir := t.TempDir()
	createTestPlugin(t, profilePluginDir, "profile-plugin", "1.0.0", "Profile plugin")

	// Project config with a default plugin and a profile that adds another
	hcl := `
package "default-plugin" {
  source = "file:` + defaultPluginDir + `"
}

profile "qa" {
  package "profile-plugin" {
    source = "file:` + profilePluginDir + `"
  }
}
`
	createTestProject(t, projectDir, hcl)

	// Sync with profile — should install both default + profile plugin
	inst, err := NewInstaller(projectDir, "qa")
	require.NoError(t, err)

	results, err := inst.Sync(false)
	require.NoError(t, err)
	require.Len(t, results, 2)

	names := make(map[string]bool)
	for _, r := range results {
		names[r.Name] = true
		assert.Equal(t, SyncInstalled, r.Action)
	}
	assert.True(t, names["default-plugin"], "default plugin should be installed")
	assert.True(t, names["profile-plugin"], "profile plugin should be installed")

	// Verify CLAUDE.md has content from both plugins
	claudeContent, err := os.ReadFile(filepath.Join(projectDir, "CLAUDE.md"))
	require.NoError(t, err)
	assert.Equal(t, "Follow this rule from default-plugin\n\nFollow this rule from profile-plugin", string(claudeContent))
}

func TestSync_DefaultIgnoresProfiles(t *testing.T) {
	projectDir := t.TempDir()

	defaultPluginDir := t.TempDir()
	createTestPlugin(t, defaultPluginDir, "default-plugin", "1.0.0", "Default plugin")

	profilePluginDir := t.TempDir()
	createTestPlugin(t, profilePluginDir, "profile-plugin", "1.0.0", "Profile plugin")

	hcl := `
package "default-plugin" {
  source = "file:` + defaultPluginDir + `"
}

profile "qa" {
  package "profile-plugin" {
    source = "file:` + profilePluginDir + `"
  }
}
`
	createTestProject(t, projectDir, hcl)

	// Sync without profile — should only install default plugin
	inst, err := NewInstaller(projectDir, "")
	require.NoError(t, err)

	results, err := inst.Sync(false)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "default-plugin", results[0].Name)
	assert.Equal(t, SyncInstalled, results[0].Action)

	// Verify CLAUDE.md only has default content
	claudeContent, err := os.ReadFile(filepath.Join(projectDir, "CLAUDE.md"))
	require.NoError(t, err)
	assert.Equal(t, "Follow this rule from default-plugin", string(claudeContent))
}

func TestSync_ProfileExcludeDefaultsPrunesDefaultPlugins(t *testing.T) {
	projectDir := t.TempDir()

	defaultPluginDir := t.TempDir()
	createTestPlugin(t, defaultPluginDir, "default-plugin", "1.0.0", "Default plugin")

	profilePluginDir := t.TempDir()
	createTestPlugin(t, profilePluginDir, "profile-plugin", "1.0.0", "Profile plugin")

	hcl := `
package "default-plugin" {
  source = "file:` + defaultPluginDir + `"
}

profile "clean" {
  exclude_defaults = true

  package "profile-plugin" {
    source = "file:` + profilePluginDir + `"
  }
}
`
	createTestProject(t, projectDir, hcl)

	// Sync with exclude_defaults — only profile plugin
	inst, err := NewInstaller(projectDir, "clean")
	require.NoError(t, err)

	results, err := inst.Sync(false)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "profile-plugin", results[0].Name)
	assert.Equal(t, SyncInstalled, results[0].Action)

	// Verify CLAUDE.md only has profile content
	claudeContent, err := os.ReadFile(filepath.Join(projectDir, "CLAUDE.md"))
	require.NoError(t, err)
	assert.Equal(t, "Follow this rule from profile-plugin", string(claudeContent))
}

func TestSync_ProfileReplacesDefaultThenRevert(t *testing.T) {
	projectDir := t.TempDir()

	pluginDir := t.TempDir()
	createTestPlugin(t, pluginDir, "shared-plugin", "1.0.0", "Shared plugin")

	profilePluginDir := t.TempDir()
	createTestPlugin(t, profilePluginDir, "extra-plugin", "1.0.0", "Extra plugin")

	hcl := `
package "shared-plugin" {
  source = "file:` + pluginDir + `"
}

profile "qa" {
  package "extra-plugin" {
    source = "file:` + profilePluginDir + `"
  }
}
`
	createTestProject(t, projectDir, hcl)

	// Step 1: Sync with profile
	inst1, err := NewInstaller(projectDir, "qa")
	require.NoError(t, err)

	results1, err := inst1.Sync(false)
	require.NoError(t, err)

	names1 := make(map[string]bool)
	for _, r := range results1 {
		names1[r.Name] = true
	}
	assert.True(t, names1["shared-plugin"])
	assert.True(t, names1["extra-plugin"])

	// Step 2: Sync back to default — extra-plugin should be pruned
	inst2, err := NewInstaller(projectDir, "")
	require.NoError(t, err)

	results2, err := inst2.Sync(false)
	require.NoError(t, err)

	var pruned []string
	var upToDate []string
	for _, r := range results2 {
		switch r.Action {
		case SyncPruned:
			pruned = append(pruned, r.Name)
		case SyncUpToDate:
			upToDate = append(upToDate, r.Name)
		}
	}
	assert.Equal(t, []string{"shared-plugin"}, upToDate, "default plugin should remain")
	assert.Equal(t, []string{"extra-plugin"}, pruned, "profile plugin should be pruned on revert")

	// Verify CLAUDE.md no longer has extra-plugin content
	claudeContent, err := os.ReadFile(filepath.Join(projectDir, "CLAUDE.md"))
	require.NoError(t, err)
	assert.Equal(t, "Follow this rule from shared-plugin", string(claudeContent))
}

func TestSync_ProfileNotFoundErrors(t *testing.T) {
	projectDir := t.TempDir()
	createTestProject(t, projectDir, `
profile "qa" {}
`)

	_, err := NewInstaller(projectDir, "bogus")
	require.Error(t, err)
	assert.Equal(t, `config error at `+filepath.Join(projectDir, "dex.hcl")+`: failed to load project config: profile "bogus" not found; available profiles: qa`, err.Error())
}

func TestSync_ProfileAgentInstructions(t *testing.T) {
	projectDir := t.TempDir()

	pluginDir := t.TempDir()
	createTestPlugin(t, pluginDir, "test-plugin", "1.0.0", "Test plugin")

	// Write full dex.hcl with project-level agent_instructions and a profile that overrides them
	content := `project {
  name = "test-project"
  default_platform = "claude-code"
  agent_instructions = "Default instructions"
}

package "test-plugin" {
  source = "file:` + pluginDir + `"
}

profile "qa" {
  agent_instructions = "QA-specific instructions"
}
`
	err := os.WriteFile(filepath.Join(projectDir, "dex.hcl"), []byte(content), 0644)
	require.NoError(t, err)

	// Sync with profile
	inst, err := NewInstaller(projectDir, "qa")
	require.NoError(t, err)

	_, err = inst.Sync(false)
	require.NoError(t, err)

	// Verify CLAUDE.md starts with profile's agent instructions
	claudeContent, err := os.ReadFile(filepath.Join(projectDir, "CLAUDE.md"))
	require.NoError(t, err)
	assert.Equal(t, "QA-specific instructions\n\nFollow this rule from test-plugin", string(claudeContent))
}
