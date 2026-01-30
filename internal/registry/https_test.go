package registry

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHTTPSRegistry(t *testing.T) {
	tests := []struct {
		name            string
		url             string
		mode            SourceMode
		wantErr         bool
		wantTarball     bool
		wantTarballName string
		wantTarballVer  string
	}{
		{
			name:    "valid https URL",
			url:     "https://example.com/registry",
			mode:    ModeRegistry,
			wantErr: false,
		},
		{
			name:    "valid http URL",
			url:     "http://example.com/registry",
			mode:    ModeRegistry,
			wantErr: false,
		},
		{
			name:    "URL with trailing slash normalized",
			url:     "https://example.com/registry/",
			mode:    ModeRegistry,
			wantErr: false,
		},
		{
			name:            "direct tarball URL",
			url:             "https://example.com/plugin-1.0.0.tar.gz",
			mode:            ModeAuto,
			wantErr:         false,
			wantTarball:     true,
			wantTarballName: "plugin",
			wantTarballVer:  "1.0.0",
		},
		{
			name:            "direct tarball with v prefix",
			url:             "https://example.com/my-plugin-v2.3.4.tar.gz",
			mode:            ModeAuto,
			wantErr:         false,
			wantTarball:     true,
			wantTarballName: "my-plugin",
			wantTarballVer:  "2.3.4",
		},
		{
			name:    "invalid scheme - ftp",
			url:     "ftp://example.com/registry",
			mode:    ModeRegistry,
			wantErr: true,
		},
		{
			name:    "invalid scheme - no scheme",
			url:     "example.com/registry",
			mode:    ModeRegistry,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg, err := NewHTTPSRegistry(tt.url, tt.mode)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, reg)
			assert.Equal(t, "https", reg.Protocol())
			assert.Equal(t, tt.wantTarball, reg.isDirectTarball)
			if tt.wantTarball && reg.tarballInfo != nil {
				assert.Equal(t, tt.wantTarballName, reg.tarballInfo.Name)
				assert.Equal(t, tt.wantTarballVer, reg.tarballInfo.Version)
			}
		})
	}
}

func TestHTTPSRegistry_GetPackageInfo_RegistryMode(t *testing.T) {
	// Create a test server that serves registry.json
	registryIndex := RegistryIndex{
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
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/registry.json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(registryIndex)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	reg, err := NewHTTPSRegistry(server.URL, ModeRegistry)
	require.NoError(t, err)

	t.Run("existing package", func(t *testing.T) {
		info, err := reg.GetPackageInfo("my-plugin")
		require.NoError(t, err)
		assert.Equal(t, "my-plugin", info.Name)
		assert.Equal(t, []string{"1.0.0", "1.1.0", "2.0.0"}, info.Versions)
		assert.Equal(t, "2.0.0", info.Latest)
	})

	t.Run("non-existent package", func(t *testing.T) {
		_, err := reg.GetPackageInfo("nonexistent")
		assert.Error(t, err)
	})
}

func TestHTTPSRegistry_GetPackageInfo_PackageMode(t *testing.T) {
	// HTTPS sources no longer support package mode - they should return an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	reg, err := NewHTTPSRegistry(server.URL, ModePackage)
	require.NoError(t, err)

	t.Run("package mode not supported", func(t *testing.T) {
		_, err := reg.GetPackageInfo("any-plugin")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "do not support package mode")
	})
}

func TestHTTPSRegistry_GetPackageInfo_DirectTarball(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Just return 200 for any request (tarball would be served here)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	reg, err := NewHTTPSRegistry(server.URL+"/my-plugin-1.2.3.tar.gz", ModeAuto)
	require.NoError(t, err)
	assert.True(t, reg.isDirectTarball)

	t.Run("get package info from tarball URL", func(t *testing.T) {
		info, err := reg.GetPackageInfo("my-plugin")
		require.NoError(t, err)
		assert.Equal(t, "my-plugin", info.Name)
		assert.Equal(t, []string{"1.2.3"}, info.Versions)
		assert.Equal(t, "1.2.3", info.Latest)
	})

	t.Run("matching name with normalization", func(t *testing.T) {
		info, err := reg.GetPackageInfo("my_plugin")
		require.NoError(t, err)
		assert.Equal(t, "my-plugin", info.Name)
	})

	t.Run("wrong package name", func(t *testing.T) {
		_, err := reg.GetPackageInfo("other-plugin")
		assert.Error(t, err)
	})

	t.Run("empty name returns package info", func(t *testing.T) {
		info, err := reg.GetPackageInfo("")
		require.NoError(t, err)
		assert.Equal(t, "my-plugin", info.Name)
	})
}

