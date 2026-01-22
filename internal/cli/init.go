// Package cli implements the command-line interface for dex.
package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new dex project",
	Long:  "Creates a dex.hcl configuration file in the current or specified directory.",
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringP("agent", "a", "claude-code", "Target AI agent platform")
	initCmd.Flags().StringP("name", "n", "", "Project name (defaults to directory name)")
	initCmd.Flags().StringP("path", "p", ".", "Project directory")
}

func runInit(cmd *cobra.Command, args []string) error {
	// Get flags
	agent, _ := cmd.Flags().GetString("agent")
	name, _ := cmd.Flags().GetString("name")
	projectPath, _ := cmd.Flags().GetString("path")

	// Resolve absolute path
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Default name to directory name
	if name == "" {
		name = filepath.Base(absPath)
	}

	// Check if dex.hcl already exists
	configPath := filepath.Join(absPath, "dex.hcl")
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("dex.hcl already exists in %s", absPath)
	}

	// Create dex.hcl with project block
	content := fmt.Sprintf(`project {
  name            = %q
  agentic_platform = %q
}
`, name, agent)

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create dex.hcl: %w", err)
	}

	// Print success message
	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s Created dex.hcl in %s\n", green("✓"), absPath)
	fmt.Printf("  Project: %s\n", name)
	fmt.Printf("  Platform: %s\n", agent)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Add plugins to dex.hcl")
	fmt.Println("  2. Run 'dex install' to install plugins")

	return nil
}
