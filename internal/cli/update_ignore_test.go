package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateDexSection_ExactOutput_EmptyExisting(t *testing.T) {
	dexSection := dexIgnoreStart + "\n" +
		".dex/\n" +
		"dex.lock\n" +
		".claude/settings.json\n" +
		".claude/skills/a-skill.md\n" +
		".mcp.json\n" +
		"CLAUDE.md\n" +
		dexIgnoreEnd + "\n"

	result := updateDexSection("", dexSection)

	expected := dexSection
	assert.Equal(t, expected, result)
}

func TestUpdateDexSection_ExactOutput_AppendsToExisting(t *testing.T) {
	existing := "node_modules/\n*.log\n"

	dexSection := dexIgnoreStart + "\n" +
		".dex/\n" +
		"dex.lock\n" +
		".mcp.json\n" +
		dexIgnoreEnd + "\n"

	result := updateDexSection(existing, dexSection)

	expected := "node_modules/\n*.log\n\n" +
		dexIgnoreStart + "\n" +
		".dex/\n" +
		"dex.lock\n" +
		".mcp.json\n" +
		dexIgnoreEnd + "\n"
	assert.Equal(t, expected, result)
}

func TestUpdateDexSection_ReplacesExisting(t *testing.T) {
	existing := "node_modules/\n\n" +
		dexIgnoreStart + "\n" +
		".dex/\n" +
		"dex.lock\n" +
		".mcp.json\n" +
		dexIgnoreEnd + "\n" +
		"\n*.log\n"

	newDexSection := dexIgnoreStart + "\n" +
		".dex/\n" +
		"dex.lock\n" +
		".claude/settings.json\n" +
		".claude/skills/a-skill.md\n" +
		".mcp.json\n" +
		"CLAUDE.md\n" +
		dexIgnoreEnd + "\n"

	result := updateDexSection(existing, newDexSection)

	expected := "node_modules/\n\n" +
		dexIgnoreStart + "\n" +
		".dex/\n" +
		"dex.lock\n" +
		".claude/settings.json\n" +
		".claude/skills/a-skill.md\n" +
		".mcp.json\n" +
		"CLAUDE.md\n" +
		dexIgnoreEnd + "\n" +
		"\n*.log\n"
	assert.Equal(t, expected, result)
}

func TestUpdateDexSection_ReplacesExisting_NothingAfter(t *testing.T) {
	existing := "node_modules/\n\n" +
		dexIgnoreStart + "\n" +
		".dex/\n" +
		"dex.lock\n" +
		dexIgnoreEnd + "\n"

	newDexSection := dexIgnoreStart + "\n" +
		".dex/\n" +
		"dex.lock\n" +
		".mcp.json\n" +
		"CLAUDE.md\n" +
		dexIgnoreEnd + "\n"

	result := updateDexSection(existing, newDexSection)

	expected := "node_modules/\n\n" +
		dexIgnoreStart + "\n" +
		".dex/\n" +
		"dex.lock\n" +
		".mcp.json\n" +
		"CLAUDE.md\n" +
		dexIgnoreEnd + "\n"
	assert.Equal(t, expected, result)
}

func TestStripDexSection_WithSection(t *testing.T) {
	content := "node_modules/\n\n" +
		dexIgnoreStart + "\n" +
		".dex/\n" +
		"dex.lock\n" +
		dexIgnoreEnd + "\n" +
		"*.log\n"

	result := stripDexSection(content)

	expected := "node_modules/\n*.log\n"
	assert.Equal(t, expected, result)
}

func TestStripDexSection_NoSection(t *testing.T) {
	content := "node_modules/\n*.log\n"
	result := stripDexSection(content)
	assert.Equal(t, content, result)
}

func TestStripDexSection_InvertedMarkers(t *testing.T) {
	// End marker appears before start marker — malformed, must be a no-op.
	content := dexIgnoreEnd + "\n" + ".dex/\n" + dexIgnoreStart + "\n"
	result := stripDexSection(content)
	assert.Equal(t, content, result)
}

func TestStripDexSection_SectionOnly(t *testing.T) {
	content := dexIgnoreStart + "\n" +
		".dex/\n" +
		"dex.lock\n" +
		dexIgnoreEnd + "\n"

	result := stripDexSection(content)
	assert.Equal(t, "", result)
}

