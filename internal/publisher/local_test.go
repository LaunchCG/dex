package publisher

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dex-tools/dex/internal/registry"
)

func TestLocalPublisher(t *testing.T) {
	t.Run("publishes tarball and creates registry.json", func(t *testing.T) {
		registryDir := t.TempDir()
		tarball := createTestTarball(t, "my-plugin", "1.0.0")
		defer os.Remove(tarball)

		pub, err := NewLocalPublisher(registryDir)
		require.NoError(t, err)

		result, err := pub.Publish(tarball)
		require.NoError(t, err)

		// Verify result
		assert.Equal(t, "my-plugin", result.Name)
		assert.Equal(t, "1.0.0", result.Version)
		assert.Contains(t, result.URL, "my-plugin-1.0.0.tar.gz")
		assert.True(t, len(result.Integrity) > 0)

		// Verify tarball was copied
		destPath := filepath.Join(registryDir, "my-plugin-1.0.0.tar.gz")
		_, err = os.Stat(destPath)
		require.NoError(t, err)

		// Verify registry.json was created
		indexPath := filepath.Join(registryDir, "registry.json")
		data, err := os.ReadFile(indexPath)
		require.NoError(t, err)

		var index registry.RegistryIndex
		require.NoError(t, json.Unmarshal(data, &index))

		assert.Equal(t, "dex-registry", index.Name)
		require.Contains(t, index.Packages, "my-plugin")
		assert.Equal(t, []string{"1.0.0"}, index.Packages["my-plugin"].Versions)
		assert.Equal(t, "1.0.0", index.Packages["my-plugin"].Latest)
	})

	t.Run("updates existing registry.json", func(t *testing.T) {
		registryDir := t.TempDir()

		// Create initial registry.json
		initialIndex := registry.RegistryIndex{
			Name:    "existing-registry",
			Version: "1.0",
			Packages: map[string]registry.PackageEntry{
				"existing-plugin": {
					Versions: []string{"1.0.0"},
					Latest:   "1.0.0",
				},
			},
		}
		data, _ := json.Marshal(initialIndex)
		require.NoError(t, os.WriteFile(filepath.Join(registryDir, "registry.json"), data, 0644))

		tarball := createTestTarball(t, "new-plugin", "2.0.0")
		defer os.Remove(tarball)

		pub, err := NewLocalPublisher(registryDir)
		require.NoError(t, err)

		_, err = pub.Publish(tarball)
		require.NoError(t, err)

		// Verify registry.json was updated
		updatedData, err := os.ReadFile(filepath.Join(registryDir, "registry.json"))
		require.NoError(t, err)

		var index registry.RegistryIndex
		require.NoError(t, json.Unmarshal(updatedData, &index))

		// Should preserve existing registry name
		assert.Equal(t, "existing-registry", index.Name)
		assert.Len(t, index.Packages, 2)
		assert.Contains(t, index.Packages, "existing-plugin")
		assert.Contains(t, index.Packages, "new-plugin")
	})

	t.Run("adds new version to existing package", func(t *testing.T) {
		registryDir := t.TempDir()

		// Publish v1.0.0
		tarball1 := createTestTarball(t, "my-plugin", "1.0.0")
		defer os.Remove(tarball1)

		pub, err := NewLocalPublisher(registryDir)
		require.NoError(t, err)

		_, err = pub.Publish(tarball1)
		require.NoError(t, err)

		// Publish v1.1.0
		tarball2 := createTestTarball(t, "my-plugin", "1.1.0")
		defer os.Remove(tarball2)

		_, err = pub.Publish(tarball2)
		require.NoError(t, err)

		// Verify registry.json has both versions
		data, err := os.ReadFile(filepath.Join(registryDir, "registry.json"))
		require.NoError(t, err)

		var index registry.RegistryIndex
		require.NoError(t, json.Unmarshal(data, &index))

		require.Contains(t, index.Packages, "my-plugin")
		assert.Equal(t, []string{"1.0.0", "1.1.0"}, index.Packages["my-plugin"].Versions)
		assert.Equal(t, "1.1.0", index.Packages["my-plugin"].Latest)
	})

	t.Run("creates registry directory if it doesn't exist", func(t *testing.T) {
		parentDir := t.TempDir()
		registryDir := filepath.Join(parentDir, "new", "registry")

		pub, err := NewLocalPublisher(registryDir)
		require.NoError(t, err)

		tarball := createTestTarball(t, "my-plugin", "1.0.0")
		defer os.Remove(tarball)

		_, err = pub.Publish(tarball)
		require.NoError(t, err)

		// Verify directory was created
		_, err = os.Stat(registryDir)
		require.NoError(t, err)
	})

	t.Run("returns error for invalid tarball name", func(t *testing.T) {
		registryDir := t.TempDir()

		// Create tarball with invalid name
		invalidTarball := filepath.Join(t.TempDir(), "invalid.tar.gz")
		require.NoError(t, os.WriteFile(invalidTarball, []byte("content"), 0644))

		pub, err := NewLocalPublisher(registryDir)
		require.NoError(t, err)

		_, err = pub.Publish(invalidTarball)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "could not parse")
	})

	t.Run("protocol returns file", func(t *testing.T) {
		pub, err := NewLocalPublisher(t.TempDir())
		require.NoError(t, err)
		assert.Equal(t, "file", pub.Protocol())
	})
}

// createTestTarball creates a minimal valid tarball with the given name and version.
func createTestTarball(t *testing.T, name, version string) string {
	t.Helper()

	tmpDir := t.TempDir()
	tarPath := filepath.Join(tmpDir, name+"-"+version+".tar.gz")

	// Create a minimal gzip file (not a real tarball, but enough for parsing tests)
	// For integration tests, use the actual packer
	content := []byte{0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff}
	require.NoError(t, os.WriteFile(tarPath, content, 0644))

	return tarPath
}
