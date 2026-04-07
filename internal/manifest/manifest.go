// Package manifest tracks files installed by dex packages.
package manifest

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/launchcg/dex/internal/jsonutil"
)

// Manifest tracks all files managed by dex.
type Manifest struct {
	// Version is the manifest format version
	Version string `json:"version"`

	// Packages maps package names to their tracked resources
	Packages map[string]*PackageManifest `json:"packages"`

	// path is the file path for saving
	path string
}

// PackageManifest tracks resources installed by a single package.
type PackageManifest struct {
	// Files are relative paths to files installed by this package
	Files []string `json:"files,omitempty"`

	// Directories are relative paths to directories created by this package
	Directories []string `json:"directories,omitempty"`

	// MCPServers are names of MCP servers contributed by this package
	MCPServers []string `json:"mcp_servers,omitempty"`

	// SettingsValues tracks settings values contributed by this package (key -> values)
	SettingsValues map[string][]string `json:"settings_values,omitempty"`

	// HasAgentContent indicates if this package contributed to the agent file
	HasAgentContent bool `json:"has_agent_content,omitempty"`

	// MergedFiles are relative paths to merged configuration files (e.g., .mcp.json, .claude/settings.json)
	MergedFiles []string `json:"merged_files,omitempty"`
}

// UntrackResult contains resources that were untracked.
type UntrackResult struct {
	Files           []string
	Directories     []string
	MCPServers      []string
	SettingsValues  map[string][]string
	HasAgentContent bool
	MergedFiles     []string
}

// Load loads a manifest from the project root.
// Creates a new manifest if the file doesn't exist.
func Load(projectRoot string) (*Manifest, error) {
	dexDir := filepath.Join(projectRoot, ".dex")
	manifestPath := filepath.Join(dexDir, "manifest.json")

	m := &Manifest{
		Version:  "1.0",
		Packages: make(map[string]*PackageManifest),
		path:     manifestPath,
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
	if m.Packages == nil {
		m.Packages = make(map[string]*PackageManifest)
	}

	return m, nil
}

// Save writes the manifest to disk.
func (m *Manifest) Save() error {
	// Ensure .dex directory exists
	if err := os.MkdirAll(filepath.Dir(m.path), 0755); err != nil {
		return err
	}

	data, err := jsonutil.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}

	data = append(data, '\n')
	return os.WriteFile(m.path, data, 0644)
}

// Track records files and directories for a package.
func (m *Manifest) Track(pkgName string, files, directories []string) {
	pm := m.getOrCreate(pkgName)
	pm.Files = uniqueStrings(append(pm.Files, files...))
	pm.Directories = uniqueStrings(append(pm.Directories, directories...))
}

// ReplaceTracked replaces (not appends) the files and directories for a package.
// Used during reinstall to remove stale entries from previous installations.
func (m *Manifest) ReplaceTracked(pkgName string, files, directories []string) {
	pm := m.getOrCreate(pkgName)
	pm.Files = uniqueStrings(files)
	pm.Directories = uniqueStrings(directories)
}

// ReplaceMergedFiles replaces the merged files list for a package.
func (m *Manifest) ReplaceMergedFiles(pkgName string, mergedFiles []string) {
	pm := m.getOrCreate(pkgName)
	pm.MergedFiles = uniqueStrings(mergedFiles)
}

// ReplaceMCPServers replaces the MCP servers list for a package.
func (m *Manifest) ReplaceMCPServers(pkgName string, servers []string) {
	pm := m.getOrCreate(pkgName)
	pm.MCPServers = uniqueStrings(servers)
}

// ReplaceSettings replaces (not merges) the settings values for a package.
func (m *Manifest) ReplaceSettings(pkgName string, values map[string][]string) {
	pm := m.getOrCreate(pkgName)
	pm.SettingsValues = values
}

// TrackMCPServer records an MCP server for a package.
func (m *Manifest) TrackMCPServer(pkgName, serverName string) {
	pm := m.getOrCreate(pkgName)
	pm.MCPServers = uniqueStrings(append(pm.MCPServers, serverName))
}

