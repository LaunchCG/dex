package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/launchcg/dex/internal/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadProject_WithProfiles(t *testing.T) {
	tmpDir := t.TempDir()
	hclContent := `
project {
  name             = "test-project"
  default_platform = "claude-code"
}

registry "default-reg" {
  path = "/default"
}

plugin "default-plugin" {
  registry = "default-reg"
}

profile "qa" {
  agent_instructions = "QA instructions"

  registry "qa-reg" {
    path = "/qa"
  }

  plugin "qa-plugin" {
    registry = "default-reg"
  }

  claude_rule "qa-rule" {
    description = "QA rule"
    content     = "QA rule content"
  }
}

profile "staging" {
  exclude_defaults = true

  plugin "staging-only" {
    registry = "default-reg"
  }
}
`
	err := os.WriteFile(filepath.Join(tmpDir, "dex.hcl"), []byte(hclContent), 0644)
	require.NoError(t, err)

	config, err := LoadProject(tmpDir)
	require.NoError(t, err)
	assert.Len(t, config.Profiles, 2)
	assert.Equal(t, "qa", config.Profiles[0].Name)
	assert.Equal(t, "staging", config.Profiles[1].Name)
	assert.False(t, config.Profiles[0].ExcludeDefaults)
	assert.True(t, config.Profiles[1].ExcludeDefaults)
}

func TestApplyProfile_AdditiveMerge(t *testing.T) {
	tmpDir := t.TempDir()
	hclContent := `
project {
  name             = "test-project"
  default_platform = "claude-code"
  agent_instructions = "Default instructions"
}

registry "default-reg" {
  path = "/default"
}

plugin "default-plugin" {
  registry = "default-reg"
}

claude_rule "default-rule" {
  description = "Default rule"
  content     = "Default rule"
}

profile "qa" {
  agent_instructions = "QA instructions"

  registry "qa-reg" {
    path = "/qa"
  }

  plugin "qa-plugin" {
    registry = "default-reg"
  }

  claude_rule "qa-rule" {
    description = "QA rule"
    content     = "QA rule"
  }
}
`
	err := os.WriteFile(filepath.Join(tmpDir, "dex.hcl"), []byte(hclContent), 0644)
	require.NoError(t, err)

	config, err := LoadProjectWithProfile(tmpDir, "qa")
	require.NoError(t, err)

	// Registries: default + qa appended
	assert.Len(t, config.Registries, 2)
	assert.Equal(t, "default-reg", config.Registries[0].Name)
	assert.Equal(t, "qa-reg", config.Registries[1].Name)

	// Plugins: default + qa appended
	assert.Len(t, config.Plugins, 2)
	assert.Equal(t, "default-plugin", config.Plugins[0].Name)
	assert.Equal(t, "qa-plugin", config.Plugins[1].Name)

	// Agent instructions: replaced by profile
	assert.Equal(t, "QA instructions", config.Project.AgentInstructions)

	// Rules: default + qa appended
	assert.Len(t, config.Rules, 2)
	assert.Equal(t, "default-rule", config.Rules[0].Name)
	assert.Equal(t, "qa-rule", config.Rules[1].Name)

	// Profiles cleared after apply
	assert.Nil(t, config.Profiles)
}

func TestApplyProfile_SameNameReplaces(t *testing.T) {
	tmpDir := t.TempDir()
	hclContent := `
project {
  name             = "test-project"
  default_platform = "claude-code"
}

registry "shared-reg" {
  path = "/default-path"
}

plugin "shared-plugin" {
  registry = "shared-reg"
  version  = "1.0.0"
}

claude_rule "shared-rule" {
  description = "Shared rule"
  content     = "Default content"
}

profile "qa" {
  registry "shared-reg" {
    path = "/qa-path"
  }

  plugin "shared-plugin" {
    registry = "shared-reg"
    version  = "2.0.0"
  }

  claude_rule "shared-rule" {
    description = "Shared rule"
    content     = "QA content"
  }
}
`
	err := os.WriteFile(filepath.Join(tmpDir, "dex.hcl"), []byte(hclContent), 0644)
	require.NoError(t, err)

	config, err := LoadProjectWithProfile(tmpDir, "qa")
	require.NoError(t, err)

	// Same-name items replaced, not duplicated
	assert.Len(t, config.Registries, 1)
	assert.Equal(t, "/qa-path", config.Registries[0].Path)

	assert.Len(t, config.Plugins, 1)
	assert.Equal(t, "2.0.0", config.Plugins[0].Version)

	assert.Len(t, config.Rules, 1)
	assert.Equal(t, "QA content", config.Rules[0].Content)
}

