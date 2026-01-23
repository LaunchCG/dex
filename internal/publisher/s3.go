package publisher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/dex-tools/dex/internal/errors"
	"github.com/dex-tools/dex/internal/registry"
)

// S3Publisher publishes packages to an S3 bucket.
type S3Publisher struct {
	url    string
	bucket string
	prefix string
	client *s3.Client
}

// NewS3Publisher creates a publisher for an S3 registry.
// URL format: s3://bucket/prefix
func NewS3Publisher(url string) (*S3Publisher, error) {
	bucket, prefix, err := parseS3URL(url)
	if err != nil {
		return nil, errors.NewPublishError("", url, "connect", err)
	}

	// Load AWS config
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, errors.NewPublishError("", url, "connect",
			fmt.Errorf("failed to load AWS config: %w", err))
	}

	client := s3.NewFromConfig(cfg)

	return &S3Publisher{
		url:    url,
		bucket: bucket,
		prefix: strings.TrimSuffix(prefix, "/"),
		client: client,
	}, nil
}

// Protocol returns "s3".
func (p *S3Publisher) Protocol() string {
	return "s3"
}

// Publish uploads the tarball to S3 and updates registry.json.
func (p *S3Publisher) Publish(tarballPath string) (*PublishResult, error) {
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
	tarballKey := p.getKey(filepath.Base(tarballPath))
	if err := p.uploadObject(tarballKey, tarballData, "application/gzip"); err != nil {
		return nil, errors.NewPublishError(info.Name, p.url, "upload", err)
	}

	// Update registry.json
	if err := p.updateIndex(info.Name, info.Version); err != nil {
		return nil, errors.NewPublishError(info.Name, p.url, "index", err)
	}

	tarballURL := fmt.Sprintf("s3://%s/%s", p.bucket, tarballKey)
	return &PublishResult{
		Name:      info.Name,
		Version:   info.Version,
		URL:       tarballURL,
		Integrity: integrity,
	}, nil
}

// updateIndex downloads registry.json, updates it, and re-uploads.
func (p *S3Publisher) updateIndex(name, version string) error {
	indexKey := p.getKey("registry.json")

	// Try to download existing index
	var index *registry.RegistryIndex

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	output, err := p.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(indexKey),
	})
	if err == nil {
		defer output.Body.Close()
		index = &registry.RegistryIndex{}
		if err := json.NewDecoder(output.Body).Decode(index); err != nil {
			return fmt.Errorf("failed to parse registry.json: %w", err)
		}
	}
	// If the object doesn't exist, index will be nil and UpdateRegistryIndex will create a new one

	// Update index
	index = UpdateRegistryIndex(index, name, version)

	// Marshal index
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry.json: %w", err)
	}

	// Upload index
	if err := p.uploadObject(indexKey, data, "application/json"); err != nil {
		return fmt.Errorf("failed to upload registry.json: %w", err)
	}

	return nil
}

// uploadObject uploads data to S3.
func (p *S3Publisher) uploadObject(key string, data []byte, contentType string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	_, err := p.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(p.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("failed to upload to s3://%s/%s: %w", p.bucket, key, err)
	}

	return nil
}

// getKey returns the full S3 key for a filename.
func (p *S3Publisher) getKey(filename string) string {
	if p.prefix == "" {
		return filename
	}
	return p.prefix + "/" + filename
}

// parseS3URL parses an S3 URL into bucket and prefix.
func parseS3URL(url string) (bucket, prefix string, err error) {
	if !strings.HasPrefix(url, "s3://") {
		return "", "", fmt.Errorf("invalid S3 URL: must start with s3://")
	}

	path := strings.TrimPrefix(url, "s3://")
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
