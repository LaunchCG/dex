// Package cli implements the command-line interface for dex.
package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/launchcg/dex/internal/config"
	"github.com/launchcg/dex/internal/installer"
)

var syncCmd = &cobra.Command{
	Use:   "sync [plugins...]",
	Short: "Synchronize plugins to match config",
	Long: `Synchronize plugins to match dex.hcl configuration.

Without arguments, syncs all plugins:
  - Installs plugins in config but not yet installed
  - Updates plugins that have newer compatible versions
  - Prunes plugins installed but no longer in config

With arguments, installs or updates specific plugins (like install).`,
	RunE: runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.Flags().StringP("source", "s", "", "Install from direct source (file://, git+)")
	syncCmd.Flags().StringP("registry", "r", "", "Registry to use")
	syncCmd.Flags().Bool("no-save", false, "Don't save to config file (plugins are saved by default)")
	syncCmd.Flags().Bool("no-lock", false, "Don't update lock file")
	syncCmd.Flags().BoolP("force", "f", false, "Overwrite non-managed files")
	syncCmd.Flags().StringP("path", "p", ".", "Project directory")
	syncCmd.Flags().Bool("namespace", false, "Namespace resources with package name (e.g., pkg-name-resource)")
	syncCmd.Flags().BoolP("dry-run", "n", false, "Show what would change without making changes")
	syncCmd.Flags().Bool("update-ignore", false, "Update .gitignore with dex-managed files")
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

func runSync(cmd *cobra.Command, args []string) error {
	// Get flags
	source, _ := cmd.Flags().GetString("source")
	registry, _ := cmd.Flags().GetString("registry")
	noSave, _ := cmd.Flags().GetBool("no-save")
	noLock, _ := cmd.Flags().GetBool("no-lock")
	force, _ := cmd.Flags().GetBool("force")
	namespace, _ := cmd.Flags().GetBool("namespace")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	projectPath, _ := cmd.Flags().GetString("path")

	// Create installer
	inst, err := installer.NewInstaller(projectPath)
	if err != nil {
		return fmt.Errorf("failed to initialize installer: %w", err)
	}

	// Configure installer options
	inst.WithForce(force).WithNoLock(noLock).WithNamespace(namespace)

	updateIgnore, _ := cmd.Flags().GetBool("update-ignore")

	// If args or --source provided, explicit install mode
	var syncErr error
	if len(args) > 0 || source != "" {
		syncErr = runSyncExplicit(cmd, inst, args, source, registry, noSave, projectPath)
	} else {
		// No args: full sync mode
		syncErr = runSyncAll(inst, dryRun)
	}

	if syncErr != nil {
		return syncErr
	}

	// After a successful non-dry-run sync, optionally update .gitignore
	if !dryRun {
		shouldUpdateIgnore := updateIgnore
		if !shouldUpdateIgnore {
			// Check config setting
			if cfg, err := config.LoadProject(projectPath); err == nil {
				shouldUpdateIgnore = cfg.Project.UpdateGitignore
			}
		}
		if shouldUpdateIgnore {
			absPath, err := filepath.Abs(projectPath)
			if err == nil {
				if err := updateIgnoreForProject(absPath); err != nil {
					fmt.Printf("%s Failed to update .gitignore: %v\n", color.YellowString("⚠"), err)
				}
			}
		}
	}

	return nil
}

func runSyncExplicit(cmd *cobra.Command, inst *installer.Installer, args []string, source, registry string, noSave bool, projectPath string) error {
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	// Parse plugin specs from args
	var specs []installer.PluginSpec

	if source != "" && len(args) == 0 {
		// Direct source install without plugin name
		specs = append(specs, installer.PluginSpec{
			Name:   "",
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

	// Print what we're doing
	if source != "" && len(args) == 0 {
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

	// Save to config by default when installing specific plugins (not "sync all")
	if !noSave && len(specs) > 0 && len(installed) > 0 {
		for idx, plugin := range installed {
			pluginSource := plugin.Source
			pluginRegistry := plugin.Registry

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

func runSyncAll(inst *installer.Installer, dryRun bool) error {
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	if dryRun {
		fmt.Println(cyan("Checking sync status (dry-run)..."))
	} else {
		fmt.Println(cyan("Syncing plugins..."))
	}

	results, err := inst.Sync(dryRun)
	if err != nil {
		return err
	}

	// Print results
	var installed, updated, upToDate, pruned int
	for _, r := range results {
		switch r.Action {
		case installer.SyncInstalled:
			installed++
			if dryRun {
				fmt.Printf("  %s %s@%s           would install\n", green("+"), r.Name, r.NewVersion)
			} else {
				fmt.Printf("  %s %s@%s           installed\n", green("+"), r.Name, r.NewVersion)
			}
		case installer.SyncUpdated:
			updated++
			if dryRun {
				fmt.Printf("  %s %s  %s → %s  would update\n", yellow("~"), r.Name, r.OldVersion, r.NewVersion)
			} else {
				fmt.Printf("  %s %s  %s → %s  updated\n", yellow("~"), r.Name, r.OldVersion, r.NewVersion)
			}
		case installer.SyncUpToDate:
			upToDate++
			fmt.Printf("  %s %s@%s           up to date\n", cyan("="), r.Name, r.NewVersion)
		case installer.SyncPruned:
			pruned++
			if dryRun {
				fmt.Printf("  %s %s@%s           would prune\n", color.RedString("-"), r.Name, r.OldVersion)
			} else {
				fmt.Printf("  %s %s@%s           pruned\n", color.RedString("-"), r.Name, r.OldVersion)
			}
		}
	}

	// Summary
	fmt.Println()
	if dryRun {
		if installed == 0 && updated == 0 && pruned == 0 {
			fmt.Println(green("Everything is up to date"))
		} else {
			parts := []string{}
			if installed > 0 {
				parts = append(parts, fmt.Sprintf("%d to install", installed))
			}
			if updated > 0 {
				parts = append(parts, fmt.Sprintf("%d to update", updated))
			}
			if pruned > 0 {
				parts = append(parts, fmt.Sprintf("%d to prune", pruned))
			}
			fmt.Println(strings.Join(parts, ", "))
		}
	} else {
		parts := []string{}
		if installed > 0 {
			parts = append(parts, fmt.Sprintf("%d installed", installed))
		}
		if updated > 0 {
			parts = append(parts, fmt.Sprintf("%d updated", updated))
		}
		if upToDate > 0 {
			parts = append(parts, fmt.Sprintf("%d up to date", upToDate))
		}
		if pruned > 0 {
			parts = append(parts, fmt.Sprintf("%d pruned", pruned))
		}
		fmt.Printf("%s Sync complete: %s\n", green("✓"), strings.Join(parts, ", "))
	}

	return nil
}
