package registry

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Cache Creation Tests
// =============================================================================

func TestNewCache(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewCache(tmpDir)

	assert.NotNil(t, cache)
	assert.Equal(t, tmpDir, cache.Dir)
}

func TestDefaultCache(t *testing.T) {
	cache, err := DefaultCache()
	require.NoError(t, err)
	assert.NotNil(t, cache)
	assert.Contains(t, cache.Dir, DefaultCacheDir)
}

// =============================================================================
// Cache GetPath Tests
// =============================================================================

func TestCache_GetPath(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewCache(tmpDir)

	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{
			name:     "simple key",
			key:      "my-package",
			expected: filepath.Join(tmpDir, "my-package"),
		},
		{
			name:     "key with slashes",
			key:      "org/repo",
			expected: filepath.Join(tmpDir, "org/repo"),
		},
		{
			name:     "empty key",
			key:      "",
			expected: tmpDir,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := cache.GetPath(tt.key)
			assert.Equal(t, tt.expected, path)
		})
	}
}

// =============================================================================
// Cache Has Tests
// =============================================================================

func TestCache_Has(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewCache(tmpDir)

	// Create a cached item
	cachedKey := "existing-package"
	err := os.MkdirAll(filepath.Join(tmpDir, cachedKey), 0755)
	require.NoError(t, err)

	t.Run("existing key", func(t *testing.T) {
		assert.True(t, cache.Has(cachedKey))
	})

	t.Run("non-existing key", func(t *testing.T) {
		assert.False(t, cache.Has("nonexistent"))
	})
}

func TestCache_Has_File(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewCache(tmpDir)

	// Create a cached file (not directory)
	cachedKey := "cached-file.tar.gz"
	err := os.WriteFile(filepath.Join(tmpDir, cachedKey), []byte("content"), 0644)
	require.NoError(t, err)

	assert.True(t, cache.Has(cachedKey))
}

// =============================================================================
// Cache GetCacheDir Tests
// =============================================================================

func TestCache_GetCacheDir(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewCache(tmpDir)

	tests := []struct {
		name     string
		protocol string
		expected string
	}{
		{
			name:     "git protocol",
			protocol: "git",
			expected: filepath.Join(tmpDir, "git"),
		},
		{
			name:     "https protocol",
			protocol: "https",
			expected: filepath.Join(tmpDir, "https"),
		},
		{
			name:     "file protocol",
			protocol: "file",
			expected: filepath.Join(tmpDir, "file"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := cache.GetCacheDir(tt.protocol)
			assert.Equal(t, tt.expected, dir)
		})
	}
}

// =============================================================================
// Cache EnsureDir Tests
// =============================================================================

