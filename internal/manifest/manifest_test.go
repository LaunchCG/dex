package manifest

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManifest_TrackMergedFile(t *testing.T) {
	tmpDir := t.TempDir()
	m, err := Load(tmpDir)
	require.NoError(t, err)

	// Track a merged file
	m.TrackMergedFile("test-plugin", ".mcp.json")
	m.TrackMergedFile("test-plugin", ".claude/settings.json")

	// Verify merged files are tracked
	plugin := m.GetPlugin("test-plugin")
	require.NotNil(t, plugin)
	assert.Contains(t, plugin.MergedFiles, ".mcp.json")
	assert.Contains(t, plugin.MergedFiles, ".claude/settings.json")

	// Verify no duplicates
	m.TrackMergedFile("test-plugin", ".mcp.json")
	assert.Len(t, plugin.MergedFiles, 2)
}

func TestManifest_AllFiles_IncludesMergedFiles(t *testing.T) {
	tmpDir := t.TempDir()
	m, err := Load(tmpDir)
	require.NoError(t, err)

	// Track regular files and merged files
	m.Track("plugin1", []string{"skills/skill1.md"}, nil)
	m.TrackMergedFile("plugin1", ".mcp.json")

	m.Track("plugin2", []string{"commands/cmd1.md"}, nil)
	m.TrackMergedFile("plugin2", ".mcp.json")
	m.TrackMergedFile("plugin2", ".claude/settings.json")

	// Get all files
	allFiles := m.AllFiles()

	// Verify all files are included
	assert.Contains(t, allFiles, "skills/skill1.md")
	assert.Contains(t, allFiles, "commands/cmd1.md")
	assert.Contains(t, allFiles, ".mcp.json")
	assert.Contains(t, allFiles, ".claude/settings.json")

	// Verify no duplicates (both plugins track .mcp.json)
	mcpCount := 0
	for _, f := range allFiles {
		if f == ".mcp.json" {
			mcpCount++
		}
	}
	assert.Equal(t, 1, mcpCount, ".mcp.json should appear only once")
}

func TestManifest_Untrack_ReturnsMergedFiles(t *testing.T) {
	tmpDir := t.TempDir()
	m, err := Load(tmpDir)
	require.NoError(t, err)

	// Track files and merged files
	m.Track("test-plugin", []string{"skills/skill1.md"}, nil)
	m.TrackMergedFile("test-plugin", ".mcp.json")
	m.TrackMergedFile("test-plugin", "CLAUDE.md")

	// Untrack the plugin
	result := m.Untrack("test-plugin")

	// Verify merged files are returned
	require.NotNil(t, result)
	assert.Contains(t, result.MergedFiles, ".mcp.json")
	assert.Contains(t, result.MergedFiles, "CLAUDE.md")
	assert.Contains(t, result.Files, "skills/skill1.md")

	// Verify plugin is removed
	assert.Nil(t, m.GetPlugin("test-plugin"))
}

func TestManifest_IsMergedFileUsedByOthers(t *testing.T) {
	tmpDir := t.TempDir()
	m, err := Load(tmpDir)
	require.NoError(t, err)

	// Track merged files for multiple plugins
	m.TrackMergedFile("plugin1", ".mcp.json")
	m.TrackMergedFile("plugin1", "CLAUDE.md")
	m.TrackMergedFile("plugin2", ".mcp.json")
	m.TrackMergedFile("plugin2", ".claude/settings.json")

	// Test that .mcp.json is used by others
	assert.True(t, m.IsMergedFileUsedByOthers("plugin1", ".mcp.json"),
		".mcp.json should be used by plugin2")
	assert.True(t, m.IsMergedFileUsedByOthers("plugin2", ".mcp.json"),
		".mcp.json should be used by plugin1")

	// Test that CLAUDE.md is not used by others
	assert.False(t, m.IsMergedFileUsedByOthers("plugin1", "CLAUDE.md"),
		"CLAUDE.md should only be used by plugin1")

	// Test that settings.json is not used by others
	assert.False(t, m.IsMergedFileUsedByOthers("plugin2", ".claude/settings.json"),
		".claude/settings.json should only be used by plugin2")

	// Test with non-existent file
	assert.False(t, m.IsMergedFileUsedByOthers("plugin1", "nonexistent.json"),
		"non-existent file should not be used by others")
}

func TestManifest_SaveAndLoad_PreservesMergedFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create and populate manifest
	m1, err := Load(tmpDir)
	require.NoError(t, err)

	m1.Track("plugin1", []string{"skills/skill1.md"}, []string{".claude/skills"})
	m1.TrackMergedFile("plugin1", ".mcp.json")
	m1.TrackMergedFile("plugin1", "CLAUDE.md")
	m1.TrackAgentContent("plugin1")

	// Save manifest
	err = m1.Save()
	require.NoError(t, err)

	// Load manifest
	m2, err := Load(tmpDir)
	require.NoError(t, err)

	// Verify merged files are preserved
	plugin := m2.GetPlugin("plugin1")
	require.NotNil(t, plugin)
	assert.Contains(t, plugin.MergedFiles, ".mcp.json")
	assert.Contains(t, plugin.MergedFiles, "CLAUDE.md")
	assert.True(t, plugin.HasAgentContent)

	// Verify all files are included
	allFiles := m2.AllFiles()
	assert.Contains(t, allFiles, "skills/skill1.md")
	assert.Contains(t, allFiles, ".mcp.json")
	assert.Contains(t, allFiles, "CLAUDE.md")
}

