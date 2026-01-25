// Package cli implements the command-line interface for dex.
package cli

import (
	"fmt"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/launchcg/dex/internal/lockfile"
	"github.com/launchcg/dex/internal/manifest"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed plugins",
	Long:  "List all installed plugins with their versions and files.",
	RunE:  runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolP("tree", "t", false, "Show dependency tree")
	listCmd.Flags().StringP("path", "p", ".", "Project directory")
}

func runList(cmd *cobra.Command, args []string) error {
	// Get flags
	showTree, _ := cmd.Flags().GetBool("tree")
	projectPath, _ := cmd.Flags().GetString("path")

	// Resolve absolute path
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Load manifest
	mf, err := manifest.Load(absPath)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	// Load lock file for version info
	lf, err := lockfile.Load(absPath)
	if err != nil {
		return fmt.Errorf("failed to load lock file: %w", err)
	}

	// Get installed plugins
	plugins := mf.InstalledPlugins()

	if len(plugins) == 0 {
		fmt.Println("No plugins installed.")
		return nil
	}

	// Print plugins
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	gray := color.New(color.FgHiBlack).SprintFunc()

	fmt.Printf("Installed plugins in %s:\n\n", absPath)

	for _, name := range plugins {
		// Get version from lock file
		version := "unknown"
		if locked := lf.Get(name); locked != nil {
			version = locked.Version
		}

		fmt.Printf("  %s %s\n", cyan(name), green("@"+version))

		if showTree {
			// Show files managed by this plugin
			pluginManifest := mf.GetPlugin(name)
			if pluginManifest != nil {
				for _, file := range pluginManifest.Files {
					fmt.Printf("    %s %s\n", gray("└──"), file)
				}
				if len(pluginManifest.MCPServers) > 0 {
					for _, server := range pluginManifest.MCPServers {
						fmt.Printf("    %s mcp: %s\n", gray("└──"), server)
					}
				}
			}
		}
	}

	fmt.Printf("\n%d plugin(s) installed\n", len(plugins))
	return nil
}
