package resource

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// File represents a universal file resource that works across all platforms.
// Files can be created with inline content or from a source file, with optional templating.
type File struct {
	// Name is the block label identifying this file resource
	Name string `hcl:"name,label"`

	// Dest is the destination path relative to project root (required)
	Dest string `hcl:"dest,attr"`

	// Content is inline file content (optional, mutually exclusive with Src)
	Content *string `hcl:"content,optional"`

	// Src is the source file path relative to plugin root (optional, mutually exclusive with Content)
	Src *string `hcl:"src,optional"`

	// Chmod specifies file permissions as octal string (e.g., "755", "644")
	Chmod string `hcl:"chmod,optional"`

	// Template enables template rendering when true
	Template bool `hcl:"template,optional"`

	// Vars provides template variables (only used when Template is true)
	Vars map[string]string `hcl:"vars,optional"`
}

// ResourceType returns the HCL block type for file resources.
func (f *File) ResourceType() string {
	return "file"
}

// ResourceName returns the file's name identifier.
func (f *File) ResourceName() string {
	return f.Name
}

// Platform returns "universal" as files work on all platforms.
func (f *File) Platform() string {
	return "universal"
}

// GetContent returns empty string as files don't have traditional content.
func (f *File) GetContent() string {
	return ""
}

// GetFiles returns empty slice as files don't have nested file blocks.
func (f *File) GetFiles() []FileBlock {
	return nil
}

// GetTemplateFiles returns empty slice as files don't have nested template file blocks.
func (f *File) GetTemplateFiles() []TemplateFileBlock {
	return nil
}

// Validate checks that the file has valid configuration.
func (f *File) Validate() error {
	if f.Name == "" {
		return fmt.Errorf("file: name is required")
	}

	if f.Dest == "" {
		return fmt.Errorf("file %q: dest is required", f.Name)
	}

	// Check for absolute paths
	if filepath.IsAbs(f.Dest) {
		return fmt.Errorf("file %q: dest must be a relative path, got absolute path %q", f.Name, f.Dest)
	}

	// Check for path traversal
	cleanDest := filepath.Clean(f.Dest)
	if strings.HasPrefix(cleanDest, ".."+string(filepath.Separator)) || cleanDest == ".." {
		return fmt.Errorf("file %q: dest must not escape project root (no .. path traversal), got %q", f.Name, f.Dest)
	}

	// Must have either content or src, but not both
	hasContent := f.Content != nil
	hasSrc := f.Src != nil

	if !hasContent && !hasSrc {
		return fmt.Errorf("file %q: must specify either 'content' or 'src'", f.Name)
	}

	if hasContent && hasSrc {
		return fmt.Errorf("file %q: cannot specify both 'content' and 'src'", f.Name)
	}

	// Validate chmod if provided
	if f.Chmod != "" {
		if err := validateChmod(f.Chmod); err != nil {
			return fmt.Errorf("file %q: %w", f.Name, err)
		}
	}

	return nil
}

// Directory represents a universal directory resource that works across all platforms.
// Directories can be created with or without parent directory creation.
type Directory struct {
	// Name is the block label identifying this directory resource
	Name string `hcl:"name,label"`

	// Path is the directory path relative to project root (required)
	Path string `hcl:"path,attr"`

	// Parents controls whether to create parent directories (default: false)
	// When true, behaves like mkdir -p
	// When false, fails if parent doesn't exist
	Parents bool `hcl:"parents,optional"`
}

// ResourceType returns the HCL block type for directory resources.
func (d *Directory) ResourceType() string {
	return "directory"
}

// ResourceName returns the directory's name identifier.
func (d *Directory) ResourceName() string {
	return d.Name
}

// Platform returns "universal" as directories work on all platforms.
func (d *Directory) Platform() string {
	return "universal"
}

// GetContent returns empty string as directories don't have content.
func (d *Directory) GetContent() string {
	return ""
}

// GetFiles returns empty slice as directories don't have file blocks.
func (d *Directory) GetFiles() []FileBlock {
	return nil
}

// GetTemplateFiles returns empty slice as directories don't have template file blocks.
func (d *Directory) GetTemplateFiles() []TemplateFileBlock {
	return nil
}

// Validate checks that the directory has valid configuration.
func (d *Directory) Validate() error {
	if d.Name == "" {
		return fmt.Errorf("directory: name is required")
	}

	if d.Path == "" {
		return fmt.Errorf("directory %q: path is required", d.Name)
	}

	// Check for absolute paths
	if filepath.IsAbs(d.Path) {
		return fmt.Errorf("directory %q: path must be relative, got absolute path %q", d.Name, d.Path)
	}

	// Check for path traversal
	cleanPath := filepath.Clean(d.Path)
	if strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) || cleanPath == ".." {
		return fmt.Errorf("directory %q: path must not escape project root (no .. path traversal), got %q", d.Name, d.Path)
	}

	return nil
}

// validateChmod checks if the chmod string is a valid 3-4 digit octal number.
func validateChmod(chmod string) error {
	if len(chmod) < 3 || len(chmod) > 4 {
		return fmt.Errorf("chmod must be 3-4 octal digits, got %q", chmod)
	}

	// Try to parse as octal
	val, err := strconv.ParseInt(chmod, 8, 32)
	if err != nil {
		return fmt.Errorf("chmod must be valid octal number, got %q", chmod)
	}

	// Check range (must fit in 12 bits for 4 digit octal)
	if val < 0 || val > 07777 {
		return fmt.Errorf("chmod must be between 000 and 7777, got %q", chmod)
	}

	return nil
}
