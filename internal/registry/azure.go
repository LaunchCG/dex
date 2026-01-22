// azure.go provides an Azure Blob Storage registry for fetching packages.

package registry

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"

	"github.com/dex-tools/dex/internal/errors"
	"github.com/dex-tools/dex/pkg/version"
)

// AzureRegistry handles az:// sources.
// It supports registry mode (registry.json), package mode (package.json),
// and direct tarball URLs.
//
// Authentication uses the Azure SDK default credential chain:
//   - Environment variables (AZURE_TENANT_ID, AZURE_CLIENT_ID, AZURE_CLIENT_SECRET)
//   - Managed Identity (for Azure VMs, App Service, etc.)
//   - Azure CLI credentials
type AzureRegistry struct {
	url             string
	account         string
	container       string
	prefix          string
	mode            SourceMode
	cache           *Cache
	client          *azblob.Client
	isDirectTarball bool
	tarballInfo     *TarballInfo
}

// NewAzureRegistry creates a registry from an Azure Blob Storage URL.
//
// URL format: az://account/container/path/to/registry/
// Direct tarball: az://account/container/path/to/plugin-1.0.0.tar.gz
//
// Authentication uses Azure SDK default credential chain.
func NewAzureRegistry(url string, mode SourceMode) (*AzureRegistry, error) {
	// Parse the Azure URL
	account, container, prefix, err := parseAzureURL(url)
	if err != nil {
		return nil, errors.NewRegistryError(url, "connect", err)
	}

	// Create cache
	defaultCache, err := DefaultCache()
	if err != nil {
		return nil, errors.NewRegistryError(url, "connect", err)
	}

	// Create Azure credential using default credential chain
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, errors.NewRegistryError(url, "connect",
			fmt.Errorf("failed to create Azure credential: %w", err))
	}

	// Create blob service client
	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net", account)
	client, err := azblob.NewClient(serviceURL, cred, nil)
	if err != nil {
		return nil, errors.NewRegistryError(url, "connect",
			fmt.Errorf("failed to create Azure blob client: %w", err))
	}

	// Check if this is a direct tarball URL
	isDirectTarball := IsTarballURL(url)
	var tarballInfo *TarballInfo

	if isDirectTarball {
		filename := GetFilenameFromURL(url)
		tarballInfo = ParseTarballFilename(filename)
	} else {
		// Normalize prefix - ensure it doesn't end with /
		prefix = strings.TrimSuffix(prefix, "/")
	}

	return &AzureRegistry{
		url:             url,
		account:         account,
		container:       container,
		prefix:          prefix,
		mode:            mode,
		cache:           defaultCache,
		client:          client,
		isDirectTarball: isDirectTarball,
		tarballInfo:     tarballInfo,
	}, nil
}

// Protocol returns "az".
func (r *AzureRegistry) Protocol() string {
	return "az"
}

// GetPackageInfo returns package info based on the mode.
func (r *AzureRegistry) GetPackageInfo(name string) (*PackageInfo, error) {
	if r.isDirectTarball {
		return r.getPackageFromTarball(name)
	}

	if r.mode == ModePackage {
		return r.getPackageFromPackageJSON(name)
	}

	// Registry mode
	return r.getPackageFromRegistry(name)
}