// TrackSettings records settings values for a package.
func (m *Manifest) TrackSettings(pkgName string, values map[string][]string) {
	pm := m.getOrCreate(pkgName)
	if pm.SettingsValues == nil {
		pm.SettingsValues = make(map[string][]string)
	}
	for k, v := range values {
		pm.SettingsValues[k] = uniqueStrings(append(pm.SettingsValues[k], v...))
	}
}

// TrackAgentContent marks that a package contributed to the agent file.
func (m *Manifest) TrackAgentContent(pkgName string) {
	pm := m.getOrCreate(pkgName)
	pm.HasAgentContent = true
}

// TrackMergedFile records a merged configuration file for a package.
func (m *Manifest) TrackMergedFile(pkgName, filePath string) {
	pm := m.getOrCreate(pkgName)
	pm.MergedFiles = uniqueStrings(append(pm.MergedFiles, filePath))
}

// Untrack removes a package and returns its tracked resources.
func (m *Manifest) Untrack(pkgName string) *UntrackResult {
	pm, ok := m.Packages[pkgName]
	if !ok {
		return &UntrackResult{}
	}

	result := &UntrackResult{
		Files:           pm.Files,
		Directories:     pm.Directories,
		MCPServers:      pm.MCPServers,
		SettingsValues:  pm.SettingsValues,
		HasAgentContent: pm.HasAgentContent,
		MergedFiles:     pm.MergedFiles,
	}

	delete(m.Packages, pkgName)
	return result
}

// GetPackage returns the manifest for a specific package.
func (m *Manifest) GetPackage(pkgName string) *PackageManifest {
	return m.Packages[pkgName]
}

// GetPackageNames returns all tracked package names.
func (m *Manifest) GetPackageNames() []string {
	names := make([]string, 0, len(m.Packages))
	for name := range m.Packages {
		names = append(names, name)
	}
	return names
}

// InstalledPackages returns all tracked package names (alias for GetPackageNames).
func (m *Manifest) InstalledPackages() []string {
	return m.GetPackageNames()
}

// AllFiles returns all tracked files across all packages, including merged files.
func (m *Manifest) AllFiles() []string {
	var files []string
	for _, pm := range m.Packages {
		files = append(files, pm.Files...)
		files = append(files, pm.MergedFiles...)
	}
	return uniqueStrings(files)
}

// AllDirectories returns all tracked directories across all packages.
func (m *Manifest) AllDirectories() []string {
	var dirs []string
	for _, pm := range m.Packages {
		dirs = append(dirs, pm.Directories...)
	}
	return uniqueStrings(dirs)
}

// IsTracked checks if a file path is tracked by any package.
// Returns the package name and true if tracked, empty string and false otherwise.
func (m *Manifest) IsTracked(filePath string) (string, bool) {
	for pkgName, pm := range m.Packages {
		for _, f := range pm.Files {
			if f == filePath {
				return pkgName, true
			}
		}
	}
	return "", false
}

// IsSettingsValueUsedByOthers checks if a settings value is used by packages other than the specified one.
func (m *Manifest) IsSettingsValueUsedByOthers(excludePkg, key, value string) bool {
	for pkgName, pm := range m.Packages {
		if pkgName == excludePkg {
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

// IsMergedFileUsedByOthers checks if a merged file is used by packages other than the specified one.
func (m *Manifest) IsMergedFileUsedByOthers(excludePkg, filePath string) bool {
	for pkgName, pm := range m.Packages {
		if pkgName == excludePkg {
			continue
		}
		for _, f := range pm.MergedFiles {
			if f == filePath {
				return true
			}
		}
	}
	return false
}

// getOrCreate gets or creates a package manifest.
func (m *Manifest) getOrCreate(pkgName string) *PackageManifest {
	pm, ok := m.Packages[pkgName]
	if !ok {
		pm = &PackageManifest{}
		m.Packages[pkgName] = pm
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
