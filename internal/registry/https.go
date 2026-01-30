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
//   - Package mode: single package with package.json
//   - Direct tarball: URL ends with .tar.gz (auto-detected)
package registry

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/launchcg/dex/internal/errors"
	"github.com/launchcg/dex/pkg/version"
)

// HTTPSRegistry handles https:// sources.
// It supports registry mode (registry.json) and direct tarball URLs.
type HTTPSRegistry struct {
	baseURL         string
	mode            SourceMode
	cache           *Cache
	client          *http.Client
	isDirectTarball bool
	tarballInfo     *TarballInfo
}

// NewHTTPSRegistry creates a registry from an HTTPS URL.
// The URL should be the base URL of the registry (without registry.json),
// or a direct tarball URL.
// The mode parameter controls how the source is interpreted.
func NewHTTPSRegistry(url string, mode SourceMode) (*HTTPSRegistry, error) {
	// Validate URL scheme
	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		return nil, errors.NewRegistryError(url, "connect", fmt.Errorf("invalid URL scheme: expected https:// or http://"))
	}

	// Create cache
	cache, err := DefaultCache()
	if err != nil {
		return nil, errors.NewRegistryError(url, "connect", err)
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Check if this is a direct tarball URL
	isDirectTarball := IsTarballURL(url)
	var tarballInfo *TarballInfo

	if isDirectTarball {
		filename := GetFilenameFromURL(url)
		tarballInfo = ParseTarballFilename(filename)
		// For direct tarballs, don't normalize the URL
	} else {
		// Normalize URL - ensure it doesn't end with /
		url = strings.TrimSuffix(url, "/")
	}

	return &HTTPSRegistry{
		baseURL:         url,
		mode:            mode,
		cache:           cache,
		client:          client,
		isDirectTarball: isDirectTarball,
		tarballInfo:     tarballInfo,
	}, nil
}

// Protocol returns "https".
func (r *HTTPSRegistry) Protocol() string {
	return "https"
}

// GetPackageInfo returns package info based on the mode.
// - Direct tarball: extracts name/version from filename
// - Registry mode: fetches registry.json and looks up the package
func (r *HTTPSRegistry) GetPackageInfo(name string) (*PackageInfo, error) {
	if r.isDirectTarball {
		return r.getPackageFromTarball(name)
	}

	if r.mode == ModePackage {
		return nil, errors.NewRegistryError(r.baseURL, "fetch",
			fmt.Errorf("HTTPS sources do not support package mode; use registry mode or direct tarball URL"))
	}

	// Registry mode
	return r.getPackageFromRegistry(name)
}

// getPackageFromTarball returns package info extracted from the tarball URL.
func (r *HTTPSRegistry) getPackageFromTarball(name string) (*PackageInfo, error) {
	if r.tarballInfo == nil {
		return nil, errors.NewRegistryError(r.baseURL, "parse",
			fmt.Errorf("could not parse package name and version from tarball URL"))
	}

	// If a name is requested, check if it matches
	if name != "" && !NamesMatch(name, r.tarballInfo.Name) {
		return nil, errors.NewNotFoundError("package", name)
	}

	return &PackageInfo{
		Name:     r.tarballInfo.Name,
		Versions: []string{r.tarballInfo.Version},
		Latest:   r.tarballInfo.Version,
	}, nil
}

