package packer

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("valid directory with package.hcl", func(t *testing.T) {
		dir := createTestPlugin(t, "test-plugin", "1.0.0")
		defer os.RemoveAll(dir)

		p, err := New(dir)
		require.NoError(t, err)
		assert.Equal(t, "test-plugin", p.Name())
		assert.Equal(t, "1.0.0", p.Version())
	})

	t.Run("directory does not exist", func(t *testing.T) {
		_, err := New("/nonexistent/path")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("directory without package.hcl", func(t *testing.T) {
		dir := t.TempDir()
		_, err := New(dir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load package.hcl")
	})

	t.Run("invalid package.hcl", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, "package.hcl"), []byte("invalid hcl {"), 0644)
		require.NoError(t, err)

		_, err = New(dir)
		require.Error(t, err)
	})
}

func TestPack(t *testing.T) {
	t.Run("creates valid tarball", func(t *testing.T) {
		dir := createTestPlugin(t, "test-plugin", "1.2.3")
		defer os.RemoveAll(dir)

		// Create some test files
		require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test Plugin"), 0644))
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "src"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "src", "main.go"), []byte("package main"), 0644))

		p, err := New(dir)
		require.NoError(t, err)

		outDir := t.TempDir()
		output := filepath.Join(outDir, "test.tar.gz")

		result, err := p.Pack(output)
		require.NoError(t, err)

		// Verify result
		assert.Equal(t, output, result.Path)
		assert.Equal(t, "test-plugin", result.Name)
		assert.Equal(t, "1.2.3", result.Version)
		assert.True(t, result.Size > 0)
		assert.True(t, strings.HasPrefix(result.Integrity, "sha256-"))

		// Verify tarball exists and is valid
		_, err = os.Stat(output)
		require.NoError(t, err)

		// Extract and verify contents
		files := extractTarballFiles(t, output)
		assert.Contains(t, files, "test-plugin-1.2.3/package.hcl")
		assert.Contains(t, files, "test-plugin-1.2.3/README.md")
		assert.Contains(t, files, "test-plugin-1.2.3/src/main.go")
	})

	t.Run("default output filename", func(t *testing.T) {
		dir := createTestPlugin(t, "my-plugin", "2.0.0")
		defer os.RemoveAll(dir)

		p, err := New(dir)
		require.NoError(t, err)

		// Change to temp dir to avoid creating file in random location
		origDir, _ := os.Getwd()
		tmpDir := t.TempDir()
		os.Chdir(tmpDir)
		defer os.Chdir(origDir)

		result, err := p.Pack("")
		require.NoError(t, err)
		defer os.Remove(result.Path)

		// Compare just the filename since temp dirs may resolve differently (e.g., /var vs /private/var on macOS)
		assert.Equal(t, "my-plugin-2.0.0.tar.gz", filepath.Base(result.Path))
	})

	t.Run("excludes default patterns", func(t *testing.T) {
		dir := createTestPlugin(t, "test-plugin", "1.0.0")
		defer os.RemoveAll(dir)

		// Create files that should be excluded
		require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git", "objects"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".git", "config"), []byte("git config"), 0644))
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "node_modules", "pkg"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "node_modules", "pkg", "index.js"), []byte("module"), 0644))
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "__pycache__"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "__pycache__", "cache.pyc"), []byte("cache"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=value"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "test.pyc"), []byte("compiled"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".DS_Store"), []byte("ds"), 0644))

		// Create a file that should be included
		require.NoError(t, os.WriteFile(filepath.Join(dir, "included.txt"), []byte("include me"), 0644))

		p, err := New(dir)
		require.NoError(t, err)

		outDir := t.TempDir()
		output := filepath.Join(outDir, "test.tar.gz")

		_, err = p.Pack(output)
		require.NoError(t, err)

		files := extractTarballFiles(t, output)

		// Should include
		assert.Contains(t, files, "test-plugin-1.0.0/package.hcl")
		assert.Contains(t, files, "test-plugin-1.0.0/included.txt")

		// Should exclude
		for _, f := range files {
			assert.NotContains(t, f, ".git/")
			assert.NotContains(t, f, "node_modules/")
			assert.NotContains(t, f, "__pycache__/")
			assert.NotContains(t, f, ".env")
			assert.NotContains(t, f, ".pyc")
			assert.NotContains(t, f, ".DS_Store")
		}
	})

	t.Run("single top-level directory", func(t *testing.T) {
		dir := createTestPlugin(t, "my-plugin", "1.0.0")
		defer os.RemoveAll(dir)

		p, err := New(dir)
		require.NoError(t, err)

		outDir := t.TempDir()
		output := filepath.Join(outDir, "test.tar.gz")

		result, err := p.Pack(output)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify all files are under single top-level directory
		files := extractTarballFiles(t, output)
		for _, f := range files {
			assert.True(t, strings.HasPrefix(f, "my-plugin-1.0.0/"),
				"file %q should be under my-plugin-1.0.0/", f)
		}
	})

	t.Run("computes correct integrity hash", func(t *testing.T) {
		dir := createTestPlugin(t, "test-plugin", "1.0.0")
		defer os.RemoveAll(dir)

		p, err := New(dir)
		require.NoError(t, err)

		outDir := t.TempDir()
		output := filepath.Join(outDir, "test.tar.gz")

		result, err := p.Pack(output)
		require.NoError(t, err)

		// Compute hash manually
		data, err := os.ReadFile(output)
		require.NoError(t, err)
		hash := sha256.Sum256(data)
		expected := "sha256-" + hex.EncodeToString(hash[:])

		assert.Equal(t, expected, result.Integrity)
	})
}

