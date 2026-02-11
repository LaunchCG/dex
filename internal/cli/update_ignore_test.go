package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