// getPackageFromRegistry fetches registry.json and returns package info.
func (r *HTTPSRegistry) getPackageFromRegistry(name string) (*PackageInfo, error) {
	index, err := r.fetchRegistryIndex()
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

// ResolvePackage resolves a version constraint and returns the resolved package.
// If versionConstraint is empty or "latest", the latest version is used.
// Returns a VersionError if no version satisfies the constraint.
func (r *HTTPSRegistry) ResolvePackage(name, versionConstraint string) (*ResolvedPackage, error) {
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
		return nil, errors.NewVersionError(name, versionConstraint, info.Versions, fmt.Sprintf("invalid constraint: %v", err))
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

	// Get the tarball URL
	resolvedVersion := best.String()
	tarballURL, err := r.getTarballURL(info.Name, resolvedVersion)
	if err != nil {
		return nil, err
	}

	return &ResolvedPackage{
		Name:    info.Name,
		Version: resolvedVersion,
		URL:     tarballURL,
	}, nil
}

// FetchPackage downloads and extracts the tarball to destDir.
// Returns the path to the extracted package directory.
func (r *HTTPSRegistry) FetchPackage(resolved *ResolvedPackage, destDir string) (string, error) {
	// Create a cache key based on the URL
	cacheKey := r.getCacheKey(resolved.URL)
	cachePath := r.cache.GetPath(cacheKey)

	// Check if already cached
	if !r.cache.Has(cacheKey) {
		// Download to cache
		if err := r.downloadFile(resolved.URL, cachePath); err != nil {
			return "", errors.NewInstallError(resolved.Name, "fetch", err)
		}
	}

	// Verify integrity if known
	if resolved.Integrity != "" {
		hash, err := computeFileHash(cachePath)
		if err != nil {
			return "", errors.NewInstallError(resolved.Name, "verify", err)
		}
		if hash != resolved.Integrity {
			// Remove corrupted cache file
			os.Remove(cachePath)
			return "", errors.NewInstallError(resolved.Name, "verify",
				fmt.Errorf("integrity mismatch: expected %s, got %s", resolved.Integrity, hash))
		}
	}

	// Extract to destination
	extractedPath, err := extractTarGz(cachePath, destDir)
	if err != nil {
		return "", errors.NewInstallError(resolved.Name, "extract", err)
	}

	return extractedPath, nil
}

// ListPackages returns all available package names.
// For direct tarballs, returns a single package.
// For registry mode, fetches and parses registry.json.
func (r *HTTPSRegistry) ListPackages() ([]string, error) {
	if r.isDirectTarball {
		if r.tarballInfo != nil {
			return []string{r.tarballInfo.Name}, nil
		}
		return nil, nil
	}

	if r.mode == ModePackage {
		return nil, errors.NewRegistryError(r.baseURL, "list",
			fmt.Errorf("HTTPS sources do not support package mode; use registry mode or direct tarball URL"))
	}

	// Registry mode
	index, err := r.fetchRegistryIndex()
	if err != nil {
		return nil, err
	}

	packages := make([]string, 0, len(index.Packages))
	for name := range index.Packages {
		packages = append(packages, name)
	}

	return packages, nil
}

// fetchRegistryIndex downloads and parses registry.json.
func (r *HTTPSRegistry) fetchRegistryIndex() (*RegistryIndex, error) {
	indexURL := r.baseURL + "/registry.json"

	resp, err := r.client.Get(indexURL)
	if err != nil {
		return nil, errors.NewRegistryError(r.baseURL, "fetch", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.NewRegistryError(r.baseURL, "fetch",
			fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status))
	}

	var index RegistryIndex
	if err := json.NewDecoder(resp.Body).Decode(&index); err != nil {
		return nil, errors.NewRegistryError(r.baseURL, "fetch",
			fmt.Errorf("failed to parse registry.json: %w", err))
	}

	return &index, nil
}

// downloadFile downloads a URL to a local file.
func (r *HTTPSRegistry) downloadFile(url, destPath string) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	resp, err := r.client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Create temporary file first
	tmpPath := destPath + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	_, err = io.Copy(out, resp.Body)
	out.Close()
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Rename to final path
	if err := os.Rename(tmpPath, destPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

// extractTarGz extracts a .tar.gz file to a directory.
// Returns the path to the extracted content. If the tarball contains a single
// top-level directory, that directory path is returned. Otherwise, destDir is returned.
func extractTarGz(tarPath, destDir string) (string, error) {
	// Open the tarball
	file, err := os.Open(tarPath)
	if err != nil {
		return "", fmt.Errorf("failed to open tarball: %w", err)
	}
	defer file.Close()

	// Create gzip reader
	gzr, err := gzip.NewReader(file)
	if err != nil {
		return "", fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	// Create tar reader
	tr := tar.NewReader(gzr)

	// Ensure destination directory exists
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Track top-level directories to detect single directory tarballs
	topLevelDirs := make(map[string]bool)

	// Extract files
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("failed to read tar header: %w", err)
		}

		// Sanitize the path to prevent directory traversal
		name := header.Name
		if strings.Contains(name, "..") {
			return "", fmt.Errorf("invalid path in tarball: %s", name)
		}

		// Track top-level directory
		parts := strings.SplitN(name, "/", 2)
		if len(parts) > 0 && parts[0] != "" && parts[0] != "." {
			topLevelDirs[parts[0]] = true
		}

		target := filepath.Join(destDir, name)

		switch header.Typeflag {
		case tar.TypeDir:
			// Create directory with permissions
			mode := os.FileMode(header.Mode)
			if mode == 0 {
				mode = 0755
			}
			if err := os.MkdirAll(target, mode); err != nil {
				return "", fmt.Errorf("failed to create directory %s: %w", target, err)
			}

		case tar.TypeReg:
			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return "", fmt.Errorf("failed to create parent directory: %w", err)
			}

			// Create file with permissions
			mode := os.FileMode(header.Mode)
			if mode == 0 {
				mode = 0644
			}

			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
			if err != nil {
				return "", fmt.Errorf("failed to create file %s: %w", target, err)
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return "", fmt.Errorf("failed to write file %s: %w", target, err)
			}
			outFile.Close()

		case tar.TypeSymlink:
			// Create symlink
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return "", fmt.Errorf("failed to create parent directory: %w", err)
			}
			if err := os.Symlink(header.Linkname, target); err != nil {
				return "", fmt.Errorf("failed to create symlink %s: %w", target, err)
			}

		case tar.TypeLink:
			// Create hard link
			linkTarget := filepath.Join(destDir, header.Linkname)
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return "", fmt.Errorf("failed to create parent directory: %w", err)
			}
			if err := os.Link(linkTarget, target); err != nil {
				return "", fmt.Errorf("failed to create hard link %s: %w", target, err)
			}
		}
	}

	// If there's exactly one top-level directory, return its path
	if len(topLevelDirs) == 1 {
		for dir := range topLevelDirs {
			return filepath.Join(destDir, dir), nil
		}
	}

	return destDir, nil
}

