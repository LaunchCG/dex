// Package cli implements the command-line interface for dex.
package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/launchcg/dex/internal/manifest"
)

var updateIgnoreCmd = &cobra.Command{
	Use:   "update-ignore",
	Short: "Update .gitignore with dex-managed files",
	Long:  "Add dex-managed files to .gitignore to prevent them from being committed.",
	RunE:  runUpdateIgnore,
}

func init() {
	rootCmd.AddCommand(updateIgnoreCmd)
	updateIgnoreCmd.Flags().Bool("print", false, "Print without modifying files")
	updateIgnoreCmd.Flags().StringP("path", "p", ".", "Project directory")
}

const (
	dexIgnoreStart = "# --- dex managed (do not edit) ---"
	dexIgnoreEnd   = "# --- end dex managed ---"
)

func runUpdateIgnore(cmd *cobra.Command, args []string) error {
	// Get flags
	printOnly, _ := cmd.Flags().GetBool("print")
	projectPath, _ := cmd.Flags().GetString("path")

	// Resolve absolute path
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	if printOnly {
		return printIgnoreSection(absPath)
	}

	return updateIgnoreForProject(absPath)
}

// printIgnoreSection prints the dex gitignore section without modifying files.
func printIgnoreSection(absPath string) error {
	mf, err := manifest.Load(absPath)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	dexSection := buildDexIgnoreSection(mf.AllFiles())
	fmt.Println(dexSection)
	return nil
}

// updateIgnoreForProject updates the .gitignore in the given project directory
// with dex-managed files. This is used by both the standalone update-ignore
// command and the sync command.
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

	// Read existing .gitignore
	gitignorePath := filepath.Join(absPath, ".gitignore")
	existingContent := ""

	if data, err := os.ReadFile(gitignorePath); err == nil {
		existingContent = string(data)
	}

	// Update the dex section
	newContent := updateDexSection(existingContent, dexSection)

	// Write the updated .gitignore
	if err := os.WriteFile(gitignorePath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write .gitignore: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s Updated .gitignore with %d managed file(s)\n", green("✓"), len(allFiles)+2)

	return nil
}

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
