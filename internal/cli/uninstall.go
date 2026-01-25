// Package cli implements the command-line interface for dex.
package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/launchcg/dex/internal/installer"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall <plugins...>",
	Short: "Remove installed plugins",
	Long:  "Remove installed plugins and their managed files.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runUninstall,
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
	uninstallCmd.Flags().BoolP("remove", "r", false, "Also remove from config file")
	uninstallCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompts")
	uninstallCmd.Flags().StringP("path", "p", ".", "Project directory")
}

func runUninstall(cmd *cobra.Command, args []string) error {
	// Get flags
	removeFromConfig, _ := cmd.Flags().GetBool("remove")
	yes, _ := cmd.Flags().GetBool("yes")
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
	red := color.New(color.FgRed).SprintFunc()

	// Check for packages that depend on the packages being uninstalled (transitive)
	allToUninstall := make([]string, 0, len(args))
	allToUninstall = append(allToUninstall, args...)
	hasDependents := false

	// Use a queue to find all transitive dependents
	queue := make([]string, len(args))
	copy(queue, args)
	checked := make(map[string]bool)

	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]

		if checked[name] {
			continue
		}
		checked[name] = true

		dependents := inst.FindDependents(name)
		if len(dependents) > 0 {
			hasDependents = true
			fmt.Printf("%s The following packages depend on %s:\n", yellow("⚠"), name)
			for _, dep := range dependents {
				fmt.Printf("    - %s\n", dep)
				// Add dependents to the list (they will also be uninstalled)
				if !contains(allToUninstall, dep) {
					allToUninstall = append(allToUninstall, dep)
					queue = append(queue, dep) // Check this package's dependents too
				}
			}
		}
	}

	// Deduplicate the list
	allToUninstall = unique(allToUninstall)

	// If there are dependents, require confirmation
	if hasDependents && !yes {
		fmt.Printf("\n%s This will uninstall %d package(s):\n", red("!"), len(allToUninstall))
		for _, name := range allToUninstall {
			fmt.Printf("    - %s\n", name)
		}
		fmt.Print("\nContinue? [y/N] ")

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	// Uninstall plugins
	for _, name := range allToUninstall {
		fmt.Printf("%s Uninstalling %s\n", cyan("→"), name)
	}

	if err := inst.Uninstall(allToUninstall, removeFromConfig); err != nil {
		return err
	}

	if removeFromConfig {
		// Intentionally not implemented: uninstall removes files but doesn't modify dex.hcl.
		// Users can manually edit dex.hcl if they want to remove the plugin permanently.
		fmt.Println(cyan("Note: --remove flag is reserved for future use"))
	}

	fmt.Printf("%s Uninstallation complete\n", green("✓"))
	return nil
}

// contains checks if a string is in a slice.
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// unique returns a deduplicated copy of the slice, preserving order.
func unique(slice []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
