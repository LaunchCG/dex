package installer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/launchcg/dex/internal/adapter"
	"github.com/launchcg/dex/internal/manifest"
)

// Executor executes installation plans by creating directories, writing files,
// and merging configuration files.
type Executor struct {
	projectRoot string
	manifest    *manifest.Manifest
	force       bool
}

// NewExecutor creates a new plan executor.
func NewExecutor(projectRoot string, m *manifest.Manifest, force bool) *Executor {
	return &Executor{
		projectRoot: projectRoot,
		manifest:    m,
		force:       force,
	}
}

// Execute executes an installation plan.
// It creates directories, writes files, and merges configuration files.
func (e *Executor) Execute(plan *adapter.Plan, vars map[string]string) error {
	if plan == nil || plan.IsEmpty() {
		return nil
	}

	// Create directories first
	if err := e.createDirectories(plan.Directories); err != nil {
		return fmt.Errorf("creating directories: %w", err)
	}

	// Write files
	if err := e.writeFiles(plan.Files, vars); err != nil {
		return fmt.Errorf("writing files: %w", err)
	}

	// Determine the MCP key to use (default to "mcpServers" for backward compatibility)
	mcpKey := plan.MCPKey
	if mcpKey == "" {
		mcpKey = "mcpServers"
	}

	// Apply MCP configuration
	if len(plan.MCPEntries) > 0 {
		if err := e.applyMCPConfig(plan.MCPEntries, plan.MCPPath, plan.MCPKey); err != nil {
			return fmt.Errorf("applying MCP config: %w", err)
		}

		// Track the merged MCP config file
		mcpPath := plan.MCPPath
		if mcpPath == "" {
			mcpPath = ".mcp.json"
		}
		e.manifest.TrackMergedFile(plan.PluginName, mcpPath)

		// Track MCP servers in manifest
		for serverName := range plan.MCPEntries {
			// The MCPEntries map contains the server configs, but we need to extract
			// server names from within the mcpServers/servers key
			if serverName == mcpKey {
				if servers, ok := plan.MCPEntries[serverName].(map[string]any); ok {
					for name := range servers {
						e.manifest.TrackMCPServer(plan.PluginName, name)
					}
				}
			}
		}
	}

	// Apply settings configuration
	if len(plan.SettingsEntries) > 0 {
		if err := e.applySettingsConfig(plan.SettingsEntries, plan.SettingsPath); err != nil {
			return fmt.Errorf("applying settings config: %w", err)
		}

		// Track the merged settings file
		settingsPath := plan.SettingsPath
		if settingsPath == "" {
			settingsPath = filepath.Join(".claude", "settings.json")
		}
		e.manifest.TrackMergedFile(plan.PluginName, settingsPath)

		// Track settings values in manifest (key -> []values)
		settingsValues := make(map[string][]string)
		for key, val := range plan.SettingsEntries {
			// Convert the values to string slice
			switch v := val.(type) {
			case []string:
				settingsValues[key] = v
			case []any:
				var strs []string
				for _, item := range v {
					if s, ok := item.(string); ok {
						strs = append(strs, s)
					}
				}
				settingsValues[key] = strs
			}
		}
		e.manifest.TrackSettings(plan.PluginName, settingsValues)
	}

	// Apply agent file content
	if plan.AgentFileContent != "" {
		if err := e.applyAgentFileContent(plan.PluginName, plan.AgentFileContent, plan.AgentFilePath); err != nil {
			return fmt.Errorf("applying agent file content: %w", err)
		}

		// Track the merged agent file
		agentPath := plan.AgentFilePath
		if agentPath == "" {
			agentPath = "CLAUDE.md"
		}
		e.manifest.TrackMergedFile(plan.PluginName, agentPath)

		// Track agent content in manifest
		e.manifest.TrackAgentContent(plan.PluginName)
	}

	// Track files and directories in manifest
	filePaths := make([]string, len(plan.Files))
	for i, f := range plan.Files {
		filePaths[i] = f.Path
	}
	dirPaths := make([]string, len(plan.Directories))
	for i, d := range plan.Directories {
		dirPaths[i] = d.Path
	}
	e.manifest.Track(plan.PluginName, filePaths, dirPaths)

	return nil
}

// createDirectories creates all directories in the plan.
func (e *Executor) createDirectories(dirs []adapter.DirectoryCreate) error {
	for _, dir := range dirs {
		path := filepath.Join(e.projectRoot, dir.Path)
		if dir.Parents {
			// Create with parents (like mkdir -p)
			if err := os.MkdirAll(path, 0755); err != nil {
				return fmt.Errorf("creating directory %s: %w", dir.Path, err)
			}
		} else {
			// Create without parents (fails if parent doesn't exist)
			if err := os.Mkdir(path, 0755); err != nil {
				return fmt.Errorf("creating directory %s: %w", dir.Path, err)
			}
		}
	}
	return nil
}

// writeFiles writes all files in the plan.
func (e *Executor) writeFiles(files []adapter.FileWrite, vars map[string]string) error {
	for _, fw := range files {
		if err := e.writeFile(fw, vars); err != nil {
			return err
		}
	}
	return nil
}

