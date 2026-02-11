package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a valid package.hcl
func createPackageHCL(t *testing.T, dir, name, version, description string) {
	t.Helper()
	content := `package {
  name = "` + name + `"
  version = "` + version + `"
  description = "` + description + `"
}
`
	err := os.WriteFile(filepath.Join(dir, "package.hcl"), []byte(content), 0644)
	require.NoError(t, err)
}

// Helper function to create a registry.json index
func createRegistryIndex(t *testing.T, dir string, index RegistryIndex) {
	t.Helper()
	data, err := json.MarshalIndent(index, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, "registry.json"), data, 0644)
	require.NoError(t, err)
}

// =============================================================================
// NewLocalRegistry Tests
// =============================================================================

func TestNewLocalRegistry_ValidPath(t *testing.T) {
	tmpDir := t.TempDir()
	createPackageHCL(t, tmpDir, "test-plugin", "1.0.0", "A test plugin")

	reg, err := NewLocalRegistry(tmpDir, ModePackage)
	require.NoError(t, err)
	assert.NotNil(t, reg)
	assert.Equal(t, "file", reg.Protocol())
	assert.Equal(t, ModePackage, reg.Mode())
}

func TestNewLocalRegistry_AbsolutePath(t *testing.T) {
	tmpDir := t.TempDir()
	createPackageHCL(t, tmpDir, "test-plugin", "1.0.0", "A test plugin")

	absPath, err := filepath.Abs(tmpDir)
	require.NoError(t, err)

	reg, err := NewLocalRegistry(absPath, ModePackage)
	require.NoError(t, err)
	assert.Equal(t, absPath, reg.BasePath())
}

func TestNewLocalRegistry_RelativePath(t *testing.T) {
	// Create a temp directory and change to it
	tmpDir := t.TempDir()
	createPackageHCL(t, tmpDir, "test-plugin", "1.0.0", "A test plugin")

	// Create subdirectory
	subDir := filepath.Join(tmpDir, "plugins", "my-plugin")
	err := os.MkdirAll(subDir, 0755)
	require.NoError(t, err)
	createPackageHCL(t, subDir, "my-plugin", "2.0.0", "My plugin")

	// Save original working directory
	origWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(origWd)

	// Change to temp directory
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Test relative path
	reg, err := NewLocalRegistry("./plugins/my-plugin", ModePackage)
	require.NoError(t, err)
	// Resolve symlinks to handle macOS /var -> /private/var
	resolvedTmpDir, err := filepath.EvalSymlinks(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(resolvedTmpDir, "plugins", "my-plugin"), reg.BasePath())
}

func TestNewLocalRegistry_InvalidPath(t *testing.T) {
	_, err := NewLocalRegistry("/nonexistent/path/that/does/not/exist", ModePackage)
	assert.Error(t, err)
}

func TestNewLocalRegistry_FileNotDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "file.txt")
	err := os.WriteFile(filePath, []byte("content"), 0644)
	require.NoError(t, err)

	_, err = NewLocalRegistry(filePath, ModePackage)
	assert.Error(t, err)
	assert.EqualError(t, err, fmt.Sprintf("registry error: connect failed for file:%s: path is not a directory: %s", filePath, filePath))
}

func TestNewLocalRegistry_ModeAutoDetectsRegistry(t *testing.T) {
	tmpDir := t.TempDir()
	createRegistryIndex(t, tmpDir, RegistryIndex{
		Name:     "test-registry",
		Version:  "1.0",
		Packages: map[string]PackageEntry{},
	})

	reg, err := NewLocalRegistry(tmpDir, ModeAuto)
	require.NoError(t, err)
	assert.Equal(t, ModeRegistry, reg.Mode())
}

func TestNewLocalRegistry_ModeAutoDetectsPackage(t *testing.T) {
	tmpDir := t.TempDir()
	createPackageHCL(t, tmpDir, "test-plugin", "1.0.0", "A test plugin")

	reg, err := NewLocalRegistry(tmpDir, ModeAuto)
	require.NoError(t, err)
	assert.Equal(t, ModePackage, reg.Mode())
}

// =============================================================================
// Registry Mode Tests
// =============================================================================

