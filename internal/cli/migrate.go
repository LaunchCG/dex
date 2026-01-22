// Package cli implements the command-line interface for dex.
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Convert package.json to dex.hcl",
	Long:  "Migrate from a JSON-based configuration to HCL format.",
	RunE:  runMigrate,
}

func init() {
	rootCmd.AddCommand(migrateCmd)
	migrateCmd.Flags().StringP("input", "i", "package.json", "Input file")
	migrateCmd.Flags().StringP("output", "o", "dex.hcl", "Output file")
	migrateCmd.Flags().StringP("path", "p", ".", "Project directory")
}

// legacyConfig represents the old JSON config format.
type legacyConfig struct {
	Name     string                     `json:"name"`
	Agent    string                     `json:"agent"`
	Platform string                     `json:"platform"`
	Plugins  map[string]*legacyPlugin   `json:"plugins"`
	Registry map[string]*legacyRegistry `json:"registries"`
}

type legacyPlugin struct {
	Version  string            `json:"version"`
	Source   string            `json:"source"`
	Registry string            `json:"registry"`
	Config   map[string]string `json:"config"`
}

type legacyRegistry struct {
	URL  string `json:"url"`
	Path string `json:"path"`
}

func runMigrate(cmd *cobra.Command, args []string) error {
	// Get flags
	inputFile, _ := cmd.Flags().GetString("input")
	outputFile, _ := cmd.Flags().GetString("output")
	projectPath, _ := cmd.Flags().GetString("path")

	// Resolve absolute path
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	inputPath := filepath.Join(absPath, inputFile)
	outputPath := filepath.Join(absPath, outputFile)

	// Check if output already exists
	if _, err := os.Stat(outputPath); err == nil {
		return fmt.Errorf("output file %s already exists", outputPath)
	}

	// Read and parse input file
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", inputFile, err)
	}

	var legacy legacyConfig
	if err := json.Unmarshal(data, &legacy); err != nil {
		return fmt.Errorf("failed to parse %s: %w", inputFile, err)
	}

	// Convert to HCL
	hcl := convertToHCL(&legacy)

	// Write output file
	if err := os.WriteFile(outputPath, []byte(hcl), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", outputFile, err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s Migrated %s to %s\n", green("✓"), inputFile, outputFile)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Review the generated dex.hcl")
	fmt.Println("  2. Run 'dex install' to reinstall plugins")
	fmt.Printf("  3. Delete %s if no longer needed\n", inputFile)

	return nil
}

func convertToHCL(legacy *legacyConfig) string {
	var b strings.Builder

	// Project block
	name := legacy.Name
	if name == "" {
		name = "my-project"
	}

	agent := legacy.Agent
	if agent == "" {
		agent = legacy.Platform // Try platform as fallback
	}
	if agent == "" {
		agent = "claude-code"
	}

	b.WriteString("project {\n")
	b.WriteString(fmt.Sprintf("  name            = %q\n", name))
	b.WriteString(fmt.Sprintf("  agentic_platform = %q\n", agent))
	b.WriteString("}\n")

	// Registry blocks
	for regName, reg := range legacy.Registry {
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("registry %q {\n", regName))
		if reg.Path != "" {
			b.WriteString(fmt.Sprintf("  path = %q\n", reg.Path))
		}
		if reg.URL != "" {
			b.WriteString(fmt.Sprintf("  url = %q\n", reg.URL))
		}
		b.WriteString("}\n")
	}

	// Plugin blocks
	for pluginName, plugin := range legacy.Plugins {
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("plugin %q {\n", pluginName))

		if plugin.Source != "" {
			b.WriteString(fmt.Sprintf("  source = %q\n", plugin.Source))
		}
		if plugin.Version != "" {
			b.WriteString(fmt.Sprintf("  version = %q\n", plugin.Version))
		}
		if plugin.Registry != "" {
			b.WriteString(fmt.Sprintf("  registry = %q\n", plugin.Registry))
		}
		if len(plugin.Config) > 0 {
			b.WriteString("  config = {\n")
			for k, v := range plugin.Config {
				b.WriteString(fmt.Sprintf("    %s = %q\n", k, v))
			}
			b.WriteString("  }\n")
		}
		b.WriteString("}\n")
	}

	return b.String()
}
