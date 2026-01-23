package registry

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// parseGitURL Tests
// =============================================================================

func TestParseGitURL_HTTPSFormat(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		wantRepoURL string
		wantRef     GitRef
		wantErr     bool
	}{
		{
			name:        "simple https URL",
			url:         "git+https://github.com/user/repo.git",
			wantRepoURL: "https://github.com/user/repo.git",
			wantRef:     GitRef{Type: "default", Value: ""},
		},
		{
			name:        "https URL with implicit tag",
			url:         "git+https://github.com/user/repo.git#v1.0.0",
			wantRepoURL: "https://github.com/user/repo.git",
			wantRef:     GitRef{Type: "tag", Value: "v1.0.0"},
		},
		{
			name:        "https URL with explicit tag",
			url:         "git+https://github.com/user/repo.git#tag=v2.0.0",
			wantRepoURL: "https://github.com/user/repo.git",
			wantRef:     GitRef{Type: "tag", Value: "v2.0.0"},
		},
		{
			name:        "https URL with branch",
			url:         "git+https://github.com/user/repo.git#branch=main",
			wantRepoURL: "https://github.com/user/repo.git",
			wantRef:     GitRef{Type: "branch", Value: "main"},
		},
		{
			name:        "https URL with commit",
			url:         "git+https://github.com/user/repo.git#commit=abc123",
			wantRepoURL: "https://github.com/user/repo.git",
			wantRef:     GitRef{Type: "commit", Value: "abc123"},
		},
		{
			name:        "https URL with feature branch",
			url:         "git+https://github.com/user/repo.git#branch=feature/my-feature",
			wantRepoURL: "https://github.com/user/repo.git",
			wantRef:     GitRef{Type: "branch", Value: "feature/my-feature"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoURL, ref, err := parseGitURL(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantRepoURL, repoURL)
			assert.Equal(t, tt.wantRef.Type, ref.Type)
			assert.Equal(t, tt.wantRef.Value, ref.Value)
		})
	}
}

func TestParseGitURL_SSHFormat(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		wantRepoURL string
		wantRef     GitRef
		wantErr     bool
	}{
		{
			name:        "ssh URL",
			url:         "git+ssh://git@github.com/user/repo.git",
			wantRepoURL: "ssh://git@github.com/user/repo.git",
			wantRef:     GitRef{Type: "default", Value: ""},
		},
		{
			name:        "ssh URL with tag",
			url:         "git+ssh://git@github.com/user/repo.git#v1.0.0",
			wantRepoURL: "ssh://git@github.com/user/repo.git",
			wantRef:     GitRef{Type: "tag", Value: "v1.0.0"},
		},
		{
			name:        "scp-style URL",
			url:         "git+git@github.com:user/repo.git",
			wantRepoURL: "git@github.com:user/repo.git",
			wantRef:     GitRef{Type: "default", Value: ""},
		},
		{
			name:        "scp-style URL with tag",
			url:         "git+git@github.com:user/repo.git#v1.0.0",
			wantRepoURL: "git@github.com:user/repo.git",
			wantRef:     GitRef{Type: "tag", Value: "v1.0.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoURL, ref, err := parseGitURL(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantRepoURL, repoURL)
			assert.Equal(t, tt.wantRef.Type, ref.Type)
			assert.Equal(t, tt.wantRef.Value, ref.Value)
		})
	}
}

func TestParseGitURL_Invalid(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{
			name: "missing git+ prefix",
			url:  "https://github.com/user/repo.git",
		},
		{
			name: "invalid scheme after git+",
			url:  "git+ftp://example.com/repo.git",
		},
		{
			name: "invalid ref type",
			url:  "git+https://github.com/user/repo.git#invalid=value",
		},
		{
			name: "empty URL",
			url:  "",
		},
		{
			name: "only git+ prefix",
			url:  "git+",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := parseGitURL(tt.url)
			assert.Error(t, err)
		})
	}
}

func TestParseGitURL_RefTypes(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		wantType string
	}{
		{
			name:     "tag ref type",
			url:      "git+https://github.com/user/repo.git#tag=v1.0.0",
			wantType: "tag",
		},
		{
			name:     "branch ref type",
			url:      "git+https://github.com/user/repo.git#branch=develop",
			wantType: "branch",
		},
		{
			name:     "commit ref type",
			url:      "git+https://github.com/user/repo.git#commit=abc123def456",
			wantType: "commit",
		},
		{
			name:     "implicit tag (no equals)",
			url:      "git+https://github.com/user/repo.git#v1.2.3",
			wantType: "tag",
		},
		{
			name:     "default (no fragment)",
			url:      "git+https://github.com/user/repo.git",
			wantType: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ref, err := parseGitURL(tt.url)
			require.NoError(t, err)
			assert.Equal(t, tt.wantType, ref.Type)
		})
	}
}