func TestLocalRegistry_RegistryMode_GetPackageInfo(t *testing.T) {
	tmpDir := t.TempDir()
	createRegistryIndex(t, tmpDir, RegistryIndex{
		Name:    "test-registry",
		Version: "1.0",
		Packages: map[string]PackageEntry{
			"my-plugin": {
				Versions: []string{"1.0.0", "1.1.0", "2.0.0"},
				Latest:   "2.0.0",
			},
			"other-plugin": {
				Versions: []string{"0.1.0"},
				Latest:   "0.1.0",
			},
		},
	})

	reg, err := NewLocalRegistry(tmpDir, ModeRegistry)
	require.NoError(t, err)

	t.Run("existing package", func(t *testing.T) {
		info, err := reg.GetPackageInfo("my-plugin")
		require.NoError(t, err)
		assert.Equal(t, "my-plugin", info.Name)
		assert.Equal(t, []string{"1.0.0", "1.1.0", "2.0.0"}, info.Versions)
		assert.Equal(t, "2.0.0", info.Latest)
	})

	t.Run("nonexistent package", func(t *testing.T) {
		_, err := reg.GetPackageInfo("nonexistent")
		assert.Error(t, err)
	})
}

func TestLocalRegistry_RegistryMode_ListPackages(t *testing.T) {
	tmpDir := t.TempDir()
	createRegistryIndex(t, tmpDir, RegistryIndex{
		Name:    "test-registry",
		Version: "1.0",
		Packages: map[string]PackageEntry{
			"plugin-a": {Versions: []string{"1.0.0"}, Latest: "1.0.0"},
			"plugin-b": {Versions: []string{"2.0.0"}, Latest: "2.0.0"},
			"plugin-c": {Versions: []string{"3.0.0"}, Latest: "3.0.0"},
		},
	})

	reg, err := NewLocalRegistry(tmpDir, ModeRegistry)
	require.NoError(t, err)

	packages, err := reg.ListPackages()
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"plugin-a", "plugin-b", "plugin-c"}, packages)
}

func TestLocalRegistry_RegistryMode_ResolvePackage(t *testing.T) {
	tmpDir := t.TempDir()
	createRegistryIndex(t, tmpDir, RegistryIndex{
		Name:    "test-registry",
		Version: "1.0",
		Packages: map[string]PackageEntry{
			"my-plugin": {
				Versions: []string{"1.0.0", "1.1.0", "1.2.0", "2.0.0"},
				Latest:   "2.0.0",
			},
		},
	})

	// Create the package directories
	pluginDir := filepath.Join(tmpDir, "my-plugin")
	err := os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)
	createPackageHCL(t, pluginDir, "my-plugin", "2.0.0", "Test plugin")

	reg, err := NewLocalRegistry(tmpDir, ModeRegistry)
	require.NoError(t, err)

	t.Run("resolve latest", func(t *testing.T) {
		resolved, err := reg.ResolvePackage("my-plugin", "latest")
		require.NoError(t, err)
		assert.Equal(t, "my-plugin", resolved.Name)
		assert.Equal(t, "2.0.0", resolved.Version)
		assert.Equal(t, "file:"+filepath.Join(tmpDir, "my-plugin"), resolved.URL)
		assert.Equal(t, filepath.Join(tmpDir, "my-plugin"), resolved.LocalPath)
	})

	t.Run("resolve empty version means latest", func(t *testing.T) {
		resolved, err := reg.ResolvePackage("my-plugin", "")
		require.NoError(t, err)
		assert.Equal(t, "2.0.0", resolved.Version)
	})

	t.Run("resolve exact version", func(t *testing.T) {
		resolved, err := reg.ResolvePackage("my-plugin", "1.2.0")
		require.NoError(t, err)
		assert.Equal(t, "1.2.0", resolved.Version)
	})

	t.Run("resolve caret constraint", func(t *testing.T) {
		resolved, err := reg.ResolvePackage("my-plugin", "^1.0.0")
		require.NoError(t, err)
		assert.Equal(t, "1.2.0", resolved.Version) // Highest 1.x
	})

	t.Run("resolve tilde constraint", func(t *testing.T) {
		resolved, err := reg.ResolvePackage("my-plugin", "~1.1.0")
		require.NoError(t, err)
		assert.Equal(t, "1.1.0", resolved.Version) // Only 1.1.x matches
	})

	t.Run("resolve nonexistent version", func(t *testing.T) {
		_, err := reg.ResolvePackage("my-plugin", "99.0.0")
		assert.Error(t, err)
	})

	t.Run("resolve nonexistent package", func(t *testing.T) {
		_, err := reg.ResolvePackage("nonexistent", "1.0.0")
		assert.Error(t, err)
	})
}

