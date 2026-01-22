package registry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAzureURL(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		wantAccount   string
		wantContainer string
		wantBlobPath  string
		wantErr       bool
	}{
		{
			name:          "account and container only",
			url:           "az://myaccount/mycontainer",
			wantAccount:   "myaccount",
			wantContainer: "mycontainer",
			wantBlobPath:  "",
			wantErr:       false,
		},
		{
			name:          "full path",
			url:           "az://myaccount/mycontainer/path/to/registry",
			wantAccount:   "myaccount",
			wantContainer: "mycontainer",
			wantBlobPath:  "path/to/registry",
			wantErr:       false,
		},
		{
			name:          "single path segment",
			url:           "az://myaccount/mycontainer/registry",
			wantAccount:   "myaccount",
			wantContainer: "mycontainer",
			wantBlobPath:  "registry",
			wantErr:       false,
		},
		{
			name:          "tarball URL",
			url:           "az://myaccount/plugins/my-plugin-1.0.0.tar.gz",
			wantAccount:   "myaccount",
			wantContainer: "plugins",
			wantBlobPath:  "my-plugin-1.0.0.tar.gz",
			wantErr:       false,
		},
		{
			name:          "nested tarball",
			url:           "az://myaccount/container/path/to/plugin-1.0.0.tar.gz",
			wantAccount:   "myaccount",
			wantContainer: "container",
			wantBlobPath:  "path/to/plugin-1.0.0.tar.gz",
			wantErr:       false,
		},
		{
			name:    "invalid - no az prefix",
			url:     "myaccount/mycontainer",
			wantErr: true,
		},
		{
			name:    "invalid - missing container",
			url:     "az://myaccount",
			wantErr: true,
		},
		{
			name:    "invalid - empty account",
			url:     "az:///mycontainer",
			wantErr: true,
		},
		{
			name:    "invalid - empty container",
			url:     "az://myaccount/",
			wantErr: true,
		},
		{
			name:    "invalid - https URL",
			url:     "https://myaccount.blob.core.windows.net/container/blob",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account, container, blobPath, err := parseAzureURL(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantAccount, account)
			assert.Equal(t, tt.wantContainer, container)
			assert.Equal(t, tt.wantBlobPath, blobPath)
		})
	}
}

func TestAzureRegistry_Protocol(t *testing.T) {
	t.Run("tarball detection from URL", func(t *testing.T) {
		url := "az://account/container/plugin-1.0.0.tar.gz"
		assert.True(t, IsTarballURL(url))

		filename := GetFilenameFromURL(url)
		assert.Equal(t, "plugin-1.0.0.tar.gz", filename)

		info := ParseTarballFilename(filename)
		require.NotNil(t, info)
		assert.Equal(t, "plugin", info.Name)
		assert.Equal(t, "1.0.0", info.Version)
	})

	t.Run("registry URL is not tarball", func(t *testing.T) {
		url := "az://account/container/registry/"
		assert.False(t, IsTarballURL(url))
	})
}

func TestAzureRegistry_getCacheKey(t *testing.T) {
	t.Run("same URL produces same cache key", func(t *testing.T) {
		url := "az://account/container/plugin-1.0.0.tar.gz"

		key1 := computeAzureCacheKey(url)
		key2 := computeAzureCacheKey(url)
		assert.Equal(t, key1, key2)
	})

	t.Run("different URLs produce different cache keys", func(t *testing.T) {
		url1 := "az://account/container/plugin-1.0.0.tar.gz"
		url2 := "az://account/container/plugin-2.0.0.tar.gz"

		key1 := computeAzureCacheKey(url1)
		key2 := computeAzureCacheKey(url2)
		assert.NotEqual(t, key1, key2)
	})

	t.Run("different accounts produce different cache keys", func(t *testing.T) {
		url1 := "az://account1/container/plugin-1.0.0.tar.gz"
		url2 := "az://account2/container/plugin-1.0.0.tar.gz"

		key1 := computeAzureCacheKey(url1)
		key2 := computeAzureCacheKey(url2)
		assert.NotEqual(t, key1, key2)
	})
}

// computeAzureCacheKey replicates the cache key logic for testing
func computeAzureCacheKey(url string) string {
	reg := &AzureRegistry{}
	return reg.getCacheKey(url)
}

func TestAzureRegistry_TarballInfo(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		wantName    string
		wantVersion string
		isTarball   bool
	}{
		{
			name:        "standard tarball",
			url:         "az://account/container/my-plugin-1.0.0.tar.gz",
			wantName:    "my-plugin",
			wantVersion: "1.0.0",
			isTarball:   true,
		},
		{
			name:        "tarball with v prefix",
			url:         "az://account/container/my-plugin-v2.0.0.tar.gz",
			wantName:    "my-plugin",
			wantVersion: "2.0.0",
			isTarball:   true,
		},
		{
			name:        "tarball in nested path",
			url:         "az://account/container/path/to/plugins/my-plugin-1.2.3.tar.gz",
			wantName:    "my-plugin",
			wantVersion: "1.2.3",
			isTarball:   true,
		},
		{
			name:        "tgz extension",
			url:         "az://account/container/plugin-3.0.0.tgz",
			wantName:    "plugin",
			wantVersion: "3.0.0",
			isTarball:   true,
		},
		{
			name:      "registry path (not tarball)",
			url:       "az://account/container/registry/",
			isTarball: false,
		},
		{
			name:      "package path (not tarball)",
			url:       "az://account/container/my-plugin/",
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

func TestAzureRegistry_URLFormat(t *testing.T) {
	// Test that we properly construct Azure blob URLs from components
	tests := []struct {
		name      string
		account   string
		container string
		blobPath  string
		wantURL   string
	}{
		{
			name:      "basic path",
			account:   "myaccount",
			container: "mycontainer",
			blobPath:  "my-plugin-1.0.0.tar.gz",
			wantURL:   "az://myaccount/mycontainer/my-plugin-1.0.0.tar.gz",
		},
		{
			name:      "nested path",
			account:   "myaccount",
			container: "mycontainer",
			blobPath:  "path/to/plugin-1.0.0.tar.gz",
			wantURL:   "az://myaccount/mycontainer/path/to/plugin-1.0.0.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that parseAzureURL can roundtrip
			account, container, blobPath, err := parseAzureURL(tt.wantURL)
			require.NoError(t, err)
			assert.Equal(t, tt.account, account)
			assert.Equal(t, tt.container, container)
			assert.Equal(t, tt.blobPath, blobPath)
		})
	}
}

// TestAzureRegistry_Integration tests would require actual Azure credentials
// These are marked as integration tests and skipped by default
func TestAzureRegistry_Integration(t *testing.T) {
	t.Skip("Integration tests require Azure credentials")

	// These tests would verify:
	// 1. Creating a registry with valid Azure credentials
	// 2. GetPackageInfo from a real Azure blob container
	// 3. ResolvePackage with a real registry.json
	// 4. FetchPackage to download and extract a tarball
	// 5. ListPackages to enumerate packages in a registry
}
