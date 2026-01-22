// Package registry provides clients for fetching plugins from various sources.
//
// This package implements support for multiple registry protocols:
//   - file:// - Local filesystem access
//   - git+https://, git+ssh:// - Git repository cloning
//   - https:// - HTTP/HTTPS downloads
//   - s3:// - Amazon S3
//   - az:// - Azure Blob Storage
//
// Each protocol supports three modes:
//   - Registry mode: registry.json index with multiple packages
//   - Package mode: single package with package.json/package.hcl
//   - Direct tarball: URL ends with .tar.gz (auto-detected)
package registry

import (
	"fmt"
	"strings"
)

// SourceMode indicates how a source should be interpreted.
type SourceMode int

const (
	// ModeAuto attempts to auto-detect the source mode
	ModeAuto SourceMode = iota
	// ModeRegistry expects a registry.json index file
	ModeRegistry
	// ModePackage expects a single package with package.json/package.hcl
	ModePackage
)

// String returns a human-readable representation of the source mode.
func (m SourceMode) String() string {
	switch m {
	case ModeAuto:
		return "auto"
	case ModeRegistry:
		return "registry"
	case ModePackage:
		return "package"
	default:
		return "unknown"
	}
}

// PackageInfo contains metadata about a package available in a registry.
type PackageInfo struct {
	// Name is the package name
	Name string
	// Versions is the list of available versions
	Versions []string
	// Latest is the latest/recommended version
	Latest string
	// Description is an optional package description
	Description string
}

// ResolvedPackage represents a fully resolved package with a specific version.
type ResolvedPackage struct {
	// Name is the package name
	Name string
	// Version is the resolved version string
	Version string
	// URL is the download URL for the package tarball or directory
	URL string
	// LocalPath is the local filesystem path (for file:// protocol)
	LocalPath string
	// Integrity is the SHA-256 hash of the package contents (if known)
	Integrity string
}

// RegistryIndex represents the registry.json file format.
// This is the index file that lists all available packages in a registry.
type RegistryIndex struct {
	// Name is the registry name
	Name string `json:"name"`
	// Version is the registry format version
	Version string `json:"version"`
	// Packages is a map of package name to package entry
	Packages map[string]PackageEntry `json:"packages"`
}

// PackageEntry represents a package in the registry index.
type PackageEntry struct {
	// Versions is the list of available versions
	Versions []string `json:"versions"`
	// Latest is the latest/recommended version
	Latest string `json:"latest"`
}

// Registry is the interface for package registries.
// Implementations handle different protocols like file://, git+https://, https://.
type Registry interface {
	// Protocol returns the protocol this registry handles (e.g., "file", "git", "https")
	Protocol() string

	// GetPackageInfo returns metadata about a package.
	// Returns a NotFoundError if the package does not exist.
	GetPackageInfo(name string) (*PackageInfo, error)

	// ResolvePackage resolves a package name and version constraint to a fetchable location.
	// If version is empty or "latest", the latest available version is used.
	// Returns a VersionError if no version satisfies the constraint.
	ResolvePackage(name, version string) (*ResolvedPackage, error)

	// FetchPackage downloads/copies a package to the destination directory.
	// Returns the path to the package directory.
	FetchPackage(resolved *ResolvedPackage, destDir string) (string, error)

	// ListPackages returns all available package names.
	// May return nil if listing is not supported by the registry type.
	ListPackages() ([]string, error)
}

// ParseSource parses a source URL string and returns the protocol and path.
// It handles various URL formats:
//
//	"file:../path"           -> ("file", "../path", nil)
//	"file:./path"            -> ("file", "./path", nil)
//	"file:///absolute/path"  -> ("file", "/absolute/path", nil)
//	"git+https://github.com" -> ("git", "https://github.com", nil)
//	"git+ssh://git@host"     -> ("git", "ssh://git@host", nil)
//	"https://example.com"    -> ("https", "https://example.com", nil)
//	"http://example.com"     -> ("http", "http://example.com", nil)
//	"s3://bucket/path"       -> ("s3", "bucket/path", nil)
//	"az://account/container" -> ("az", "account/container", nil)
func ParseSource(source string) (protocol, path string, err error) {
	if source == "" {
		return "", "", fmt.Errorf("empty source URL")
	}

	// Handle file:// protocol (special case with multiple formats)
	if strings.HasPrefix(source, "file:") {
		rest := source[5:]
		// file:///absolute/path -> /absolute/path
		if strings.HasPrefix(rest, "//") {
			return "file", rest[2:], nil
		}
		// file:./relative or file:../relative or file:relative
		return "file", rest, nil
	}

	// Handle git+ prefixed protocols
	if strings.HasPrefix(source, "git+") {
		rest := source[4:]
		if strings.HasPrefix(rest, "https://") || strings.HasPrefix(rest, "ssh://") || strings.HasPrefix(rest, "git@") {
			return "git", rest, nil
		}
		return "", "", fmt.Errorf("invalid git URL: must be git+https://, git+ssh://, or git+git@: %s", source)
	}

	// Handle standard URL schemes
	if strings.HasPrefix(source, "https://") {
		return "https", source, nil
	}
	if strings.HasPrefix(source, "http://") {
		return "http", source, nil
	}

	// Handle cloud storage protocols
	if strings.HasPrefix(source, "s3://") {
		return "s3", strings.TrimPrefix(source, "s3://"), nil
	}
	if strings.HasPrefix(source, "az://") {
		return "az", strings.TrimPrefix(source, "az://"), nil
	}

	return "", "", fmt.Errorf("unsupported source URL format: %s", source)
}