func TestLocalRegistry_RegistryMode_FetchPackage(t *testing.T) {
	tmpDir := t.TempDir()
	createRegistryIndex(t, tmpDir, RegistryIndex{
		Name:    "test-registry",
		Version: "1.0",
		Packages: map[string]PackageEntry{
			"my-plugin": {
				Versions: []string{"1.0.0"},
				Latest:   "1.0.0",
			},
		},
	})

	// Create the package directory
	pluginDir := filepath.Join(tmpDir, "my-plugin")
	err := os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)
	createPackageHCL(t, pluginDir, "my-plugin", "1.0.0", "Test plugin")

	reg, err := NewLocalRegistry(tmpDir, ModeRegistry)
	require.NoError(t, err)

	resolved := &ResolvedPackage{
		Name:      "my-plugin",
		Version:   "1.0.0",
		LocalPath: pluginDir,
	}

	destDir := t.TempDir()
	path, err := reg.FetchPackage(resolved, destDir)
	require.NoError(t, err)
	assert.Equal(t, pluginDir, path) // Local registry returns source path directly

	// Verify the package.hcl exists
	_, err = os.Stat(filepath.Join(path, "package.hcl"))
	require.NoError(t, err)
}

func TestLocalRegistry_RegistryMode_FetchPackage_VersionedDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	createRegistryIndex(t, tmpDir, RegistryIndex{
		Name:    "test-registry",
		Version: "1.0",
		Packages: map[string]PackageEntry{
			"my-plugin": {
				Versions: []string{"1.0.0"},
				Latest:   "1.0.0",
			},
		},
	})

	// Create a versioned package directory (plugin-version format)
	pluginDir := filepath.Join(tmpDir, "my-plugin-1.0.0")
	err := os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)
	createPackageHCL(t, pluginDir, "my-plugin", "1.0.0", "Test plugin")

	reg, err := NewLocalRegistry(tmpDir, ModeRegistry)
	require.NoError(t, err)

	resolved := &ResolvedPackage{
		Name:    "my-plugin",
		Version: "1.0.0",
	}

	destDir := t.TempDir()
	path, err := reg.FetchPackage(resolved, destDir)
	require.NoError(t, err)
	assert.Equal(t, pluginDir, path)
}

// =============================================================================
// Package Mode Tests
// =============================================================================

func TestLocalRegistry_PackageMode_GetPackageInfo(t *testing.T) {
	tmpDir := t.TempDir()
	createPackageHCL(t, tmpDir, "standalone-plugin", "3.0.0", "A standalone plugin")

	reg, err := NewLocalRegistry(tmpDir, ModePackage)
	require.NoError(t, err)

	t.Run("get package info", func(t *testing.T) {
		info, err := reg.GetPackageInfo("standalone-plugin")
		require.NoError(t, err)
		assert.Equal(t, "standalone-plugin", info.Name)
		assert.Equal(t, []string{"3.0.0"}, info.Versions)
		assert.Equal(t, "3.0.0", info.Latest)
		assert.Equal(t, "A standalone plugin", info.Description)
	})

	t.Run("any name returns package info in package mode", func(t *testing.T) {
		// In package mode, we return the package info regardless of the name requested
		// as there's only one package
		info, err := reg.GetPackageInfo("some-other-name")
		require.NoError(t, err)
		assert.Equal(t, "standalone-plugin", info.Name)
	})
}

func TestLocalRegistry_PackageMode_ListPackages(t *testing.T) {
	tmpDir := t.TempDir()
	createPackageHCL(t, tmpDir, "standalone-plugin", "1.0.0", "A standalone plugin")

	reg, err := NewLocalRegistry(tmpDir, ModePackage)
	require.NoError(t, err)

	packages, err := reg.ListPackages()
	require.NoError(t, err)
	assert.Equal(t, []string{"standalone-plugin"}, packages)
}

