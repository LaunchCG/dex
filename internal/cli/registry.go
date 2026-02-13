// Package cli implements the command-line interface for dex.
package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/launchcg/dex/internal/config"
	"github.com/launchcg/dex/internal/registry"
)

var registryCmd = &cobra.Command{
	Use:   "registry",
	Short: "Manage registries",
	Long:  "Commands for managing plugin registries in your dex project.",
}

var registryAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a registry to the project config",
	Long:  "Validates the registry is reachable, then adds a registry block to dex.hcl.",
	Args:  cobra.ExactArgs(1),
	RunE:  runRegistryAdd,
}

func init() {
	rootCmd.AddCommand(registryCmd)
	registryCmd.AddCommand(registryAddCmd)
	registryAddCmd.Flags().StringP("url", "u", "", "Remote registry URL")
	registryAddCmd.Flags().StringP("local", "l", "", "Local filesystem path for the registry")
	registryAddCmd.Flags().StringP("path", "p", ".", "Project directory")
}

func runRegistryAdd(cmd *cobra.Command, args []string) error {
	name := args[0]
	url, _ := cmd.Flags().GetString("url")
	local, _ := cmd.Flags().GetString("local")
	projectPath, _ := cmd.Flags().GetString("path")

	// Validate exactly one of url or local is set
	if url == "" && local == "" {
		return fmt.Errorf("exactly one of --url or --local must be provided")
	}
	if url != "" && local != "" {
		return fmt.Errorf("cannot specify both --url and --local")
	}

	// Build registry source string for validation
	var source string
	if url != "" {
		source = url
	} else {
		source = "file:" + local
	}

	// Validate the registry is reachable
	cyan := color.New(color.FgCyan).SprintFunc()
	fmt.Printf("%s Validating registry %q...\n", cyan("~"), name)

	reg, err := registry.NewRegistry(source, registry.ModeRegistry)
	if err != nil {
		return fmt.Errorf("invalid registry source: %w", err)
	}

	_, err = reg.ListPackages()
	if err != nil {
		return fmt.Errorf("failed to reach registry: %w", err)
	}

	// Add registry to config
	if err := config.AddRegistry(projectPath, name, url, local); err != nil {
		return err
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s Added registry %q to dex.hcl\n", green("~"), name)
	return nil
}