// getTarballURL returns the URL for a package tarball.
// For direct tarballs, returns the baseURL.
// For package mode, looks for the tarball in the package directory.
// For registry mode, tries multiple naming conventions.
func (r *HTTPSRegistry) getTarballURL(name, ver string) (string, error) {
	// Direct tarball URL
	if r.isDirectTarball {
		return r.baseURL, nil
	}

	// Try different naming conventions
	patterns := []string{
		fmt.Sprintf("%s-%s.tar.gz", name, ver),
		fmt.Sprintf("%s-v%s.tar.gz", name, ver),
		fmt.Sprintf("%s_%s.tar.gz", name, ver),
		fmt.Sprintf("%s-%s.tgz", name, ver),
	}

	for _, pattern := range patterns {
		url := r.baseURL + "/" + pattern

		req, err := http.NewRequest("HEAD", url, nil)
		if err != nil {
			continue
		}

		resp, err := r.client.Do(req)
		if err != nil {
			continue
		}
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			return url, nil
		}
	}

	// If no HEAD request succeeded, return the first pattern and let download fail
	// with a more descriptive error
	return r.baseURL + "/" + patterns[0], nil
}

// getCacheKey returns a unique cache key for this URL.
// The key is a SHA-256 hash of the URL, suitable for use as a filename.
func (r *HTTPSRegistry) getCacheKey(url string) string {
	hash := sha256.Sum256([]byte(url))
	// Use https subdirectory and add extension
	return filepath.Join("https", hex.EncodeToString(hash[:])+".tar.gz")
}

// computeFileHash computes the SHA-256 hash of a file.
// Returns the hash in the format "sha256-{hex}".
func computeFileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return "sha256-" + hex.EncodeToString(hash.Sum(nil)), nil
}
