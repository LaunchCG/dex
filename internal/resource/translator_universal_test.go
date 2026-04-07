package resource

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Skill Translator Tests
// =============================================================================

func TestTranslateToClaudeSkill_BasicFields(t *testing.T) {
	s := &Skill{
		Name:        "code-review",
		Description: "Reviews code",
		Content:     "Review all code",
		Files:       []FileBlock{{Src: "a.txt"}},
	}

	cs := TranslateToClaudeSkill(s)
	require.NotNil(t, cs)
	assert.Equal(t, "code-review", cs.Name)
	assert.Equal(t, "Reviews code", cs.Description)
	assert.Equal(t, "Review all code", cs.Content)
	assert.Len(t, cs.Files, 1)
}

func TestTranslateToClaudeSkill_ClaudeOverride(t *testing.T) {
	s := &Skill{
		Name:        "test",
		Description: "d",
		Content:     "base content",
		Claude: &SkillClaudeOverride{
			Content:      "claude content",
			AllowedTools: []string{"Bash"},
			Model:        "sonnet",
		},
	}

	cs := TranslateToClaudeSkill(s)
	require.NotNil(t, cs)
	assert.Equal(t, "claude content", cs.Content)
	assert.Equal(t, []string{"Bash"}, cs.AllowedTools)
	assert.Equal(t, "sonnet", cs.Model)
}

func TestTranslateToClaudeSkill_DisabledByOverride(t *testing.T) {
	s := &Skill{
		Name:        "test",
		Description: "d",
		Content:     "c",
		Claude:      &SkillClaudeOverride{Disabled: true},
	}

	assert.Nil(t, TranslateToClaudeSkill(s))
}

func TestTranslateToClaudeSkill_DisabledByPlatforms(t *testing.T) {
	s := &Skill{
		Name:        "test",
		Description: "d",
		Content:     "c",
		Platforms:   []string{"github-copilot"},
	}

	assert.Nil(t, TranslateToClaudeSkill(s))
}

func TestTranslateToCopilotSkill_BasicFields(t *testing.T) {
	s := &Skill{
		Name:        "test",
		Description: "d",
		Content:     "base",
	}

	cs := TranslateToCopilotSkill(s)
	require.NotNil(t, cs)
	assert.Equal(t, "test", cs.Name)
	assert.Equal(t, "base", cs.Content)
}

func TestTranslateToCopilotSkill_DisabledByOverride(t *testing.T) {
	s := &Skill{
		Name:        "test",
		Description: "d",
		Content:     "c",
		Copilot:     &PlatformOverride{Disabled: true},
	}

	assert.Nil(t, TranslateToCopilotSkill(s))
}

// =============================================================================
// Command Translator Tests
// =============================================================================

func TestTranslateToClaudeCommand_BasicFields(t *testing.T) {
	c := &Command{
		Name:        "deploy",
		Description: "Deploys app",
		Content:     "Deploy instructions",
	}

	cc := TranslateToClaudeCommand(c)
	require.NotNil(t, cc)
	assert.Equal(t, "deploy", cc.Name)
	assert.Equal(t, "Deploy instructions", cc.Content)
}

func TestTranslateToCopilotPrompt_BasicFields(t *testing.T) {
	c := &Command{
		Name:        "deploy",
		Description: "Deploys app",
		Content:     "Deploy instructions",
	}

	cp := TranslateToCopilotPrompt(c)
	require.NotNil(t, cp)
	assert.Equal(t, "deploy", cp.Name)
	assert.Equal(t, "Deploy instructions", cp.Content)
}

func TestTranslateToCopilotPrompt_CopilotOverride(t *testing.T) {
	c := &Command{
		Name:        "deploy",
		Description: "d",
		Content:     "base",
		Copilot: &CommandCopilotOverride{
			Agent: "edit",
			Model: "gpt-4",
			Tools: []string{"terminal"},
		},
	}

	cp := TranslateToCopilotPrompt(c)
	require.NotNil(t, cp)
	assert.Equal(t, "edit", cp.Agent)
	assert.Equal(t, "gpt-4", cp.Model)
	assert.Equal(t, []string{"terminal"}, cp.Tools)
}

func TestTranslateToCursorCommand_BasicFields(t *testing.T) {
	c := &Command{
		Name:        "deploy",
		Description: "Deploys app",
		Content:     "Deploy instructions",
	}

	cc := TranslateToCursorCommand(c)
	require.NotNil(t, cc)
	assert.Equal(t, "deploy", cc.Name)
	assert.Equal(t, "Deploy instructions", cc.Content)
}

func TestTranslateToCursorCommand_Disabled(t *testing.T) {
	c := &Command{
		Name:        "deploy",
		Description: "d",
		Content:     "c",
		Cursor:      &PlatformOverride{Disabled: true},
	}

	assert.Nil(t, TranslateToCursorCommand(c))
}

// =============================================================================
// Rule Translator Tests
// =============================================================================

func TestTranslateToClaudeRule_WithPaths(t *testing.T) {
	r := &Rule{
		Name:        "lint",
		Description: "Lint rules",
		Content:     "Run lint",
		Claude: &RuleClaudeOverride{
			Paths: []string{"src/**/*.ts"},
		},
	}

	cr := TranslateToClaudeRule(r)
	require.NotNil(t, cr)
	assert.Equal(t, []string{"src/**/*.ts"}, cr.Paths)
}

