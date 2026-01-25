// Package packer provides functionality for creating distributable plugin tarballs.
package packer

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/launchcg/dex/internal/config"
	"github.com/launchcg/dex/internal/errors"
)

// DefaultExcludes are the default patterns to exclude when packing.
var DefaultExcludes = []string{
	".git",
	"node_modules",
	"__pycache__",
	".env",
	"*.pyc",
	"build",
	"dist",
	".DS_Store",
	"*.swp",
	"*.swo",
	".vscode",
	".idea",
}

// PackResult contains the result of a pack operation.
type PackResult struct {
	// Path is the output tarball path
	Path string
	// Size is the size in bytes
	Size int64
	// Integrity is the SHA-256 hash in format "sha256-{hex}"
	Integrity string
	// Name is the package name from package.hcl
	Name string
	// Version is the package version from package.hcl
	Version string
}

// Packer handles creating distributable tarballs from plugin directories.
type Packer struct {
	dir       string
	pkgConfig *config.PackageConfig
	excludes  []string
}

// New creates a new Packer for the given directory.
// It loads and validates package.hcl to get name/version.
func New(dir string) (*Packer, error) {
	// Resolve to absolute path
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, errors.NewPackError(dir, "read", fmt.Errorf("failed to resolve path: %w", err))
	}

	// Check directory exists
	info, err := os.Stat(absDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.NewPackError(dir, "read", fmt.Errorf("directory does not exist: %s", absDir))
		}
		return nil, errors.NewPackError(dir, "read", err)
	}
	if !info.IsDir() {
		return nil, errors.NewPackError(dir, "read", fmt.Errorf("not a directory: %s", absDir))
	}

	// Load package.hcl
	pkgConfig, err := config.LoadPackage(absDir)
	if err != nil {
		return nil, errors.NewPackError(dir, "read", fmt.Errorf("failed to load package.hcl: %w", err))
	}

	// Validate package config
	if err := pkgConfig.Validate(); err != nil {
		return nil, errors.NewPackError(dir, "validate", err)
	}

	return &Packer{
		dir:       absDir,
		pkgConfig: pkgConfig,
		excludes:  DefaultExcludes,
	}, nil
}

// WithExcludes sets custom exclude patterns.
func (p *Packer) WithExcludes(excludes []string) *Packer {
	p.excludes = excludes
	return p
}

// Pack creates a tarball from the plugin directory.
// If output is empty, it defaults to {name}-{version}.tar.gz in the current directory.
func (p *Packer) Pack(output string) (*PackResult, error) {
	name := p.pkgConfig.Package.Name
	version := p.pkgConfig.Package.Version

	// Default output filename
	if output == "" {
		output = fmt.Sprintf("%s-%s.tar.gz", name, version)
	}

	// Resolve output to absolute path
	absOutput, err := filepath.Abs(output)
	if err != nil {
		return nil, errors.NewPackError(p.dir, "compress", fmt.Errorf("failed to resolve output path: %w", err))
	}

	// Create output file
	outFile, err := os.Create(absOutput)
	if err != nil {
		return nil, errors.NewPackError(p.dir, "compress", fmt.Errorf("failed to create output file: %w", err))
	}
	defer outFile.Close()

	// Create hash writer to compute integrity while writing
	hash := sha256.New()
	multiWriter := io.MultiWriter(outFile, hash)

	// Create gzip writer
	gzw := gzip.NewWriter(multiWriter)
	defer gzw.Close()

	// Create tar writer
	tw := tar.NewWriter(gzw)
	defer tw.Close()

	// Top-level directory in tarball
	topDir := fmt.Sprintf("%s-%s", name, version)

	// Walk directory and add files
	err = filepath.Walk(p.dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(p.dir, path)
		if err != nil {
			return err
		}

		// Skip root
		if relPath == "." {
			return nil
		}

		// Check if excluded
		if p.shouldExclude(relPath, info.IsDir()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("failed to create header for %s: %w", relPath, err)
		}

		// Set name with top-level directory prefix
		header.Name = filepath.ToSlash(filepath.Join(topDir, relPath))

		// Handle symlinks
		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return fmt.Errorf("failed to read symlink %s: %w", relPath, err)
			}
			header.Linkname = link
		}

		// Write header
		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write header for %s: %w", relPath, err)
		}

		// Write file content if regular file
		if info.Mode().IsRegular() {
			file, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open %s: %w", relPath, err)
			}
			defer file.Close()

			if _, err := io.Copy(tw, file); err != nil {
				return fmt.Errorf("failed to write %s: %w", relPath, err)
			}
		}

		return nil
	})

	if err != nil {
		// Clean up partial file
		outFile.Close()
		os.Remove(absOutput)
		return nil, errors.NewPackError(p.dir, "compress", err)
	}

	// Close writers to flush
	if err := tw.Close(); err != nil {
		os.Remove(absOutput)
		return nil, errors.NewPackError(p.dir, "compress", fmt.Errorf("failed to close tar writer: %w", err))
	}
	if err := gzw.Close(); err != nil {
		os.Remove(absOutput)
		return nil, errors.NewPackError(p.dir, "compress", fmt.Errorf("failed to close gzip writer: %w", err))
	}

	// Get file size
	fileInfo, err := os.Stat(absOutput)
	if err != nil {
		return nil, errors.NewPackError(p.dir, "compress", fmt.Errorf("failed to stat output file: %w", err))
	}

	return &PackResult{
		Path:      absOutput,
		Size:      fileInfo.Size(),
		Integrity: "sha256-" + hex.EncodeToString(hash.Sum(nil)),
		Name:      name,
		Version:   version,
	}, nil
}

// shouldExclude checks if a path should be excluded.
func (p *Packer) shouldExclude(relPath string, isDir bool) bool {
	// Get the base name for matching
	baseName := filepath.Base(relPath)

	for _, pattern := range p.excludes {
		// Handle glob patterns
		if strings.Contains(pattern, "*") {
			matched, err := filepath.Match(pattern, baseName)
			if err == nil && matched {
				return true
			}
			continue
		}

		// Exact match on base name
		if baseName == pattern {
			return true
		}

		// Check if any path component matches
		parts := strings.Split(filepath.ToSlash(relPath), "/")
		for _, part := range parts {
			if part == pattern {
				return true
			}
		}
	}

	return false
}

// Name returns the package name.
func (p *Packer) Name() string {
	return p.pkgConfig.Package.Name
}

// Version returns the package version.
func (p *Packer) Version() string {
	return p.pkgConfig.Package.Version
}
