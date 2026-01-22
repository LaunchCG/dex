package registry

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSource(t *testing.T) {
	tests := []struct {
		name         string
		source       string
		wantProtocol string
		wantPath     string
		wantErr      bool
	}{
		// File protocol
		{
			name:         "file relative path with ./",
			source:       "file:./my-plugin",
			wantProtocol: "file",
			wantPath:     "./my-plugin",
		},
		{
			name:         "file relative path with ../",
			source:       "file:../my-plugin",
			wantProtocol: "file",
			wantPath:     "../my-plugin",
		},
		{
			name:         "file absolute path",
			source:       "file:///home/user/plugins",
			wantProtocol: "file",
			wantPath:     "/home/user/plugins",
		},
		{
			name:         "file path without prefix dots",
			source:       "file:my-plugin",
			wantProtocol: "file",
			wantPath:     "my-plugin",
		},

		// Git protocol
		{
			name:         "git https",
			source:       "git+https://github.com/user/repo.git",
			wantProtocol: "git",
			wantPath:     "https://github.com/user/repo.git",
		},
		{
			name:         "git https with tag",
			source:       "git+https://github.com/user/repo.git#v1.0.0",
			wantProtocol: "git",
			wantPath:     "https://github.com/user/repo.git#v1.0.0",
		},
		{
			name:         "git ssh",
			source:       "git+ssh://git@github.com/user/repo.git",
			wantProtocol: "git",
			wantPath:     "ssh://git@github.com/user/repo.git",
		},
		{
			name:         "git scp-style URL",
			source:       "git+git@github.com:user/repo.git",
			wantProtocol: "git",
			wantPath:     "git@github.com:user/repo.git",
		},
		{
			name:    "git invalid URL",
			source:  "git+ftp://example.com/repo",
			wantErr: true,
		},

		// HTTPS protocol
		{
			name:         "https URL",
			source:       "https://example.com/registry",
			wantProtocol: "https",
			wantPath:     "https://example.com/registry",
		},
		{
			name:         "https tarball URL",
			source:       "https://example.com/plugin-1.0.0.tar.gz",
			wantProtocol: "https",
			wantPath:     "https://example.com/plugin-1.0.0.tar.gz",
		},

		// HTTP protocol
		{
			name:         "http URL",
			source:       "http://example.com/registry",
			wantProtocol: "http",
			wantPath:     "http://example.com/registry",
		},

		// S3 protocol
		{
			name:         "s3 URL",
			source:       "s3://my-bucket/path/to/registry",
			wantProtocol: "s3",
			wantPath:     "my-bucket/path/to/registry",
		},
		{
			name:         "s3 tarball URL",
			source:       "s3://my-bucket/plugin-1.0.0.tar.gz",
			wantProtocol: "s3",
			wantPath:     "my-bucket/plugin-1.0.0.tar.gz",
		},

		// Azure protocol
		{
			name:         "azure URL",
			source:       "az://myaccount/mycontainer/path",
			wantProtocol: "az",
			wantPath:     "myaccount/mycontainer/path",
		},
		{
			name:         "azure tarball URL",
			source:       "az://myaccount/mycontainer/plugin-1.0.0.tar.gz",
			wantProtocol: "az",
			wantPath:     "myaccount/mycontainer/plugin-1.0.0.tar.gz",
		},

		// Invalid sources
		{
			name:    "empty source",
			source:  "",
			wantErr: true,
		},
		{
			name:    "unsupported protocol",
			source:  "ftp://example.com/files",
			wantErr: true,
		},
		{
			name:    "no protocol",
			source:  "example.com/path",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			protocol, path, err := ParseSource(tt.source)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantProtocol, protocol)
			assert.Equal(t, tt.wantPath, path)
		})
	}
}

