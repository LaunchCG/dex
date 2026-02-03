package resource

import (
	"strings"
	"testing"
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
			errMsg:  "name is required",
		},
		{
			name: "missing dest",
			file: File{
				Name:    "test",
				Content: strPtr("test"),
			},
			wantErr: true,
			errMsg:  "dest is required",
		},
		{
			name: "absolute dest path",
			file: File{
				Name:    "test",
				Dest:    "/absolute/path",
				Content: strPtr("test"),
			},
			wantErr: true,
			errMsg:  "must be a relative path",
		},
		{
			name: "path traversal",
			file: File{
				Name:    "test",
				Dest:    "../../../etc/passwd",
				Content: strPtr("test"),
			},
			wantErr: true,
			errMsg:  "must not escape project root",
		},
		{
			name: "neither content nor src",
			file: File{
				Name: "test",
				Dest: "config/test.txt",
			},
			wantErr: true,
			errMsg:  "must specify either 'content' or 'src'",
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
			errMsg:  "cannot specify both 'content' and 'src'",
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
			errMsg:  "chmod must be 3-4 octal digits",
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
			errMsg:  "chmod must be 3-4 octal digits",
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
			errMsg:  "chmod must be valid octal number",
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
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
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
			errMsg:  "name is required",
		},
		{
			name: "missing path",
			dir: Directory{
				Name: "test",
			},
			wantErr: true,
			errMsg:  "path is required",
		},
		{
			name: "absolute path",
			dir: Directory{
				Name: "test",
				Path: "/absolute/path",
			},
			wantErr: true,
			errMsg:  "must be relative",
		},
		{
			name: "path traversal",
			dir: Directory{
				Name: "test",
				Path: "../../etc",
			},
			wantErr: true,
			errMsg:  "must not escape project root",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.dir.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
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

	if file.ResourceType() != "file" {
		t.Errorf("expected ResourceType() = %q, got %q", "file", file.ResourceType())
	}

	if file.ResourceName() != "test" {
		t.Errorf("expected ResourceName() = %q, got %q", "test", file.ResourceName())
	}

	if file.Platform() != "universal" {
		t.Errorf("expected Platform() = %q, got %q", "universal", file.Platform())
	}

	if file.GetContent() != "" {
		t.Errorf("expected GetContent() = %q, got %q", "", file.GetContent())
	}

	if len(file.GetFiles()) != 0 {
		t.Errorf("expected GetFiles() to return empty slice, got %d items", len(file.GetFiles()))
	}

	if len(file.GetTemplateFiles()) != 0 {
		t.Errorf("expected GetTemplateFiles() to return empty slice, got %d items", len(file.GetTemplateFiles()))
	}
}

// TestDirectoryResourceInterface tests that Directory implements the Resource interface correctly.
func TestDirectoryResourceInterface(t *testing.T) {
	dir := &Directory{
		Name: "test",
		Path: "data/cache",
	}

	if dir.ResourceType() != "directory" {
		t.Errorf("expected ResourceType() = %q, got %q", "directory", dir.ResourceType())
	}

	if dir.ResourceName() != "test" {
		t.Errorf("expected ResourceName() = %q, got %q", "test", dir.ResourceName())
	}

	if dir.Platform() != "universal" {
		t.Errorf("expected Platform() = %q, got %q", "universal", dir.Platform())
	}

	if dir.GetContent() != "" {
		t.Errorf("expected GetContent() = %q, got %q", "", dir.GetContent())
	}

	if len(dir.GetFiles()) != 0 {
		t.Errorf("expected GetFiles() to return empty slice, got %d items", len(dir.GetFiles()))
	}

	if len(dir.GetTemplateFiles()) != 0 {
		t.Errorf("expected GetTemplateFiles() to return empty slice, got %d items", len(dir.GetTemplateFiles()))
	}
}

// strPtr is a helper to get a pointer to a string.
func strPtr(s string) *string {
	return &s
}