func TestHTTPSRegistry_ResolvePackage(t *testing.T) {
	registryIndex := RegistryIndex{
		Name:    "test-registry",
		Version: "1.0",
		Packages: map[string]PackageEntry{
			"my-plugin": {
				Versions: []string{"1.0.0", "1.1.0", "1.2.0", "2.0.0"},
				Latest:   "2.0.0",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/registry.json":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(registryIndex)
		case "/my-plugin-2.0.0.tar.gz":
			w.WriteHeader(http.StatusOK)
		case "/my-plugin-1.2.0.tar.gz":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	reg, err := NewHTTPSRegistry(server.URL, ModeRegistry)
	require.NoError(t, err)

	t.Run("resolve latest", func(t *testing.T) {
		resolved, err := reg.ResolvePackage("my-plugin", "latest")
		require.NoError(t, err)
		assert.Equal(t, "my-plugin", resolved.Name)
		assert.Equal(t, "2.0.0", resolved.Version)
		assert.Contains(t, resolved.URL, "my-plugin-2.0.0.tar.gz")
	})

	t.Run("resolve empty version (means latest)", func(t *testing.T) {
		resolved, err := reg.ResolvePackage("my-plugin", "")
		require.NoError(t, err)
		assert.Equal(t, "2.0.0", resolved.Version)
	})

	t.Run("resolve exact version", func(t *testing.T) {
		resolved, err := reg.ResolvePackage("my-plugin", "1.2.0")
		require.NoError(t, err)
		assert.Equal(t, "1.2.0", resolved.Version)
	})

	t.Run("resolve version range", func(t *testing.T) {
		resolved, err := reg.ResolvePackage("my-plugin", "^1.0.0")
		require.NoError(t, err)
		assert.Equal(t, "1.2.0", resolved.Version) // Highest 1.x
	})

	t.Run("resolve non-existent version", func(t *testing.T) {
		_, err := reg.ResolvePackage("my-plugin", "99.0.0")
		assert.Error(t, err)
	})

	t.Run("resolve non-existent package", func(t *testing.T) {
		_, err := reg.ResolvePackage("nonexistent", "1.0.0")
		assert.Error(t, err)
	})
}

func TestHTTPSRegistry_ResolvePackage_DirectTarball(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tarballURL := server.URL + "/my-plugin-1.0.0.tar.gz"
	reg, err := NewHTTPSRegistry(tarballURL, ModeAuto)
	require.NoError(t, err)

	t.Run("resolve returns tarball URL", func(t *testing.T) {
		resolved, err := reg.ResolvePackage("my-plugin", "1.0.0")
		require.NoError(t, err)
		assert.Equal(t, "my-plugin", resolved.Name)
		assert.Equal(t, "1.0.0", resolved.Version)
		assert.Equal(t, tarballURL, resolved.URL)
	})

	t.Run("resolve with empty version", func(t *testing.T) {
		resolved, err := reg.ResolvePackage("my-plugin", "")
		require.NoError(t, err)
		assert.Equal(t, "1.0.0", resolved.Version)
	})
}

func TestHTTPSRegistry_FetchPackage(t *testing.T) {
	// Create a test tarball
	tarballContent := createTestTarball(t, "my-plugin", map[string]string{
		"package.json": `{"name": "my-plugin", "version": "1.0.0"}`,
		"README.md":    "# My Plugin",
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/registry.json":
			json.NewEncoder(w).Encode(RegistryIndex{
				Packages: map[string]PackageEntry{
					"my-plugin": {Versions: []string{"1.0.0"}, Latest: "1.0.0"},
				},
			})
		case "/my-plugin-1.0.0.tar.gz":
			w.Header().Set("Content-Type", "application/gzip")
			w.Write(tarballContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	reg, err := NewHTTPSRegistry(server.URL, ModeRegistry)
	require.NoError(t, err)

	// Create temp directory for extraction
	destDir, err := os.MkdirTemp("", "dex-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	t.Run("fetch and extract package", func(t *testing.T) {
		resolved := &ResolvedPackage{
			Name:    "my-plugin",
			Version: "1.0.0",
			URL:     server.URL + "/my-plugin-1.0.0.tar.gz",
		}

		extractedPath, err := reg.FetchPackage(resolved, destDir)
		require.NoError(t, err)

		// Verify extracted files
		packageJSON, err := os.ReadFile(filepath.Join(extractedPath, "package.json"))
		require.NoError(t, err)
		assert.Contains(t, string(packageJSON), `"name": "my-plugin"`)

		readme, err := os.ReadFile(filepath.Join(extractedPath, "README.md"))
		require.NoError(t, err)
		assert.Equal(t, "# My Plugin", string(readme))
	})
}

func TestHTTPSRegistry_ListPackages(t *testing.T) {
	t.Run("registry mode", func(t *testing.T) {
		registryIndex := RegistryIndex{
			Packages: map[string]PackageEntry{
				"plugin-a": {Versions: []string{"1.0.0"}, Latest: "1.0.0"},
				"plugin-b": {Versions: []string{"2.0.0"}, Latest: "2.0.0"},
				"plugin-c": {Versions: []string{"3.0.0"}, Latest: "3.0.0"},
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(registryIndex)
		}))
		defer server.Close()

		reg, err := NewHTTPSRegistry(server.URL, ModeRegistry)
		require.NoError(t, err)

		packages, err := reg.ListPackages()
		require.NoError(t, err)
		assert.Len(t, packages, 3)
		assert.Contains(t, packages, "plugin-a")
		assert.Contains(t, packages, "plugin-b")
		assert.Contains(t, packages, "plugin-c")
	})

	t.Run("package mode not supported", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		}))
		defer server.Close()

		reg, err := NewHTTPSRegistry(server.URL, ModePackage)
		require.NoError(t, err)

		packages, err := reg.ListPackages()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "do not support package mode")
		assert.Nil(t, packages)
	})

	t.Run("direct tarball mode", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		reg, err := NewHTTPSRegistry(server.URL+"/my-plugin-1.0.0.tar.gz", ModeAuto)
		require.NoError(t, err)

		packages, err := reg.ListPackages()
		require.NoError(t, err)
		assert.Equal(t, []string{"my-plugin"}, packages)
	})
}

