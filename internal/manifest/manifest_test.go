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
	m.TrackMergedFile("test-package", ".mcp.json")
	m.TrackMergedFile("test-package", ".claude/settings.json")

	// Verify merged files are tracked
	pkg := m.GetPackage("test-package")
	require.NotNil(t, pkg)
	assert.Equal(t, []string{".mcp.json", ".claude/settings.json"}, pkg.MergedFiles)

	// Verify no duplicates
	m.TrackMergedFile("test-package", ".mcp.json")
	assert.Len(t, pkg.MergedFiles, 2)
}

func TestManifest_AllFiles_IncludesMergedFiles(t *testing.T) {
	tmpDir := t.TempDir()
	m, err := Load(tmpDir)
	require.NoError(t, err)

	// Track regular files and merged files
	m.Track("package1", []string{"skills/skill1.md"}, nil)
	m.TrackMergedFile("package1", ".mcp.json")

	m.Track("pkg2", []string{"commands/cmd1.md"}, nil)
	m.TrackMergedFile("pkg2", ".mcp.json")
	m.TrackMergedFile("pkg2", ".claude/settings.json")

	// Get all files
	allFiles := m.AllFiles()

	// Verify all files are included (no duplicates; map iteration order is non-deterministic)
	assert.ElementsMatch(t, []string{
		"skills/skill1.md",
		"commands/cmd1.md",
		".mcp.json",
		".claude/settings.json",
	}, allFiles)
}

func TestManifest_Untrack_ReturnsMergedFiles(t *testing.T) {
	tmpDir := t.TempDir()
	m, err := Load(tmpDir)
	require.NoError(t, err)

	// Track files and merged files
	m.Track("test-package", []string{"skills/skill1.md"}, nil)
	m.TrackMergedFile("test-package", ".mcp.json")
	m.TrackMergedFile("test-package", "CLAUDE.md")

	// Untrack the package
	result := m.Untrack("test-package")

	// Verify merged files are returned
	require.NotNil(t, result)
	assert.Equal(t, []string{".mcp.json", "CLAUDE.md"}, result.MergedFiles)
	assert.Equal(t, []string{"skills/skill1.md"}, result.Files)

	// Verify package is removed
	assert.Nil(t, m.GetPackage("test-package"))
}

func TestManifest_IsMergedFileUsedByOthers(t *testing.T) {
	tmpDir := t.TempDir()
	m, err := Load(tmpDir)
	require.NoError(t, err)

	// Track merged files for multiple packages
	m.TrackMergedFile("package1", ".mcp.json")
	m.TrackMergedFile("package1", "CLAUDE.md")
	m.TrackMergedFile("pkg2", ".mcp.json")
	m.TrackMergedFile("pkg2", ".claude/settings.json")

	// Test that .mcp.json is used by others
	assert.True(t, m.IsMergedFileUsedByOthers("package1", ".mcp.json"),
		".mcp.json should be used by pkg2")
	assert.True(t, m.IsMergedFileUsedByOthers("pkg2", ".mcp.json"),
		".mcp.json should be used by package1")

	// Test that CLAUDE.md is not used by others
	assert.False(t, m.IsMergedFileUsedByOthers("package1", "CLAUDE.md"),
		"CLAUDE.md should only be used by package1")

	// Test that settings.json is not used by others
	assert.False(t, m.IsMergedFileUsedByOthers("pkg2", ".claude/settings.json"),
		".claude/settings.json should only be used by pkg2")

	// Test with non-existent file
	assert.False(t, m.IsMergedFileUsedByOthers("package1", "nonexistent.json"),
		"non-existent file should not be used by others")
}

func TestManifest_SaveAndLoad_PreservesMergedFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create and populate manifest
	m1, err := Load(tmpDir)
	require.NoError(t, err)

	m1.Track("package1", []string{"skills/skill1.md"}, []string{".claude/skills"})
	m1.TrackMergedFile("package1", ".mcp.json")
	m1.TrackMergedFile("package1", "CLAUDE.md")
	m1.TrackAgentContent("package1")

	// Save manifest
	err = m1.Save()
	require.NoError(t, err)

	// Load manifest
	m2, err := Load(tmpDir)
	require.NoError(t, err)

	// Verify merged files are preserved
	pkg := m2.GetPackage("package1")
	require.NotNil(t, pkg)
	assert.Equal(t, []string{".mcp.json", "CLAUDE.md"}, pkg.MergedFiles)
	assert.True(t, pkg.HasAgentContent)

	// Verify all files are included (single package, so order is deterministic)
	allFiles := m2.AllFiles()
	assert.ElementsMatch(t, []string{"skills/skill1.md", ".mcp.json", "CLAUDE.md"}, allFiles)
}

