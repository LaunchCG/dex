package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dex-tools/dex/internal/config"
	"github.com/dex-tools/dex/internal/errors"
	"github.com/dex-tools/dex/pkg/version"
)

// LocalRegistry handles file:// sources.
// It supports both registry mode (with registry.json index) and package mode
// (single package with package.hcl).
type LocalRegistry struct {
	basePath string     // Resolved absolute path to the registry/package directory
	mode     SourceMode // Registry or package mode
}

// NewLocalRegistry creates a registry from a local filesystem path.
// The path can be absolute or relative. Relative paths are resolved
// against the current working directory.
//
// In ModeAuto, the registry will detect the mode based on the presence
// of registry.json (registry mode) or package.hcl (package mode).
func NewLocalRegistry(path string, mode SourceMode) (*LocalRegistry, error) {
	// Resolve the path to absolute
	absPath, err := resolveLocalPath(path)
	if err != nil {
		return nil, errors.NewRegistryError("file:"+path, "connect", err)
	}

	// Verify the path exists
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.NewNotFoundError("path", absPath)
		}
		return nil, errors.NewRegistryError("file:"+path, "connect", err)
	}

	// Must be a directory
	if !info.IsDir() {
		return nil, errors.NewRegistryError("file:"+path, "connect",
			fmt.Errorf("path is not a directory: %s", absPath))
	}

	// Auto-detect mode if needed
	if mode == ModeAuto {
		mode = detectMode(absPath)
	}

	return &LocalRegistry{
		basePath: absPath,
		mode:     mode,
	}, nil
}

// Protocol returns "file".
func (r *LocalRegistry) Protocol() string {
	return "file"
}

// Mode returns the source mode (registry or package).
func (r *LocalRegistry) Mode() SourceMode {
	return r.mode
}

// BasePath returns the resolved base path of the registry.
func (r *LocalRegistry) BasePath() string {
	return r.basePath
}

// GetPackageInfo returns metadata about a package.
// In registry mode, it reads registry.json and looks up the package.
// In package mode, it reads package.hcl from the base path.
func (r *LocalRegistry) GetPackageInfo(name string) (*PackageInfo, error) {
	if r.mode == ModeRegistry {
		return r.getPackageFromRegistry(name)
	}
	return r.getPackageFromManifest(name)
}

// getPackageFromRegistry reads registry.json and returns package info.
func (r *LocalRegistry) getPackageFromRegistry(name string) (*PackageInfo, error) {
	index, err := r.loadRegistryIndex()
	if err != nil {
		return nil, err
	}

	entry, ok := index.Packages[name]
	if !ok {
		return nil, errors.NewNotFoundError("package", name)
	}

	return &PackageInfo{
		Name:     name,
		Versions: entry.Versions,
		Latest:   entry.Latest,
	}, nil
}

// getPackageFromManifest reads package.hcl and returns package info.
func (r *LocalRegistry) getPackageFromManifest(name string) (*PackageInfo, error) {
	pkgConfig, err := config.LoadPackage(r.basePath)
	if err != nil {
		return nil, errors.NewRegistryError("file:"+r.basePath, "fetch", err)
	}

	// For local packages, the package defines its own name and version
	return &PackageInfo{
		Name:        pkgConfig.Package.Name,
		Versions:    []string{pkgConfig.Package.Version},
		Latest:      pkgConfig.Package.Version,
		Description: pkgConfig.Package.Description,
	}, nil
}