func TestManifest_MergedFiles_EmptyByDefault(t *testing.T) {
	tmpDir := t.TempDir()
	m, err := Load(tmpDir)
	require.NoError(t, err)

	// Track a plugin without merged files
	m.Track("test-plugin", []string{"skills/skill1.md"}, nil)

	// Verify MergedFiles is empty
	plugin := m.GetPlugin("test-plugin")
	require.NotNil(t, plugin)
	assert.Empty(t, plugin.MergedFiles)

	// Verify manifest can still be saved
	err = m.Save()
	require.NoError(t, err)

	// Verify it can be loaded again
	m2, err := Load(tmpDir)
	require.NoError(t, err)
	plugin2 := m2.GetPlugin("test-plugin")
	require.NotNil(t, plugin2)
	assert.Empty(t, plugin2.MergedFiles)
}

func TestManifest_MultiplePlugins_SharedMergedFiles(t *testing.T) {
	tmpDir := t.TempDir()
	m, err := Load(tmpDir)
	require.NoError(t, err)

	// Simulate three plugins all contributing to .mcp.json
	m.TrackMergedFile("plugin1", ".mcp.json")
	m.TrackMergedFile("plugin2", ".mcp.json")
	m.TrackMergedFile("plugin3", ".mcp.json")

	// Untrack plugin1
	result := m.Untrack("plugin1")
	assert.Contains(t, result.MergedFiles, ".mcp.json")

	// Verify .mcp.json is still used by others
	assert.True(t, m.IsMergedFileUsedByOthers("plugin1", ".mcp.json"))

	// Untrack plugin2
	m.Untrack("plugin2")

	// Verify .mcp.json is still used by plugin3
	plugin3 := m.GetPlugin("plugin3")
	require.NotNil(t, plugin3)
	assert.Contains(t, plugin3.MergedFiles, ".mcp.json")

	// Untrack plugin3
	result = m.Untrack("plugin3")
	assert.Contains(t, result.MergedFiles, ".mcp.json")

	// Now no plugin uses .mcp.json
	assert.Empty(t, m.AllFiles())
}

func TestManifest_ProjectPlugin_MergedFiles(t *testing.T) {
	tmpDir := t.TempDir()
	m, err := Load(tmpDir)
	require.NoError(t, err)

	// Track project-level resources
	m.TrackAgentContent("__project__")
	m.TrackMergedFile("__project__", "CLAUDE.md")

	// Track plugin resources
	m.TrackMergedFile("plugin1", "CLAUDE.md")
	m.TrackMergedFile("plugin1", ".mcp.json")

	// Verify both use CLAUDE.md
	assert.True(t, m.IsMergedFileUsedByOthers("__project__", "CLAUDE.md"))
	assert.True(t, m.IsMergedFileUsedByOthers("plugin1", "CLAUDE.md"))

	// Untrack project - CLAUDE.md should still be used by plugin1
	result := m.Untrack("__project__")
	assert.Contains(t, result.MergedFiles, "CLAUDE.md")
	assert.True(t, m.IsMergedFileUsedByOthers("__project__", "CLAUDE.md"))

	// Verify plugin still has CLAUDE.md
	plugin := m.GetPlugin("plugin1")
	require.NotNil(t, plugin)
	assert.Contains(t, plugin.MergedFiles, "CLAUDE.md")
}

func TestManifest_AllFiles_MultiplePlugins_ExactOutput(t *testing.T) {
	tmpDir := t.TempDir()
	m, err := Load(tmpDir)
	require.NoError(t, err)

	// Track 3 plugins with different merged files
	m.Track("plugin-a", []string{".claude/skills/a-skill.md"}, nil)
	m.TrackMergedFile("plugin-a", ".mcp.json")
	m.TrackMergedFile("plugin-a", ".claude/settings.json")
	m.TrackMergedFile("plugin-a", "CLAUDE.md")

	m.Track("plugin-b", []string{".claude/skills/b-skill.md"}, nil)
	m.TrackMergedFile("plugin-b", ".mcp.json")
	m.TrackMergedFile("plugin-b", "CLAUDE.md")

	m.Track("plugin-c", []string{".claude/skills/c-skill.md"}, nil)
	m.TrackMergedFile("plugin-c", ".claude/settings.json")
	m.TrackMergedFile("plugin-c", "CLAUDE.md")

	allFiles := m.AllFiles()
	sort.Strings(allFiles)

	expected := []string{
		".claude/settings.json",
		".claude/skills/a-skill.md",
		".claude/skills/b-skill.md",
		".claude/skills/c-skill.md",
		".mcp.json",
		"CLAUDE.md",
	}
	assert.Equal(t, expected, allFiles)
}

func TestManifest_RemoveString_Helper(t *testing.T) {
	// Test the removeString helper function used in installer
	slice := []string{"a", "b", "c", "d"}

	result := removeString(slice, "b")
	assert.Equal(t, []string{"a", "c", "d"}, result)

	result = removeString(slice, "a")
	assert.Equal(t, []string{"b", "c", "d"}, result)

	result = removeString(slice, "d")
	assert.Equal(t, []string{"a", "b", "c"}, result)

	result = removeString(slice, "x")
	assert.Equal(t, []string{"a", "b", "c", "d"}, result)

	result = removeString([]string{}, "a")
	assert.Empty(t, result)
}

// removeString helper for testing (duplicated from installer package)
func removeString(slice []string, s string) []string {
	result := make([]string, 0, len(slice))
	for _, v := range slice {
		if v != s {
			result = append(result, v)
		}
	}
	return result
}
