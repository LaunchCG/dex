// Package cli implements the command-line interface for dex.
package cli

import (
	"fmt"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/launchcg/dex/internal/manifest"
)

var manifestCmd = &cobra.Command{
	Use:   "manifest [package]",
	Short: "Show files managed by dex",
	Long:  "Display all files tracked by dex, optionally filtered by package.",
	RunE:  runManifest,
}

func init() {
	rootCmd.AddCommand(manifestCmd)
	manifestCmd.Flags().StringP("path", "p", ".", "Project directory")
}

func runManifest(cmd *cobra.Command, args []string) error {
	// Get flags
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

	cyan := color.New(color.FgCyan).SprintFunc()
	gray := color.New(color.FgHiBlack).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()

	// If a specific package is requested
	if len(args) > 0 {
		pkgName := args[0]
		pkgManifest := mf.GetPackage(pkgName)
		if pkgManifest == nil {
			return fmt.Errorf("package %q is not installed", pkgName)
		}

		fmt.Printf("%s %s\n\n", bold("Files managed by"), cyan(pkgName))

		if len(pkgManifest.Files) == 0 {
			fmt.Println("  (no files)")
		} else {
			for _, file := range pkgManifest.Files {
				fmt.Printf("  %s\n", file)
			}
		}

		if len(pkgManifest.Directories) > 0 {
			fmt.Printf("\n%s\n", bold("Directories:"))
			for _, dir := range pkgManifest.Directories {
				fmt.Printf("  %s\n", dir)
			}
		}

		if len(pkgManifest.MCPServers) > 0 {
			fmt.Printf("\n%s\n", bold("MCP Servers:"))
			for _, server := range pkgManifest.MCPServers {
				fmt.Printf("  %s\n", server)
			}
		}

		return nil
	}

	// Show all files
	packages := mf.InstalledPackages()
	if len(packages) == 0 {
		fmt.Println("No files are currently managed by dex.")
		return nil
	}

	fmt.Printf("%s\n\n", bold("Files managed by dex:"))

	totalFiles := 0
	for _, pkgName := range packages {
		pkgManifest := mf.GetPackage(pkgName)
		if pkgManifest == nil {
			continue
		}

		fmt.Printf("%s\n", cyan(pkgName))

		if len(pkgManifest.Files) == 0 {
			fmt.Printf("  %s\n", gray("(no files)"))
		} else {
			for _, file := range pkgManifest.Files {
				fmt.Printf("  %s\n", file)
				totalFiles++
			}
		}

		if len(pkgManifest.MCPServers) > 0 {
			for _, server := range pkgManifest.MCPServers {
				fmt.Printf("  %s %s\n", gray("mcp:"), server)
			}
		}

		fmt.Println()
	}

	fmt.Printf("%d file(s) managed across %d package(s)\n", totalFiles, len(packages))
	return nil
}
