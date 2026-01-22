// Package lockfile provides lock file management for reproducible installs.
//
// The lock file is stored at dex.lock and pins exact versions of all installed
// plugins. This ensures that `dex install` produces identical results across
// different machines and times.
package lockfile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

const (
	// LockFileVersion is the current lock file format version
	LockFileVersion = "1.0"

	// LockFileName is the lock file name
	LockFileName = "dex.lock"
)

// LockFile pins exact versions for reproducible installs.
// Stored at dex.lock (JSON format)
type LockFile struct {
	// Version is the lock file format version
	Version string `json:"version"`

	// Agent is the agentic platform (e.g., "claude-code", "cursor")
	Agent string `json:"agent"`

	// Plugins maps plugin names to their locked versions
	Plugins map[string]*LockedPlugin `json:"plugins"`

	// path is the path to the lock file (not serialized)
	path string
}

// LockedPlugin represents a locked plugin version.
type LockedPlugin struct {
	// Version is the exact version string
	Version string `json:"version"`

	// Resolved is the full URL or path used to fetch the plugin
	Resolved string `json:"resolved"`

	// Integrity is the content hash in format "sha256-{base64}"
	Integrity string `json:"integrity"`

	// Dependencies maps dependency plugin names to version constraints
	Dependencies map[string]string `json:"dependencies"`
}

// Load loads a lock file from the project root.
// Returns an empty lock file if the file doesn't exist.
func Load(projectRoot string) (*LockFile, error) {
	lockPath := filepath.Join(projectRoot, LockFileName)

	l := &LockFile{
		Version: LockFileVersion,
		Plugins: make(map[string]*LockedPlugin),
		path:    lockPath,
	}

	data, err := os.ReadFile(lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return l, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, l); err != nil {
		return nil, err
	}

	// Ensure path is set after unmarshaling
	l.path = lockPath

	// Ensure Plugins map is initialized
	if l.Plugins == nil {
		l.Plugins = make(map[string]*LockedPlugin)
	}

	return l, nil
}

// Save writes the lock file to disk.
func (l *LockFile) Save() error {
	// Sort plugin entries for consistent output
	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(l.path, data, 0644)
}

// Get returns the locked version for a plugin (nil if not locked).
func (l *LockFile) Get(pluginName string) *LockedPlugin {
	return l.Plugins[pluginName]
}

// Set updates or adds a locked plugin.
func (l *LockFile) Set(pluginName string, locked *LockedPlugin) {
	if l.Plugins == nil {
		l.Plugins = make(map[string]*LockedPlugin)
	}

	// Ensure Dependencies map is initialized
	if locked.Dependencies == nil {
		locked.Dependencies = make(map[string]string)
	}

	l.Plugins[pluginName] = locked
}

// Remove removes a plugin from the lock file.
func (l *LockFile) Remove(pluginName string) {
	delete(l.Plugins, pluginName)
}

// Has checks if a plugin is in the lock file.
func (l *LockFile) Has(pluginName string) bool {
	_, exists := l.Plugins[pluginName]
	return exists
}

// LockedPlugins returns all locked plugin names (sorted).
func (l *LockFile) LockedPlugins() []string {
	names := make([]string, 0, len(l.Plugins))
	for name := range l.Plugins {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
