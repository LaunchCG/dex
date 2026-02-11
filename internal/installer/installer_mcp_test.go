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

	// Verify full MCP config
	assert.Equal(t, map[string]any{
		"mcpServers": map[string]any{
			"filesystem": map[string]any{
				"command": "npx",
				"args":    []any{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"},
				"env":     map[string]any{"DEBUG": "true"},
			},
		},
	}, mcpConfig)
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

	// Verify full MCP config with namespaced server name
	assert.Equal(t, map[string]any{
		"mcpServers": map[string]any{
			"mcp-test-filesystem": map[string]any{
				"command": "npx",
				"args":    []any{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"},
			},
		},
	}, mcpConfig)
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

	// Verify full MCP config for Cursor
	assert.Equal(t, map[string]any{
		"mcpServers": map[string]any{
			"context7": map[string]any{
				"url":     "https://mcp.context7.com/mcp",
				"headers": map[string]any{"Authorization": "Bearer test-token"},
			},
		},
	}, mcpConfig)
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

	// Verify full MCP config (Copilot uses "servers" key, not "mcpServers")
	assert.Equal(t, map[string]any{
		"servers": map[string]any{
			"database": map[string]any{
				"type":    "stdio",
				"command": "db-server",
				"args":    []any{"--host", "localhost"},
				"env":     map[string]any{"DB_HOST": "localhost", "DB_PORT": "5432"},
			},
		},
	}, mcpConfig)
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

		assert.Equal(t, map[string]any{
			"mcpServers": map[string]any{
				"multi-platform": map[string]any{
					"command": "claude-command",
					"args":    []any{"--claude"},
				},
			},
		}, mcpConfig)
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

		// The MCP file should not exist since the server is disabled for Cursor
		mcpPath := filepath.Join(cursorProjectDir, ".cursor", "mcp.json")
		_, err = os.Stat(mcpPath)
		assert.True(t, os.IsNotExist(err), ".cursor/mcp.json should not exist when server is disabled")
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

	// Verify full MCP config with all three servers
	assert.Equal(t, map[string]any{
		"mcpServers": map[string]any{
			"filesystem": map[string]any{
				"command": "npx",
				"args":    []any{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"},
			},
			"github": map[string]any{
				"command": "npx",
				"args":    []any{"-y", "@modelcontextprotocol/server-github"},
				"env":     map[string]any{"GITHUB_TOKEN": "test-gh-token"},
			},
			"api": map[string]any{
				"url": "https://api.example.com/mcp",
			},
		},
	}, mcpConfig)
}

func TestInstaller_MCPServer_Copilot_WithInputs(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Set up a local plugin with a copilot_mcp_server that has inputs
	pluginDir := t.TempDir()
	pluginContent := `package {
  name = "ado-plugin"
  version = "1.0.0"
  description = "Plugin with MCP server inputs"
}

copilot_mcp_server "ado" {
  type    = "stdio"
  command = "npx"
  args    = ["-y", "@azure-devops/mcp", "$${input:ado_org}"]

  input "ado_org" {
    type        = "promptString"
    description = "Azure DevOps organization name"
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

plugin "ado-plugin" {
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

	// Verify full MCP config with inputs
	assert.Equal(t, map[string]any{
		"inputs": []any{
			map[string]any{
				"id":          "ado_org",
				"type":        "promptString",
				"description": "Azure DevOps organization name",
			},
		},
		"servers": map[string]any{
			"ado": map[string]any{
				"type":    "stdio",
				"command": "npx",
				"args":    []any{"-y", "@azure-devops/mcp", "${input:ado_org}"},
			},
		},
	}, mcpConfig)
}

func TestInstaller_MCPServer_Universal_WithCopilotInputOverride(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Set up a local plugin with a universal mcp_server + copilot input override
	pluginDir := t.TempDir()
	pluginContent := `package {
  name = "ado-plugin"
  version = "1.0.0"
  description = "Plugin with universal MCP server and copilot input override"
}

mcp_server "ado" {
  command = "npx"
  args    = ["-y", "@azure-devops/mcp", "$${input:ado_org}"]

  copilot {
    input "ado_org" {
      type        = "promptString"
      description = "Azure DevOps organization name"
    }
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

plugin "ado-plugin" {
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

	// Verify full MCP config with inputs from copilot override
	assert.Equal(t, map[string]any{
		"inputs": []any{
			map[string]any{
				"id":          "ado_org",
				"type":        "promptString",
				"description": "Azure DevOps organization name",
			},
		},
		"servers": map[string]any{
			"ado": map[string]any{
				"type":    "stdio",
				"command": "npx",
				"args":    []any{"-y", "@azure-devops/mcp", "${input:ado_org}"},
			},
		},
	}, mcpConfig)
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

	assert.Equal(t, map[string]any{
		"mcpServers": map[string]any{
			"test-server": map[string]any{
				"command": "test-command",
				"args":    []any{"--test"},
			},
		},
	}, mcpConfig)

	// Now uninstall
	err = installer.Uninstall([]string{"mcp-test"}, false)
	require.NoError(t, err)

	// Verify .mcp.json was deleted since it's now empty
	_, err = os.Stat(mcpPath)
	assert.True(t, os.IsNotExist(err), ".mcp.json should be deleted when all servers are removed")
}