// ResolvePackage resolves a version constraint and returns the resolved package.
// If version is empty or "latest", the latest version is used.
func (r *LocalRegistry) ResolvePackage(name, versionConstraint string) (*ResolvedPackage, error) {
	info, err := r.GetPackageInfo(name)
	if err != nil {
		return nil, err
	}

	// Handle "latest" or empty version
	if versionConstraint == "" || strings.ToLower(versionConstraint) == "latest" {
		if info.Latest == "" && len(info.Versions) > 0 {
			// If no latest is specified, use the highest version
			info.Latest = info.Versions[len(info.Versions)-1]
		}
		if info.Latest == "" {
			return nil, errors.NewVersionError(name, "latest", info.Versions, "no versions available")
		}
		versionConstraint = info.Latest
	}

	// Parse the constraint
	constraint, err := version.ParseConstraint(versionConstraint)
	if err != nil {
		return nil, errors.NewVersionError(name, versionConstraint, info.Versions,
			fmt.Sprintf("invalid constraint: %v", err))
	}

	// Parse all available versions
	var parsedVersions []*version.Version
	for _, v := range info.Versions {
		parsed, err := version.Parse(v)
		if err != nil {
			// Skip invalid versions
			continue
		}
		parsedVersions = append(parsedVersions, parsed)
	}

	// Find the best matching version
	best := constraint.FindBest(parsedVersions)
	if best == nil {
		return nil, errors.NewVersionError(name, versionConstraint, info.Versions, "")
	}

	resolvedVersion := best.String()

	// Determine the local path
	var localPath string
	if r.mode == ModeRegistry {
		// In registry mode, look for package directory
		localPath = filepath.Join(r.basePath, name)
		if _, err := os.Stat(localPath); os.IsNotExist(err) {
			// Try versioned directory
			localPath = filepath.Join(r.basePath, name+"-"+resolvedVersion)
		}
	} else {
		// In package mode, the base path is the package
		localPath = r.basePath
	}

	// Compute integrity hash
	integrity, err := ComputeIntegrity(localPath)
	if err != nil {
		// Non-fatal: just leave integrity empty
		integrity = ""
	}

	return &ResolvedPackage{
		Name:      name,
		Version:   resolvedVersion,
		URL:       "file:" + localPath,
		LocalPath: localPath,
		Integrity: integrity,
	}, nil
}

// FetchPackage returns the path to the package directory.
// For local file:// sources, this returns the source path directly (no copy needed).
func (r *LocalRegistry) FetchPackage(resolved *ResolvedPackage, destDir string) (string, error) {
	// For local sources, just return the path directly - no need to copy
	if r.mode == ModePackage {
		return r.basePath, nil
	}

	// Registry mode: find the package directory
	srcPath := filepath.Join(r.basePath, resolved.Name)
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		// Try versioned directory
		srcPath = filepath.Join(r.basePath, resolved.Name+"-"+resolved.Version)
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			return "", errors.NewNotFoundError("package directory", resolved.Name)
		}
	}

	return srcPath, nil
}

// ListPackages returns all available package names.
// In registry mode, this reads registry.json.
// In package mode, this returns the single package name.
func (r *LocalRegistry) ListPackages() ([]string, error) {
	if r.mode == ModeRegistry {
		index, err := r.loadRegistryIndex()
		if err != nil {
			return nil, err
		}

		packages := make([]string, 0, len(index.Packages))
		for name := range index.Packages {
			packages = append(packages, name)
		}
		return packages, nil
	}

	// Package mode: return the package name from manifest
	pkgConfig, err := config.LoadPackage(r.basePath)
	if err != nil {
		return nil, errors.NewRegistryError("file:"+r.basePath, "list", err)
	}

	return []string{pkgConfig.Package.Name}, nil
}

// loadRegistryIndex reads and parses registry.json.
func (r *LocalRegistry) loadRegistryIndex() (*RegistryIndex, error) {
	indexPath := filepath.Join(r.basePath, "registry.json")

	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.NewNotFoundError("registry.json", r.basePath)
		}
		return nil, errors.NewRegistryError("file:"+r.basePath, "fetch", err)
	}

	var index RegistryIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, errors.NewRegistryError("file:"+r.basePath, "fetch",
			fmt.Errorf("failed to parse registry.json: %w", err))
	}

	return &index, nil
}

// resolveLocalPath resolves a local path to an absolute path.
// It handles both absolute and relative paths.
func resolveLocalPath(path string) (string, error) {
	// Clean the path first
	path = filepath.Clean(path)

	// If already absolute, return as-is
	if filepath.IsAbs(path) {
		return path, nil
	}

	// Resolve relative to current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	return filepath.Join(cwd, path), nil
}

// detectMode auto-detects whether the path is a registry or a single package.
// Returns ModeRegistry if registry.json exists, otherwise ModePackage.
func detectMode(path string) SourceMode {
	registryPath := filepath.Join(path, "registry.json")
	if _, err := os.Stat(registryPath); err == nil {
		return ModeRegistry
	}

	// Default to package mode
	return ModePackage
}
