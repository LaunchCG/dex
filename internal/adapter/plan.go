package adapter

// Plan represents an installation plan for a resource.
// Plans describe what files to write and configurations to merge
// without actually performing the installation.
type Plan struct {
	// Directories to create before writing files
	Directories []string

	// Files to write during installation
	Files []FileWrite

	// MCPEntries to merge into the MCP config file
	MCPEntries map[string]any

	// MCPPath overrides the default MCP config file path (default: ".mcp.json")
	MCPPath string

	// MCPKey overrides the key used in the MCP config (default: "mcpServers")
	MCPKey string

	// SettingsEntries to merge into .claude/settings.json
	SettingsEntries map[string]any

	// SettingsPath overrides the default settings file path (default: ".claude/settings.json")
	SettingsPath string

	// AgentFileContent to merge into the agent file (e.g., CLAUDE.md, .github/copilot-instructions.md)
	AgentFileContent string

	// AgentFilePath overrides the default agent file path (default: "CLAUDE.md")
	AgentFilePath string

	// PluginName that owns these files (for manifest tracking)
	PluginName string
}

// FileWrite represents a file to write during installation.
type FileWrite struct {
	// Path is relative to the project root
	Path string

	// Content is the file content to write
	Content string

	// Chmod specifies file permissions (e.g., "755", "644")
	// Empty string means use default permissions
	Chmod string
}

// NewPlan creates a new empty installation plan for the given plugin.
func NewPlan(pluginName string) *Plan {
	return &Plan{
		PluginName:      pluginName,
		MCPEntries:      make(map[string]any),
		SettingsEntries: make(map[string]any),
	}
}

// AddDirectory adds a directory to create.
func (p *Plan) AddDirectory(dir string) {
	p.Directories = append(p.Directories, dir)
}

// AddFile adds a file to write.
func (p *Plan) AddFile(path, content, chmod string) {
	p.Files = append(p.Files, FileWrite{
		Path:    path,
		Content: content,
		Chmod:   chmod,
	})
}

// MergePlans combines multiple plans into one.
// This is useful when a plugin has multiple resources to install.
func MergePlans(plans ...*Plan) *Plan {
	if len(plans) == 0 {
		return NewPlan("")
	}

	// Use the plugin name from the first plan
	merged := NewPlan(plans[0].PluginName)

	// Track directories to avoid duplicates
	dirsSeen := make(map[string]bool)

	for _, plan := range plans {
		if plan == nil {
			continue
		}

		// Merge directories (deduplicate)
		for _, dir := range plan.Directories {
			if !dirsSeen[dir] {
				dirsSeen[dir] = true
				merged.Directories = append(merged.Directories, dir)
			}
		}

		// Merge files
		merged.Files = append(merged.Files, plan.Files...)

		// Merge MCP entries (deep merge for mcpServers/servers)
		mergeMCPEntries(merged.MCPEntries, plan.MCPEntries)

		// Use MCP path/key from first plan that specifies them
		if merged.MCPPath == "" && plan.MCPPath != "" {
			merged.MCPPath = plan.MCPPath
		}
		if merged.MCPKey == "" && plan.MCPKey != "" {
			merged.MCPKey = plan.MCPKey
		}

		// Merge settings entries (deep merge for arrays)
		mergeSettingsEntries(merged.SettingsEntries, plan.SettingsEntries)

		// Use settings path from first plan that specifies it
		if merged.SettingsPath == "" && plan.SettingsPath != "" {
			merged.SettingsPath = plan.SettingsPath
		}

		// Concatenate agent file content
		if plan.AgentFileContent != "" {
			if merged.AgentFileContent != "" {
				merged.AgentFileContent += "\n"
			}
			merged.AgentFileContent += plan.AgentFileContent
		}

		// Use agent file path from first plan that specifies it
		if merged.AgentFilePath == "" && plan.AgentFilePath != "" {
			merged.AgentFilePath = plan.AgentFilePath
		}
	}

	return merged
}

// IsEmpty returns true if the plan has nothing to do.
func (p *Plan) IsEmpty() bool {
	return len(p.Directories) == 0 &&
		len(p.Files) == 0 &&
		len(p.MCPEntries) == 0 &&
		len(p.SettingsEntries) == 0 &&
		p.AgentFileContent == ""
}

// FilePaths returns all file paths in this plan.
// Useful for manifest tracking.
func (p *Plan) FilePaths() []string {
	paths := make([]string, len(p.Files))
	for i, f := range p.Files {
		paths[i] = f.Path
	}
	return paths
}

// mergeMCPEntries deep merges MCP entries, properly combining mcpServers maps.
func mergeMCPEntries(dst, src map[string]any) {
	for k, v := range src {
		if k == "mcpServers" {
			// Deep merge the mcpServers map
			srcServers, srcOK := v.(map[string]any)
			dstServers, dstOK := dst[k].(map[string]any)
			if srcOK {
				if !dstOK {
					dstServers = make(map[string]any)
				}
				for name, config := range srcServers {
					dstServers[name] = config
				}
				dst[k] = dstServers
			}
		} else {
			dst[k] = v
		}
	}
}

// mergeSettingsEntries deep merges settings entries, appending arrays.
func mergeSettingsEntries(dst, src map[string]any) {
	for k, v := range src {
		switch srcVal := v.(type) {
		case []string:
			// Append arrays
			if dstVal, ok := dst[k].([]string); ok {
				dst[k] = append(dstVal, srcVal...)
			} else {
				dst[k] = srcVal
			}
		case []any:
			// Append arrays
			if dstVal, ok := dst[k].([]any); ok {
				dst[k] = append(dstVal, srcVal...)
			} else {
				dst[k] = srcVal
			}
		default:
			dst[k] = v
		}
	}
}
