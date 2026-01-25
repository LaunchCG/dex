// Package manifest tracks files installed by dex plugins.
package manifest

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Manifest tracks all files managed by dex.
type Manifest struct {
	// Version is the manifest format version
	Version string `json:"version"`

	// Plugins maps plugin names to their tracked resources
	Plugins map[string]*PluginManifest `json:"plugins"`

	// path is the file path for saving
	path string
}

// PluginManifest tracks resources installed by a single plugin.
type PluginManifest struct {
	// Files are relative paths to files installed by this plugin
	Files []string `json:"files,omitempty"`

	// Directories are relative paths to directories created by this plugin
	Directories []string `json:"directories,omitempty"`

	// MCPServers are names of MCP servers contributed by this plugin
	MCPServers []string `json:"mcp_servers,omitempty"`

	// SettingsValues tracks settings values contributed by this plugin (key -> values)
	SettingsValues map[string][]string `json:"settings_values,omitempty"`

	// HasAgentContent indicates if this plugin contributed to the agent file
	HasAgentContent bool `json:"has_agent_content,omitempty"`
}

// UntrackResult contains resources that were untracked.
type UntrackResult struct {
	Files           []string
	Directories     []string
	MCPServers      []string
	SettingsValues  map[string][]string
	HasAgentContent bool
}

// Load loads a manifest from the project root.
// Creates a new manifest if the file doesn't exist.
func Load(projectRoot string) (*Manifest, error) {
	dexDir := filepath.Join(projectRoot, ".dex")
	manifestPath := filepath.Join(dexDir, "manifest.json")

	m := &Manifest{
		Version: "1.0",
		Plugins: make(map[string]*PluginManifest),
		path:    manifestPath,
	}

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return m, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, m); err != nil {
		return nil, err
	}

	m.path = manifestPath
	if m.Plugins == nil {
		m.Plugins = make(map[string]*PluginManifest)
	}

	return m, nil
}

// Save writes the manifest to disk.
func (m *Manifest) Save() error {
	// Ensure .dex directory exists
	if err := os.MkdirAll(filepath.Dir(m.path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}

	data = append(data, '\n')
	return os.WriteFile(m.path, data, 0644)
}

// Track records files and directories for a plugin.
func (m *Manifest) Track(pluginName string, files, directories []string) {
	pm := m.getOrCreate(pluginName)
	pm.Files = uniqueStrings(append(pm.Files, files...))
	pm.Directories = uniqueStrings(append(pm.Directories, directories...))
}

// TrackMCPServer records an MCP server for a plugin.
func (m *Manifest) TrackMCPServer(pluginName, serverName string) {
	pm := m.getOrCreate(pluginName)
	pm.MCPServers = uniqueStrings(append(pm.MCPServers, serverName))
}

// TrackSettings records settings values for a plugin.
func (m *Manifest) TrackSettings(pluginName string, values map[string][]string) {
	pm := m.getOrCreate(pluginName)
	if pm.SettingsValues == nil {
		pm.SettingsValues = make(map[string][]string)
	}
	for k, v := range values {
		pm.SettingsValues[k] = uniqueStrings(append(pm.SettingsValues[k], v...))
	}
}

// TrackAgentContent marks that a plugin contributed to the agent file.
func (m *Manifest) TrackAgentContent(pluginName string) {
	pm := m.getOrCreate(pluginName)
	pm.HasAgentContent = true
}

// Untrack removes a plugin and returns its tracked resources.
func (m *Manifest) Untrack(pluginName string) *UntrackResult {
	pm, ok := m.Plugins[pluginName]
	if !ok {
		return &UntrackResult{}
	}

	result := &UntrackResult{
		Files:           pm.Files,
		Directories:     pm.Directories,
		MCPServers:      pm.MCPServers,
		SettingsValues:  pm.SettingsValues,
		HasAgentContent: pm.HasAgentContent,
	}

	delete(m.Plugins, pluginName)
	return result
}

// GetPlugin returns the manifest for a specific plugin.
func (m *Manifest) GetPlugin(pluginName string) *PluginManifest {
	return m.Plugins[pluginName]
}

// GetPluginNames returns all tracked plugin names.
func (m *Manifest) GetPluginNames() []string {
	names := make([]string, 0, len(m.Plugins))
	for name := range m.Plugins {
		names = append(names, name)
	}
	return names
}

// InstalledPlugins returns all tracked plugin names (alias for GetPluginNames).
func (m *Manifest) InstalledPlugins() []string {
	return m.GetPluginNames()
}

// AllFiles returns all tracked files across all plugins.
func (m *Manifest) AllFiles() []string {
	var files []string
	for _, pm := range m.Plugins {
		files = append(files, pm.Files...)
	}
	return files
}

// AllDirectories returns all tracked directories across all plugins.
func (m *Manifest) AllDirectories() []string {
	var dirs []string
	for _, pm := range m.Plugins {
		dirs = append(dirs, pm.Directories...)
	}
	return uniqueStrings(dirs)
}

// IsTracked checks if a file path is tracked by any plugin.
// Returns the plugin name and true if tracked, empty string and false otherwise.
func (m *Manifest) IsTracked(filePath string) (string, bool) {
	for pluginName, pm := range m.Plugins {
		for _, f := range pm.Files {
			if f == filePath {
				return pluginName, true
			}
		}
	}
	return "", false
}

// IsSettingsValueUsedByOthers checks if a settings value is used by plugins other than the specified one.
func (m *Manifest) IsSettingsValueUsedByOthers(excludePlugin, key, value string) bool {
	for pluginName, pm := range m.Plugins {
		if pluginName == excludePlugin {
			continue
		}
		if pm.SettingsValues != nil {
			for _, v := range pm.SettingsValues[key] {
				if v == value {
					return true
				}
			}
		}
	}
	return false
}

// getOrCreate gets or creates a plugin manifest.
func (m *Manifest) getOrCreate(pluginName string) *PluginManifest {
	pm, ok := m.Plugins[pluginName]
	if !ok {
		pm = &PluginManifest{}
		m.Plugins[pluginName] = pm
	}
	return pm
}

// uniqueStrings returns a slice with duplicates removed.
func uniqueStrings(s []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(s))
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}
