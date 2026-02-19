package installer

import (
	"bytes"
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
// and tracking resources in the manifest.
// Shared files (MCP config, settings, agent files) are NOT written by the executor;
// they are handled by generateSharedFiles() in the installer after all plugins are processed.
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

// RemoveStaleEntries deletes dedicated files and empty directories that were
// previously tracked for a plugin but are absent from the new plan.
// Must be called before Execute so that even plugins whose new plan is empty
// (no resources match the current platform) have their old files cleaned up.
func (e *Executor) RemoveStaleEntries(pluginName string, newFilePaths, newDirPaths map[string]bool) error {
	old := e.manifest.GetPlugin(pluginName)
	if old == nil {
		return nil
	}

	// Remove stale files
	for _, oldFile := range old.Files {
		if !newFilePaths[oldFile] {
			path := filepath.Join(e.projectRoot, oldFile)
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("removing stale file %s: %w", oldFile, err)
			}
		}
	}

	// Remove stale directories (reverse order so deepest dirs are removed first)
	for j := len(old.Directories) - 1; j >= 0; j-- {
		oldDir := old.Directories[j]
		if !newDirPaths[oldDir] {
			path := filepath.Join(e.projectRoot, oldDir)
			entries, err := os.ReadDir(path)
			if err == nil && len(entries) == 0 {
				os.Remove(path)
			}
		}
	}

	return nil
}

// Execute executes an installation plan.
// It creates directories, writes dedicated files, and tracks all resources in the manifest.
// Shared files (MCP, settings, agent content) are tracked but NOT written;
// use generateSharedFiles() to write them after all plugins are processed.
func (e *Executor) Execute(plan *adapter.Plan, vars map[string]string) error {
	if plan == nil || plan.IsEmpty() {
		return nil
	}

	// Create directories first
	if err := e.createDirectories(plan.Directories); err != nil {
		return fmt.Errorf("creating directories: %w", err)
	}

	// Write dedicated files
	if err := e.writeFiles(plan.Files, vars); err != nil {
		return fmt.Errorf("writing files: %w", err)
	}

	// Determine the MCP key to use (default to "mcpServers" for backward compatibility)
	mcpKey := plan.MCPKey
	if mcpKey == "" {
		mcpKey = "mcpServers"
	}

	// Build new merged files, MCP servers, and settings values from this plan.
	var newMergedFiles []string
	var newMCPServers []string
	newSettingsValues := make(map[string][]string)

	// Track MCP entries
	if len(plan.MCPEntries) > 0 {
		mcpPath := plan.MCPPath
		if mcpPath == "" {
			mcpPath = ".mcp.json"
		}
		newMergedFiles = append(newMergedFiles, mcpPath)

		for serverName := range plan.MCPEntries {
			if serverName == mcpKey {
				if servers, ok := plan.MCPEntries[serverName].(map[string]any); ok {
					for name := range servers {
						newMCPServers = append(newMCPServers, name)
					}
				}
			}
		}
	}

	// Track settings entries
	if len(plan.SettingsEntries) > 0 {
		settingsPath := plan.SettingsPath
		if settingsPath == "" {
			settingsPath = filepath.Join(".claude", "settings.json")
		}
		newMergedFiles = append(newMergedFiles, settingsPath)

		for key, val := range plan.SettingsEntries {
			switch v := val.(type) {
			case []string:
				newSettingsValues[key] = v
			case []any:
				var strs []string
				for _, item := range v {
					if s, ok := item.(string); ok {
						strs = append(strs, s)
					}
				}
				newSettingsValues[key] = strs
			}
		}
	}

	// Track agent file content
	hasAgentContent := plan.AgentFileContent != ""
	if hasAgentContent {
		agentPath := plan.AgentFilePath
		if agentPath == "" {
			agentPath = "CLAUDE.md"
		}
		newMergedFiles = append(newMergedFiles, agentPath)
	}

	// Replace (not append) all manifest entries for this plugin so stale
	// entries from previous installs on a different platform are cleared.
	filePaths := make([]string, len(plan.Files))
	for i, f := range plan.Files {
		filePaths[i] = f.Path
	}
	dirPaths := make([]string, len(plan.Directories))
	for i, d := range plan.Directories {
		dirPaths[i] = d.Path
	}
	e.manifest.ReplaceTracked(plan.PluginName, filePaths, dirPaths)
	e.manifest.ReplaceMergedFiles(plan.PluginName, newMergedFiles)
	e.manifest.ReplaceMCPServers(plan.PluginName, newMCPServers)
	e.manifest.ReplaceSettings(plan.PluginName, newSettingsValues)
	if hasAgentContent {
		e.manifest.TrackAgentContent(plan.PluginName)
	} else {
		// Clear agent content flag if this plugin no longer contributes
		if pm := e.manifest.GetPlugin(plan.PluginName); pm != nil {
			pm.HasAgentContent = false
		}
	}

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

// contentChanged returns true if the file at path doesn't exist or has
// different content than newContent.
func contentChanged(path string, newContent []byte) bool {
	existing, err := os.ReadFile(path)
	if err != nil {
		return true // file doesn't exist or can't read, needs write
	}
	return !bytes.Equal(existing, newContent)
}
