// Package cli implements the command-line interface for dex.
package cli

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Global flags
	verbose int
)

// rootCmd is the base command for dex
var rootCmd = &cobra.Command{
	Use:   "dex",
	Short: "A universal package manager for AI coding agents",
	Long: `Dex is a package manager for AI coding agents like Claude Code, Cursor, and others.
It allows you to install and manage packages that provide skills, commands, rules,
and other resources for your AI coding assistant.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		level := slog.LevelWarn
		switch {
		case verbose >= 3:
			level = slog.LevelDebug - 4 // trace
		case verbose >= 2:
			level = slog.LevelDebug
		case verbose >= 1:
			level = slog.LevelInfo
		}
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: level,
		})))
	},
}

func init() {
	rootCmd.PersistentFlags().CountVarP(&verbose, "verbose", "v", "Increase verbosity (-v info, -vv debug, -vvv trace)")
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

// GetRootCmd returns the root command for testing
func GetRootCmd() *cobra.Command {
	return rootCmd
}
