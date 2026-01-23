// Package cli implements the command-line interface for dex.
package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/dex-tools/dex/internal/installer"
)

var updateCmd = &cobra.Command{
	Use:   "update [plugins...]",
	Short: "Update plugins to newer versions",
	Long:  "Update plugins to newer versions respecting version constraints. Without arguments, updates all plugins.",
	RunE:  runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.Flags().BoolP("dry-run", "n", false, "Show what would be updated without making changes")
	updateCmd.Flags().StringP("path", "p", ".", "Project directory")
}

func runUpdate(cmd *cobra.Command, args []string) error {
	// Get flags
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	projectPath, _ := cmd.Flags().GetString("path")

	// Create installer
	inst, err := installer.NewInstaller(projectPath)
	if err != nil {
		return fmt.Errorf("failed to initialize installer: %w", err)
	}

	// Colors for output
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	if dryRun {
		fmt.Println(cyan("Checking for updates (dry-run)..."))
	} else if len(args) == 0 {
		fmt.Println(cyan("Updating all plugins..."))
	} else {
		for _, name := range args {
			fmt.Printf("%s Checking %s for updates\n", cyan("→"), name)
		}
	}

	// Run update
	results, err := inst.Update(args, dryRun)
	if err != nil {
		return err
	}

	// Report results
	updatedCount := 0
	for _, result := range results {
		if result.Skipped {
			fmt.Printf("  %s %s: %s\n", yellow("-"), result.Name, result.Reason)
		} else {
			if dryRun {
				fmt.Printf("  %s %s: %s → %s\n", cyan("~"), result.Name, result.OldVersion, result.NewVersion)
			} else {
				fmt.Printf("  %s Updated %s: %s → %s\n", green("✓"), result.Name, result.OldVersion, result.NewVersion)
			}
			updatedCount++
		}
	}

	// Summary
	if dryRun {
		if updatedCount == 0 {
			fmt.Println(green("All plugins are up to date"))
		} else {
			fmt.Printf("%s %d plugin(s) would be updated\n", cyan("→"), updatedCount)
		}
	} else {
		if updatedCount == 0 {
			fmt.Println(green("All plugins are up to date"))
		} else {
			fmt.Printf("%s Updated %d plugin(s)\n", green("✓"), updatedCount)
		}
	}

	return nil
}
