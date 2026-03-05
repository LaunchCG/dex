// Package cli implements the command-line interface for dex.
package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"

	"github.com/launchcg/dex/internal/manifest"
)

const (
	dexIgnoreStart = "# --- dex managed (do not edit) ---"
	dexIgnoreEnd   = "# --- end dex managed ---"
)

// updateIgnoreForProject updates .git/info/exclude in the given project directory
// with dex-managed files. Called by sync when --git-exclude is set or
// git_exclude = true in dex.hcl. Also migrates any existing dex section
// from .gitignore to the new location.
func updateIgnoreForProject(absPath string) error {
	// Load manifest
	mf, err := manifest.Load(absPath)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	// Get all managed files
	allFiles := mf.AllFiles()

	// Build the dex section
	dexSection := buildDexIgnoreSection(allFiles)

	// Verify this is a git repository before touching anything under .git/.
	gitDir := filepath.Join(absPath, ".git")
	if _, err := os.Stat(gitDir); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("not a git repository: %s", absPath)
	} else if err != nil {
		return fmt.Errorf("failed to stat .git directory: %w", err)
	}

	// Ensure .git/info/ directory exists
	gitInfoDir := filepath.Join(gitDir, "info")
	if err := os.MkdirAll(gitInfoDir, 0755); err != nil {
		return fmt.Errorf("failed to create .git/info directory: %w", err)
	}

	// Read existing .git/info/exclude
	excludePath := filepath.Join(gitInfoDir, "exclude")
	existingContent := ""
	if data, err := os.ReadFile(excludePath); err == nil {
		existingContent = string(data)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to read .git/info/exclude: %w", err)
	}

	// Update the dex section in exclude
	newContent := updateDexSection(existingContent, dexSection)

	// Write the updated .git/info/exclude
	if err := os.WriteFile(excludePath, []byte(newContent), 0600); err != nil {
		return fmt.Errorf("failed to write .git/info/exclude: %w", err)
	}

	// Migrate: remove old dex section from .gitignore if present.
	// Non-fatal: the core write to .git/info/exclude already succeeded, so a failure
	// here (e.g. read-only .gitignore) leaves the section in both files temporarily
	// until the user resolves the permission issue.
	if err := migrateGitignore(absPath); err != nil {
		fmt.Printf("%s Warning: failed to migrate .gitignore: %v\n", color.YellowString("⚠"), err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s Updated .git/info/exclude with %d managed file(s)\n", green("✓"), len(allFiles)+dexIgnoreAlwaysCount)

	return nil
}

// migrateGitignore removes the dex managed section from .gitignore if present.
// This is called after writing to .git/info/exclude to clean up the old location.
func migrateGitignore(projectDir string) error {
	gitignorePath := filepath.Join(projectDir, ".gitignore")
	data, err := os.ReadFile(gitignorePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read .gitignore: %w", err)
	}

	content := string(data)
	if !strings.Contains(content, dexIgnoreStart) {
		return nil
	}

	newContent := stripDexSection(content)
	if err := os.WriteFile(gitignorePath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write .gitignore: %w", err)
	}
	return nil
}

// stripDexSection removes the dex managed section from content, if present.
// Returns the modified content unchanged if no section is found.
func stripDexSection(content string) string {
	startIdx := strings.Index(content, dexIgnoreStart)
	endIdx := strings.Index(content, dexIgnoreEnd)
	if startIdx == -1 || endIdx == -1 || endIdx <= startIdx {
		return content
	}
	endIdx += len(dexIgnoreEnd)
	// Skip trailing newline after end marker
	if endIdx < len(content) && content[endIdx] == '\n' {
		endIdx++
	}
	before := content[:startIdx]
	after := content[endIdx:]
	// Remove the extra blank line that preceded the section
	if strings.HasSuffix(before, "\n\n") {
		before = before[:len(before)-1]
	}
	return before + after
}

// dexIgnoreAlwaysCount is the number of entries buildDexIgnoreSection always writes
// regardless of manifest contents (currently: .dex/ and dex.lock).
// If buildDexIgnoreSection changes its always-included entries, update this constant
// and TestBuildDexIgnoreSection_AlwaysCount in the test file to match.
const dexIgnoreAlwaysCount = 2

// buildDexIgnoreSection builds the dex-managed section content for .gitignore.
func buildDexIgnoreSection(allFiles []string) string {
	var dexSection strings.Builder
	dexSection.WriteString(dexIgnoreStart)
	dexSection.WriteString("\n")

	// Always include .dex directory and lock file
	dexSection.WriteString(".dex/\n")
	dexSection.WriteString("dex.lock\n")

	for _, file := range allFiles {
		dexSection.WriteString(file)
		dexSection.WriteString("\n")
	}

	dexSection.WriteString(dexIgnoreEnd)
	dexSection.WriteString("\n")
	return dexSection.String()
}

// updateDexSection replaces or appends the dex section in gitignore content.
func updateDexSection(existingContent, dexSection string) string {
	// Check if there's an existing dex section
	startIdx := strings.Index(existingContent, dexIgnoreStart)
	endIdx := strings.Index(existingContent, dexIgnoreEnd)

	if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
		// Replace existing section
		endIdx += len(dexIgnoreEnd)
		// Skip any trailing newline
		if endIdx < len(existingContent) && existingContent[endIdx] == '\n' {
			endIdx++
		}
		return existingContent[:startIdx] + dexSection + existingContent[endIdx:]
	}

	// Append new section
	if existingContent != "" && !strings.HasSuffix(existingContent, "\n") {
		existingContent += "\n"
	}
	if existingContent != "" {
		existingContent += "\n"
	}
	return existingContent + dexSection
}
