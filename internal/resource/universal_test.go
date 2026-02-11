package resource

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFileValidation tests the validation logic for File resources.
func TestFileValidation(t *testing.T) {
	tests := []struct {
		name    string
		file    File
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid file with content",
			file: File{
				Name:    "test",
				Dest:    "config/test.txt",
				Content: strPtr("test content"),
			},
			wantErr: false,
		},
		{
			name: "valid file with src",
			file: File{
				Name: "test",
				Dest: "config/test.txt",
				Src:  strPtr("files/test.txt"),
			},
			wantErr: false,
		},
		{
			name: "valid file with chmod",
			file: File{
				Name:    "test",
				Dest:    "bin/test.sh",
				Content: strPtr("#!/bin/bash"),
				Chmod:   "755",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			file: File{
				Dest:    "config/test.txt",
				Content: strPtr("test"),
			},
			wantErr: true,
			errMsg:  "file: name is required",
		},
		{
			name: "missing dest",
			file: File{
				Name:    "test",
				Content: strPtr("test"),
			},
			wantErr: true,
			errMsg:  `file "test": dest is required`,
		},
		{
			name: "absolute dest path",
			file: File{
				Name:    "test",
				Dest:    "/absolute/path",
				Content: strPtr("test"),
			},
			wantErr: true,
			errMsg:  `file "test": dest must be a relative path, got absolute path "/absolute/path"`,
		},
		{
			name: "path traversal",
			file: File{
				Name:    "test",
				Dest:    "../../../etc/passwd",
				Content: strPtr("test"),
			},
			wantErr: true,
			errMsg:  `file "test": dest must not escape project root (no .. path traversal), got "../../../etc/passwd"`,
		},
		{
			name: "neither content nor src",
			file: File{
				Name: "test",
				Dest: "config/test.txt",
			},
			wantErr: true,
			errMsg:  `file "test": must specify either 'content' or 'src'`,
		},
		{
			name: "both content and src",
			file: File{
				Name:    "test",
				Dest:    "config/test.txt",
				Content: strPtr("test"),
				Src:     strPtr("files/test.txt"),
			},
			wantErr: true,
			errMsg:  `file "test": cannot specify both 'content' and 'src'`,
		},
		{
			name: "invalid chmod - too short",
			file: File{
				Name:    "test",
				Dest:    "bin/test.sh",
				Content: strPtr("test"),
				Chmod:   "5",
			},
			wantErr: true,
			errMsg:  `file "test": chmod must be 3-4 octal digits, got "5"`,
		},
		{
			name: "invalid chmod - too long",
			file: File{
				Name:    "test",
				Dest:    "bin/test.sh",
				Content: strPtr("test"),
				Chmod:   "77777",
			},
			wantErr: true,
			errMsg:  `file "test": chmod must be 3-4 octal digits, got "77777"`,
		},
		{
			name: "invalid chmod - not octal",
			file: File{
				Name:    "test",
				Dest:    "bin/test.sh",
				Content: strPtr("test"),
				Chmod:   "999",
			},
			wantErr: true,
			errMsg:  `file "test": chmod must be valid octal number, got "999"`,
		},
		{
			name: "valid chmod - 4 digits",
			file: File{
				Name:    "test",
				Dest:    "bin/test.sh",
				Content: strPtr("test"),
				Chmod:   "0755",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.file.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.EqualError(t, err, tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestDirectoryValidation tests the validation logic for Directory resources.
func TestDirectoryValidation(t *testing.T) {
	tests := []struct {
		name    string
		dir     Directory
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid directory",
			dir: Directory{
				Name: "test",
				Path: "data/cache",
			},
			wantErr: false,
		},
		{
			name: "valid directory with parents",
			dir: Directory{
				Name:    "test",
				Path:    "data/cache/sessions",
				Parents: true,
			},
			wantErr: false,
		},
		{
			name: "missing name",
			dir: Directory{
				Path: "data/cache",
			},
			wantErr: true,
			errMsg:  "directory: name is required",
		},
		{
			name: "missing path",
			dir: Directory{
				Name: "test",
			},
			wantErr: true,
			errMsg:  `directory "test": path is required`,
		},
		{
			name: "absolute path",
			dir: Directory{
				Name: "test",
				Path: "/absolute/path",
			},
			wantErr: true,
			errMsg:  `directory "test": path must be relative, got absolute path "/absolute/path"`,
		},
		{
			name: "path traversal",
			dir: Directory{
				Name: "test",
				Path: "../../etc",
			},
			wantErr: true,
			errMsg:  `directory "test": path must not escape project root (no .. path traversal), got "../../etc"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.dir.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.EqualError(t, err, tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestFileResourceInterface tests that File implements the Resource interface correctly.
func TestFileResourceInterface(t *testing.T) {
	content := "test content"
	file := &File{
		Name:    "test",
		Dest:    "config/test.txt",
		Content: &content,
	}

	assert.Equal(t, "file", file.ResourceType())
	assert.Equal(t, "test", file.ResourceName())
	assert.Equal(t, "universal", file.Platform())
	assert.Equal(t, "", file.GetContent())
	assert.Nil(t, file.GetFiles())
	assert.Nil(t, file.GetTemplateFiles())
}

// TestDirectoryResourceInterface tests that Directory implements the Resource interface correctly.
func TestDirectoryResourceInterface(t *testing.T) {
	dir := &Directory{
		Name: "test",
		Path: "data/cache",
	}

	assert.Equal(t, "directory", dir.ResourceType())
	assert.Equal(t, "test", dir.ResourceName())
	assert.Equal(t, "universal", dir.Platform())
	assert.Equal(t, "", dir.GetContent())
	assert.Nil(t, dir.GetFiles())
	assert.Nil(t, dir.GetTemplateFiles())
}

// strPtr is a helper to get a pointer to a string.
func strPtr(s string) *string {
	return &s
}
