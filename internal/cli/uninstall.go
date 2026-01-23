// Package cli implements the command-line interface for dex.
package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/dex-tools/dex/internal/installer"
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
	uninstallCmd.Flags().StringP("path", "p", ".", "Project directory")
}

func runUninstall(cmd *cobra.Command, args []string) error {
	// Get flags
	removeFromConfig, _ := cmd.Flags().GetBool("remove")
	projectPath, _ := cmd.Flags().GetString("path")

	// Create installer
	inst, err := installer.NewInstaller(projectPath)
	if err != nil {
		return fmt.Errorf("failed to initialize installer: %w", err)
	}

	// Uninstall plugins
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	for _, name := range args {
		fmt.Printf("%s Uninstalling %s\n", cyan("→"), name)
	}

	if err := inst.Uninstall(args, removeFromConfig); err != nil {
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