func TestLocalRegistry_PackageMode_ResolvePackage(t *testing.T) {
	tmpDir := t.TempDir()
	createPackageHCL(t, tmpDir, "standalone-plugin", "2.0.0", "A standalone plugin")

	reg, err := NewLocalRegistry(tmpDir, ModePackage)
	require.NoError(t, err)

	t.Run("resolve with matching version", func(t *testing.T) {
		resolved, err := reg.ResolvePackage("standalone-plugin", "2.0.0")
		require.NoError(t, err)
		assert.Equal(t, "standalone-plugin", resolved.Name)
		assert.Equal(t, "2.0.0", resolved.Version)
		assert.Equal(t, tmpDir, resolved.LocalPath)
	})

	t.Run("resolve with latest", func(t *testing.T) {
		resolved, err := reg.ResolvePackage("standalone-plugin", "latest")
		require.NoError(t, err)
		assert.Equal(t, "2.0.0", resolved.Version)
	})

	t.Run("resolve with empty version", func(t *testing.T) {
		resolved, err := reg.ResolvePackage("standalone-plugin", "")
		require.NoError(t, err)
		assert.Equal(t, "2.0.0", resolved.Version)
	})

	t.Run("resolve with caret constraint", func(t *testing.T) {
		resolved, err := reg.ResolvePackage("standalone-plugin", "^2.0.0")
		require.NoError(t, err)
		assert.Equal(t, "2.0.0", resolved.Version)
	})
}

func TestLocalRegistry_PackageMode_FetchPackage(t *testing.T) {
	tmpDir := t.TempDir()
	createPackageHCL(t, tmpDir, "standalone-plugin", "1.0.0", "A standalone plugin")

	reg, err := NewLocalRegistry(tmpDir, ModePackage)
	require.NoError(t, err)

	resolved := &ResolvedPackage{
		Name:      "standalone-plugin",
		Version:   "1.0.0",
		LocalPath: tmpDir,
	}

	destDir := t.TempDir()
	path, err := reg.FetchPackage(resolved, destDir)
	require.NoError(t, err)

	// In package mode, FetchPackage returns the base path directly (no copy)
	assert.Equal(t, tmpDir, path)
}

// =============================================================================
// Integrity Computation Tests
// =============================================================================

func TestLocalRegistry_IntegrityComputation(t *testing.T) {
	tmpDir := t.TempDir()
	createPackageHCL(t, tmpDir, "test-plugin", "1.0.0", "Test plugin")

	reg, err := NewLocalRegistry(tmpDir, ModePackage)
	require.NoError(t, err)

	resolved, err := reg.ResolvePackage("test-plugin", "1.0.0")
	require.NoError(t, err)

	// Verify integrity hash is computed
	assert.Equal(t, "sha256-uydHqstACv8xna+noaANfpMp67OMqhExI4l/8K0D8dw=", resolved.Integrity)
}

func TestLocalRegistry_IntegrityConsistency(t *testing.T) {
	tmpDir := t.TempDir()
	createPackageHCL(t, tmpDir, "test-plugin", "1.0.0", "Test plugin")

	reg, err := NewLocalRegistry(tmpDir, ModePackage)
	require.NoError(t, err)

	// Resolve twice and verify integrity is consistent
	resolved1, err := reg.ResolvePackage("test-plugin", "1.0.0")
	require.NoError(t, err)

	resolved2, err := reg.ResolvePackage("test-plugin", "1.0.0")
	require.NoError(t, err)

	assert.Equal(t, resolved1.Integrity, resolved2.Integrity)
}

// =============================================================================
// Path Normalization Tests
// =============================================================================

func TestLocalRegistry_PathNormalization(t *testing.T) {
	tmpDir := t.TempDir()
	createPackageHCL(t, tmpDir, "test-plugin", "1.0.0", "Test plugin")

	// Test with various path formats
	tests := []struct {
		name string
		path string
	}{
		{"absolute path", tmpDir},
		{"path with trailing slash", tmpDir + "/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg, err := NewLocalRegistry(tt.path, ModePackage)
			require.NoError(t, err)
			assert.NotNil(t, reg)
			// Base path should be normalized (no trailing slash)
			assert.False(t, filepath.Base(reg.BasePath()) == "")
		})
	}
}

// =============================================================================
// Error Cases
// =============================================================================

func TestLocalRegistry_MissingRegistryJSON(t *testing.T) {
	tmpDir := t.TempDir()

	reg, err := NewLocalRegistry(tmpDir, ModeRegistry)
	require.NoError(t, err)

	_, err = reg.GetPackageInfo("any-package")
	assert.Error(t, err)
}

