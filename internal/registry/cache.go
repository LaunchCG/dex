package registry

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DefaultCacheDir is the default cache directory relative to the user's home directory.
const DefaultCacheDir = ".dex/cache"

// Cache manages downloaded packages.
// It provides a simple key-value store for caching downloaded content.
type Cache struct {
	// Dir is the base cache directory
	Dir string
}

// DefaultCache returns a cache using the default location (~/.dex/cache).
func DefaultCache() (*Cache, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	cacheDir := filepath.Join(homeDir, DefaultCacheDir)
	return NewCache(cacheDir), nil
}

// NewCache creates a cache at the specified directory.
// The directory is created if it doesn't exist when needed.
func NewCache(baseDir string) *Cache {
	return &Cache{Dir: baseDir}
}

// GetPath returns the cache path for a given key.
func (c *Cache) GetPath(key string) string {
	return filepath.Join(c.Dir, key)
}

// Has returns true if the cache contains the given key.
func (c *Cache) Has(key string) bool {
	path := c.GetPath(key)
	_, err := os.Stat(path)
	return err == nil
}

// GetCacheDir returns the cache directory for a specific protocol.
// This allows organizing cached content by protocol (e.g., git, https, file).
func (c *Cache) GetCacheDir(protocol string) string {
	return filepath.Join(c.Dir, protocol)
}

// EnsureDir ensures the cache directory for a protocol exists.
func (c *Cache) EnsureDir(protocol string) error {
	dir := c.GetCacheDir(protocol)
	return os.MkdirAll(dir, 0755)
}

// Clear removes all cached content for a specific protocol.
// If protocol is empty, clears the entire cache.
func (c *Cache) Clear(protocol string) error {
	var targetDir string
	if protocol == "" {
		targetDir = c.Dir
	} else {
		targetDir = c.GetCacheDir(protocol)
	}

	if err := os.RemoveAll(targetDir); err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}
	return nil
}

// ComputeIntegrity computes a sha256-{base64} integrity hash for a directory or file.
// For directories, it hashes all files recursively in a deterministic order.
// For files, it hashes the file contents directly.
//
// The hash format follows the Subresource Integrity (SRI) specification:
// "sha256-{base64-encoded-hash}"
func ComputeIntegrity(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("failed to stat path: %w", err)
	}

	if info.IsDir() {
		return computeDirectoryIntegrity(path)
	}
	return computeFileIntegrity(path)
}

// computeFileIntegrity computes the integrity hash of a single file.
func computeFileIntegrity(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return "sha256-" + base64.StdEncoding.EncodeToString(hash.Sum(nil)), nil
}

// computeDirectoryIntegrity computes the integrity hash of a directory.
// Files are processed in sorted order for deterministic results.
// The hash includes both file paths (relative to the directory) and contents.
func computeDirectoryIntegrity(dirPath string) (string, error) {
	hash := sha256.New()

	// Collect all files with relative paths
	var files []string
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories themselves (we'll hash their contents)
		if info.IsDir() {
			// Skip .git directories
			if info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return err
		}

		// Normalize path separators for cross-platform consistency
		relPath = filepath.ToSlash(relPath)
		files = append(files, relPath)
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to walk directory: %w", err)
	}

	// Sort files for deterministic ordering
	sort.Strings(files)

	// Hash each file: path + content
	for _, relPath := range files {
		// Write the relative path (for structural integrity)
		hash.Write([]byte(relPath))
		hash.Write([]byte{0}) // null separator

		// Write file content
		absPath := filepath.Join(dirPath, filepath.FromSlash(relPath))
		file, err := os.Open(absPath)
		if err != nil {
			return "", fmt.Errorf("failed to open file %s: %w", relPath, err)
		}

		if _, err := io.Copy(hash, file); err != nil {
			file.Close()
			return "", fmt.Errorf("failed to read file %s: %w", relPath, err)
		}
		file.Close()

		hash.Write([]byte{0}) // null separator between files
	}

	return "sha256-" + base64.StdEncoding.EncodeToString(hash.Sum(nil)), nil
}

// VerifyIntegrity checks if a path matches the expected integrity hash.
// Returns nil if the integrity matches, or an error describing the mismatch.
func VerifyIntegrity(path, expected string) error {
	if expected == "" {
		// No integrity to verify
		return nil
	}

	// Validate expected format
	if !strings.HasPrefix(expected, "sha256-") {
		return fmt.Errorf("unsupported integrity format: %s (expected sha256-{base64})", expected)
	}

	actual, err := ComputeIntegrity(path)
	if err != nil {
		return fmt.Errorf("failed to compute integrity: %w", err)
	}

	if actual != expected {
		return fmt.Errorf("integrity mismatch: expected %s, got %s", expected, actual)
	}

	return nil
}