func TestTranslateToCopilotInstruction_BasicFields(t *testing.T) {
	r := &Rule{
		Name:        "lint",
		Description: "Lint rules",
		Content:     "Run lint",
	}

	ci := TranslateToCopilotInstruction(r)
	require.NotNil(t, ci)
	assert.Equal(t, "lint", ci.Name)
	assert.Equal(t, "Run lint", ci.Content)
}

func TestTranslateToCursorRule_BasicFields(t *testing.T) {
	r := &Rule{
		Name:        "lint",
		Description: "Lint rules",
		Content:     "Run lint",
	}

	cr := TranslateToCursorRule(r)
	require.NotNil(t, cr)
	assert.Equal(t, "lint", cr.Name)
	assert.Equal(t, "Run lint", cr.Content)
}

// =============================================================================
// Rules Translator Tests
// =============================================================================

func TestTranslateToCursorRules_WithGlobs(t *testing.T) {
	r := &Rules{
		Name:        "review",
		Description: "Review rules",
		Content:     "Review code",
		Cursor: &RulesCursorOverride{
			Globs: []string{"*.go"},
		},
	}

	cr := TranslateToCursorRules(r)
	require.NotNil(t, cr)
	assert.Equal(t, []string{"*.go"}, cr.Globs)
}

func TestTranslateToCopilotInstructions_WithApplyTo(t *testing.T) {
	r := &Rules{
		Name:        "review",
		Description: "Review rules",
		Content:     "Review code",
		Copilot: &RulesCopilotOverride{
			ApplyTo: "**/*.ts",
		},
	}

	ci := TranslateToCopilotInstructions(r)
	require.NotNil(t, ci)
	assert.Equal(t, "**/*.ts", ci.ApplyTo)
}

// =============================================================================
// Agent Translator Tests
// =============================================================================

func TestTranslateToClaudeSubagent_BasicFields(t *testing.T) {
	a := &Agent{
		Name:        "explorer",
		Description: "Explores code",
		Content:     "Explore the codebase",
	}

	cs := TranslateToClaudeSubagent(a)
	require.NotNil(t, cs)
	assert.Equal(t, "explorer", cs.Name)
	assert.Equal(t, "Explore the codebase", cs.Content)
}

func TestTranslateToClaudeSubagent_ClaudeOverride(t *testing.T) {
	a := &Agent{
		Name:        "explorer",
		Description: "d",
		Content:     "base",
		Claude: &AgentClaudeOverride{
			Model: "opus",
			Color: "blue",
			Tools: []string{"Read", "Grep"},
		},
	}

	cs := TranslateToClaudeSubagent(a)
	require.NotNil(t, cs)
	assert.Equal(t, "opus", cs.Model)
	assert.Equal(t, "blue", cs.Color)
	assert.Equal(t, []string{"Read", "Grep"}, cs.Tools)
}

func TestTranslateToCopilotAgent_BasicFields(t *testing.T) {
	a := &Agent{
		Name:        "explorer",
		Description: "Explores code",
		Content:     "Explore the codebase",
	}

	ca := TranslateToCopilotAgent(a)
	require.NotNil(t, ca)
	assert.Equal(t, "explorer", ca.Name)
	assert.Equal(t, "Explore the codebase", ca.Content)
}

// =============================================================================
// Settings Translator Tests
// =============================================================================

func TestTranslateToClaudeSettings_BasicFields(t *testing.T) {
	s := &Settings{
		Name: "perms",
		Claude: &SettingsClaudeOverride{
			Allow:                      []string{"Bash"},
			EnableAllProjectMCPServers: true,
		},
	}

	cs := TranslateToClaudeSettings(s)
	require.NotNil(t, cs)
	assert.Equal(t, "perms", cs.Name)
	assert.Equal(t, []string{"Bash"}, cs.Allow)
	assert.True(t, cs.EnableAllProjectMCPServers)
}

func TestTranslateToClaudeSettings_NoClaude(t *testing.T) {
	s := &Settings{Name: "empty"}

	cs := TranslateToClaudeSettings(s)
	require.NotNil(t, cs)
	assert.Equal(t, "empty", cs.Name)
	assert.Empty(t, cs.Allow)
}

func TestTranslateToClaudeSettings_Disabled(t *testing.T) {
	s := &Settings{
		Name:   "perms",
		Claude: &SettingsClaudeOverride{Disabled: true},
	}

	assert.Nil(t, TranslateToClaudeSettings(s))
}

// =============================================================================
// Content Override Tests (applies to all types)
// =============================================================================

func TestTranslate_ContentOverride(t *testing.T) {
	s := &Skill{
		Name:        "test",
		Description: "d",
		Content:     "base content",
		Copilot: &PlatformOverride{
			Content: "copilot-specific content",
		},
	}

	cs := TranslateToCopilotSkill(s)
	require.NotNil(t, cs)
	assert.Equal(t, "copilot-specific content", cs.Content)

	// Claude should still get base content
	claude := TranslateToClaudeSkill(s)
	require.NotNil(t, claude)
	assert.Equal(t, "base content", claude.Content)
}
