package registry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseTarballFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     *TarballInfo
	}{
		{
			name:     "standard format with hyphen",
			filename: "my-plugin-1.0.0.tar.gz",
			want:     &TarballInfo{Name: "my-plugin", Version: "1.0.0"},
		},
		{
			name:     "standard format with v prefix",
			filename: "my-plugin-v1.0.0.tar.gz",
			want:     &TarballInfo{Name: "my-plugin", Version: "1.0.0"},
		},
		{
			name:     "underscore separator",
			filename: "my_plugin_1.0.0.tar.gz",
			want:     &TarballInfo{Name: "my_plugin", Version: "1.0.0"},
		},
		{
			name:     "tgz extension",
			filename: "plugin-2.3.4.tgz",
			want:     &TarballInfo{Name: "plugin", Version: "2.3.4"},
		},
		{
			name:     "version with prerelease",
			filename: "my-plugin-1.0.0-beta.1.tar.gz",
			want:     &TarballInfo{Name: "my-plugin", Version: "1.0.0-beta.1"},
		},
		{
			name:     "version with build metadata",
			filename: "my-plugin-1.0.0+build.123.tar.gz",
			want:     &TarballInfo{Name: "my-plugin", Version: "1.0.0+build.123"},
		},
		{
			name:     "complex name with multiple hyphens",
			filename: "my-cool-plugin-1.2.3.tar.gz",
			want:     &TarballInfo{Name: "my-cool-plugin", Version: "1.2.3"},
		},
		{
			name:     "invalid - no version",
			filename: "my-plugin.tar.gz",
			want:     nil,
		},
		{
			name:     "invalid - wrong extension",
			filename: "my-plugin-1.0.0.zip",
			want:     nil,
		},
		{
			name:     "invalid - empty string",
			filename: "",
			want:     nil,
		},
		{
			name:     "invalid - just version",
			filename: "1.0.0.tar.gz",
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseTarballFilename(tt.filename)
			if tt.want == nil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
				assert.Equal(t, tt.want.Name, got.Name)
				assert.Equal(t, tt.want.Version, got.Version)
			}
		})
	}
}

func TestIsTarballURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{
			name: "https tar.gz",
			url:  "https://example.com/plugin-1.0.0.tar.gz",
			want: true,
		},
		{
			name: "https tgz",
			url:  "https://example.com/plugin-1.0.0.tgz",
			want: true,
		},
		{
			name: "s3 tar.gz",
			url:  "s3://bucket/path/plugin-1.0.0.tar.gz",
			want: true,
		},
		{
			name: "azure tar.gz",
			url:  "az://account/container/plugin-1.0.0.tar.gz",
			want: true,
		},
		{
			name: "uppercase TAR.GZ",
			url:  "https://example.com/plugin-1.0.0.TAR.GZ",
			want: true,
		},
		{
			name: "mixed case Tar.Gz",
			url:  "https://example.com/plugin-1.0.0.Tar.Gz",
			want: true,
		},
		{
			name: "registry URL (no tarball)",
			url:  "https://example.com/registry/",
			want: false,
		},
		{
			name: "package.json URL",
			url:  "https://example.com/plugin/package.json",
			want: false,
		},
		{
			name: "zip file",
			url:  "https://example.com/plugin-1.0.0.zip",
			want: false,
		},
		{
			name: "empty string",
			url:  "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsTarballURL(tt.url)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "lowercase with hyphens",
			in:   "my-plugin",
			want: "my-plugin",
		},
		{
			name: "underscores to hyphens",
			in:   "my_plugin",
			want: "my-plugin",
		},
		{
			name: "uppercase to lowercase",
			in:   "My-Plugin",
			want: "my-plugin",
		},
		{
			name: "mixed underscores and uppercase",
			in:   "My_Cool_Plugin",
			want: "my-cool-plugin",
		},
		{
			name: "already normalized",
			in:   "plugin",
			want: "plugin",
		},
		{
			name: "empty string",
			in:   "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeName(tt.in)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNamesMatch(t *testing.T) {
	tests := []struct {
		name  string
		name1 string
		name2 string
		want  bool
	}{
		{
			name:  "exact match",
			name1: "my-plugin",
			name2: "my-plugin",
			want:  true,
		},
		{
			name:  "hyphen vs underscore",
			name1: "my-plugin",
			name2: "my_plugin",
			want:  true,
		},
		{
			name:  "case insensitive",
			name1: "My-Plugin",
			name2: "my-plugin",
			want:  true,
		},
		{
			name:  "all variations",
			name1: "My_Cool_Plugin",
			name2: "my-cool-plugin",
			want:  true,
		},
		{
			name:  "different names",
			name1: "plugin-a",
			name2: "plugin-b",
			want:  false,
		},
		{
			name:  "similar but different",
			name1: "my-plugin",
			name2: "my-plugins",
			want:  false,
		},
		{
			name:  "empty strings match",
			name1: "",
			name2: "",
			want:  true,
		},
		{
			name:  "empty vs non-empty",
			name1: "",
			name2: "plugin",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NamesMatch(tt.name1, tt.name2)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetFilenameFromURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "simple https URL",
			url:  "https://example.com/plugin-1.0.0.tar.gz",
			want: "plugin-1.0.0.tar.gz",
		},
		{
			name: "URL with path",
			url:  "https://example.com/registry/packages/plugin-1.0.0.tar.gz",
			want: "plugin-1.0.0.tar.gz",
		},
		{
			name: "URL with query string",
			url:  "https://example.com/plugin-1.0.0.tar.gz?token=abc123",
			want: "plugin-1.0.0.tar.gz",
		},
		{
			name: "URL with fragment",
			url:  "https://example.com/plugin-1.0.0.tar.gz#section",
			want: "plugin-1.0.0.tar.gz",
		},
		{
			name: "URL with query and fragment",
			url:  "https://example.com/plugin-1.0.0.tar.gz?v=1#ref",
			want: "plugin-1.0.0.tar.gz",
		},
		{
			name: "S3 URL",
			url:  "s3://bucket/path/to/plugin-1.0.0.tar.gz",
			want: "plugin-1.0.0.tar.gz",
		},
		{
			name: "Azure URL",
			url:  "az://account/container/plugin-1.0.0.tar.gz",
			want: "plugin-1.0.0.tar.gz",
		},
		{
			name: "no path",
			url:  "plugin-1.0.0.tar.gz",
			want: "plugin-1.0.0.tar.gz",
		},
		{
			name: "trailing slash removed",
			url:  "https://example.com/registry/",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetFilenameFromURL(tt.url)
			assert.Equal(t, tt.want, got)
		})
	}
}
