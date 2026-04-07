// Package lockfile provides lock file management for reproducible installs.
//
// The lock file is stored at dex.lock and pins exact versions of all installed
// packages. This ensures that `dex sync` produces identical results across
// different machines and times.
package lockfile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/launchcg/dex/internal/jsonutil"
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

	// Packages maps package names to their locked versions
	Packages map[string]*LockedPackage `json:"packages"`

	// path is the path to the lock file (not serialized)
	path string
}

// LockedPackage represents a locked package version.
type LockedPackage struct {
	// Version is the exact version string
	Version string `json:"version"`

	// Resolved is the full URL or path used to fetch the package
	Resolved string `json:"resolved"`

	// Integrity is the content hash in format "sha256-{base64}"
	Integrity string `json:"integrity"`

	// Dependencies maps dependency package names to version constraints
	Dependencies map[string]string `json:"dependencies"`
}

// Load loads a lock file from the project root.
// Returns an empty lock file if the file doesn't exist.
func Load(projectRoot string) (*LockFile, error) {
	lockPath := filepath.Join(projectRoot, LockFileName)

	l := &LockFile{
		Version:  LockFileVersion,
		Packages: make(map[string]*LockedPackage),
		path:     lockPath,
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

	// Ensure Packages map is initialized
	if l.Packages == nil {
		l.Packages = make(map[string]*LockedPackage)
	}

	return l, nil
}

// Save writes the lock file to disk.
func (l *LockFile) Save() error {
	// Sort package entries for consistent output
	data, err := jsonutil.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(l.path, data, 0644)
}

// Get returns the locked version for a package (nil if not locked).
func (l *LockFile) Get(pkgName string) *LockedPackage {
	return l.Packages[pkgName]
}

// Set updates or adds a locked package.
func (l *LockFile) Set(pkgName string, locked *LockedPackage) {
	if l.Packages == nil {
		l.Packages = make(map[string]*LockedPackage)
	}

	// Ensure Dependencies map is initialized
	if locked.Dependencies == nil {
		locked.Dependencies = make(map[string]string)
	}

	l.Packages[pkgName] = locked
}

// Remove removes a package from the lock file.
func (l *LockFile) Remove(pkgName string) {
	delete(l.Packages, pkgName)
}

// Has checks if a package is in the lock file.
func (l *LockFile) Has(pkgName string) bool {
	_, exists := l.Packages[pkgName]
	return exists
}

// LockedPackages returns all locked package names (sorted).
func (l *LockFile) LockedPackages() []string {
	names := make([]string, 0, len(l.Packages))
	for name := range l.Packages {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
