package resource

import "fmt"

// =============================================================================
// MERGED RESOURCES (multiple plugins contribute to same file)
// =============================================================================

// CursorRule (singular) represents a rule merged into AGENTS.md file.
// Similar to ClaudeRule which merges into CLAUDE.md, multiple plugins can contribute
// rules which are combined together using marker-based sections.
type CursorRule struct {
	// Name is the block label identifying this rule
	Name string `hcl:"name,label"`

	// Description explains what this rule provides
	Description string `hcl:"description,attr"`

	// Content is the rule text to merge
	Content string `hcl:"content,optional"`
}

// ResourceType returns the HCL block type for Cursor rules (singular).
func (r *CursorRule) ResourceType() string {
	return "cursor_rule"
}

// ResourceName returns the rule's name identifier.
func (r *CursorRule) ResourceName() string {
	return r.Name
}

// Platform returns the target platform for Cursor rules.
func (r *CursorRule) Platform() string {
	return "cursor"
}

// GetContent returns the rule's content.
func (r *CursorRule) GetContent() string {
	return r.Content
}

// GetFiles returns nil as merged rules don't have associated files.
func (r *CursorRule) GetFiles() []FileBlock {
	return nil
}

// GetTemplateFiles returns nil as merged rules don't have associated template files.
func (r *CursorRule) GetTemplateFiles() []TemplateFileBlock {
	return nil
}

// Validate checks that the rule has all required fields.
func (r *CursorRule) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("cursor_rule: name is required")
	}
	if r.Description == "" {
		return fmt.Errorf("cursor_rule %q: description is required", r.Name)
	}
	if r.Content == "" {
		return fmt.Errorf("cursor_rule %q: content is required", r.Name)
	}
	return nil
}

// CursorMCPServer represents an MCP server configuration merged into .cursor/mcp.json.
// MCP servers provide additional tools and capabilities to Cursor.
type CursorMCPServer struct {
	// Name is the unique identifier for this MCP server
	Name string `hcl:"name,label"`

	// Description explains what this MCP server provides
	Description string `hcl:"description,optional"`

	// Type specifies the server type: "stdio", "http", or "sse"
	Type string `hcl:"type,attr"`

	// Command is the executable to run (required for type="stdio")
	Command string `hcl:"command,optional"`

	// Args are the command-line arguments for the command
	Args []string `hcl:"args,optional"`

	// Env contains environment variables for the server
	Env map[string]string `hcl:"env,optional"`

	// EnvFile is the path to an env file to load
	EnvFile string `hcl:"env_file,optional"`

	// URL is the HTTP/SSE endpoint (required for type="http" or type="sse")
	URL string `hcl:"url,optional"`

	// Headers contains HTTP headers for http/sse type servers
	Headers map[string]string `hcl:"headers,optional"`
}

// ResourceType returns the HCL block type for Cursor MCP servers.
func (m *CursorMCPServer) ResourceType() string {
	return "cursor_mcp_server"
}

// ResourceName returns the MCP server's name identifier.
func (m *CursorMCPServer) ResourceName() string {
	return m.Name
}

// Platform returns the target platform for Cursor MCP servers.
func (m *CursorMCPServer) Platform() string {
	return "cursor"
}

// GetContent returns an empty string as MCP servers don't have content.
func (m *CursorMCPServer) GetContent() string {
	return ""
}

// GetFiles returns nil as MCP servers don't have associated files.
func (m *CursorMCPServer) GetFiles() []FileBlock {
	return nil
}

// GetTemplateFiles returns nil as MCP servers don't have associated template files.
func (m *CursorMCPServer) GetTemplateFiles() []TemplateFileBlock {
	return nil
}

// Validate checks that the MCP server has all required fields.
func (m *CursorMCPServer) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("cursor_mcp_server: name is required")
	}
	if m.Type == "" {
		return fmt.Errorf("cursor_mcp_server %q: type is required", m.Name)
	}
	validTypes := map[string]bool{"stdio": true, "http": true, "sse": true}
	if !validTypes[m.Type] {
		return fmt.Errorf("cursor_mcp_server %q: type must be 'stdio', 'http', or 'sse', got %q", m.Name, m.Type)
	}
	if m.Type == "stdio" {
		if m.Command == "" {
			return fmt.Errorf("cursor_mcp_server %q: command is required for type 'stdio'", m.Name)
		}
	}
	if m.Type == "http" || m.Type == "sse" {
		if m.URL == "" {
			return fmt.Errorf("cursor_mcp_server %q: url is required for type %q", m.Name, m.Type)
		}
	}
	return nil
}