func TestMigrateGitignore_StripsOldSection(t *testing.T) {
	dir := t.TempDir()

	// Write a .gitignore with a dex managed section
	gitignoreContent := "node_modules/\n\n" +
		dexIgnoreStart + "\n" +
		".dex/\n" +
		"dex.lock\n" +
		dexIgnoreEnd + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(gitignoreContent), 0644))

	err := migrateGitignore(dir)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	require.NoError(t, err)
	assert.Equal(t, "node_modules/\n", string(data))
}

func TestMigrateGitignore_NoOpWhenNoSection(t *testing.T) {
	dir := t.TempDir()

	original := "node_modules/\n*.log\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(original), 0644))

	err := migrateGitignore(dir)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	require.NoError(t, err)
	assert.Equal(t, original, string(data))
}

func TestMigrateGitignore_NoOpWhenNoFile(t *testing.T) {
	dir := t.TempDir()
	err := migrateGitignore(dir)
	assert.NoError(t, err)
}

func TestUpdateIgnore_MigratesGitignore(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".dex"), 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, ".dex", "manifest.json"),
		[]byte(`{"version":"1.0","plugins":{}}`),
		0644,
	))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0755))

	// Pre-populate .gitignore with an old dex managed section.
	gitignoreContent := "node_modules/\n\n" +
		dexIgnoreStart + "\n" +
		".dex/\n" +
		"dex.lock\n" +
		dexIgnoreEnd + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(gitignoreContent), 0644))

	err := updateIgnoreForProject(dir)
	require.NoError(t, err)

	// The dex section must have been removed from .gitignore.
	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	require.NoError(t, err)
	assert.Equal(t, "node_modules/\n", string(data))

	// And written to .git/info/exclude instead.
	excludeData, err := os.ReadFile(filepath.Join(dir, ".git", "info", "exclude"))
	require.NoError(t, err)
	excludeContent := string(excludeData)
	assert.True(t, len(excludeContent) > 0 && excludeContent[:len(dexIgnoreStart)] == dexIgnoreStart,
		"exclude file should start with dex ignore marker")
}

func TestUpdateIgnore_ErrorsInNonGitDir(t *testing.T) {
	dir := t.TempDir()
	// No .git directory — should return an error, not create one.
	err := updateIgnoreForProject(dir)
	require.Error(t, err)
	assert.Equal(t, "not a git repository: "+dir, err.Error())

	// Confirm .git was NOT created.
	_, statErr := os.Stat(filepath.Join(dir, ".git"))
	assert.True(t, os.IsNotExist(statErr), ".git should not have been created")
}

func TestUpdateIgnore_WritesToGitInfoExclude(t *testing.T) {
	dir := t.TempDir()

	// Create minimal .dex directory (manifest.Load creates an empty manifest if missing)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".dex"), 0755))
	// Write an empty manifest
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, ".dex", "manifest.json"),
		[]byte(`{"version":"1.0","plugins":{}}`),
		0644,
	))

	// Create .git directory (but not .git/info — updateIgnoreForProject should create it)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0755))

	err := updateIgnoreForProject(dir)
	require.NoError(t, err)

	excludePath := filepath.Join(dir, ".git", "info", "exclude")
	data, err := os.ReadFile(excludePath)
	require.NoError(t, err)

	expectedContent := dexIgnoreStart + "\n.dex/\ndex.lock\n" + dexIgnoreEnd + "\n"
	assert.Equal(t, expectedContent, string(data))
}

// TestBuildDexIgnoreSection_AlwaysCount verifies that dexIgnoreAlwaysCount matches the
// number of entries buildDexIgnoreSection writes when allFiles is empty.
// If buildDexIgnoreSection changes its always-included entries, this test will fail,
// prompting an update to dexIgnoreAlwaysCount.
func TestBuildDexIgnoreSection_AlwaysCount(t *testing.T) {
	section := buildDexIgnoreSection(nil)
	lines := 0
	for _, line := range strings.Split(section, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && trimmed != dexIgnoreStart && trimmed != dexIgnoreEnd {
			lines++
		}
	}
	assert.Equal(t, dexIgnoreAlwaysCount, lines,
		"dexIgnoreAlwaysCount must equal the number of always-included entries in buildDexIgnoreSection")
}
