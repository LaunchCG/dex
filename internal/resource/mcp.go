package resource

import (
	"fmt"
)

// MCPServer represents a unified MCP server that works across all platforms.
// It defines once and translates to platform-specific configurations (Claude, Cursor, Copilot).
type MCPServer struct {
	// Name is the unique identifier for this MCP server
	Name string `hcl:"name,label"`

	// Description explains what this MCP server provides
	Description string `hcl:"description,optional"`

	// Transport configuration (mutually exclusive: command or URL)
	Command string            `hcl:"command,optional"`
	Args    []string          `hcl:"args,optional"`
	Env     map[string]string `hcl:"env,optional"`
	URL     string            `hcl:"url,optional"`

	// Optional fields
	EnvFile string            `hcl:"env_file,optional"`
	Headers map[string]string `hcl:"headers,optional"`

	// Platform-specific overrides
	Claude  *MCPServerPlatformOverride `hcl:"claude,block"`
	Cursor  *MCPServerPlatformOverride `hcl:"cursor,block"`
	Copilot *MCPServerPlatformOverride `hcl:"copilot,block"`

	// Platform filtering (empty = all platforms)
	Platforms []string `hcl:"platforms,optional"`
}

// MCPServerPlatformOverride allows platform-specific customization of MCP server configuration.
type MCPServerPlatformOverride struct {
	Command  string            `hcl:"command,optional"`
	Args     []string          `hcl:"args,optional"`
	Env      map[string]string `hcl:"env,optional"`
	URL      string            `hcl:"url,optional"`
	EnvFile  string            `hcl:"env_file,optional"`
	Headers  map[string]string `hcl:"headers,optional"`
	Disabled bool              `hcl:"disabled,optional"`
}

// ResourceType returns the HCL block type for unified MCP servers.
func (m *MCPServer) ResourceType() string {
	return "mcp_server"
}

// ResourceName returns the MCP server's name identifier.
func (m *MCPServer) ResourceName() string {
	return m.Name
}

// Platform returns "universal" as this resource works across all platforms.
func (m *MCPServer) Platform() string {
	return "universal"
}

// GetContent returns an empty string as MCP servers don't have content.
func (m *MCPServer) GetContent() string {
	return ""
}

// GetFiles returns nil as MCP servers don't have associated files.
func (m *MCPServer) GetFiles() []FileBlock {
	return nil
}

// GetTemplateFiles returns nil as MCP servers don't have associated template files.
func (m *MCPServer) GetTemplateFiles() []TemplateFileBlock {
	return nil
}

// Validate checks that the MCP server has valid configuration.
func (m *MCPServer) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("mcp_server: name is required")
	}

	hasCommand := m.Command != ""
	hasURL := m.URL != ""

	if !hasCommand && !hasURL {
		return fmt.Errorf("mcp_server %q: must specify command or url", m.Name)
	}
	if hasCommand && hasURL {
		return fmt.Errorf("mcp_server %q: cannot specify both command and url", m.Name)
	}
	if hasURL && len(m.Args) > 0 {
		return fmt.Errorf("mcp_server %q: cannot use args with url", m.Name)
	}
	if hasCommand && len(m.Headers) > 0 {
		return fmt.Errorf("mcp_server %q: headers only valid for url", m.Name)
	}

	return nil
}

// IsEnabledForPlatform checks if this MCP server should be installed for the given platform.
func (m *MCPServer) IsEnabledForPlatform(platform string) bool {
	// Empty platforms list means enabled for all platforms
	if len(m.Platforms) == 0 {
		return true
	}

	// Check if platform is in the allowed list
	for _, p := range m.Platforms {
		if p == platform {
			return true
		}
	}

	return false
}
