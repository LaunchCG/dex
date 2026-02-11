package resource

// TranslateToClaudeMCPServer converts a unified MCPServer to a Claude-specific MCP server.
// Returns nil if the server is disabled for Claude platform.
func TranslateToClaudeMCPServer(mcp *MCPServer) *ClaudeMCPServer {
	// Check if server is enabled for Claude platform
	if !mcp.IsEnabledForPlatform("claude-code") {
		return nil
	}

	// Start with base configuration
	server := &ClaudeMCPServer{
		Name:        mcp.Name,
		Description: mcp.Description,
		Command:     mcp.Command,
		Args:        mcp.Args,
		Env:         mcp.Env,
		URL:         mcp.URL,
	}

	// Determine type based on transport
	if mcp.Command != "" {
		server.Type = "command"
	} else if mcp.URL != "" {
		server.Type = "http"
	}

	// Apply Claude-specific overrides if present
	if mcp.Claude != nil {
		if mcp.Claude.Disabled {
			return nil
		}
		applyClaudeOverride(server, mcp.Claude)
	}

	return server
}

// applyClaudeOverride applies platform-specific overrides to a Claude MCP server.
func applyClaudeOverride(server *ClaudeMCPServer, override *MCPServerPlatformOverride) {
	if override.Command != "" {
		server.Command = override.Command
		server.Type = "command"
	}
	if override.URL != "" {
		server.URL = override.URL
		server.Type = "http"
	}
	if len(override.Args) > 0 {
		server.Args = override.Args
	}
	if len(override.Env) > 0 {
		server.Env = override.Env
	}
}

// TranslateToCursorMCPServer converts a unified MCPServer to a Cursor-specific MCP server.
// Returns nil if the server is disabled for Cursor platform.
func TranslateToCursorMCPServer(mcp *MCPServer) *CursorMCPServer {
	// Check if server is enabled for Cursor platform
	if !mcp.IsEnabledForPlatform("cursor") {
		return nil
	}

	// Start with base configuration
	server := &CursorMCPServer{
		Name:        mcp.Name,
		Description: mcp.Description,
		Command:     mcp.Command,
		Args:        mcp.Args,
		Env:         mcp.Env,
		URL:         mcp.URL,
		EnvFile:     mcp.EnvFile,
		Headers:     mcp.Headers,
	}

	// Determine type based on transport
	if mcp.Command != "" {
		server.Type = "stdio"
	} else if mcp.URL != "" {
		server.Type = "http"
	}

	// Apply Cursor-specific overrides if present
	if mcp.Cursor != nil {
		if mcp.Cursor.Disabled {
			return nil
		}
		applyCursorOverride(server, mcp.Cursor)
	}

	return server
}

// applyCursorOverride applies platform-specific overrides to a Cursor MCP server.
func applyCursorOverride(server *CursorMCPServer, override *MCPServerPlatformOverride) {
	if override.Command != "" {
		server.Command = override.Command
		server.Type = "stdio"
	}
	if override.URL != "" {
		server.URL = override.URL
		server.Type = "http"
	}
	if len(override.Args) > 0 {
		server.Args = override.Args
	}
	if len(override.Env) > 0 {
		server.Env = override.Env
	}
	if override.EnvFile != "" {
		server.EnvFile = override.EnvFile
	}
	if len(override.Headers) > 0 {
		server.Headers = override.Headers
	}
}

// TranslateToCopilotMCPServer converts a unified MCPServer to a Copilot-specific MCP server.
// Returns nil if the server is disabled for Copilot platform.
func TranslateToCopilotMCPServer(mcp *MCPServer) *CopilotMCPServer {
	// Check if server is enabled for Copilot platform
	if !mcp.IsEnabledForPlatform("github-copilot") {
		return nil
	}

	// Start with base configuration
	server := &CopilotMCPServer{
		Name:        mcp.Name,
		Description: mcp.Description,
		Command:     mcp.Command,
		Args:        mcp.Args,
		Env:         mcp.Env,
		URL:         mcp.URL,
		EnvFile:     mcp.EnvFile,
		Headers:     mcp.Headers,
	}

	// Determine type based on transport
	if mcp.Command != "" {
		server.Type = "stdio"
	} else if mcp.URL != "" {
		server.Type = "http"
	}

	// Apply Copilot-specific overrides if present
	if mcp.Copilot != nil {
		if mcp.Copilot.Disabled {
			return nil
		}
		applyCopilotOverride(server, mcp.Copilot)
	}

	return server
}

// applyCopilotOverride applies platform-specific overrides to a Copilot MCP server.
func applyCopilotOverride(server *CopilotMCPServer, override *MCPServerPlatformOverride) {
	if override.Command != "" {
		server.Command = override.Command
		server.Type = "stdio"
	}
	if override.URL != "" {
		server.URL = override.URL
		server.Type = "http"
	}
	if len(override.Args) > 0 {
		server.Args = override.Args
	}
	if len(override.Env) > 0 {
		server.Env = override.Env
	}
	if override.EnvFile != "" {
		server.EnvFile = override.EnvFile
	}
	if len(override.Headers) > 0 {
		server.Headers = override.Headers
	}
	if len(override.Inputs) > 0 {
		server.Inputs = override.Inputs
	}
}
