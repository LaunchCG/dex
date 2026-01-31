package installer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===========================================================================
// MCP Server Installation Tests
// ===========================================================================

func TestInstaller_MCPServer_ClaudeCode(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Set up a local plugin with an MCP server
	pluginDir := t.TempDir()
	pluginContent := `package {
  name = "mcp-test"
  version = "1.0.0"
  description = "Plugin with MCP server"
}

mcp_server "filesystem" {
  description = "Filesystem MCP server"
  command = "npx"
  args = ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
  env = {
    DEBUG = "true"
  }
}
`
	err := os.WriteFile(filepath.Join(pluginDir, "package.hcl"), []byte(pluginContent), 0644)
	require.NoError(t, err)

	// Create project config
	createTestProject(t, projectDir, `
plugin "mcp-test" {
  source = "file:`+pluginDir+`"
}
`)

	// Create installer
	installer, err := NewInstaller(projectDir)
	require.NoError(t, err)

	// Install
	err = installer.InstallAll()
	require.NoError(t, err)

	// Verify .mcp.json was created with the correct content
	mcpPath := filepath.Join(projectDir, ".mcp.json")
	require.FileExists(t, mcpPath)

	// Read and parse MCP config
	mcpData, err := os.ReadFile(mcpPath)
	require.NoError(t, err)

	var mcpConfig map[string]any
	err = json.Unmarshal(mcpData, &mcpConfig)
	require.NoError(t, err)

	// Verify structure
	assert.Contains(t, mcpConfig, "mcpServers")
	mcpServers := mcpConfig["mcpServers"].(map[string]any)

	// Verify the filesystem server was installed (non-namespaced by default)
	assert.Contains(t, mcpServers, "filesystem")
	fsServer := mcpServers["filesystem"].(map[string]any)

	assert.Equal(t, "npx", fsServer["command"])
	assert.Equal(t, []any{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"}, fsServer["args"])

	env := fsServer["env"].(map[string]any)
	assert.Equal(t, "true", env["DEBUG"])
}

func TestInstaller_MCPServer_ClaudeCode_WithNamespacing(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Set up a local plugin with an MCP server
	pluginDir := t.TempDir()
	pluginContent := `package {
  name = "mcp-test"
  version = "1.0.0"
  description = "Plugin with MCP server"
}

mcp_server "filesystem" {
  description = "Filesystem MCP server"
  command = "npx"
  args = ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
}
`
	err := os.WriteFile(filepath.Join(pluginDir, "package.hcl"), []byte(pluginContent), 0644)
	require.NoError(t, err)

	// Create project config with namespace_all enabled
	projectContent := `project {
  name = "test-project"
  agentic_platform = "claude-code"
  namespace_all = true
}

plugin "mcp-test" {
  source = "file:` + pluginDir + `"
}
`
	err = os.WriteFile(filepath.Join(projectDir, "dex.hcl"), []byte(projectContent), 0644)
	require.NoError(t, err)

	// Create installer
	installer, err := NewInstaller(projectDir)
	require.NoError(t, err)

	// Install
	err = installer.InstallAll()
	require.NoError(t, err)

	// Verify .mcp.json was created with namespaced server name
	mcpPath := filepath.Join(projectDir, ".mcp.json")
	require.FileExists(t, mcpPath)

	// Read and parse MCP config
	mcpData, err := os.ReadFile(mcpPath)
	require.NoError(t, err)

	var mcpConfig map[string]any
	err = json.Unmarshal(mcpData, &mcpConfig)
	require.NoError(t, err)

	// Verify structure
	assert.Contains(t, mcpConfig, "mcpServers")
	mcpServers := mcpConfig["mcpServers"].(map[string]any)

	// Verify the server was namespaced
	assert.Contains(t, mcpServers, "mcp-test-filesystem")
	assert.NotContains(t, mcpServers, "filesystem")

	fsServer := mcpServers["mcp-test-filesystem"].(map[string]any)
	assert.Equal(t, "npx", fsServer["command"])
}

func TestInstaller_MCPServer_Cursor(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Set up a local plugin with an MCP server
	pluginDir := t.TempDir()
	pluginContent := `package {
  name = "mcp-test"
  version = "1.0.0"
  description = "Plugin with MCP server"
}

mcp_server "context7" {
  description = "Context7 MCP server"
  url = "https://mcp.context7.com/mcp"
  headers = {
    Authorization = "Bearer test-token"
  }
}
`
	err := os.WriteFile(filepath.Join(pluginDir, "package.hcl"), []byte(pluginContent), 0644)
	require.NoError(t, err)

	// Create project config for Cursor
	projectContent := `project {
  name = "test-project"
  agentic_platform = "cursor"
}

plugin "mcp-test" {
  source = "file:` + pluginDir + `"
}
`
	err = os.WriteFile(filepath.Join(projectDir, "dex.hcl"), []byte(projectContent), 0644)
	require.NoError(t, err)

	// Create installer
	installer, err := NewInstaller(projectDir)
	require.NoError(t, err)

	// Install
	err = installer.InstallAll()
	require.NoError(t, err)

	// Verify .cursor/mcp.json was created
	mcpPath := filepath.Join(projectDir, ".cursor", "mcp.json")
	require.FileExists(t, mcpPath)

	// Read and parse MCP config
	mcpData, err := os.ReadFile(mcpPath)
	require.NoError(t, err)

	var mcpConfig map[string]any
	err = json.Unmarshal(mcpData, &mcpConfig)
	require.NoError(t, err)

	// Verify structure
	assert.Contains(t, mcpConfig, "mcpServers")
	mcpServers := mcpConfig["mcpServers"].(map[string]any)

	// Verify the context7 server was installed
	assert.Contains(t, mcpServers, "context7")
	ctx7Server := mcpServers["context7"].(map[string]any)

	assert.Equal(t, "https://mcp.context7.com/mcp", ctx7Server["url"])

	headers := ctx7Server["headers"].(map[string]any)
	assert.Equal(t, "Bearer test-token", headers["Authorization"])
}

func TestInstaller_MCPServer_Copilot(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Set up a local plugin with an MCP server
	pluginDir := t.TempDir()
	pluginContent := `package {
  name = "mcp-test"
  version = "1.0.0"
  description = "Plugin with MCP server"
}

mcp_server "database" {
  description = "Database MCP server"
  command = "db-server"
  args = ["--host", "localhost"]
  env = {
    DB_HOST = "localhost"
    DB_PORT = "5432"
  }
}
`
	err := os.WriteFile(filepath.Join(pluginDir, "package.hcl"), []byte(pluginContent), 0644)
	require.NoError(t, err)

	// Create project config for GitHub Copilot
	projectContent := `project {
  name = "test-project"
  agentic_platform = "github-copilot"
}

plugin "mcp-test" {
  source = "file:` + pluginDir + `"
}
`
	err = os.WriteFile(filepath.Join(projectDir, "dex.hcl"), []byte(projectContent), 0644)
	require.NoError(t, err)

	// Create installer
	installer, err := NewInstaller(projectDir)
	require.NoError(t, err)

	// Install
	err = installer.InstallAll()
	require.NoError(t, err)

	// Verify .vscode/mcp.json was created
	mcpPath := filepath.Join(projectDir, ".vscode", "mcp.json")
	require.FileExists(t, mcpPath)

	// Read and parse MCP config
	mcpData, err := os.ReadFile(mcpPath)
	require.NoError(t, err)

	var mcpConfig map[string]any
	err = json.Unmarshal(mcpData, &mcpConfig)
	require.NoError(t, err)

	// Verify structure (Copilot uses "servers" key, not "mcpServers")
	assert.Contains(t, mcpConfig, "servers")
	servers := mcpConfig["servers"].(map[string]any)

	// Verify the database server was installed
	assert.Contains(t, servers, "database")
	dbServer := servers["database"].(map[string]any)

	assert.Equal(t, "stdio", dbServer["type"])
	assert.Equal(t, "db-server", dbServer["command"])
	assert.Equal(t, []any{"--host", "localhost"}, dbServer["args"])

	env := dbServer["env"].(map[string]any)
	assert.Equal(t, "localhost", env["DB_HOST"])
	assert.Equal(t, "5432", env["DB_PORT"])
}

func TestInstaller_MCPServer_PlatformOverride(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Set up a local plugin with platform-specific overrides
	pluginDir := t.TempDir()
	pluginContent := `package {
  name = "mcp-test"
  version = "1.0.0"
  description = "Plugin with platform overrides"
}

mcp_server "multi-platform" {
  description = "Multi-platform server"
  command = "default-command"
  args = ["--default"]

  claude {
    command = "claude-command"
    args = ["--claude"]
  }

  cursor {
    disabled = true
  }
}
`
	err := os.WriteFile(filepath.Join(pluginDir, "package.hcl"), []byte(pluginContent), 0644)
	require.NoError(t, err)

	// Test Claude Code (should use override)
	t.Run("claude-code uses override", func(t *testing.T) {
		projectContent := `project {
  name = "test-project"
  agentic_platform = "claude-code"
}

plugin "mcp-test" {
  source = "file:` + pluginDir + `"
}
`
		err = os.WriteFile(filepath.Join(projectDir, "dex.hcl"), []byte(projectContent), 0644)
		require.NoError(t, err)

		installer, err := NewInstaller(projectDir)
		require.NoError(t, err)

		err = installer.InstallAll()
		require.NoError(t, err)

		// Read MCP config
		mcpData, err := os.ReadFile(filepath.Join(projectDir, ".mcp.json"))
		require.NoError(t, err)

		var mcpConfig map[string]any
		err = json.Unmarshal(mcpData, &mcpConfig)
		require.NoError(t, err)

		mcpServers := mcpConfig["mcpServers"].(map[string]any)
		assert.Contains(t, mcpServers, "multi-platform")

		server := mcpServers["multi-platform"].(map[string]any)
		assert.Equal(t, "claude-command", server["command"])
		assert.Equal(t, []any{"--claude"}, server["args"])
	})

	// Test Cursor (should be disabled)
	t.Run("cursor disabled via override", func(t *testing.T) {
		cursorProjectDir := t.TempDir()
		projectContent := `project {
  name = "test-project"
  agentic_platform = "cursor"
}

plugin "mcp-test" {
  source = "file:` + pluginDir + `"
}
`
		err = os.WriteFile(filepath.Join(cursorProjectDir, "dex.hcl"), []byte(projectContent), 0644)
		require.NoError(t, err)

		installer, err := NewInstaller(cursorProjectDir)
		require.NoError(t, err)

		err = installer.InstallAll()
		require.NoError(t, err)

		// The MCP file might not even exist if no servers are installed
		mcpPath := filepath.Join(cursorProjectDir, ".cursor", "mcp.json")
		if _, err := os.Stat(mcpPath); err == nil {
			// File exists, check it doesn't contain the server
			mcpData, err := os.ReadFile(mcpPath)
			require.NoError(t, err)

			var mcpConfig map[string]any
			err = json.Unmarshal(mcpData, &mcpConfig)
			require.NoError(t, err)

			if mcpServers, ok := mcpConfig["mcpServers"].(map[string]any); ok {
				assert.NotContains(t, mcpServers, "multi-platform")
			}
		}
	})
}

func TestInstaller_MCPServer_MultipleServers(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Set up a local plugin with multiple MCP servers
	pluginDir := t.TempDir()
	pluginContent := `package {
  name = "multi-mcp"
  version = "1.0.0"
  description = "Plugin with multiple MCP servers"
}

mcp_server "filesystem" {
  command = "npx"
  args = ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
}

mcp_server "github" {
  command = "npx"
  args = ["-y", "@modelcontextprotocol/server-github"]
  env = {
    GITHUB_TOKEN = "test-gh-token"
  }
}

mcp_server "api" {
  url = "https://api.example.com/mcp"
  headers = {
    Authorization = "Bearer secret"
  }
}
`
	err := os.WriteFile(filepath.Join(pluginDir, "package.hcl"), []byte(pluginContent), 0644)
	require.NoError(t, err)

	// Create project config
	createTestProject(t, projectDir, `
plugin "multi-mcp" {
  source = "file:`+pluginDir+`"
}
`)

	// Create installer
	installer, err := NewInstaller(projectDir)
	require.NoError(t, err)

	// Install
	err = installer.InstallAll()
	require.NoError(t, err)

	// Verify .mcp.json was created with all servers
	mcpPath := filepath.Join(projectDir, ".mcp.json")
	require.FileExists(t, mcpPath)

	// Read and parse MCP config
	mcpData, err := os.ReadFile(mcpPath)
	require.NoError(t, err)

	var mcpConfig map[string]any
	err = json.Unmarshal(mcpData, &mcpConfig)
	require.NoError(t, err)

	mcpServers := mcpConfig["mcpServers"].(map[string]any)

	// Verify all three servers are present
	assert.Contains(t, mcpServers, "filesystem")
	assert.Contains(t, mcpServers, "github")
	assert.Contains(t, mcpServers, "api")

	// Verify filesystem server
	fsServer := mcpServers["filesystem"].(map[string]any)
	assert.Equal(t, "npx", fsServer["command"])

	// Verify github server with env var substitution
	ghServer := mcpServers["github"].(map[string]any)
	assert.Equal(t, "npx", ghServer["command"])
	ghEnv := ghServer["env"].(map[string]any)
	assert.Equal(t, "test-gh-token", ghEnv["GITHUB_TOKEN"])

	// Verify HTTP server
	apiServer := mcpServers["api"].(map[string]any)
	assert.Equal(t, "https://api.example.com/mcp", apiServer["url"])
}

func TestInstaller_MCPServer_Uninstall(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Set up a local plugin with an MCP server
	pluginDir := t.TempDir()
	pluginContent := `package {
  name = "mcp-test"
  version = "1.0.0"
  description = "Plugin with MCP server"
}

mcp_server "test-server" {
  command = "test-command"
  args = ["--test"]
}
`
	err := os.WriteFile(filepath.Join(pluginDir, "package.hcl"), []byte(pluginContent), 0644)
	require.NoError(t, err)

	// Create project config
	createTestProject(t, projectDir, `
plugin "mcp-test" {
  source = "file:`+pluginDir+`"
}
`)

	// Create installer and install
	installer, err := NewInstaller(projectDir)
	require.NoError(t, err)

	err = installer.InstallAll()
	require.NoError(t, err)

	// Verify server was installed
	mcpPath := filepath.Join(projectDir, ".mcp.json")
	mcpData, err := os.ReadFile(mcpPath)
	require.NoError(t, err)

	var mcpConfig map[string]any
	err = json.Unmarshal(mcpData, &mcpConfig)
	require.NoError(t, err)

	mcpServers := mcpConfig["mcpServers"].(map[string]any)
	assert.Contains(t, mcpServers, "test-server")

	// Now uninstall
	err = installer.Uninstall([]string{"mcp-test"}, false)
	require.NoError(t, err)

	// Verify server was removed
	mcpData, err = os.ReadFile(mcpPath)
	require.NoError(t, err)

	err = json.Unmarshal(mcpData, &mcpConfig)
	require.NoError(t, err)

	mcpServers = mcpConfig["mcpServers"].(map[string]any)
	assert.NotContains(t, mcpServers, "test-server")
}
