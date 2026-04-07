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
	Use:   "info <package>",
	Short: "Show package information",
	Long:  "Display detailed information about an installed package.",
	Args:  cobra.ExactArgs(1),
	RunE:  runInfo,
}

func init() {
	rootCmd.AddCommand(infoCmd)
	infoCmd.Flags().StringP("path", "p", ".", "Project directory")
}

func runInfo(cmd *cobra.Command, args []string) error {
	pkgName := args[0]

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

	// Get locked package info
	locked := lf.Get(pkgName)
	if locked == nil {
		return fmt.Errorf("package %q is not installed", pkgName)
	}

	// Get package manifest
	pkgManifest := mf.GetPackage(pkgName)

	// Print package info
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()

	fmt.Printf("%s\n\n", bold(pkgName))

	fmt.Printf("  %s: %s\n", cyan("Version"), green(locked.Version))
	fmt.Printf("  %s: %s\n", cyan("Resolved"), locked.Resolved)
	if locked.Integrity != "" {
		fmt.Printf("  %s: %s\n", cyan("Integrity"), locked.Integrity)
	}

	if pkgManifest != nil {
		fmt.Println()
		fmt.Printf("  %s:\n", cyan("Files"))
		if len(pkgManifest.Files) == 0 {
			fmt.Println("    (none)")
		} else {
			for _, file := range pkgManifest.Files {
				fmt.Printf("    - %s\n", file)
			}
		}

		if len(pkgManifest.Directories) > 0 {
			fmt.Println()
			fmt.Printf("  %s:\n", cyan("Directories"))
			for _, dir := range pkgManifest.Directories {
				fmt.Printf("    - %s\n", dir)
			}
		}

		if len(pkgManifest.MCPServers) > 0 {
			fmt.Println()
			fmt.Printf("  %s:\n", cyan("MCP Servers"))
			for _, server := range pkgManifest.MCPServers {
				fmt.Printf("    - %s\n", server)
			}
		}

		if pkgManifest.HasAgentContent {
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