func TestWithExcludes(t *testing.T) {
	dir := createTestPlugin(t, "test-plugin", "1.0.0")
	defer os.RemoveAll(dir)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "custom.txt"), []byte("custom"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "keep.txt"), []byte("keep"), 0644))

	p, err := New(dir)
	require.NoError(t, err)

	p.WithExcludes([]string{"custom.txt"})

	outDir := t.TempDir()
	output := filepath.Join(outDir, "test.tar.gz")

	_, err = p.Pack(output)
	require.NoError(t, err)

	files := extractTarballFiles(t, output)
	assert.Contains(t, files, "test-plugin-1.0.0/keep.txt")

	hasCustom := false
	for _, f := range files {
		if strings.Contains(f, "custom.txt") {
			hasCustom = true
			break
		}
	}
	assert.False(t, hasCustom, "custom.txt should be excluded")
}

// createTestPlugin creates a temporary directory with a valid package.hcl.
func createTestPlugin(t *testing.T, name, version string) string {
	t.Helper()

	dir := t.TempDir()
	content := `package {
  name    = "` + name + `"
  version = "` + version + `"
}
`
	err := os.WriteFile(filepath.Join(dir, "package.hcl"), []byte(content), 0644)
	require.NoError(t, err)

	return dir
}

func TestPackWithVariousFileTypes(t *testing.T) {
	t.Run("packs directory with various file types and sizes", func(t *testing.T) {
		dir := createTestPlugin(t, "comprehensive-test", "1.0.0")
		defer os.RemoveAll(dir)

		// Create files of various sizes to test the race condition fix
		// Small file
		require.NoError(t, os.WriteFile(filepath.Join(dir, "small.txt"), []byte("small content"), 0644))

		// Medium file
		mediumContent := strings.Repeat("medium content line\n", 100)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "medium.txt"), []byte(mediumContent), 0644))

		// Large file
		largeContent := strings.Repeat("large content line with more data\n", 1000)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "large.txt"), []byte(largeContent), 0644))

		// Binary file (simulate binary data)
		binaryData := make([]byte, 512)
		for i := range binaryData {
			binaryData[i] = byte(i % 256)
		}
		require.NoError(t, os.WriteFile(filepath.Join(dir, "binary.dat"), binaryData, 0644))

		// Executable file
		require.NoError(t, os.WriteFile(filepath.Join(dir, "script.sh"), []byte("#!/bin/bash\necho 'test'"), 0755))

		// Create nested directory structure
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "nested", "deep", "path"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "nested", "file1.txt"), []byte("nested file 1"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "nested", "deep", "file2.txt"), []byte("nested file 2"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "nested", "deep", "path", "file3.txt"), []byte("deeply nested file"), 0644))

		// Create empty file
		require.NoError(t, os.WriteFile(filepath.Join(dir, "empty.txt"), []byte(""), 0644))

		// Pack the directory
		p, err := New(dir)
		require.NoError(t, err)

		outDir := t.TempDir()
		output := filepath.Join(outDir, "comprehensive-test.tar.gz")

		result, err := p.Pack(output)
		require.NoError(t, err, "Pack should succeed without 'write too long' error")

		// Verify result metadata
		assert.Equal(t, output, result.Path)
		assert.Equal(t, "comprehensive-test", result.Name)
		assert.Equal(t, "1.0.0", result.Version)
		assert.True(t, result.Size > 0)
		assert.True(t, strings.HasPrefix(result.Integrity, "sha256-"))

		// Verify tarball file exists
		fileInfo, err := os.Stat(output)
		require.NoError(t, err)
		assert.Equal(t, result.Size, fileInfo.Size())

		// Extract and verify all files are present
		files := extractTarballFiles(t, output)

		expectedFiles := []string{
			"comprehensive-test-1.0.0/package.hcl",
			"comprehensive-test-1.0.0/small.txt",
			"comprehensive-test-1.0.0/medium.txt",
			"comprehensive-test-1.0.0/large.txt",
			"comprehensive-test-1.0.0/binary.dat",
			"comprehensive-test-1.0.0/script.sh",
			"comprehensive-test-1.0.0/nested/file1.txt",
			"comprehensive-test-1.0.0/nested/deep/file2.txt",
			"comprehensive-test-1.0.0/nested/deep/path/file3.txt",
			"comprehensive-test-1.0.0/empty.txt",
		}

		for _, expected := range expectedFiles {
			assert.Contains(t, files, expected, "tarball should contain %s", expected)
		}

		// Verify file contents match by extracting and comparing
		contents := extractTarballContents(t, output)

		assert.Equal(t, "small content", contents["comprehensive-test-1.0.0/small.txt"])
		assert.Equal(t, mediumContent, contents["comprehensive-test-1.0.0/medium.txt"])
		assert.Equal(t, largeContent, contents["comprehensive-test-1.0.0/large.txt"])
		assert.Equal(t, string(binaryData), contents["comprehensive-test-1.0.0/binary.dat"])
		assert.Equal(t, "#!/bin/bash\necho 'test'", contents["comprehensive-test-1.0.0/script.sh"])
		assert.Equal(t, "nested file 1", contents["comprehensive-test-1.0.0/nested/file1.txt"])
		assert.Equal(t, "nested file 2", contents["comprehensive-test-1.0.0/nested/deep/file2.txt"])
		assert.Equal(t, "deeply nested file", contents["comprehensive-test-1.0.0/nested/deep/path/file3.txt"])
		assert.Equal(t, "", contents["comprehensive-test-1.0.0/empty.txt"])

		// Verify integrity hash matches actual file
		data, err := os.ReadFile(output)
		require.NoError(t, err)
		hash := sha256.Sum256(data)
		expectedIntegrity := "sha256-" + hex.EncodeToString(hash[:])
		assert.Equal(t, expectedIntegrity, result.Integrity)
	})
}

// extractTarballFiles extracts all file paths from a tarball.
func extractTarballFiles(t *testing.T, tarPath string) []string {
	t.Helper()

	file, err := os.Open(tarPath)
	require.NoError(t, err)
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	require.NoError(t, err)
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	var files []string
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		// Only include regular files
		if header.Typeflag == tar.TypeReg {
			files = append(files, header.Name)
		}
	}

	return files
}

// extractTarballContents extracts all file contents from a tarball.
func extractTarballContents(t *testing.T, tarPath string) map[string]string {
	t.Helper()

	file, err := os.Open(tarPath)
	require.NoError(t, err)
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	require.NoError(t, err)
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	contents := make(map[string]string)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		// Only read regular files
		if header.Typeflag == tar.TypeReg {
			data, err := io.ReadAll(tr)
			require.NoError(t, err)
			contents[header.Name] = string(data)
		}
	}

	return contents
}
