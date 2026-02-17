// Package cli implements the command-line interface for dex.
package cli

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/launchcg/dex/internal/config"
	"github.com/launchcg/dex/internal/installer"
)

var installCmd = &cobra.Command{
	Use:   "install [plugins...]",
	Short: "Install plugins",
	Long:  "Install plugins from registry or direct source. Without arguments, installs all plugins from config.",
	RunE:  runInstall,
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().StringP("source", "s", "", "Install from direct source (file://, git+)")
	installCmd.Flags().StringP("registry", "r", "", "Registry to use")
	installCmd.Flags().Bool("no-save", false, "Don't save to config file (plugins are saved by default)")
	installCmd.Flags().Bool("no-lock", false, "Don't update lock file")
	installCmd.Flags().BoolP("force", "f", false, "Overwrite non-managed files")
	installCmd.Flags().StringP("path", "p", ".", "Project directory")
	installCmd.Flags().Bool("namespace", false, "Namespace resources with package name (e.g., pkg-name-resource)")
}

// parsePluginSpec parses a plugin specification in name@version format.
func parsePluginSpec(spec string) (name, version string) {
	parts := strings.SplitN(spec, "@", 2)
	name = parts[0]
	if len(parts) > 1 {
		version = parts[1]
	}
	return name, version
}

func runInstall(cmd *cobra.Command, args []string) error {
	// Get flags
	source, _ := cmd.Flags().GetString("source")
	registry, _ := cmd.Flags().GetString("registry")
	noSave, _ := cmd.Flags().GetBool("no-save")
	noLock, _ := cmd.Flags().GetBool("no-lock")
	force, _ := cmd.Flags().GetBool("force")
	namespace, _ := cmd.Flags().GetBool("namespace")
	projectPath, _ := cmd.Flags().GetString("path")

	// Create installer
	inst, err := installer.NewInstaller(projectPath)
	if err != nil {
		return fmt.Errorf("failed to initialize installer: %w", err)
	}

	// Configure installer options
	inst.WithForce(force).WithNoLock(noLock).WithNamespace(namespace)

	// Parse plugin specs from args
	var specs []installer.PluginSpec

	// If --source is provided without plugin names, install from that source directly
	if source != "" && len(args) == 0 {
		// Use a placeholder name - installer will derive from package.hcl
		specs = append(specs, installer.PluginSpec{
			Name:   "", // Will be derived from package.hcl
			Source: source,
		})
	} else {
		for _, arg := range args {
			name, version := parsePluginSpec(arg)
			spec := installer.PluginSpec{
				Name:     name,
				Version:  version,
				Source:   source,
				Registry: registry,
			}
			specs = append(specs, spec)
		}
	}

	// Install plugins
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	if len(specs) == 0 {
		fmt.Println(cyan("Installing all plugins from config..."))
	} else if source != "" && len(args) == 0 {
		fmt.Printf("%s Installing from source: %s\n", cyan("→"), source)
	} else {
		for _, spec := range specs {
			if spec.Version != "" {
				fmt.Printf("%s Installing %s@%s\n", cyan("→"), spec.Name, spec.Version)
			} else {
				fmt.Printf("%s Installing %s\n", cyan("→"), spec.Name)
			}
		}
	}

	installed, err := inst.Install(specs)
	if err != nil {
		return err
	}

	// Save to config by default when installing specific plugins (not "install all")
	if !noSave && len(specs) > 0 && len(installed) > 0 {
		for idx, plugin := range installed {
			pluginSource := plugin.Source
			pluginRegistry := plugin.Registry

			// If neither source nor registry came from the installer,
			// try to get them from the original spec
			if pluginSource == "" && pluginRegistry == "" && idx < len(specs) {
				pluginSource = specs[idx].Source
				pluginRegistry = specs[idx].Registry
			}

			if pluginSource != "" || pluginRegistry != "" {
				if err := config.AddPluginToConfig(projectPath, plugin.Name, pluginSource, pluginRegistry, ""); err != nil {
					fmt.Printf("%s Failed to save plugin %s to config: %v\n", color.YellowString("⚠"), plugin.Name, err)
				} else {
					fmt.Printf("  %s Saved %s to dex.hcl\n", green("✓"), plugin.Name)
				}
			} else {
				fmt.Printf("%s Cannot save plugin %s: no source or registry information available\n", color.YellowString("⚠"), plugin.Name)
			}
		}
	}

	fmt.Printf("%s Installation complete\n", green("✓"))
	return nil
}
