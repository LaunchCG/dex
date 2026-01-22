// Package cli implements the command-line interface for dex.
package cli

import (
	"fmt"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/dex-tools/dex/internal/manifest"
)

var manifestCmd = &cobra.Command{
	Use:   "manifest [plugin]",
	Short: "Show files managed by dex",
	Long:  "Display all files tracked by dex, optionally filtered by plugin.",
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

	// If a specific plugin is requested
	if len(args) > 0 {
		pluginName := args[0]
		pluginManifest := mf.GetPlugin(pluginName)
		if pluginManifest == nil {
			return fmt.Errorf("plugin %q is not installed", pluginName)
		}

		fmt.Printf("%s %s\n\n", bold("Files managed by"), cyan(pluginName))

		if len(pluginManifest.Files) == 0 {
			fmt.Println("  (no files)")
		} else {
			for _, file := range pluginManifest.Files {
				fmt.Printf("  %s\n", file)
			}
		}

		if len(pluginManifest.Directories) > 0 {
			fmt.Printf("\n%s\n", bold("Directories:"))
			for _, dir := range pluginManifest.Directories {
				fmt.Printf("  %s\n", dir)
			}
		}

		if len(pluginManifest.MCPServers) > 0 {
			fmt.Printf("\n%s\n", bold("MCP Servers:"))
			for _, server := range pluginManifest.MCPServers {
				fmt.Printf("  %s\n", server)
			}
		}

		return nil
	}

	// Show all files
	plugins := mf.InstalledPlugins()
	if len(plugins) == 0 {
		fmt.Println("No files are currently managed by dex.")
		return nil
	}

	fmt.Printf("%s\n\n", bold("Files managed by dex:"))

	totalFiles := 0
	for _, pluginName := range plugins {
		pluginManifest := mf.GetPlugin(pluginName)
		if pluginManifest == nil {
			continue
		}

		fmt.Printf("%s\n", cyan(pluginName))

		if len(pluginManifest.Files) == 0 {
			fmt.Printf("  %s\n", gray("(no files)"))
		} else {
			for _, file := range pluginManifest.Files {
				fmt.Printf("  %s\n", file)
				totalFiles++
			}
		}

		if len(pluginManifest.MCPServers) > 0 {
			for _, server := range pluginManifest.MCPServers {
				fmt.Printf("  %s %s\n", gray("mcp:"), server)
			}
		}

		fmt.Println()
	}

	fmt.Printf("%d file(s) managed across %d plugin(s)\n", totalFiles, len(plugins))
	return nil
}