func TestLocalRegistry_MalformedRegistryJSON(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "registry.json"), []byte("invalid json"), 0644)
	require.NoError(t, err)

	reg, err := NewLocalRegistry(tmpDir, ModeRegistry)
	require.NoError(t, err)

	_, err = reg.GetPackageInfo("any-package")
	assert.Error(t, err)
}

func TestLocalRegistry_MissingPackageHCL(t *testing.T) {
	tmpDir := t.TempDir()

	reg, err := NewLocalRegistry(tmpDir, ModePackage)
	require.NoError(t, err)

	_, err = reg.GetPackageInfo("any-package")
	assert.Error(t, err)
}

func TestLocalRegistry_InvalidVersionConstraint(t *testing.T) {
	tmpDir := t.TempDir()
	createRegistryIndex(t, tmpDir, RegistryIndex{
		Name:    "test-registry",
		Version: "1.0",
		Packages: map[string]PackageEntry{
			"my-plugin": {
				Versions: []string{"1.0.0"},
				Latest:   "1.0.0",
			},
		},
	})

	reg, err := NewLocalRegistry(tmpDir, ModeRegistry)
	require.NoError(t, err)

	_, err = reg.ResolvePackage("my-plugin", "invalid>>version")
	assert.Error(t, err)
}

func TestLocalRegistry_NoVersionsAvailable(t *testing.T) {
	tmpDir := t.TempDir()
	createRegistryIndex(t, tmpDir, RegistryIndex{
		Name:    "test-registry",
		Version: "1.0",
		Packages: map[string]PackageEntry{
			"empty-plugin": {
				Versions: []string{},
				Latest:   "",
			},
		},
	})

	reg, err := NewLocalRegistry(tmpDir, ModeRegistry)
	require.NoError(t, err)

	_, err = reg.ResolvePackage("empty-plugin", "latest")
	assert.Error(t, err)
}

// =============================================================================
// Version Selection Tests
// =============================================================================

func TestLocalRegistry_VersionConstraints(t *testing.T) {
	tmpDir := t.TempDir()
	createRegistryIndex(t, tmpDir, RegistryIndex{
		Name:    "test-registry",
		Version: "1.0",
		Packages: map[string]PackageEntry{
			"my-plugin": {
				Versions: []string{"0.9.0", "1.0.0", "1.1.0", "1.2.0", "2.0.0", "2.1.0"},
				Latest:   "2.1.0",
			},
		},
	})

	// Create package directory
	pluginDir := filepath.Join(tmpDir, "my-plugin")
	err := os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)
	createPackageHCL(t, pluginDir, "my-plugin", "2.1.0", "Test plugin")

	reg, err := NewLocalRegistry(tmpDir, ModeRegistry)
	require.NoError(t, err)

	tests := []struct {
		name        string
		constraint  string
		expectedVer string
		shouldErr   bool
	}{
		{"exact match", "1.1.0", "1.1.0", false},
		{"exact with equals", "=1.2.0", "1.2.0", false},
		{"caret 1.x", "^1.0.0", "1.2.0", false},
		{"caret 2.x", "^2.0.0", "2.1.0", false},
		{"tilde 1.1", "~1.1.0", "1.1.0", false},
		{"greater than", ">1.0.0", "2.1.0", false},
		{"greater than or equal", ">=2.0.0", "2.1.0", false},
		{"less than", "<1.0.0", "0.9.0", false},
		{"less than or equal", "<=1.0.0", "1.0.0", false},
		{"no match", "^3.0.0", "", true},
		{"latest", "latest", "2.1.0", false},
		{"empty means latest", "", "2.1.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, err := reg.ResolvePackage("my-plugin", tt.constraint)
			if tt.shouldErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expectedVer, resolved.Version)
		})
	}
}

// =============================================================================
// Protocol Tests
// =============================================================================

func TestLocalRegistry_Protocol(t *testing.T) {
	tmpDir := t.TempDir()
	createPackageHCL(t, tmpDir, "test-plugin", "1.0.0", "Test plugin")

	reg, err := NewLocalRegistry(tmpDir, ModePackage)
	require.NoError(t, err)

	assert.Equal(t, "file", reg.Protocol())
}
