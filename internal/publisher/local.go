package publisher

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/dex-tools/dex/internal/errors"
	"github.com/dex-tools/dex/internal/registry"
)

// LocalPublisher publishes packages to a local filesystem registry.
type LocalPublisher struct {
	basePath string
}

// NewLocalPublisher creates a publisher for a local filesystem registry.
func NewLocalPublisher(path string) (*LocalPublisher, error) {
	// Resolve to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, errors.NewPublishError("", "file:"+path, "connect",
			fmt.Errorf("failed to resolve path: %w", err))
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(absPath, 0755); err != nil {
		return nil, errors.NewPublishError("", "file:"+path, "connect",
			fmt.Errorf("failed to create registry directory: %w", err))
	}

	return &LocalPublisher{
		basePath: absPath,
	}, nil
}

// Protocol returns "file".
func (p *LocalPublisher) Protocol() string {
	return "file"
}

// Publish copies the tarball to the registry and updates registry.json.
func (p *LocalPublisher) Publish(tarballPath string) (*PublishResult, error) {
	// Parse tarball to get name and version
	info, err := ParseTarball(tarballPath)
	if err != nil {
		return nil, errors.NewPublishError("", "file:"+p.basePath, "validate", err)
	}

	// Compute integrity hash
	integrity, err := ComputeTarballHash(tarballPath)
	if err != nil {
		return nil, errors.NewPublishError(info.Name, "file:"+p.basePath, "validate", err)
	}

	// Copy tarball to registry
	destPath := filepath.Join(p.basePath, filepath.Base(tarballPath))
	if err := copyFile(tarballPath, destPath); err != nil {
		return nil, errors.NewPublishError(info.Name, "file:"+p.basePath, "upload", err)
	}

	// Update registry.json
	if err := p.updateIndex(info.Name, info.Version); err != nil {
		// Clean up copied tarball on failure
		os.Remove(destPath)
		return nil, errors.NewPublishError(info.Name, "file:"+p.basePath, "index", err)
	}

	return &PublishResult{
		Name:      info.Name,
		Version:   info.Version,
		URL:       "file:" + destPath,
		Integrity: integrity,
	}, nil
}

// updateIndex updates the registry.json file with the new package version.
func (p *LocalPublisher) updateIndex(name, version string) error {
	indexPath := filepath.Join(p.basePath, "registry.json")

	// Load existing index or create new one
	var index *registry.RegistryIndex

	data, err := os.ReadFile(indexPath)
	if err == nil {
		index = &registry.RegistryIndex{}
		if err := json.Unmarshal(data, index); err != nil {
			return fmt.Errorf("failed to parse registry.json: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to read registry.json: %w", err)
	}

	// Update index
	index = UpdateRegistryIndex(index, name, version)

	// Write back
	data, err = json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry.json: %w", err)
	}

	if err := os.WriteFile(indexPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write registry.json: %w", err)
	}

	return nil
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}