func TestApplyProfile_ExcludeDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	hclContent := `
project {
  name             = "test-project"
  default_platform = "claude-code"
  agent_instructions = "Default instructions"
}

registry "default-reg" {
  path = "/default"
}

plugin "default-plugin" {
  registry = "default-reg"
}

claude_rule "default-rule" {
  description = "Default rule"
  content     = "Default rule"
}

profile "clean" {
  exclude_defaults = true
  agent_instructions = "Clean instructions"

  registry "clean-reg" {
    path = "/clean"
  }

  plugin "clean-plugin" {
    registry = "clean-reg"
  }
}
`
	err := os.WriteFile(filepath.Join(tmpDir, "dex.hcl"), []byte(hclContent), 0644)
	require.NoError(t, err)

	config, err := LoadProjectWithProfile(tmpDir, "clean")
	require.NoError(t, err)

	// Only profile items, no defaults
	assert.Len(t, config.Registries, 1)
	assert.Equal(t, "clean-reg", config.Registries[0].Name)

	assert.Len(t, config.Plugins, 1)
	assert.Equal(t, "clean-plugin", config.Plugins[0].Name)

	assert.Equal(t, "Clean instructions", config.Project.AgentInstructions)

	// Default rules excluded
	assert.Empty(t, config.Rules)
}

func TestApplyProfile_FallbackDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	hclContent := `
project {
  name             = "test-project"
  default_platform = "claude-code"
  agent_instructions = "Default instructions"
}

registry "default-reg" {
  path = "/default"
}

plugin "default-plugin" {
  registry = "default-reg"
}

claude_rule "default-rule" {
  description = "Default rule"
  content     = "Default rule"
}

profile "empty" {
}
`
	err := os.WriteFile(filepath.Join(tmpDir, "dex.hcl"), []byte(hclContent), 0644)
	require.NoError(t, err)

	config, err := LoadProjectWithProfile(tmpDir, "empty")
	require.NoError(t, err)

	// Everything preserved from defaults
	assert.Len(t, config.Registries, 1)
	assert.Equal(t, "default-reg", config.Registries[0].Name)
	assert.Len(t, config.Plugins, 1)
	assert.Equal(t, "default-plugin", config.Plugins[0].Name)
	assert.Equal(t, "Default instructions", config.Project.AgentInstructions)
	assert.Len(t, config.Rules, 1)
	assert.Equal(t, "default-rule", config.Rules[0].Name)
}

func TestApplyProfile_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	hclContent := `
project {
  name             = "test-project"
  default_platform = "claude-code"
}

profile "qa" {
}
`
	err := os.WriteFile(filepath.Join(tmpDir, "dex.hcl"), []byte(hclContent), 0644)
	require.NoError(t, err)

	config, err := LoadProjectWithProfile(tmpDir, "nonexistent")
	assert.Nil(t, config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
	assert.Contains(t, err.Error(), "qa")
}

func TestApplyProfile_NoProfileFlag(t *testing.T) {
	tmpDir := t.TempDir()
	hclContent := `
project {
  name             = "test-project"
  default_platform = "claude-code"
}

plugin "default-plugin" {
  source = "file:///test"
}

profile "qa" {
  plugin "qa-plugin" {
    source = "file:///qa"
  }
}
`
	err := os.WriteFile(filepath.Join(tmpDir, "dex.hcl"), []byte(hclContent), 0644)
	require.NoError(t, err)

	// Empty profile string = no profile applied
	config, err := LoadProjectWithProfile(tmpDir, "")
	require.NoError(t, err)

	// Profile blocks are parsed but not applied
	assert.Len(t, config.Profiles, 1)
	assert.Len(t, config.Plugins, 1)
	assert.Equal(t, "default-plugin", config.Plugins[0].Name)
}

func TestValidate_DuplicateProfileNames(t *testing.T) {
	cfg := &ProjectConfig{
		Project: ProjectBlock{
			Name:            "test",
			AgenticPlatform: "claude-code",
		},
		Profiles: []ProfileBlock{
			{Name: "qa"},
			{Name: "qa"},
		},
	}

	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate profile name")
}