func TestManifest_MergedFiles_EmptyByDefault(t *testing.T) {
	tmpDir := t.TempDir()
	m, err := Load(tmpDir)
	require.NoError(t, err)

	// Track a package without merged files
	m.Track("test-package", []string{"skills/skill1.md"}, nil)

	// Verify MergedFiles is empty
	pkg := m.GetPackage("test-package")
	require.NotNil(t, pkg)
	assert.Empty(t, pkg.MergedFiles)

	// Verify manifest can still be saved
	err = m.Save()
	require.NoError(t, err)

	// Verify it can be loaded again
	m2, err := Load(tmpDir)
	require.NoError(t, err)
	pkg2 := m2.GetPackage("test-package")
	require.NotNil(t, pkg2)
	assert.Empty(t, pkg2.MergedFiles)
}

func TestManifest_MultiplePackages_SharedMergedFiles(t *testing.T) {
	tmpDir := t.TempDir()
	m, err := Load(tmpDir)
	require.NoError(t, err)

	// Simulate three packages all contributing to .mcp.json
	m.TrackMergedFile("package1", ".mcp.json")
	m.TrackMergedFile("pkg2", ".mcp.json")
	m.TrackMergedFile("package3", ".mcp.json")

	// Untrack package1
	result := m.Untrack("package1")
	assert.Equal(t, []string{".mcp.json"}, result.MergedFiles)

	// Verify .mcp.json is still used by others
	assert.True(t, m.IsMergedFileUsedByOthers("package1", ".mcp.json"))

	// Untrack pkg2
	m.Untrack("pkg2")

	// Verify .mcp.json is still used by package3
	package3 := m.GetPackage("package3")
	require.NotNil(t, package3)
	assert.Equal(t, []string{".mcp.json"}, package3.MergedFiles)

	// Untrack package3
	result = m.Untrack("package3")
	assert.Equal(t, []string{".mcp.json"}, result.MergedFiles)

	// Now no package uses .mcp.json
	assert.Empty(t, m.AllFiles())
}

func TestManifest_ProjectPackage_MergedFiles(t *testing.T) {
	tmpDir := t.TempDir()
	m, err := Load(tmpDir)
	require.NoError(t, err)

	// Track project-level resources
	m.TrackAgentContent("__project__")
	m.TrackMergedFile("__project__", "CLAUDE.md")

	// Track package resources
	m.TrackMergedFile("package1", "CLAUDE.md")
	m.TrackMergedFile("package1", ".mcp.json")

	// Verify both use CLAUDE.md
	assert.True(t, m.IsMergedFileUsedByOthers("__project__", "CLAUDE.md"))
	assert.True(t, m.IsMergedFileUsedByOthers("package1", "CLAUDE.md"))

	// Untrack project - CLAUDE.md should still be used by package1
	result := m.Untrack("__project__")
	assert.Equal(t, []string{"CLAUDE.md"}, result.MergedFiles)
	assert.True(t, m.IsMergedFileUsedByOthers("__project__", "CLAUDE.md"))

	// Verify package still has CLAUDE.md
	pkg := m.GetPackage("package1")
	require.NotNil(t, pkg)
	assert.Equal(t, []string{"CLAUDE.md", ".mcp.json"}, pkg.MergedFiles)
}

func TestManifest_AllFiles_MultiplePackages_ExactOutput(t *testing.T) {
	tmpDir := t.TempDir()
	m, err := Load(tmpDir)
	require.NoError(t, err)

	// Track 3 packages with different merged files
	m.Track("package-a", []string{".claude/skills/a-skill.md"}, nil)
	m.TrackMergedFile("package-a", ".mcp.json")
	m.TrackMergedFile("package-a", ".claude/settings.json")
	m.TrackMergedFile("package-a", "CLAUDE.md")

	m.Track("package-b", []string{".claude/skills/b-skill.md"}, nil)
	m.TrackMergedFile("package-b", ".mcp.json")
	m.TrackMergedFile("package-b", "CLAUDE.md")

	m.Track("package-c", []string{".claude/skills/c-skill.md"}, nil)
	m.TrackMergedFile("package-c", ".claude/settings.json")
	m.TrackMergedFile("package-c", "CLAUDE.md")

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