func TestCache_EnsureDir(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewCache(tmpDir)

	protocol := "git"
	expectedDir := filepath.Join(tmpDir, protocol)

	// Directory shouldn't exist yet
	_, err := os.Stat(expectedDir)
	assert.True(t, os.IsNotExist(err))

	// Ensure the directory
	err = cache.EnsureDir(protocol)
	require.NoError(t, err)

	// Directory should now exist
	info, err := os.Stat(expectedDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestCache_EnsureDir_AlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewCache(tmpDir)

	protocol := "git"
	expectedDir := filepath.Join(tmpDir, protocol)

	// Create the directory first
	err := os.MkdirAll(expectedDir, 0755)
	require.NoError(t, err)

	// EnsureDir should succeed even if directory exists
	err = cache.EnsureDir(protocol)
	require.NoError(t, err)

	// Directory should still exist
	info, err := os.Stat(expectedDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

// =============================================================================
// Cache Clear Tests
// =============================================================================

func TestCache_Clear_Protocol(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewCache(tmpDir)

	// Create directories for multiple protocols
	gitDir := filepath.Join(tmpDir, "git")
	httpsDir := filepath.Join(tmpDir, "https")
	err := os.MkdirAll(gitDir, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(httpsDir, 0755)
	require.NoError(t, err)

	// Add some content
	err = os.WriteFile(filepath.Join(gitDir, "file.txt"), []byte("content"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(httpsDir, "file.txt"), []byte("content"), 0644)
	require.NoError(t, err)

	// Clear only git protocol
	err = cache.Clear("git")
	require.NoError(t, err)

	// Git directory should be gone
	_, err = os.Stat(gitDir)
	assert.True(t, os.IsNotExist(err))

	// HTTPS directory should still exist
	_, err = os.Stat(httpsDir)
	require.NoError(t, err)
}

func TestCache_Clear_All(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewCache(tmpDir)

	// Create directories for multiple protocols
	gitDir := filepath.Join(tmpDir, "git")
	httpsDir := filepath.Join(tmpDir, "https")
	err := os.MkdirAll(gitDir, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(httpsDir, 0755)
	require.NoError(t, err)

	// Clear all (empty protocol string)
	err = cache.Clear("")
	require.NoError(t, err)

	// Entire cache directory should be gone
	_, err = os.Stat(tmpDir)
	assert.True(t, os.IsNotExist(err))
}

func TestCache_Clear_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewCache(tmpDir)

	// Clearing non-existent directory should not error
	err := cache.Clear("nonexistent")
	require.NoError(t, err)
}

// =============================================================================
// ComputeIntegrity Tests
// =============================================================================

func TestComputeIntegrity_File(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")

	err := os.WriteFile(filePath, []byte("test content"), 0644)
	require.NoError(t, err)

	integrity, err := ComputeIntegrity(filePath)
	require.NoError(t, err)

	assert.NotEmpty(t, integrity)
	assert.Contains(t, integrity, "sha256-")
}

func TestComputeIntegrity_File_Deterministic(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")

	err := os.WriteFile(filePath, []byte("test content"), 0644)
	require.NoError(t, err)

	// Compute twice and verify same result
	integrity1, err := ComputeIntegrity(filePath)
	require.NoError(t, err)

	integrity2, err := ComputeIntegrity(filePath)
	require.NoError(t, err)

	assert.Equal(t, integrity1, integrity2)
}

func TestComputeIntegrity_File_DifferentContent(t *testing.T) {
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "test1.txt")
	file2 := filepath.Join(tmpDir, "test2.txt")

	err := os.WriteFile(file1, []byte("content1"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(file2, []byte("content2"), 0644)
	require.NoError(t, err)

	integrity1, err := ComputeIntegrity(file1)
	require.NoError(t, err)

	integrity2, err := ComputeIntegrity(file2)
	require.NoError(t, err)

	assert.NotEqual(t, integrity1, integrity2)
}

func TestComputeIntegrity_Directory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directory structure
	err := os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content1"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "subdir", "file2.txt"), []byte("content2"), 0644)
	require.NoError(t, err)

	integrity, err := ComputeIntegrity(tmpDir)
	require.NoError(t, err)

	assert.NotEmpty(t, integrity)
	assert.Contains(t, integrity, "sha256-")
}

func TestComputeIntegrity_Directory_Deterministic(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directory structure
	err := os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte("a"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "b.txt"), []byte("b"), 0644)
	require.NoError(t, err)

	// Compute twice
	integrity1, err := ComputeIntegrity(tmpDir)
	require.NoError(t, err)

	integrity2, err := ComputeIntegrity(tmpDir)
	require.NoError(t, err)

	assert.Equal(t, integrity1, integrity2)
}

func TestComputeIntegrity_Directory_OrderIndependent(t *testing.T) {
	// Test that file order in the directory doesn't affect the hash
	// (since we sort files alphabetically)
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	// Create files in different order
	err := os.WriteFile(filepath.Join(tmpDir1, "b.txt"), []byte("b"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir1, "a.txt"), []byte("a"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tmpDir2, "a.txt"), []byte("a"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir2, "b.txt"), []byte("b"), 0644)
	require.NoError(t, err)

	integrity1, err := ComputeIntegrity(tmpDir1)
	require.NoError(t, err)

	integrity2, err := ComputeIntegrity(tmpDir2)
	require.NoError(t, err)

	assert.Equal(t, integrity1, integrity2)
}

func TestComputeIntegrity_Directory_SkipsGit(t *testing.T) {
	tmpDir := t.TempDir()

	// Create normal file and .git directory
	err := os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("content"), 0644)
	require.NoError(t, err)

	gitDir := filepath.Join(tmpDir, ".git")
	err = os.MkdirAll(gitDir, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main"), 0644)
	require.NoError(t, err)

	// Compute integrity - should not include .git
	integrity, err := ComputeIntegrity(tmpDir)
	require.NoError(t, err)

	// Now create another directory without .git but same file
	tmpDir2 := t.TempDir()
	err = os.WriteFile(filepath.Join(tmpDir2, "file.txt"), []byte("content"), 0644)
	require.NoError(t, err)

	integrity2, err := ComputeIntegrity(tmpDir2)
	require.NoError(t, err)

	// Both should have the same integrity since .git is skipped
	assert.Equal(t, integrity, integrity2)
}

func TestComputeIntegrity_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	integrity, err := ComputeIntegrity(tmpDir)
	require.NoError(t, err)

	// Empty directory should still have a valid hash
	assert.NotEmpty(t, integrity)
	assert.Contains(t, integrity, "sha256-")
}

func TestComputeIntegrity_NonExistent(t *testing.T) {
	_, err := ComputeIntegrity("/nonexistent/path")
	assert.Error(t, err)
}

// =============================================================================
// VerifyIntegrity Tests
// =============================================================================

func TestVerifyIntegrity_Match(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")

	err := os.WriteFile(filePath, []byte("test content"), 0644)
	require.NoError(t, err)

	// Compute integrity
	integrity, err := ComputeIntegrity(filePath)
	require.NoError(t, err)

	// Verify should pass
	err = VerifyIntegrity(filePath, integrity)
	require.NoError(t, err)
}

func TestVerifyIntegrity_Mismatch(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")

	err := os.WriteFile(filePath, []byte("test content"), 0644)
	require.NoError(t, err)

	// Verify with wrong hash
	err = VerifyIntegrity(filePath, "sha256-wronghash")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "integrity mismatch")
}

func TestVerifyIntegrity_EmptyExpected(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")

	err := os.WriteFile(filePath, []byte("test content"), 0644)
	require.NoError(t, err)

	// Empty expected should pass (no verification)
	err = VerifyIntegrity(filePath, "")
	require.NoError(t, err)
}

func TestVerifyIntegrity_InvalidFormat(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")

	err := os.WriteFile(filePath, []byte("test content"), 0644)
	require.NoError(t, err)

	// Verify with invalid format (not sha256-)
	err = VerifyIntegrity(filePath, "md5-somehash")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported integrity format")
}

func TestVerifyIntegrity_NonExistentPath(t *testing.T) {
	err := VerifyIntegrity("/nonexistent/path", "sha256-test")
	assert.Error(t, err)
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestCache_FullWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewCache(tmpDir)

	// 1. Ensure protocol directory exists
	err := cache.EnsureDir("git")
	require.NoError(t, err)

	// 2. Store some content
	key := "org/repo#v1.0.0"
	contentDir := filepath.Join(cache.GetCacheDir("git"), key)
	err = os.MkdirAll(contentDir, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(contentDir, "package.json"), []byte(`{"name":"test"}`), 0644)
	require.NoError(t, err)

	// 3. Verify it's cached
	assert.True(t, cache.Has(filepath.Join("git", key)))

	// 4. Compute and verify integrity
	integrity, err := ComputeIntegrity(contentDir)
	require.NoError(t, err)

	err = VerifyIntegrity(contentDir, integrity)
	require.NoError(t, err)

	// 5. Clear the protocol
	err = cache.Clear("git")
	require.NoError(t, err)

	// 6. Verify it's gone
	assert.False(t, cache.Has(filepath.Join("git", key)))
}

func TestComputeIntegrity_IncludesRelativePaths(t *testing.T) {
	// Test that changing a filename changes the integrity
	// (since we include relative paths in the hash)
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	// Same content, different filenames
	err := os.WriteFile(filepath.Join(tmpDir1, "file-a.txt"), []byte("content"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tmpDir2, "file-b.txt"), []byte("content"), 0644)
	require.NoError(t, err)

	integrity1, err := ComputeIntegrity(tmpDir1)
	require.NoError(t, err)

	integrity2, err := ComputeIntegrity(tmpDir2)
	require.NoError(t, err)

	// Different filenames should produce different hashes
	assert.NotEqual(t, integrity1, integrity2)
}

func TestComputeIntegrity_NestedDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create deeply nested structure
	nested := filepath.Join(tmpDir, "a", "b", "c")
	err := os.MkdirAll(nested, 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(nested, "deep.txt"), []byte("deep content"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "root.txt"), []byte("root content"), 0644)
	require.NoError(t, err)

	integrity, err := ComputeIntegrity(tmpDir)
	require.NoError(t, err)

	assert.NotEmpty(t, integrity)
	assert.Contains(t, integrity, "sha256-")
}