// =============================================================================
// STANDALONE RESOURCES (one file per resource)
// =============================================================================

// CursorRules (plural) represents a standalone rules file.
// Similar to ClaudeRules which creates standalone files in .claude/rules/,
// this creates standalone files in .cursor/rules/.
type CursorRules struct {
	// Name is the block label identifying this rules file
	Name string `hcl:"name,label"`

	// Description explains what these rules provide
	Description string `hcl:"description,attr"`

	// Content is the rules text
	Content string `hcl:"content,optional"`

	// Globs are optional file patterns for selective application
	Globs []string `hcl:"globs,optional"`

	// AlwaysApply indicates whether the rule should always be applied
	AlwaysApply *bool `hcl:"always_apply,optional"`

	// Files lists static files to copy alongside the rules
	Files []FileBlock `hcl:"file,block"`

	// TemplateFiles lists template files to render and copy
	TemplateFiles []TemplateFileBlock `hcl:"template_file,block"`
}

// ResourceType returns the HCL block type for Cursor rules (plural).
func (r *CursorRules) ResourceType() string {
	return "cursor_rules"
}

// ResourceName returns the rules file's name identifier.
func (r *CursorRules) ResourceName() string {
	return r.Name
}

// Platform returns the target platform for Cursor rules files.
func (r *CursorRules) Platform() string {
	return "cursor"
}

// GetContent returns the rules content.
func (r *CursorRules) GetContent() string {
	return r.Content
}

// GetFiles returns the rules file blocks.
func (r *CursorRules) GetFiles() []FileBlock {
	return r.Files
}

// GetTemplateFiles returns the rules template file blocks.
func (r *CursorRules) GetTemplateFiles() []TemplateFileBlock {
	return r.TemplateFiles
}

// Validate checks that the rules have all required fields.
func (r *CursorRules) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("cursor_rules: name is required")
	}
	if r.Description == "" {
		return fmt.Errorf("cursor_rules %q: description is required", r.Name)
	}
	if r.Content == "" {
		return fmt.Errorf("cursor_rules %q: content is required", r.Name)
	}
	return nil
}

// CursorCommand represents a command for Cursor.
// Commands are installed to .cursor/commands/{plugin}-{name}.md
// and can be invoked with the / prefix in chat.
type CursorCommand struct {
	// Name is the block label identifying this command
	Name string `hcl:"name,label"`

	// Description explains what this command does
	Description string `hcl:"description,attr"`

	// Content is the command body
	Content string `hcl:"content,optional"`

	// Files lists static files to copy alongside the command
	Files []FileBlock `hcl:"file,block"`

	// TemplateFiles lists template files to render and copy
	TemplateFiles []TemplateFileBlock `hcl:"template_file,block"`
}

// ResourceType returns the HCL block type for Cursor commands.
func (c *CursorCommand) ResourceType() string {
	return "cursor_command"
}

// ResourceName returns the command's name identifier.
func (c *CursorCommand) ResourceName() string {
	return c.Name
}

// Platform returns the target platform for Cursor commands.
func (c *CursorCommand) Platform() string {
	return "cursor"
}

// GetContent returns the command's content.
func (c *CursorCommand) GetContent() string {
	return c.Content
}

// GetFiles returns the command's file blocks.
func (c *CursorCommand) GetFiles() []FileBlock {
	return c.Files
}

// GetTemplateFiles returns the command's template file blocks.
func (c *CursorCommand) GetTemplateFiles() []TemplateFileBlock {
	return c.TemplateFiles
}

// Validate checks that the command has all required fields.
func (c *CursorCommand) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("cursor_command: name is required")
	}
	if c.Description == "" {
		return fmt.Errorf("cursor_command %q: description is required", c.Name)
	}
	if c.Content == "" {
		return fmt.Errorf("cursor_command %q: content is required", c.Name)
	}
	return nil
}
