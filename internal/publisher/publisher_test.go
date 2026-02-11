package publisher

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/launchcg/dex/internal/registry"
)

func TestParseTarball(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		wantName    string
		wantVersion string
		wantErr     bool
	}{
		{
			name:        "standard format",
			path:        "/path/to/my-plugin-1.0.0.tar.gz",
			wantName:    "my-plugin",
			wantVersion: "1.0.0",
		},
		{
			name:        "with v prefix",
			path:        "plugin-v2.3.4.tar.gz",
			wantName:    "plugin",
			wantVersion: "2.3.4",
		},
		{
			name:        "with underscore",
			path:        "my_plugin_1.0.0.tar.gz",
			wantName:    "my_plugin",
			wantVersion: "1.0.0",
		},
		{
			name:        "tgz extension",
			path:        "plugin-1.2.3.tgz",
			wantName:    "plugin",
			wantVersion: "1.2.3",
		},
		{
			name:        "with prerelease",
			path:        "plugin-1.0.0-beta.1.tar.gz",
			wantName:    "plugin",
			wantVersion: "1.0.0-beta.1",
		},
		{
			name:        "with build metadata",
			path:        "plugin-1.0.0+build.123.tar.gz",
			wantName:    "plugin",
			wantVersion: "1.0.0+build.123",
		},
		{
			name:    "invalid format - no version",
			path:    "plugin.tar.gz",
			wantErr: true,
		},
		{
			name:    "invalid format - no extension",
			path:    "plugin-1.0.0",
			wantErr: true,
		},
		{
			name:    "invalid format - wrong extension",
			path:    "plugin-1.0.0.zip",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := ParseTarball(tt.path)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantName, info.Name)
			assert.Equal(t, tt.wantVersion, info.Version)
		})
	}
}

func TestUpdateRegistryIndex(t *testing.T) {
	t.Run("creates new index when nil", func(t *testing.T) {
		index := UpdateRegistryIndex(nil, "my-plugin", "1.0.0")

		require.NotNil(t, index)
		assert.Equal(t, "dex-registry", index.Name)
		assert.Equal(t, "1.0", index.Version)
		require.Contains(t, index.Packages, "my-plugin")
		assert.Equal(t, []string{"1.0.0"}, index.Packages["my-plugin"].Versions)
		assert.Equal(t, "1.0.0", index.Packages["my-plugin"].Latest)
	})

	t.Run("adds new package to existing index", func(t *testing.T) {
		existing := &registry.RegistryIndex{
			Name:    "test-registry",
			Version: "1.0",
			Packages: map[string]registry.PackageEntry{
				"existing-plugin": {
					Versions: []string{"1.0.0"},
					Latest:   "1.0.0",
				},
			},
		}

		index := UpdateRegistryIndex(existing, "new-plugin", "2.0.0")

		assert.Equal(t, "test-registry", index.Name)
		assert.Len(t, index.Packages, 2)
		require.Contains(t, index.Packages, "new-plugin")
		assert.Equal(t, []string{"2.0.0"}, index.Packages["new-plugin"].Versions)
		assert.Equal(t, "2.0.0", index.Packages["new-plugin"].Latest)
	})

	t.Run("adds new version to existing package", func(t *testing.T) {
		existing := &registry.RegistryIndex{
			Name:    "test-registry",
			Version: "1.0",
			Packages: map[string]registry.PackageEntry{
				"my-plugin": {
					Versions: []string{"1.0.0"},
					Latest:   "1.0.0",
				},
			},
		}

		index := UpdateRegistryIndex(existing, "my-plugin", "1.1.0")

		require.Contains(t, index.Packages, "my-plugin")
		assert.Equal(t, []string{"1.0.0", "1.1.0"}, index.Packages["my-plugin"].Versions)
		assert.Equal(t, "1.1.0", index.Packages["my-plugin"].Latest)
	})

	t.Run("does not duplicate existing version", func(t *testing.T) {
		existing := &registry.RegistryIndex{
			Name:    "test-registry",
			Version: "1.0",
			Packages: map[string]registry.PackageEntry{
				"my-plugin": {
					Versions: []string{"1.0.0", "1.1.0"},
					Latest:   "1.1.0",
				},
			},
		}

		index := UpdateRegistryIndex(existing, "my-plugin", "1.0.0")

		require.Contains(t, index.Packages, "my-plugin")
		assert.Equal(t, []string{"1.0.0", "1.1.0"}, index.Packages["my-plugin"].Versions)
		// Latest should update to the "published" version
		assert.Equal(t, "1.0.0", index.Packages["my-plugin"].Latest)
	})

	t.Run("handles nil packages map", func(t *testing.T) {
		existing := &registry.RegistryIndex{
			Name:     "test-registry",
			Version:  "1.0",
			Packages: nil,
		}

		index := UpdateRegistryIndex(existing, "my-plugin", "1.0.0")

		require.NotNil(t, index.Packages)
		require.Contains(t, index.Packages, "my-plugin")
	})
}

func TestNew(t *testing.T) {
	t.Run("file protocol", func(t *testing.T) {
		tmpDir := t.TempDir()
		pub, err := New("file:" + tmpDir)
		require.NoError(t, err)
		assert.Equal(t, "file", pub.Protocol())
	})

	t.Run("unsupported protocol", func(t *testing.T) {
		_, err := New("ftp://example.com/registry")
		require.Error(t, err)
		assert.EqualError(t, err, "publish error for package to ftp://example.com/registry during connect: unsupported source URL format: ftp://example.com/registry")
	})

	// Note: s3:// and az:// would require credentials, so we skip them in unit tests
}

func TestSortVersions(t *testing.T) {
	versions := []string{"1.2.0", "1.0.0", "2.0.0", "1.1.0"}
	sorted := SortVersions(versions)

	assert.Equal(t, []string{"1.0.0", "1.1.0", "1.2.0", "2.0.0"}, sorted)
}
