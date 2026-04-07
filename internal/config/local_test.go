package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/launchcg/dex/internal/resource"
)

func TestLoadLocalConfigs_NoFiles(t *testing.T) {
	// Use a fake home directory with no .dex files
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)

	result, err := LoadLocalConfigs("my-project")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestLoadLocalConfigs_GlobalOnly(t *testing.T) {
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)

	dexDir := filepath.Join(fakeHome, ".dex")
	require.NoError(t, os.MkdirAll(dexDir, 0755))

	globalContent := `
mcp_server "my-server" {
  command = "npx"
  args    = ["-y", "my-mcp-server"]
}
`
	require.NoError(t, os.WriteFile(filepath.Join(dexDir, "local.hcl"), []byte(globalContent), 0644))

	result, err := LoadLocalConfigs("my-project")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.MCPServers, 1)
	assert.Equal(t, "my-server", result.MCPServers[0].Name)
}

func TestLoadLocalConfigs_ProjectOverride(t *testing.T) {
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)

	dexDir := filepath.Join(fakeHome, ".dex")
	require.NoError(t, os.MkdirAll(dexDir, 0755))

	// Global config with one MCP server
	globalContent := `
mcp_server "global-server" {
  command = "npx"
  args    = ["-y", "global-mcp"]
}
`
	require.NoError(t, os.WriteFile(filepath.Join(dexDir, "local.hcl"), []byte(globalContent), 0644))

	// Per-project config with another MCP server
	projectDir := filepath.Join(dexDir, "projects", "my-project")
	require.NoError(t, os.MkdirAll(projectDir, 0755))

	projectContent := `
mcp_server "project-server" {
  command = "npx"
  args    = ["-y", "project-mcp"]
}
`
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "project.hcl"), []byte(projectContent), 0644))

	result, err := LoadLocalConfigs("my-project")
	require.NoError(t, err)
	require.NotNil(t, result)

	// Both servers should be present (global first, then project)
	assert.Len(t, result.MCPServers, 2)
	assert.Equal(t, "global-server", result.MCPServers[0].Name)
	assert.Equal(t, "project-server", result.MCPServers[1].Name)
}

func TestMergeLocal_AppendsResources(t *testing.T) {
	project := &ProjectConfig{
		Project: ProjectBlock{
			Name:            "test",
			AgenticPlatform: "claude-code",
		},
	}
	project.buildResources()

	local := &LocalConfig{
		Packages: []PackageBlock{
			{Name: "local-plugin", Source: "file:///tmp/plugin"},
		},
	}

	project.MergeLocal(local)

	require.Len(t, project.Packages, 1)
	assert.Equal(t, "local-plugin", project.Packages[0].Name)
}

// TestMergeLocal_AllResourceFields verifies that every resource field in LocalConfig is
// transferred to ProjectConfig by MergeLocal. This acts as a compiler-equivalent guard:
// if a new field is added to LocalConfig but not wired into toLocalConfig/applyLocalConfig,
// this test will fail.
func TestMergeLocal_AllResourceFields(t *testing.T) {
	project := &ProjectConfig{
		Project: ProjectBlock{Name: "test", AgenticPlatform: "claude-code"},
	}

	local := &LocalConfig{
		Registries:   []RegistryBlock{{Name: "r", Path: "/tmp"}},
		Packages:     []PackageBlock{{Name: "p", Source: "file:///tmp"}},
		Skills:       []resource.Skill{{Name: "sk", Description: "d", Content: "c"}},
		Commands:     []resource.Command{{Name: "cmd", Description: "d", Content: "c"}},
		Agents:       []resource.Agent{{Name: "ag", Description: "d", Content: "c"}},
		Rules:        []resource.Rule{{Name: "rule", Description: "d", Content: "c"}},
		RulesFiles:   []resource.Rules{{Name: "rules", Description: "d", Content: "c"}},
		Settings:     []resource.Settings{{Name: "cfg"}},
		MCPServers:   []resource.MCPServer{{Name: "mcp", Command: "test"}},
		Variables:    []ProjectVariableBlock{{Name: "v", Default: "val"}},
		ResolvedVars: map[string]string{"v": "val"},
	}

	project.MergeLocal(local)

	assert.Len(t, project.Registries, 1, "Registries")
	assert.Len(t, project.Packages, 1, "Packages")
	assert.Len(t, project.Skills, 1, "Skills")
	assert.Len(t, project.Commands, 1, "Commands")
	assert.Len(t, project.Agents, 1, "Agents")
	assert.Len(t, project.Rules, 1, "Rules")
	assert.Len(t, project.RulesFiles, 1, "RulesFiles")
	assert.Len(t, project.Settings, 1, "Settings")
	assert.Len(t, project.MCPServers, 1, "MCPServers")
	assert.Len(t, project.Variables, 1, "Variables")
	assert.Equal(t, "val", project.ResolvedVars["v"], "ResolvedVars")
}

func TestLoadLocalConfigs_ProjectOnly(t *testing.T) {
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)

	projectDir := filepath.Join(fakeHome, ".dex", "projects", "my-project")
	require.NoError(t, os.MkdirAll(projectDir, 0755))

	projectContent := `
mcp_server "project-server" {
  command = "node"
  args    = ["server.js"]
}
`
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "project.hcl"), []byte(projectContent), 0644))

	result, err := LoadLocalConfigs("my-project")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.MCPServers, 1)
	assert.Equal(t, "project-server", result.MCPServers[0].Name)
}