// =============================================================================
// NewGitRegistry Tests
// =============================================================================

func TestNewGitRegistry_ValidURLs(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "valid https URL",
			url:     "git+https://github.com/user/repo.git",
			wantErr: false,
		},
		{
			name:    "valid ssh URL",
			url:     "git+ssh://git@github.com/user/repo.git",
			wantErr: false,
		},
		{
			name:    "valid scp-style URL",
			url:     "git+git@github.com:user/repo.git",
			wantErr: false,
		},
		{
			name:    "https with tag",
			url:     "git+https://github.com/user/repo.git#v1.0.0",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg, err := NewGitRegistry(tt.url, ModePackage)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, reg)
			assert.Equal(t, "git", reg.Protocol())
		})
	}
}

func TestNewGitRegistry_InvalidURLs(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{
			name: "invalid scheme",
			url:  "ftp://example.com/repo.git",
		},
		{
			name: "empty URL",
			url:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewGitRegistry(tt.url, ModePackage)
			assert.Error(t, err)
		})
	}
}

// =============================================================================
// GitRegistry Properties Tests
// =============================================================================

func TestGitRegistry_Protocol(t *testing.T) {
	reg, err := NewGitRegistry("git+https://github.com/user/repo.git", ModePackage)
	require.NoError(t, err)
	assert.Equal(t, "git", reg.Protocol())
}

func TestGitRegistry_Mode(t *testing.T) {
	t.Run("package mode", func(t *testing.T) {
		reg, err := NewGitRegistry("git+https://github.com/user/repo.git", ModePackage)
		require.NoError(t, err)
		assert.Equal(t, ModePackage, reg.Mode())
	})

	t.Run("registry mode", func(t *testing.T) {
		reg, err := NewGitRegistry("git+https://github.com/user/repo.git", ModeRegistry)
		require.NoError(t, err)
		assert.Equal(t, ModeRegistry, reg.Mode())
	})

	t.Run("auto mode", func(t *testing.T) {
		reg, err := NewGitRegistry("git+https://github.com/user/repo.git", ModeAuto)
		require.NoError(t, err)
		assert.Equal(t, ModeAuto, reg.Mode())
	})
}

func TestGitRegistry_RepoURL(t *testing.T) {
	reg, err := NewGitRegistry("git+https://github.com/user/repo.git#v1.0.0", ModePackage)
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/user/repo.git", reg.RepoURL())
}

func TestGitRegistry_Ref(t *testing.T) {
	t.Run("with tag", func(t *testing.T) {
		reg, err := NewGitRegistry("git+https://github.com/user/repo.git#tag=v1.0.0", ModePackage)
		require.NoError(t, err)
		ref := reg.Ref()
		assert.Equal(t, "tag", ref.Type)
		assert.Equal(t, "v1.0.0", ref.Value)
	})

	t.Run("with branch", func(t *testing.T) {
		reg, err := NewGitRegistry("git+https://github.com/user/repo.git#branch=develop", ModePackage)
		require.NoError(t, err)
		ref := reg.Ref()
		assert.Equal(t, "branch", ref.Type)
		assert.Equal(t, "develop", ref.Value)
	})

	t.Run("without ref", func(t *testing.T) {
		reg, err := NewGitRegistry("git+https://github.com/user/repo.git", ModePackage)
		require.NoError(t, err)
		ref := reg.Ref()
		assert.Equal(t, "default", ref.Type)
		assert.Equal(t, "", ref.Value)
	})
}

// =============================================================================
// getCacheKey Tests
// =============================================================================

func TestGitRegistry_getCacheKey(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "default ref",
			url:      "git+https://github.com/user/repo.git",
			expected: "https://github.com/user/repo.git#HEAD",
		},
		{
			name:     "with tag",
			url:      "git+https://github.com/user/repo.git#tag=v1.0.0",
			expected: "https://github.com/user/repo.git#tag=v1.0.0",
		},
		{
			name:     "with branch",
			url:      "git+https://github.com/user/repo.git#branch=main",
			expected: "https://github.com/user/repo.git#branch=main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg, err := NewGitRegistry(tt.url, ModePackage)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, reg.getCacheKey())
		})
	}
}

