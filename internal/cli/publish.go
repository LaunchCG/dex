package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/launchcg/dex/internal/publisher"
)

var publishCmd = &cobra.Command{
	Use:   "publish <tarball>",
	Short: "Publish a plugin tarball to a registry",
	Long: `Publish a plugin tarball to a registry.

The tarball must follow the naming convention {name}-{version}.tar.gz.
The registry URL determines the upload method:

  file://path     - Copy tarball and update local registry.json
  s3://bucket/... - Upload via AWS SDK
  az://account/...    - Upload via Azure SDK
  https://...     - Output manual upload instructions (read-only)

Examples:
  # Publish to a local registry
  dex publish my-plugin-1.0.0.tar.gz -r file:///path/to/registry

  # Publish to S3
  dex publish my-plugin-1.0.0.tar.gz -r s3://my-bucket/registry

  # Publish to Azure Blob Storage
  dex publish my-plugin-1.0.0.tar.gz -r az://myaccount/mycontainer/registry

  # Get manual instructions for HTTPS registry
  dex publish my-plugin-1.0.0.tar.gz -r https://example.com/registry`,
	Args: cobra.ExactArgs(1),
	RunE: runPublish,
}

func init() {
	rootCmd.AddCommand(publishCmd)
	publishCmd.Flags().StringP("registry", "r", "", "Registry URL (required)")
	publishCmd.MarkFlagRequired("registry")
}

func runPublish(cmd *cobra.Command, args []string) error {
	tarballPath := args[0]
	registryURL, _ := cmd.Flags().GetString("registry")

	// Colors for output
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	fmt.Printf("%s Publishing %s to %s\n", cyan("→"), tarballPath, registryURL)

	// Create publisher
	pub, err := publisher.New(registryURL)
	if err != nil {
		return err
	}

	// Publish
	result, err := pub.Publish(tarballPath)
	if err != nil {
		return err
	}

	// Display results
	fmt.Printf("  Package: %s@%s\n", result.Name, result.Version)
	fmt.Printf("  URL: %s\n", result.URL)
	fmt.Printf("  Integrity: %s\n", result.Integrity)

	// If manual instructions are provided (HTTPS), display them
	if result.ManualInstructions != "" {
		fmt.Printf("\n%s Manual steps required:\n\n", yellow("!"))
		fmt.Println(result.ManualInstructions)
	} else {
		fmt.Printf("%s Publish complete\n", green("✓"))
	}

	return nil
}
