// Package resource defines the core interfaces and types for dex resources.
// Resources represent installable components like skills, commands, rules, and MCP servers
// that can be managed by dex for various AI agent platforms.
package resource

// Resource is the interface all dex resources must implement.
// Each resource type (skill, command, rule, etc.) implements this interface
// to provide consistent handling during installation and management.
type Resource interface {
	// ResourceType returns the HCL block type name (e.g., "claude_skill", "claude_command")
	ResourceType() string

	// ResourceName returns the unique identifier for this resource instance
	ResourceName() string

	// Platform returns the target platform for this resource (e.g., "claude-code", "github-copilot")
	Platform() string

	// GetContent returns the main content/body of the resource
	GetContent() string

	// GetFiles returns the list of files to copy alongside this resource
	GetFiles() []FileBlock

	// GetTemplateFiles returns the list of template files to render and copy
	GetTemplateFiles() []TemplateFileBlock

	// Validate checks that the resource has all required fields and valid values.
	// Returns an error describing any validation failures.
	Validate() error
}

// FileBlock represents a static file to copy alongside a resource.
// Files are copied from the plugin source to the installation directory.
type FileBlock struct {
	// Src is the source path relative to the plugin root
	Src string `hcl:"src,attr"`

	// Dest is the destination filename (defaults to basename of Src if not specified)
	Dest string `hcl:"dest,optional"`

	// Chmod specifies file permissions (e.g., "755", "600")
	Chmod string `hcl:"chmod,optional"`
}

// TemplateFileBlock represents a template file to render and copy.
// Templates are processed with template variables before being written.
type TemplateFileBlock struct {
	// Src is the source template path relative to the plugin root
	Src string `hcl:"src,attr"`

	// Dest is the destination filename (defaults to basename without .tmpl suffix)
	Dest string `hcl:"dest,optional"`

	// Chmod specifies file permissions (e.g., "755", "600")
	Chmod string `hcl:"chmod,optional"`

	// Vars provides additional template variables specific to this file
	Vars map[string]string `hcl:"vars,optional"`
}
