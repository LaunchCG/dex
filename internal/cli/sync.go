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
	Use:   "sync [packages...]",
	Short: "Synchronize packages to match config",
	Long: `Synchronize packages to match dex.hcl configuration.

Without arguments, syncs all packages:
  - Installs packages in config but not yet installed
  - Updates packages that have newer compatible versions
  - Prunes packages installed but no longer in config

With arguments, installs or updates specific packages (like install).`,
	RunE: runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.Flags().StringP("source", "s", "", "Install from direct source (file://, git+)")
	syncCmd.Flags().StringP("registry", "r", "", "Registry to use")
	syncCmd.Flags().Bool("no-save", false, "Don't save to config file (packages are saved by default)")
	syncCmd.Flags().Bool("no-lock", false, "Don't update lock file")
	syncCmd.Flags().BoolP("force", "f", false, "Overwrite non-managed files")
	syncCmd.Flags().StringP("path", "p", ".", "Project directory")
	syncCmd.Flags().Bool("namespace", false, "Namespace resources with package name (e.g., pkg-name-resource)")
	syncCmd.Flags().BoolP("dry-run", "n", false, "Show what would change without making changes")
	syncCmd.Flags().Bool("git-exclude", false, "Update .git/info/exclude to locally hide dex-managed files from git")
	syncCmd.Flags().StringP("platform", "P", "", "Override the target AI agent platform")
	syncCmd.Flags().String("profile", "", "Use a named configuration profile")
}

// parsePackageSpec parses a package specification in name@version format.
func parsePackageSpec(spec string) (name, version string) {
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
	profile, _ := cmd.Flags().GetString("profile")

	// Create installer
	inst, err := installer.NewInstaller(projectPath, profile)
	if err != nil {
		return fmt.Errorf("failed to initialize installer: %w", err)
	}

	// Configure installer options
	inst.WithForce(force).WithNoLock(noLock).WithNamespace(namespace)

	// Apply platform override if provided
	platform, _ := cmd.Flags().GetString("platform")
	if platform != "" {
		if err := inst.WithPlatform(platform); err != nil {
			return fmt.Errorf("unsupported platform %q: %w", platform, err)
		}
	}

	gitExclude, _ := cmd.Flags().GetBool("git-exclude")

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

	// After a successful non-dry-run sync, optionally update .git/info/exclude
	if !dryRun {
		shouldGitExclude := gitExclude
		if !shouldGitExclude {
			shouldGitExclude = inst.ProjectConfig().Project.GitExclude
		}
		if shouldGitExclude {
			absPath, err := filepath.Abs(projectPath)
			if err != nil {
				fmt.Printf("%s Failed to resolve path for .git/info/exclude update: %v\n", color.YellowString("⚠"), err)
			} else if err := updateIgnoreForProject(absPath); err != nil {
				fmt.Printf("%s Failed to update .git/info/exclude: %v\n", color.YellowString("⚠"), err)
			}
		}
	}

	return nil
}

func runSyncExplicit(cmd *cobra.Command, inst *installer.Installer, args []string, source, registry string, noSave bool, projectPath string) error {
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	// Parse package specs from args
	var specs []installer.PackageSpec

	if source != "" && len(args) == 0 {
		// Direct source install without package name
		specs = append(specs, installer.PackageSpec{
			Name:   "",
			Source: source,
		})
	} else {
		for _, arg := range args {
			name, version := parsePackageSpec(arg)
			spec := installer.PackageSpec{
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

	// Save to config by default when installing specific packages (not "sync all")
	if !noSave && len(specs) > 0 && len(installed) > 0 {
		for idx, pkg := range installed {
			pkgSource := pkg.Source
			pkgRegistry := pkg.Registry

			if pkgSource == "" && pkgRegistry == "" && idx < len(specs) {
				pkgSource = specs[idx].Source
				pkgRegistry = specs[idx].Registry
			}

			if pkgSource != "" || pkgRegistry != "" {
				if err := config.AddPackageToConfig(projectPath, pkg.Name, pkgSource, pkgRegistry, ""); err != nil {
					fmt.Printf("%s Failed to save package %s to config: %v\n", color.YellowString("⚠"), pkg.Name, err)
				} else {
					fmt.Printf("  %s Saved %s to dex.hcl\n", green("✓"), pkg.Name)
				}
			} else {
				fmt.Printf("%s Cannot save package %s: no source or registry information available\n", color.YellowString("⚠"), pkg.Name)
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
		fmt.Println(cyan("Syncing packages..."))
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