func TestApplyProfile_ResourcesRebuilt(t *testing.T) {
	tmpDir := t.TempDir()
	hclContent := `
project {
  name             = "test-project"
  default_platform = "claude-code"
}

claude_rule "default-rule" {
  description = "Default"
  content     = "Default"
}

profile "qa" {
  claude_rule "qa-rule" {
    description = "QA"
    content     = "QA"
  }
}
`
	err := os.WriteFile(filepath.Join(tmpDir, "dex.hcl"), []byte(hclContent), 0644)
	require.NoError(t, err)

	config, err := LoadProjectWithProfile(tmpDir, "qa")
	require.NoError(t, err)

	// Resources slice should include both default and profile rules
	assert.Len(t, config.Resources, 2)
}

func TestMergeByName(t *testing.T) {
	getName := func(p PluginBlock) string { return p.Name }

	defaults := []PluginBlock{
		{Name: "a", Version: "1.0"},
		{Name: "b", Version: "1.0"},
		{Name: "c", Version: "1.0"},
	}

	t.Run("no overrides", func(t *testing.T) {
		result := mergeByName(defaults, nil, getName)
		assert.Equal(t, defaults, result)
	})

	t.Run("new items appended", func(t *testing.T) {
		overrides := []PluginBlock{{Name: "d", Version: "2.0"}}
		result := mergeByName(defaults, overrides, getName)
		assert.Len(t, result, 4)
		assert.Equal(t, "d", result[3].Name)
	})

	t.Run("same name replaced in place", func(t *testing.T) {
		overrides := []PluginBlock{{Name: "b", Version: "2.0"}}
		result := mergeByName(defaults, overrides, getName)
		assert.Len(t, result, 3)
		assert.Equal(t, "a", result[0].Name)
		assert.Equal(t, "1.0", result[0].Version)
		assert.Equal(t, "b", result[1].Name)
		assert.Equal(t, "2.0", result[1].Version) // replaced
		assert.Equal(t, "c", result[2].Name)
	})

	t.Run("mix of replace and append", func(t *testing.T) {
		overrides := []PluginBlock{
			{Name: "a", Version: "3.0"},
			{Name: "e", Version: "1.0"},
		}
		result := mergeByName(defaults, overrides, getName)
		assert.Len(t, result, 4)
		assert.Equal(t, "a", result[0].Name)
		assert.Equal(t, "3.0", result[0].Version) // replaced
		assert.Equal(t, "b", result[1].Name)
		assert.Equal(t, "c", result[2].Name)
		assert.Equal(t, "e", result[3].Name) // appended
	})
}

func TestApplyProfile_AgentInstructionsFallback(t *testing.T) {
	cfg := &ProjectConfig{
		Project: ProjectBlock{
			Name:              "test",
			AgenticPlatform:   "claude-code",
			AgentInstructions: "Default instructions",
		},
		Profiles: []ProfileBlock{
			{Name: "no-instructions"},
		},
	}

	err := cfg.ApplyProfile("no-instructions")
	require.NoError(t, err)
	assert.Equal(t, "Default instructions", cfg.Project.AgentInstructions)
}

func TestApplyProfile_MultipleResourceTypes(t *testing.T) {
	cfg := &ProjectConfig{
		Project: ProjectBlock{
			Name:            "test",
			AgenticPlatform: "claude-code",
		},
		Skills: []resource.ClaudeSkill{
			{Name: "default-skill", Description: "Default"},
		},
		Rules: []resource.ClaudeRule{
			{Name: "default-rule", Content: "Default"},
		},
		Profiles: []ProfileBlock{
			{
				Name: "qa",
				Rules: []resource.ClaudeRule{
					{Name: "qa-rule", Content: "QA"},
				},
			},
		},
	}

	err := cfg.ApplyProfile("qa")
	require.NoError(t, err)

	// Skills preserved (profile didn't define any)
	assert.Len(t, cfg.Skills, 1)
	assert.Equal(t, "default-skill", cfg.Skills[0].Name)

	// Rules: default + qa
	assert.Len(t, cfg.Rules, 2)
	assert.Equal(t, "default-rule", cfg.Rules[0].Name)
	assert.Equal(t, "qa-rule", cfg.Rules[1].Name)
}