func TestExtractTarGz(t *testing.T) {
	t.Run("extract tarball with single top-level directory", func(t *testing.T) {
		tarballContent := createTestTarball(t, "my-plugin", map[string]string{
			"file1.txt": "content1",
			"file2.txt": "content2",
		})

		// Write tarball to temp file
		tarballPath := filepath.Join(t.TempDir(), "test.tar.gz")
		err := os.WriteFile(tarballPath, tarballContent, 0644)
		require.NoError(t, err)

		destDir := t.TempDir()
		extractedPath, err := extractTarGz(tarballPath, destDir)
		require.NoError(t, err)

		// Should return path to the single top-level directory
		assert.Equal(t, filepath.Join(destDir, "my-plugin"), extractedPath)

		// Verify files
		content1, err := os.ReadFile(filepath.Join(extractedPath, "file1.txt"))
		require.NoError(t, err)
		assert.Equal(t, "content1", string(content1))
	})

	t.Run("extract tarball without single top-level directory", func(t *testing.T) {
		tarballContent := createTestTarballFlat(t, map[string]string{
			"file1.txt": "content1",
			"file2.txt": "content2",
		})

		tarballPath := filepath.Join(t.TempDir(), "test.tar.gz")
		err := os.WriteFile(tarballPath, tarballContent, 0644)
		require.NoError(t, err)

		destDir := t.TempDir()
		extractedPath, err := extractTarGz(tarballPath, destDir)
		require.NoError(t, err)

		// Should return destDir since no single top-level directory
		assert.Equal(t, destDir, extractedPath)

		// Verify files
		content1, err := os.ReadFile(filepath.Join(destDir, "file1.txt"))
		require.NoError(t, err)
		assert.Equal(t, "content1", string(content1))
	})

	t.Run("reject path traversal", func(t *testing.T) {
		tarballContent := createTestTarballWithPaths(t, map[string]string{
			"../evil.txt": "malicious",
		})

		tarballPath := filepath.Join(t.TempDir(), "test.tar.gz")
		err := os.WriteFile(tarballPath, tarballContent, 0644)
		require.NoError(t, err)

		destDir := t.TempDir()
		_, err = extractTarGz(tarballPath, destDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid path")
	})
}

// Helper functions for creating test tarballs

func createTestTarball(t *testing.T, dirName string, files map[string]string) []byte {
	t.Helper()

	var buf []byte
	gzipBuf := &bytesBuffer{}
	gzWriter := gzip.NewWriter(gzipBuf)
	tarWriter := tar.NewWriter(gzWriter)

	// Add directory entry
	err := tarWriter.WriteHeader(&tar.Header{
		Name:     dirName + "/",
		Mode:     0755,
		Typeflag: tar.TypeDir,
	})
	require.NoError(t, err)

	// Add files
	for name, content := range files {
		err := tarWriter.WriteHeader(&tar.Header{
			Name:     dirName + "/" + name,
			Mode:     0644,
			Size:     int64(len(content)),
			Typeflag: tar.TypeReg,
		})
		require.NoError(t, err)
		_, err = tarWriter.Write([]byte(content))
		require.NoError(t, err)
	}

	err = tarWriter.Close()
	require.NoError(t, err)
	err = gzWriter.Close()
	require.NoError(t, err)

	buf = gzipBuf.Bytes()
	return buf
}

func createTestTarballFlat(t *testing.T, files map[string]string) []byte {
	t.Helper()

	gzipBuf := &bytesBuffer{}
	gzWriter := gzip.NewWriter(gzipBuf)
	tarWriter := tar.NewWriter(gzWriter)

	for name, content := range files {
		err := tarWriter.WriteHeader(&tar.Header{
			Name:     name,
			Mode:     0644,
			Size:     int64(len(content)),
			Typeflag: tar.TypeReg,
		})
		require.NoError(t, err)
		_, err = tarWriter.Write([]byte(content))
		require.NoError(t, err)
	}

	err := tarWriter.Close()
	require.NoError(t, err)
	err = gzWriter.Close()
	require.NoError(t, err)

	return gzipBuf.Bytes()
}

func createTestTarballWithPaths(t *testing.T, files map[string]string) []byte {
	t.Helper()

	gzipBuf := &bytesBuffer{}
	gzWriter := gzip.NewWriter(gzipBuf)
	tarWriter := tar.NewWriter(gzWriter)

	for path, content := range files {
		err := tarWriter.WriteHeader(&tar.Header{
			Name:     path,
			Mode:     0644,
			Size:     int64(len(content)),
			Typeflag: tar.TypeReg,
		})
		require.NoError(t, err)
		_, err = tarWriter.Write([]byte(content))
		require.NoError(t, err)
	}

	err := tarWriter.Close()
	require.NoError(t, err)
	err = gzWriter.Close()
	require.NoError(t, err)

	return gzipBuf.Bytes()
}

// bytesBuffer is a simple bytes.Buffer wrapper
type bytesBuffer struct {
	data []byte
}

func (b *bytesBuffer) Write(p []byte) (n int, err error) {
	b.data = append(b.data, p...)
	return len(p), nil
}

func (b *bytesBuffer) Bytes() []byte {
	return b.data
}
