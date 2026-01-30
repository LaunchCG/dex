// s3.go provides an S3 registry for fetching packages from Amazon S3.

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

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/launchcg/dex/internal/errors"
	"github.com/launchcg/dex/pkg/version"
)

// S3Registry handles s3:// sources.
// It supports registry mode (registry.json) and direct tarball URLs.
//
// Authentication uses the AWS SDK default credential chain:
//   - Environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY)
//   - Shared credentials file (~/.aws/credentials)
//   - IAM role (for EC2/ECS/Lambda)
type S3Registry struct {
	url             string
	bucket          string
	prefix          string
	mode            SourceMode
	cache           *Cache
	client          *s3.Client
	isDirectTarball bool
	tarballInfo     *TarballInfo
}

// NewS3Registry creates a registry from an S3 URL.
//
// URL format: s3://bucket/path/to/registry/
// Direct tarball: s3://bucket/path/to/plugin-1.0.0.tar.gz
//
// Authentication uses AWS SDK default credential chain.
func NewS3Registry(url string, mode SourceMode) (*S3Registry, error) {
	// Parse the S3 URL
	bucket, prefix, err := parseS3URL(url)
	if err != nil {
		return nil, errors.NewRegistryError(url, "connect", err)
	}

	// Create cache
	defaultCache, err := DefaultCache()
	if err != nil {
		return nil, errors.NewRegistryError(url, "connect", err)
	}

	// Load AWS config with default credential chain
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, errors.NewRegistryError(url, "connect",
			fmt.Errorf("failed to load AWS config: %w", err))
	}

	// Create S3 client
	client := s3.NewFromConfig(cfg)

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

	return &S3Registry{
		url:             url,
		bucket:          bucket,
		prefix:          prefix,
		mode:            mode,
		cache:           defaultCache,
		client:          client,
		isDirectTarball: isDirectTarball,
		tarballInfo:     tarballInfo,
	}, nil
}

// Protocol returns "s3".
func (r *S3Registry) Protocol() string {
	return "s3"
}

// GetPackageInfo returns package info based on the mode.
func (r *S3Registry) GetPackageInfo(name string) (*PackageInfo, error) {
	if r.isDirectTarball {
		return r.getPackageFromTarball(name)
	}

	if r.mode == ModePackage {
		return nil, errors.NewRegistryError(r.url, "fetch",
			fmt.Errorf("S3 sources do not support package mode; use registry mode or direct tarball URL"))
	}

	// Registry mode
	return r.getPackageFromRegistry(name)
}

// getPackageFromTarball returns package info extracted from the tarball URL.
func (r *S3Registry) getPackageFromTarball(name string) (*PackageInfo, error) {
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

// getPackageFromRegistry fetches registry.json and returns package info.
func (r *S3Registry) getPackageFromRegistry(name string) (*PackageInfo, error) {
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
func (r *S3Registry) ResolvePackage(name, versionConstraint string) (*ResolvedPackage, error) {
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
func (r *S3Registry) FetchPackage(resolved *ResolvedPackage, destDir string) (string, error) {
	// Create a cache key based on the URL
	cacheKey := r.getCacheKey(resolved.URL)
	cachePath := r.cache.GetPath(cacheKey)

	// Check if already cached
	if !r.cache.Has(cacheKey) {
		// Parse the S3 URL from resolved.URL
		bucket, key, err := parseS3URL(resolved.URL)
		if err != nil {
			return "", errors.NewInstallError(resolved.Name, "fetch", err)
		}

		// Download to cache
		data, err := r.downloadObjectFromBucket(bucket, key)
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
func (r *S3Registry) ListPackages() ([]string, error) {
	if r.isDirectTarball {
		if r.tarballInfo != nil {
			return []string{r.tarballInfo.Name}, nil
		}
		return nil, nil
	}

	if r.mode == ModePackage {
		return nil, errors.NewRegistryError(r.url, "list",
			fmt.Errorf("S3 sources do not support package mode; use registry mode or direct tarball URL"))
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
func (r *S3Registry) fetchRegistryIndex() (*RegistryIndex, error) {
	key := r.prefix + "/registry.json"
	if r.prefix == "" {
		key = "registry.json"
	}

	data, err := r.downloadObject(key)
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

// downloadObject downloads an object from the registry's bucket.
func (r *S3Registry) downloadObject(key string) ([]byte, error) {
	return r.downloadObjectFromBucket(r.bucket, key)
}

// downloadObjectFromBucket downloads an object from a specific bucket.
func (r *S3Registry) downloadObjectFromBucket(bucket, key string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	output, err := r.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object s3://%s/%s: %w", bucket, key, err)
	}
	defer output.Body.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, output.Body); err != nil {
		return nil, fmt.Errorf("failed to read object: %w", err)
	}

	return buf.Bytes(), nil
}

// getTarballURL returns the S3 URL for a package tarball.
func (r *S3Registry) getTarballURL(name, ver string) (string, error) {
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
		key := pattern
		if r.prefix != "" {
			key = r.prefix + "/" + pattern
		}

		// Check if object exists
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_, err := r.client.HeadObject(ctx, &s3.HeadObjectInput{
			Bucket: aws.String(r.bucket),
			Key:    aws.String(key),
		})
		cancel()

		if err == nil {
			return fmt.Sprintf("s3://%s/%s", r.bucket, key), nil
		}
	}

	// Return first pattern even if not found (let download fail with better error)
	key := patterns[0]
	if r.prefix != "" {
		key = r.prefix + "/" + patterns[0]
	}
	return fmt.Sprintf("s3://%s/%s", r.bucket, key), nil
}

// getCacheKey returns a unique cache key for this URL.
func (r *S3Registry) getCacheKey(url string) string {
	hash := sha256.Sum256([]byte(url))
	return filepath.Join("s3", hex.EncodeToString(hash[:])+".tar.gz")
}

// parseS3URL parses an S3 URL into bucket and key/prefix.
// URL format: s3://bucket/path/to/object
func parseS3URL(url string) (bucket, prefix string, err error) {
	if !strings.HasPrefix(url, "s3://") {
		return "", "", fmt.Errorf("invalid S3 URL: must start with s3://")
	}

	// Remove s3:// prefix
	path := strings.TrimPrefix(url, "s3://")

	// Split into bucket and key
	parts := strings.SplitN(path, "/", 2)
	if len(parts) == 0 || parts[0] == "" {
		return "", "", fmt.Errorf("invalid S3 URL: missing bucket name")
	}

	bucket = parts[0]
	if len(parts) > 1 {
		prefix = parts[1]
	}

	return bucket, prefix, nil
}