func TestNewRegistry_ProtocolRouting(t *testing.T) {
	// Test that NewRegistry correctly routes to the right implementation
	// We can only test protocols that don't require external resources

	t.Run("file protocol creates LocalRegistry", func(t *testing.T) {
		// Create a temp directory as a valid local path
		tmpDir, err := os.MkdirTemp("", "dex-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		// Create a package.hcl so it's valid
		err = os.WriteFile(filepath.Join(tmpDir, "package.hcl"), []byte(`
package {
  name = "test-plugin"
  version = "1.0.0"
}
`), 0644)
		require.NoError(t, err)

		reg, err := NewRegistry("file:"+tmpDir, ModePackage)
		require.NoError(t, err)
		assert.Equal(t, "file", reg.Protocol())

		// Verify it's actually a LocalRegistry
		_, ok := reg.(*LocalRegistry)
		assert.True(t, ok, "expected *LocalRegistry")
	})

	t.Run("https protocol creates HTTPSRegistry", func(t *testing.T) {
		reg, err := NewRegistry("https://example.com/registry", ModeRegistry)
		require.NoError(t, err)
		assert.Equal(t, "https", reg.Protocol())

		_, ok := reg.(*HTTPSRegistry)
		assert.True(t, ok, "expected *HTTPSRegistry")
	})

	t.Run("http protocol creates HTTPSRegistry", func(t *testing.T) {
		reg, err := NewRegistry("http://example.com/registry", ModeRegistry)
		require.NoError(t, err)
		assert.Equal(t, "https", reg.Protocol()) // Note: still returns "https" as protocol name

		_, ok := reg.(*HTTPSRegistry)
		assert.True(t, ok, "expected *HTTPSRegistry")
	})

	t.Run("git protocol creates GitRegistry", func(t *testing.T) {
		reg, err := NewRegistry("git+https://github.com/user/repo.git", ModePackage)
		require.NoError(t, err)
		assert.Equal(t, "git", reg.Protocol())

		_, ok := reg.(*GitRegistry)
		assert.True(t, ok, "expected *GitRegistry")
	})

	// S3 and Azure require credentials, so we can only test that they return errors
	// when credentials aren't available (which is expected in tests)
	t.Run("s3 protocol attempts to create S3Registry", func(t *testing.T) {
		// This will fail because no AWS credentials are available in test environment
		// but it verifies the routing is correct
		_, err := NewRegistry("s3://bucket/path", ModeRegistry)
		// The error should be about credentials, not about unknown protocol
		if err != nil {
			assert.NotContains(t, err.Error(), "unsupported protocol")
		}
	})

	t.Run("az protocol attempts to create AzureRegistry", func(t *testing.T) {
		// This will fail because no Azure credentials are available in test environment
		// but it verifies the routing is correct
		_, err := NewRegistry("az://account/container/path", ModeRegistry)
		// The error should be about credentials, not about unknown protocol
		if err != nil {
			assert.NotContains(t, err.Error(), "unsupported protocol")
		}
	})

	t.Run("unsupported protocol returns error", func(t *testing.T) {
		_, err := NewRegistry("ftp://example.com/files", ModeRegistry)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported")
	})
}

func TestSourceMode_String(t *testing.T) {
	tests := []struct {
		mode SourceMode
		want string
	}{
		{ModeAuto, "auto"},
		{ModeRegistry, "registry"},
		{ModePackage, "package"},
		{SourceMode(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.mode.String())
		})
	}
}

func TestMustNewRegistry(t *testing.T) {
	t.Run("valid URL does not panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			MustNewRegistry("https://example.com/registry", ModeRegistry)
		})
	})

	t.Run("invalid URL panics", func(t *testing.T) {
		assert.Panics(t, func() {
			MustNewRegistry("invalid://url", ModeRegistry)
		})
	})
}

func TestNewRegistry_TarballURLs(t *testing.T) {
	t.Run("https tarball URL creates direct tarball registry", func(t *testing.T) {
		reg, err := NewRegistry("https://example.com/plugin-1.0.0.tar.gz", ModeAuto)
		require.NoError(t, err)

		httpsReg, ok := reg.(*HTTPSRegistry)
		require.True(t, ok)
		assert.True(t, httpsReg.isDirectTarball)
		require.NotNil(t, httpsReg.tarballInfo)
		assert.Equal(t, "plugin", httpsReg.tarballInfo.Name)
		assert.Equal(t, "1.0.0", httpsReg.tarballInfo.Version)
	})
}
