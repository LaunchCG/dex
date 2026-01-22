package registry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseS3URL(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		wantBucket string
		wantPrefix string
		wantErr    bool
	}{
		{
			name:       "bucket only",
			url:        "s3://my-bucket",
			wantBucket: "my-bucket",
			wantPrefix: "",
			wantErr:    false,
		},
		{
			name:       "bucket with path",
			url:        "s3://my-bucket/path/to/registry",
			wantBucket: "my-bucket",
			wantPrefix: "path/to/registry",
			wantErr:    false,
		},
		{
			name:       "bucket with single path segment",
			url:        "s3://my-bucket/registry",
			wantBucket: "my-bucket",
			wantPrefix: "registry",
			wantErr:    false,
		},
		{
			name:       "bucket with trailing slash",
			url:        "s3://my-bucket/path/",
			wantBucket: "my-bucket",
			wantPrefix: "path/",
			wantErr:    false,
		},
		{
			name:       "tarball URL",
			url:        "s3://my-bucket/plugins/my-plugin-1.0.0.tar.gz",
			wantBucket: "my-bucket",
			wantPrefix: "plugins/my-plugin-1.0.0.tar.gz",
			wantErr:    false,
		},
		{
			name:    "invalid - no s3 prefix",
			url:     "my-bucket/path",
			wantErr: true,
		},
		{
			name:    "invalid - empty bucket",
			url:     "s3://",
			wantErr: true,
		},
		{
			name:    "invalid - https URL",
			url:     "https://s3.amazonaws.com/bucket/key",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bucket, prefix, err := parseS3URL(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantBucket, bucket)
			assert.Equal(t, tt.wantPrefix, prefix)
		})
	}
}

func TestS3Registry_Protocol(t *testing.T) {
	// We can't fully test NewS3Registry without AWS credentials,
	// but we can test the URL parsing and tarball detection logic
	t.Run("tarball detection from URL", func(t *testing.T) {
		url := "s3://bucket/path/plugin-1.0.0.tar.gz"
		assert.True(t, IsTarballURL(url))

		filename := GetFilenameFromURL(url)
		assert.Equal(t, "plugin-1.0.0.tar.gz", filename)

		info := ParseTarballFilename(filename)
		require.NotNil(t, info)
		assert.Equal(t, "plugin", info.Name)
		assert.Equal(t, "1.0.0", info.Version)
	})

	t.Run("registry URL is not tarball", func(t *testing.T) {
		url := "s3://bucket/registry/"
		assert.False(t, IsTarballURL(url))
	})
}

func TestS3Registry_getCacheKey(t *testing.T) {
	// Test that cache keys are deterministic and unique
	t.Run("same URL produces same cache key", func(t *testing.T) {
		url := "s3://bucket/plugin-1.0.0.tar.gz"

		// Since we can't create the registry without AWS creds,
		// test the hash logic directly
		key1 := computeS3CacheKey(url)
		key2 := computeS3CacheKey(url)
		assert.Equal(t, key1, key2)
	})

	t.Run("different URLs produce different cache keys", func(t *testing.T) {
		url1 := "s3://bucket/plugin-1.0.0.tar.gz"
		url2 := "s3://bucket/plugin-2.0.0.tar.gz"

		key1 := computeS3CacheKey(url1)
		key2 := computeS3CacheKey(url2)
		assert.NotEqual(t, key1, key2)
	})
}

// computeS3CacheKey replicates the cache key logic for testing
func computeS3CacheKey(url string) string {
	// This matches the logic in S3Registry.getCacheKey
	// We test it here without needing AWS credentials
	reg := &S3Registry{}
	return reg.getCacheKey(url)
}

func TestS3Registry_TarballInfo(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		wantName    string
		wantVersion string
		isTarball   bool
	}{
		{
			name:        "standard tarball",
			url:         "s3://bucket/my-plugin-1.0.0.tar.gz",
			wantName:    "my-plugin",
			wantVersion: "1.0.0",
			isTarball:   true,
		},
		{
			name:        "tarball with v prefix",
			url:         "s3://bucket/my-plugin-v2.0.0.tar.gz",
			wantName:    "my-plugin",
			wantVersion: "2.0.0",
			isTarball:   true,
		},
		{
			name:        "tarball in nested path",
			url:         "s3://bucket/path/to/plugins/my-plugin-1.2.3.tar.gz",
			wantName:    "my-plugin",
			wantVersion: "1.2.3",
			isTarball:   true,
		},
		{
			name:        "tgz extension",
			url:         "s3://bucket/plugin-3.0.0.tgz",
			wantName:    "plugin",
			wantVersion: "3.0.0",
			isTarball:   true,
		},
		{
			name:      "registry path (not tarball)",
			url:       "s3://bucket/registry/",
			isTarball: false,
		},
		{
			name:      "package path (not tarball)",
			url:       "s3://bucket/my-plugin/",
			isTarball: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isTarball := IsTarballURL(tt.url)
			assert.Equal(t, tt.isTarball, isTarball)

			if tt.isTarball {
				filename := GetFilenameFromURL(tt.url)
				info := ParseTarballFilename(filename)
				require.NotNil(t, info)
				assert.Equal(t, tt.wantName, info.Name)
				assert.Equal(t, tt.wantVersion, info.Version)
			}
		})
	}
}

// TestS3Registry_Integration tests would require actual AWS credentials
// These are marked as integration tests and skipped by default
func TestS3Registry_Integration(t *testing.T) {
	t.Skip("Integration tests require AWS credentials")

	// These tests would verify:
	// 1. Creating a registry with valid AWS credentials
	// 2. GetPackageInfo from a real S3 bucket
	// 3. ResolvePackage with a real registry.json
	// 4. FetchPackage to download and extract a tarball
	// 5. ListPackages to enumerate packages in a registry
}
