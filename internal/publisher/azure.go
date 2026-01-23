package publisher

import (
	"context"
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
	"github.com/dex-tools/dex/internal/registry"
)

// AzurePublisher publishes packages to Azure Blob Storage.
type AzurePublisher struct {
	url       string
	account   string
	container string
	prefix    string
	client    *azblob.Client
}

// NewAzurePublisher creates a publisher for an Azure Blob Storage registry.
// URL format: az://account/container/prefix
func NewAzurePublisher(url string) (*AzurePublisher, error) {
	account, container, prefix, err := parseAzureURL(url)
	if err != nil {
		return nil, errors.NewPublishError("", url, "connect", err)
	}

	// Create Azure credential
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, errors.NewPublishError("", url, "connect",
			fmt.Errorf("failed to create Azure credential: %w", err))
	}

	// Create blob client
	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net", account)
	client, err := azblob.NewClient(serviceURL, cred, nil)
	if err != nil {
		return nil, errors.NewPublishError("", url, "connect",
			fmt.Errorf("failed to create Azure blob client: %w", err))
	}

	return &AzurePublisher{
		url:       url,
		account:   account,
		container: container,
		prefix:    strings.TrimSuffix(prefix, "/"),
		client:    client,
	}, nil
}

// Protocol returns "az".
func (p *AzurePublisher) Protocol() string {
	return "az"
}

// Publish uploads the tarball to Azure and updates registry.json.
func (p *AzurePublisher) Publish(tarballPath string) (*PublishResult, error) {
	// Parse tarball to get name and version
	info, err := ParseTarball(tarballPath)
	if err != nil {
		return nil, errors.NewPublishError("", p.url, "validate", err)
	}

	// Compute integrity hash
	integrity, err := ComputeTarballHash(tarballPath)
	if err != nil {
		return nil, errors.NewPublishError(info.Name, p.url, "validate", err)
	}

	// Read tarball content
	tarballData, err := os.ReadFile(tarballPath)
	if err != nil {
		return nil, errors.NewPublishError(info.Name, p.url, "upload",
			fmt.Errorf("failed to read tarball: %w", err))
	}

	// Upload tarball
	tarballBlobPath := p.getBlobPath(filepath.Base(tarballPath))
	if err := p.uploadBlob(tarballBlobPath, tarballData); err != nil {
		return nil, errors.NewPublishError(info.Name, p.url, "upload", err)
	}

	// Update registry.json
	if err := p.updateIndex(info.Name, info.Version); err != nil {
		return nil, errors.NewPublishError(info.Name, p.url, "index", err)
	}

	tarballURL := fmt.Sprintf("az://%s/%s/%s", p.account, p.container, tarballBlobPath)
	return &PublishResult{
		Name:      info.Name,
		Version:   info.Version,
		URL:       tarballURL,
		Integrity: integrity,
	}, nil
}

// updateIndex downloads registry.json, updates it, and re-uploads.
func (p *AzurePublisher) updateIndex(name, version string) error {
	indexBlobPath := p.getBlobPath("registry.json")

	// Try to download existing index
	var index *registry.RegistryIndex

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := p.client.DownloadStream(ctx, p.container, indexBlobPath, nil)
	if err == nil {
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if err == nil {
			index = &registry.RegistryIndex{}
			if err := json.Unmarshal(data, index); err != nil {
				return fmt.Errorf("failed to parse registry.json: %w", err)
			}
		}
	}
	// If the blob doesn't exist, index will be nil and UpdateRegistryIndex will create a new one

	// Update index
	index = UpdateRegistryIndex(index, name, version)

	// Marshal index
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry.json: %w", err)
	}

	// Upload index
	if err := p.uploadBlob(indexBlobPath, data); err != nil {
		return fmt.Errorf("failed to upload registry.json: %w", err)
	}

	return nil
}

// uploadBlob uploads data to Azure Blob Storage.
func (p *AzurePublisher) uploadBlob(blobPath string, data []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	_, err := p.client.UploadBuffer(ctx, p.container, blobPath, data, nil)
	if err != nil {
		return fmt.Errorf("failed to upload to az://%s/%s/%s: %w", p.account, p.container, blobPath, err)
	}

	return nil
}

// getBlobPath returns the full blob path for a filename.
func (p *AzurePublisher) getBlobPath(filename string) string {
	if p.prefix == "" {
		return filename
	}
	return p.prefix + "/" + filename
}

// parseAzureURL parses an Azure Blob Storage URL into account, container, and prefix.
func parseAzureURL(url string) (account, container, prefix string, err error) {
	if !strings.HasPrefix(url, "az://") {
		return "", "", "", fmt.Errorf("invalid Azure URL: must start with az://")
	}

	path := strings.TrimPrefix(url, "az://")
	parts := strings.SplitN(path, "/", 3)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", "", fmt.Errorf("invalid Azure URL: must be az://account/container[/prefix]")
	}

	account = parts[0]
	container = parts[1]
	if len(parts) > 2 {
		prefix = parts[2]
	}

	return account, container, prefix, nil
}
