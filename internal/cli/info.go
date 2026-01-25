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

var infoCmd = &cobra.Command{
	Use:   "info <plugin>",
	Short: "Show plugin information",
	Long:  "Display detailed information about an installed plugin.",
	Args:  cobra.ExactArgs(1),
	RunE:  runInfo,
}

func init() {
	rootCmd.AddCommand(infoCmd)
	infoCmd.Flags().StringP("path", "p", ".", "Project directory")
}

func runInfo(cmd *cobra.Command, args []string) error {
	pluginName := args[0]

	// Get flags
	projectPath, _ := cmd.Flags().GetString("path")

	// Resolve absolute path
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Load lock file for version info
	lf, err := lockfile.Load(absPath)
	if err != nil {
		return fmt.Errorf("failed to load lock file: %w", err)
	}

	// Load manifest for file info
	mf, err := manifest.Load(absPath)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	// Get locked plugin info
	locked := lf.Get(pluginName)
	if locked == nil {
		return fmt.Errorf("plugin %q is not installed", pluginName)
	}

	// Get plugin manifest
	pluginManifest := mf.GetPlugin(pluginName)

	// Print plugin info
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()

	fmt.Printf("%s\n\n", bold(pluginName))

	fmt.Printf("  %s: %s\n", cyan("Version"), green(locked.Version))
	fmt.Printf("  %s: %s\n", cyan("Resolved"), locked.Resolved)
	if locked.Integrity != "" {
		fmt.Printf("  %s: %s\n", cyan("Integrity"), locked.Integrity)
	}

	if pluginManifest != nil {
		fmt.Println()
		fmt.Printf("  %s:\n", cyan("Files"))
		if len(pluginManifest.Files) == 0 {
			fmt.Println("    (none)")
		} else {
			for _, file := range pluginManifest.Files {
				fmt.Printf("    - %s\n", file)
			}
		}

		if len(pluginManifest.Directories) > 0 {
			fmt.Println()
			fmt.Printf("  %s:\n", cyan("Directories"))
			for _, dir := range pluginManifest.Directories {
				fmt.Printf("    - %s\n", dir)
			}
		}

		if len(pluginManifest.MCPServers) > 0 {
			fmt.Println()
			fmt.Printf("  %s:\n", cyan("MCP Servers"))
			for _, server := range pluginManifest.MCPServers {
				fmt.Printf("    - %s\n", server)
			}
		}

		if pluginManifest.HasAgentContent {
			fmt.Println()
			fmt.Printf("  %s: yes\n", cyan("Agent Content"))
		}
	}

	// Show dependencies if any
	if len(locked.Dependencies) > 0 {
		fmt.Println()
		fmt.Printf("  %s:\n", cyan("Dependencies"))
		for dep, version := range locked.Dependencies {
			fmt.Printf("    - %s@%s\n", dep, version)
		}
	}

	return nil
}