func TestGitRegistry_getCacheKeyForRef(t *testing.T) {
	reg, err := NewGitRegistry("git+https://github.com/user/repo.git", ModePackage)
	require.NoError(t, err)

	tests := []struct {
		name     string
		ref      GitRef
		expected string
	}{
		{
			name:     "default ref",
			ref:      GitRef{Type: "default", Value: ""},
			expected: "https://github.com/user/repo.git#HEAD",
		},
		{
			name:     "tag ref",
			ref:      GitRef{Type: "tag", Value: "v2.0.0"},
			expected: "https://github.com/user/repo.git#tag=v2.0.0",
		},
		{
			name:     "branch ref",
			ref:      GitRef{Type: "branch", Value: "develop"},
			expected: "https://github.com/user/repo.git#branch=develop",
		},
		{
			name:     "commit ref",
			ref:      GitRef{Type: "commit", Value: "abc123"},
			expected: "https://github.com/user/repo.git#commit=abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, reg.getCacheKeyForRef(tt.ref))
		})
	}
}

// =============================================================================
// contains helper function Tests
// =============================================================================

func TestContains(t *testing.T) {
	tests := []struct {
		name   string
		slice  []string
		value  string
		expect bool
	}{
		{
			name:   "value present",
			slice:  []string{"a", "b", "c"},
			value:  "b",
			expect: true,
		},
		{
			name:   "value absent",
			slice:  []string{"a", "b", "c"},
			value:  "d",
			expect: false,
		},
		{
			name:   "empty slice",
			slice:  []string{},
			value:  "a",
			expect: false,
		},
		{
			name:   "nil slice",
			slice:  nil,
			value:  "a",
			expect: false,
		},
		{
			name:   "empty value in slice",
			slice:  []string{"", "a"},
			value:  "",
			expect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, contains(tt.slice, tt.value))
		})
	}
}

// =============================================================================
// copyDir and copyFile Tests
// =============================================================================

func TestCopyDir(t *testing.T) {
	// Create source directory with files
	srcDir := t.TempDir()

	// Create nested structure
	err := os.MkdirAll(filepath.Join(srcDir, "subdir"), 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("content1"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(srcDir, "subdir", "file2.txt"), []byte("content2"), 0644)
	require.NoError(t, err)

	// Copy to destination
	dstDir := filepath.Join(t.TempDir(), "copied")
	err = copyDir(srcDir, dstDir)
	require.NoError(t, err)

	// Verify files were copied
	content1, err := os.ReadFile(filepath.Join(dstDir, "file1.txt"))
	require.NoError(t, err)
	assert.Equal(t, "content1", string(content1))

	content2, err := os.ReadFile(filepath.Join(dstDir, "subdir", "file2.txt"))
	require.NoError(t, err)
	assert.Equal(t, "content2", string(content2))
}

func TestCopyFile(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	srcFile := filepath.Join(srcDir, "source.txt")
	dstFile := filepath.Join(dstDir, "dest.txt")

	// Create source file
	err := os.WriteFile(srcFile, []byte("test content"), 0644)
	require.NoError(t, err)

	// Copy file
	err = copyFile(srcFile, dstFile)
	require.NoError(t, err)

	// Verify content
	content, err := os.ReadFile(dstFile)
	require.NoError(t, err)
	assert.Equal(t, "test content", string(content))
}

func TestCopyFile_PreservesPermissions(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	srcFile := filepath.Join(srcDir, "source.txt")
	dstFile := filepath.Join(dstDir, "dest.txt")

	// Create source file with specific permissions
	err := os.WriteFile(srcFile, []byte("test content"), 0755)
	require.NoError(t, err)

	// Copy file
	err = copyFile(srcFile, dstFile)
	require.NoError(t, err)

	// Verify permissions are preserved
	srcInfo, err := os.Stat(srcFile)
	require.NoError(t, err)
	dstInfo, err := os.Stat(dstFile)
	require.NoError(t, err)

	assert.Equal(t, srcInfo.Mode(), dstInfo.Mode())
}

// =============================================================================
// ListPackages Tests (Package Mode)
// =============================================================================

func TestGitRegistry_ListPackages_PackageMode(t *testing.T) {
	reg, err := NewGitRegistry("git+https://github.com/user/repo.git", ModePackage)
	require.NoError(t, err)

	// In package mode without cloning, ListPackages returns nil
	packages, err := reg.ListPackages()
	require.NoError(t, err)
	assert.Nil(t, packages)
}

// =============================================================================
// GitRef Tests
// =============================================================================

func TestGitRef_Empty(t *testing.T) {
	ref := GitRef{Type: "default", Value: ""}
	assert.Equal(t, "default", ref.Type)
	assert.Equal(t, "", ref.Value)
}

func TestGitRef_WithValue(t *testing.T) {
	ref := GitRef{Type: "tag", Value: "v1.0.0"}
	assert.Equal(t, "tag", ref.Type)
	assert.Equal(t, "v1.0.0", ref.Value)
}
