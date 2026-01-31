package resource

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTranslateToClaudeMCPServer(t *testing.T) {
	t.Run("basic stdio server", func(t *testing.T) {
		mcp := &MCPServer{
			Name:    "filesystem",
			Command: "npx",
			Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"},
			Env: map[string]string{
				"DEBUG": "true",
			},
		}

		result := TranslateToClaudeMCPServer(mcp)
		require.NotNil(t, result)

		assert.Equal(t, "filesystem", result.Name)
		assert.Equal(t, "command", result.Type)
		assert.Equal(t, "npx", result.Command)
		assert.Equal(t, []string{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"}, result.Args)
		assert.Equal(t, map[string]string{"DEBUG": "true"}, result.Env)
	})

	t.Run("HTTP server", func(t *testing.T) {
		mcp := &MCPServer{
			Name: "api",
			URL:  "https://api.example.com/mcp",
			Headers: map[string]string{
				"Authorization": "Bearer token",
			},
		}

		result := TranslateToClaudeMCPServer(mcp)
		require.NotNil(t, result)

		assert.Equal(t, "api", result.Name)
		assert.Equal(t, "http", result.Type)
		assert.Equal(t, "https://api.example.com/mcp", result.URL)
	})

	t.Run("disabled for claude", func(t *testing.T) {
		disabled := true
		mcp := &MCPServer{
			Name:    "test",
			Command: "test-cmd",
			Claude: &MCPServerPlatformOverride{
				Disabled: disabled,
			},
		}

		result := TranslateToClaudeMCPServer(mcp)
		assert.Nil(t, result)
	})

	t.Run("platform override", func(t *testing.T) {
		mcp := &MCPServer{
			Name:    "test",
			Command: "default-cmd",
			Args:    []string{"--default"},
			Claude: &MCPServerPlatformOverride{
				Command: "claude-cmd",
				Args:    []string{"--claude"},
			},
		}

		result := TranslateToClaudeMCPServer(mcp)
		require.NotNil(t, result)

		assert.Equal(t, "claude-cmd", result.Command)
		assert.Equal(t, []string{"--claude"}, result.Args)
	})
}
