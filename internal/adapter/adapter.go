// Package adapter provides platform-specific adapters for installing dex resources.
// Each adapter knows how to install skills, commands, rules, and other resources
// for a specific AI agent platform (e.g., Claude Code, Cursor).
package adapter

import (
	"fmt"
	"sort"
	"sync"

	"github.com/dex-tools/dex/internal/config"
	"github.com/dex-tools/dex/internal/resource"
)

// Adapter defines the interface for platform-specific adapters.
// Each adapter handles installation of resources for a specific AI agent platform.
type Adapter interface {
	// Name returns the adapter name (e.g., "claude-code")
	Name() string

	// BaseDir returns the base directory for this adapter (e.g., ".claude")
	BaseDir(root string) string

	// SkillsDir returns the directory for skills
	SkillsDir(root string) string

	// CommandsDir returns the directory for commands
	CommandsDir(root string) string

	// SubagentsDir returns the directory for subagents (agents)
	SubagentsDir(root string) string

	// RulesDir returns the directory for rules files
	RulesDir(root string) string

	// PlanInstallation creates an installation plan for a resource.
	// The plan describes what files to create and what configurations to merge.
	PlanInstallation(res resource.Resource, pkg *config.PackageConfig, pluginDir, projectRoot string) (*Plan, error)

	// GenerateFrontmatter generates YAML frontmatter for a resource.
	// Different resource types have different frontmatter fields.
	GenerateFrontmatter(res resource.Resource, pkg *config.PackageConfig) string

	// MergeMCPConfig merges MCP server configurations into existing config.
	// The existing map is in the format used by the platform's MCP config file.
	MergeMCPConfig(existing map[string]any, pluginName string, servers []*resource.ClaudeMCPServer) map[string]any

	// MergeSettingsConfig merges settings configurations into existing config.
	// Handles merging of arrays (allow, deny, ask) and maps (env).
	MergeSettingsConfig(existing map[string]any, settings *resource.ClaudeSettings) map[string]any

	// MergeAgentFile merges content into the agent file (e.g., CLAUDE.md) with markers.
	// Uses marker comments to allow multiple plugins to contribute content.
	MergeAgentFile(existing, pluginName, content string) string
}

// adapters holds registered adapters
var (
	adapters   = make(map[string]Adapter)
	adaptersMu sync.RWMutex
)

// Get returns the adapter for the given platform name.
// Returns an error if no adapter is registered for the platform.
func Get(platform string) (Adapter, error) {
	adaptersMu.RLock()
	defer adaptersMu.RUnlock()

	adapter, ok := adapters[platform]
	if !ok {
		return nil, fmt.Errorf("unknown platform adapter: %q", platform)
	}
	return adapter, nil
}

// Register registers an adapter for a platform name.
// Should be called from adapter init() functions.
func Register(platform string, adapter Adapter) {
	adaptersMu.Lock()
	defer adaptersMu.Unlock()

	adapters[platform] = adapter
}

// RegisteredAdapters returns all registered adapter names in sorted order.
func RegisteredAdapters() []string {
	adaptersMu.RLock()
	defer adaptersMu.RUnlock()

	names := make([]string, 0, len(adapters))
	for name := range adapters {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