// writeFile writes a single file with proper permissions and conflict handling.
func (e *Executor) writeFile(fw adapter.FileWrite, vars map[string]string) error {
	path := filepath.Join(e.projectRoot, fw.Path)

	// Check for conflicts
	if _, err := os.Stat(path); err == nil {
		// File exists - check if it's tracked by manifest
		if _, tracked := e.manifest.IsTracked(fw.Path); !tracked {
			if !e.force {
				return fmt.Errorf("file %s already exists and is not managed by dex (use --force to overwrite)", fw.Path)
			}
			// Force mode - warn but continue
			fmt.Fprintf(os.Stderr, "warning: overwriting non-managed file %s\n", fw.Path)
		}
		// If file is tracked by a plugin (reinstall case), this is OK - proceed to overwrite
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating parent directory for %s: %w", fw.Path, err)
	}

	// Process content with template variables if any
	content := fw.Content
	if len(vars) > 0 {
		content = processTemplate(content, vars)
	}

	// Determine file mode
	mode := os.FileMode(0644)
	if fw.Chmod != "" {
		parsedMode, err := parseChmod(fw.Chmod)
		if err == nil {
			mode = parsedMode
		}
	}

	// Write the file
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		return fmt.Errorf("writing file %s: %w", fw.Path, err)
	}

	return nil
}

// applyMCPConfig merges MCP config into the MCP config file.
// Uses mcpPath if provided, otherwise defaults to ".mcp.json".
// Uses mcpKey if provided, otherwise defaults to "mcpServers".
func (e *Executor) applyMCPConfig(entries map[string]any, mcpPath, mcpKey string) error {
	// Default paths
	if mcpPath == "" {
		mcpPath = ".mcp.json"
	}
	if mcpKey == "" {
		mcpKey = "mcpServers"
	}

	fullPath := filepath.Join(e.projectRoot, mcpPath)

	// Read existing config
	existing, err := ReadJSONFile(fullPath)
	if err != nil {
		return err
	}

	// Merge the new entries using the specified key
	merged := MergeMCPServersWithKey(existing, entries, mcpKey)

	// Write back
	return WriteJSONFile(fullPath, merged)
}

// applySettingsConfig merges settings into the settings file.
// Uses settingsPath if provided, otherwise defaults to ".claude/settings.json".
func (e *Executor) applySettingsConfig(entries map[string]any, settingsPath string) error {
	// Default path
	if settingsPath == "" {
		settingsPath = filepath.Join(".claude", "settings.json")
	}

	fullPath := filepath.Join(e.projectRoot, settingsPath)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}

	// Read existing config
	existing, err := ReadJSONFile(fullPath)
	if err != nil {
		return err
	}

	// Merge the new entries
	merged := MergeJSON(existing, entries)

	// Write back
	return WriteJSONFile(fullPath, merged)
}

// applyAgentFileContent merges content into the agent file.
// Uses agentPath if provided, otherwise defaults to "CLAUDE.md".
func (e *Executor) applyAgentFileContent(pluginName, content, agentPath string) error {
	// Default path
	if agentPath == "" {
		agentPath = "CLAUDE.md"
	}

	fullPath := filepath.Join(e.projectRoot, agentPath)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}

	// Read existing content
	existing := ""
	data, err := os.ReadFile(fullPath)
	if err == nil {
		existing = string(data)
	} else if !os.IsNotExist(err) {
		return err
	}

	// Merge content with markers
	merged := MergeAgentContent(existing, pluginName, content)

	// Write back
	return os.WriteFile(fullPath, []byte(merged), 0644)
}

// applyProjectAgentInstructions merges project-level instructions into the agent file.
// Project instructions appear at the top without markers, before any plugin sections.
// Uses agentPath if provided, otherwise defaults to "CLAUDE.md".
func (e *Executor) applyProjectAgentInstructions(content, agentPath string) error {
	// Default path
	if agentPath == "" {
		agentPath = "CLAUDE.md"
	}

	fullPath := filepath.Join(e.projectRoot, agentPath)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}

	// Read existing content
	existing := ""
	data, err := os.ReadFile(fullPath)
	if err == nil {
		existing = string(data)
	} else if !os.IsNotExist(err) {
		return err
	}

	// Merge project instructions with existing plugin sections
	merged := MergeProjectAgentContent(existing, content)

	// Write back
	return os.WriteFile(fullPath, []byte(merged), 0644)
}

// processTemplate performs simple variable substitution on content.
// Variables are in the format ${var_name} or {{var_name}}.
func processTemplate(content string, vars map[string]string) string {
	result := content
	for k, v := range vars {
		result = strings.ReplaceAll(result, "${"+k+"}", v)
		result = strings.ReplaceAll(result, "{{"+k+"}}", v)
	}
	return result
}

// parseChmod parses a chmod string like "755" into a FileMode.
func parseChmod(s string) (os.FileMode, error) {
	val, err := strconv.ParseUint(s, 8, 32)
	if err != nil {
		return 0, err
	}
	return os.FileMode(val), nil
}

// ReadJSONFile reads and parses a JSON file.
// Returns empty map if file doesn't exist.
func ReadJSONFile(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]any), nil
		}
		return nil, err
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing JSON file %s: %w", path, err)
	}

	if result == nil {
		result = make(map[string]any)
	}

	return result, nil
}

// WriteJSONFile writes a map as formatted JSON.
func WriteJSONFile(path string, data map[string]any) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	// Add trailing newline
	content = append(content, '\n')

	return os.WriteFile(path, content, 0644)
}
