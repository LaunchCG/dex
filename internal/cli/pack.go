package cli

import (
	"fmt"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/dex-tools/dex/internal/packer"
)

var packCmd = &cobra.Command{
	Use:   "pack [directory]",
	Short: "Create a distributable tarball from a plugin directory",
	Long: `Create a distributable tarball from a plugin directory.

The directory must contain a valid package.hcl file. The tarball will be
created with the naming convention {name}-{version}.tar.gz.

The tarball includes a single top-level directory ({name}-{version}/)
containing all plugin files except excluded patterns like .git,
node_modules, __pycache__, .env, etc.

Examples:
  # Pack the current directory
  dex pack

  # Pack a specific directory
  dex pack /path/to/plugin

  # Pack with custom output path
  dex pack -o my-plugin.tar.gz

  # Pack a specific directory with custom output
  dex pack /path/to/plugin -o /tmp/my-plugin.tar.gz`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPack,
}

func init() {
	rootCmd.AddCommand(packCmd)
	packCmd.Flags().StringP("output", "o", "", "Output file path (default: {name}-{version}.tar.gz)")
}

func runPack(cmd *cobra.Command, args []string) error {
	// Get the directory to pack
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}

	// Get output flag
	output, _ := cmd.Flags().GetString("output")

	// Colors for output
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	fmt.Printf("%s Packing plugin from %s\n", cyan("→"), dir)

	// Create packer
	p, err := packer.New(dir)
	if err != nil {
		return err
	}

	fmt.Printf("  Package: %s@%s\n", p.Name(), p.Version())

	// Pack
	result, err := p.Pack(output)
	if err != nil {
		return err
	}

	fmt.Printf("  Output: %s\n", result.Path)
	fmt.Printf("  Size: %s\n", humanize.Bytes(uint64(result.Size)))
	fmt.Printf("  Integrity: %s\n", result.Integrity)
	fmt.Printf("%s Pack complete\n", green("✓"))

	return nil
}