// getPackageFromTarball returns package info extracted from the tarball URL.
func (r *AzureRegistry) getPackageFromTarball(name string) (*PackageInfo, error) {
	if r.tarballInfo == nil {
		return nil, errors.NewRegistryError(r.url, "parse",
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

// getPackageFromPackageJSON fetches and parses package.json for single-package mode.
func (r *AzureRegistry) getPackageFromPackageJSON(name string) (*PackageInfo, error) {
	blobPath := r.prefix + "/package.json"
	if r.prefix == "" {
		blobPath = "package.json"
	}

	data, err := r.downloadBlob(blobPath)
	if err != nil {
		return nil, errors.NewNotFoundError("package.json", r.url)
	}

	var pkg struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		Description string `json:"description"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, errors.NewRegistryError(r.url, "fetch",
			fmt.Errorf("failed to parse package.json: %w", err))
	}

	// If a name is requested, check if it matches
	if name != "" && !NamesMatch(name, pkg.Name) {
		return nil, errors.NewNotFoundError("package", name)
	}

	return &PackageInfo{
		Name:        pkg.Name,
		Versions:    []string{pkg.Version},
		Latest:      pkg.Version,
		Description: pkg.Description,
	}, nil
}

// getPackageFromRegistry fetches registry.json and returns package info.
func (r *AzureRegistry) getPackageFromRegistry(name string) (*PackageInfo, error) {
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
func (r *AzureRegistry) ResolvePackage(name, versionConstraint string) (*ResolvedPackage, error) {
	info, err := r.GetPackageInfo(name)
	if err != nil {
		return nil, err
	}

	// Handle "latest" or empty version
	if versionConstraint == "" || strings.ToLower(versionConstraint) == "latest" {
		if info.Latest == "" && len(info.Versions) > 0 {
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
func (r *AzureRegistry) FetchPackage(resolved *ResolvedPackage, destDir string) (string, error) {
	// Create a cache key based on the URL
	cacheKey := r.getCacheKey(resolved.URL)
	cachePath := r.cache.GetPath(cacheKey)

	// Check if already cached
	if !r.cache.Has(cacheKey) {
		// Parse the Azure URL from resolved.URL
		_, container, blobPath, err := parseAzureURL(resolved.URL)
		if err != nil {
			return "", errors.NewInstallError(resolved.Name, "fetch", err)
		}

		// Download to cache
		data, err := r.downloadBlobFromContainer(container, blobPath)
		if err != nil {
			return "", errors.NewInstallError(resolved.Name, "fetch", err)
		}

		// Ensure cache directory exists
		if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
			return "", errors.NewInstallError(resolved.Name, "fetch", err)
		}

		// Write to cache file
		if err := os.WriteFile(cachePath, data, 0644); err != nil {
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
func (r *AzureRegistry) ListPackages() ([]string, error) {
	if r.isDirectTarball {
		if r.tarballInfo != nil {
			return []string{r.tarballInfo.Name}, nil
		}
		return nil, nil
	}

	if r.mode == ModePackage {
		info, err := r.getPackageFromPackageJSON("")
		if err != nil {
			return nil, err
		}
		return []string{info.Name}, nil
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
func (r *AzureRegistry) fetchRegistryIndex() (*RegistryIndex, error) {
	blobPath := r.prefix + "/registry.json"
	if r.prefix == "" {
		blobPath = "registry.json"
	}

	data, err := r.downloadBlob(blobPath)
	if err != nil {
		return nil, errors.NewRegistryError(r.url, "fetch",
			fmt.Errorf("failed to fetch registry.json: %w", err))
	}

	var index RegistryIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, errors.NewRegistryError(r.url, "fetch",
			fmt.Errorf("failed to parse registry.json: %w", err))
	}

	return &index, nil
}

// downloadBlob downloads a blob from the registry's container.
func (r *AzureRegistry) downloadBlob(blobPath string) ([]byte, error) {
	return r.downloadBlobFromContainer(r.container, blobPath)
}

// downloadBlobFromContainer downloads a blob from a specific container.
func (r *AzureRegistry) downloadBlobFromContainer(container, blobPath string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	resp, err := r.client.DownloadStream(ctx, container, blobPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to download blob az://%s/%s/%s: %w",
			r.account, container, blobPath, err)
	}
	defer resp.Body.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, resp.Body); err != nil {
		return nil, fmt.Errorf("failed to read blob: %w", err)
	}

	return buf.Bytes(), nil
}

// getTarballURL returns the Azure URL for a package tarball.
func (r *AzureRegistry) getTarballURL(name, ver string) (string, error) {
	if r.isDirectTarball {
		return r.url, nil
	}

	// Try different naming conventions
	patterns := []string{
		fmt.Sprintf("%s-%s.tar.gz", name, ver),
		fmt.Sprintf("%s-v%s.tar.gz", name, ver),
		fmt.Sprintf("%s_%s.tar.gz", name, ver),
		fmt.Sprintf("%s-%s.tgz", name, ver),
	}

	for _, pattern := range patterns {
		blobPath := pattern
		if r.prefix != "" {
			blobPath = r.prefix + "/" + pattern
		}

		// Check if blob exists using GetProperties
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_, err := r.client.ServiceClient().NewContainerClient(r.container).NewBlobClient(blobPath).GetProperties(ctx, nil)
		cancel()

		if err == nil {
			return fmt.Sprintf("az://%s/%s/%s", r.account, r.container, blobPath), nil
		}
	}

	// Return first pattern even if not found (let download fail with better error)
	blobPath := patterns[0]
	if r.prefix != "" {
		blobPath = r.prefix + "/" + patterns[0]
	}
	return fmt.Sprintf("az://%s/%s/%s", r.account, r.container, blobPath), nil
}

// getCacheKey returns a unique cache key for this URL.
func (r *AzureRegistry) getCacheKey(url string) string {
	hash := sha256.Sum256([]byte(url))
	return filepath.Join("azure", hex.EncodeToString(hash[:])+".tar.gz")
}

// parseAzureURL parses an Azure Blob Storage URL into account, container, and blob path.
// URL format: az://account/container/path/to/blob
func parseAzureURL(url string) (account, container, blobPath string, err error) {
	if !strings.HasPrefix(url, "az://") {
		return "", "", "", fmt.Errorf("invalid Azure URL: must start with az://")
	}

	// Remove az:// prefix
	path := strings.TrimPrefix(url, "az://")

	// Split into parts
	parts := strings.SplitN(path, "/", 3)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", "", fmt.Errorf("invalid Azure URL: must be az://account/container[/path]")
	}

	account = parts[0]
	container = parts[1]
	if len(parts) > 2 {
		blobPath = parts[2]
	}

	return account, container, blobPath, nil
}
