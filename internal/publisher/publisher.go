// Package publisher provides functionality for publishing plugin tarballs to registries.
package publisher

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"github.com/dex-tools/dex/internal/errors"
	"github.com/dex-tools/dex/internal/registry"
	"github.com/dex-tools/dex/pkg/version"
)

// Publisher is the interface for publishing packages to registries.
type Publisher interface {
	// Publish uploads a tarball to the registry and updates the index.
	Publish(tarballPath string) (*PublishResult, error)

	// Protocol returns the protocol this publisher handles (e.g., "file", "s3", "az").
	Protocol() string
}

// PublishResult contains the result of a publish operation.
type PublishResult struct {
	// Name is the package name
	Name string
	// Version is the package version
	Version string
	// URL is the full URL to the published tarball
	URL string
	// Integrity is the SHA-256 hash in format "sha256-{hex}"
	Integrity string
	// ManualInstructions contains manual upload instructions (for HTTPS only)
	ManualInstructions string
}

// tarballPattern matches common tarball naming conventions.
var tarballPattern = regexp.MustCompile(`^(.+?)[-_]v?(\d+\.\d+\.\d+(?:[-+].+)?)\.(tar\.gz|tgz)$`)

// TarballInfo holds parsed name and version from a tarball filename.
type TarballInfo struct {
	Name    string
	Version string
}

// ParseTarball extracts package name and version from a tarball filename.
// Returns an error if the filename doesn't match expected patterns.
func ParseTarball(path string) (*TarballInfo, error) {
	filename := filepath.Base(path)
	matches := tarballPattern.FindStringSubmatch(filename)
	if matches == nil {
		return nil, fmt.Errorf("could not parse package name and version from tarball filename: %s", filename)
	}

	return &TarballInfo{
		Name:    matches[1],
		Version: matches[2],
	}, nil
}

// ComputeTarballHash computes the SHA-256 hash of a tarball file.
// Returns the hash in format "sha256-{hex}".
func ComputeTarballHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open tarball: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to read tarball: %w", err)
	}

	return "sha256-" + hex.EncodeToString(hash.Sum(nil)), nil
}

// New creates a Publisher for the given registry URL.
// Supported protocols:
//   - file:// - Local filesystem
//   - s3://   - Amazon S3
//   - az://   - Azure Blob Storage
//   - https:// - Manual instructions only (read-only)
func New(registryURL string) (Publisher, error) {
	protocol, path, err := registry.ParseSource(registryURL)
	if err != nil {
		return nil, errors.NewPublishError("", registryURL, "connect", err)
	}

	switch protocol {
	case "file":
		return NewLocalPublisher(path)
	case "s3":
		return NewS3Publisher(registryURL)
	case "az":
		return NewAzurePublisher(registryURL)
	case "https", "http":
		return NewHTTPSPublisher(registryURL)
	default:
		return nil, errors.NewPublishError("", registryURL, "connect",
			fmt.Errorf("unsupported protocol: %s", protocol))
	}
}

// UpdateRegistryIndex updates a registry.json with a new package version.
// If the index doesn't exist, it creates a new one.
func UpdateRegistryIndex(index *registry.RegistryIndex, name, version string) *registry.RegistryIndex {
	if index == nil {
		index = &registry.RegistryIndex{
			Name:     "dex-registry",
			Version:  "1.0",
			Packages: make(map[string]registry.PackageEntry),
		}
	}

	if index.Packages == nil {
		index.Packages = make(map[string]registry.PackageEntry)
	}

	entry, ok := index.Packages[name]
	if !ok {
		entry = registry.PackageEntry{
			Versions: []string{},
		}
	}

	// Add version if not already present
	found := false
	for _, v := range entry.Versions {
		if v == version {
			found = true
			break
		}
	}
	if !found {
		entry.Versions = append(entry.Versions, version)
	}

	// Update latest
	entry.Latest = version

	index.Packages[name] = entry
	return index
}

// SortVersions sorts versions in proper semver order.
// Invalid versions are filtered out.
func SortVersions(versions []string) []string {
	parsed := version.SortStrings(versions)
	result := make([]string, len(parsed))
	for i, v := range parsed {
		result[i] = v.String()
	}
	return result
}
