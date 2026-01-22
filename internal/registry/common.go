// common.go provides shared utilities for registry implementations.

package registry

import (
	"regexp"
	"strings"
)

// TarballInfo holds parsed name and version from a tarball filename.
type TarballInfo struct {
	Name    string
	Version string
}

// tarballPattern matches common tarball naming conventions:
// - pkg-1.0.0.tar.gz
// - pkg-v1.0.0.tar.gz
// - pkg_1.0.0.tar.gz
// - pkg-1.0.0.tgz
var tarballPattern = regexp.MustCompile(`^(.+?)[-_]v?(\d+\.\d+\.\d+(?:[-+].+)?)\.(tar\.gz|tgz)$`)

// ParseTarballFilename extracts package name and version from a tarball filename.
// Returns nil if the filename doesn't match expected patterns.
//
// Supported formats:
//   - pkg-1.0.0.tar.gz
//   - pkg-v1.0.0.tar.gz
//   - pkg_1.0.0.tar.gz
//   - pkg-1.0.0.tgz
func ParseTarballFilename(filename string) *TarballInfo {
	matches := tarballPattern.FindStringSubmatch(filename)
	if matches == nil {
		return nil
	}

	return &TarballInfo{
		Name:    matches[1],
		Version: matches[2],
	}
}

// IsTarballURL checks if a URL points directly to a tarball file.
// Returns true for URLs ending in .tar.gz or .tgz.
func IsTarballURL(url string) bool {
	lower := strings.ToLower(url)
	return strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz")
}

// NormalizeName normalizes a package name for comparison.
// It converts to lowercase and replaces underscores with hyphens.
func NormalizeName(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, "_", "-"))
}

// NamesMatch checks if two package names match after normalization.
// This handles variations like "my-plugin" vs "my_plugin".
func NamesMatch(name1, name2 string) bool {
	return NormalizeName(name1) == NormalizeName(name2)
}

// GetFilenameFromURL extracts the filename from a URL path.
// For example, "https://example.com/path/to/pkg-1.0.0.tar.gz" returns "pkg-1.0.0.tar.gz".
func GetFilenameFromURL(url string) string {
	// Remove query string and fragment
	if idx := strings.Index(url, "?"); idx != -1 {
		url = url[:idx]
	}
	if idx := strings.Index(url, "#"); idx != -1 {
		url = url[:idx]
	}

	// Find the last path segment
	if idx := strings.LastIndex(url, "/"); idx != -1 {
		return url[idx+1:]
	}
	return url
}
