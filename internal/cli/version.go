// Package cli implements the command-line interface for dex.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version information, set at build time via ldflags.
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show dex version",
	Long:  "Display the version, commit hash, and build date of dex.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("dex %s (%s) built %s\n", Version, Commit, Date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
