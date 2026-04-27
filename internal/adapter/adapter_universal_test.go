package adapter

import (
	"context"
	"log/slog"
	"sync"
	"testing"

	"github.com/launchcg/dex/internal/config"
	"github.com/launchcg/dex/internal/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureHandler is a test slog.Handler that records all log records.
type captureHandler struct {
	mu      sync.Mutex
	records []slog.Record
}

func (h *captureHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }
func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, r)
	return nil
}
func (h *captureHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *captureHandler) WithGroup(_ string) slog.Handler      { return h }

// findRecord searches captured records for a matching message and returns its attrs.
func (h *captureHandler) findRecord(msg string) *slog.Record {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i := range h.records {
		if h.records[i].Message == msg {
			return &h.records[i]
		}
	}
	return nil
}

// getAttr extracts a named attribute from a slog.Record.
func getAttr(r *slog.Record, key string) string {
	var val string
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == key {
			val = a.Value.String()
			return false
		}
		return true
	})
	return val
}

func withCapturedLogs(t *testing.T) *captureHandler {
	t.Helper()
	h := &captureHandler{}
	prev := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return h
}

func emptyPkg() *config.PackageConfig {
	return &config.PackageConfig{
		Meta: config.MetaBlock{Name: "test", Version: "1.0.0"},
	}
}

// =============================================================================
// Cursor Adapter: Unsupported Types Are Logged and Skipped
// =============================================================================

func TestCursorAdapter_SkillProducesPlan(t *testing.T) {
	adapter, err := Get("cursor")
	require.NoError(t, err)

	skill := &resource.Skill{
		Name:        "my-skill",
		Description: "A skill",
		Content:     "Skill content",
	}

	plan, err := adapter.PlanInstallation(skill, emptyPkg(), "/tmp", "/tmp", &InstallContext{})
	require.NoError(t, err)
	require.Len(t, plan.Files, 1)
	assert.Equal(t, ".cursor/skills/my-skill/SKILL.md", plan.Files[0].Path)
	expected := "---\nname: my-skill\ndescription: A skill\n---\nSkill content"
	assert.Equal(t, expected, plan.Files[0].Content)
}

func TestCursorAdapter_AgentSkippedAndLogged(t *testing.T) {
	h := withCapturedLogs(t)
	adapter, err := Get("cursor")
	require.NoError(t, err)

	agent := &resource.Agent{
		Name:        "my-agent",
		Description: "An agent",
		Content:     "Agent content",
	}

	plan, err := adapter.PlanInstallation(agent, emptyPkg(), "/tmp", "/tmp", &InstallContext{})
	require.NoError(t, err)
	assert.Empty(t, plan.Files)

	rec := h.findRecord("resource skipped: not supported by platform")
	require.NotNil(t, rec, "expected warning log for unsupported agent on cursor")
	assert.Equal(t, "my-agent", getAttr(rec, "resource"))
	assert.Equal(t, "agent", getAttr(rec, "type"))
	assert.Equal(t, "cursor", getAttr(rec, "platform"))
}

func TestCursorAdapter_SettingsSkippedAndLogged(t *testing.T) {
	h := withCapturedLogs(t)
	adapter, err := Get("cursor")
	require.NoError(t, err)

	settings := &resource.Settings{
		Name: "my-settings",
		Claude: &resource.SettingsClaudeOverride{
			Allow: []string{"Bash"},
		},
	}

	plan, err := adapter.PlanInstallation(settings, emptyPkg(), "/tmp", "/tmp", &InstallContext{})
	require.NoError(t, err)
	assert.Empty(t, plan.Files)

	rec := h.findRecord("resource skipped: not supported by platform")
	require.NotNil(t, rec, "expected warning log for unsupported settings on cursor")
	assert.Equal(t, "my-settings", getAttr(rec, "resource"))
	assert.Equal(t, "settings", getAttr(rec, "type"))
	assert.Equal(t, "cursor", getAttr(rec, "platform"))
}

// =============================================================================
// Copilot Adapter: Settings Unsupported
// =============================================================================

func TestCopilotAdapter_SettingsSkippedAndLogged(t *testing.T) {
	h := withCapturedLogs(t)
	adapter, err := Get("github-copilot")
	require.NoError(t, err)

	settings := &resource.Settings{
		Name: "my-settings",
		Claude: &resource.SettingsClaudeOverride{
			Allow: []string{"Bash"},
		},
	}

	plan, err := adapter.PlanInstallation(settings, emptyPkg(), "/tmp", "/tmp", &InstallContext{})
	require.NoError(t, err)
	assert.Empty(t, plan.Files)

	rec := h.findRecord("resource skipped: not supported by platform")
	require.NotNil(t, rec, "expected warning log for unsupported settings on copilot")
	assert.Equal(t, "my-settings", getAttr(rec, "resource"))
	assert.Equal(t, "settings", getAttr(rec, "type"))
	assert.Equal(t, "github-copilot", getAttr(rec, "platform"))
}

// =============================================================================
// Claude Adapter: Disabled By Override
// =============================================================================

func TestClaudeAdapter_SkillDisabledByOverrideLogged(t *testing.T) {
	h := withCapturedLogs(t)
	adapter, err := Get("claude-code")
	require.NoError(t, err)

	skill := &resource.Skill{
		Name:        "disabled-skill",
		Description: "A skill",
		Content:     "Content",
		Claude:      &resource.SkillClaudeOverride{Disabled: true},
	}

	plan, err := adapter.PlanInstallation(skill, emptyPkg(), "/tmp", "/tmp", &InstallContext{})
	require.NoError(t, err)
	assert.Empty(t, plan.Files, "disabled skill should produce no files")

	rec := h.findRecord("resource skipped: disabled for platform")
	require.NotNil(t, rec, "expected warning log for disabled skill on claude")
	assert.Equal(t, "disabled-skill", getAttr(rec, "resource"))
	assert.Equal(t, "skill", getAttr(rec, "type"))
	assert.Equal(t, "claude-code", getAttr(rec, "platform"))
}

// =============================================================================
// Supported Types Produce Plans (Not Skipped)
// =============================================================================

func TestCursorAdapter_CommandProducesPlan(t *testing.T) {
	adapter, err := Get("cursor")
	require.NoError(t, err)

	cmd := &resource.Command{
		Name:        "test-cmd",
		Description: "A command",
		Content:     "Command content",
	}

	plan, err := adapter.PlanInstallation(cmd, emptyPkg(), "/tmp", "/tmp", &InstallContext{})
	require.NoError(t, err)
	require.Len(t, plan.Files, 1)

	assert.Equal(t, ".cursor/commands/test-cmd.md", plan.Files[0].Path)
	// Cursor commands are plain markdown — no frontmatter is emitted.
	assert.Equal(t, "Command content", plan.Files[0].Content)
}

func TestCursorAdapter_RuleProducesPlan(t *testing.T) {
	adapter, err := Get("cursor")
	require.NoError(t, err)

	rule := &resource.Rule{
		Name:        "test-rule",
		Description: "A rule",
		Content:     "Rule content",
	}

	plan, err := adapter.PlanInstallation(rule, emptyPkg(), "/tmp", "/tmp", &InstallContext{})
	require.NoError(t, err)
	assert.Equal(t, "Rule content", plan.AgentFileContent)
	assert.Empty(t, plan.Files)
}

func TestClaudeAdapter_SkillProducesPlan(t *testing.T) {
	adapter, err := Get("claude-code")
	require.NoError(t, err)

	skill := &resource.Skill{
		Name:        "test-skill",
		Description: "A skill",
		Content:     "Skill content",
	}

	plan, err := adapter.PlanInstallation(skill, emptyPkg(), "/tmp", "/tmp", &InstallContext{})
	require.NoError(t, err)
	require.Len(t, plan.Files, 1)

	assert.Equal(t, ".claude/skills/test-skill/SKILL.md", plan.Files[0].Path)
	assert.Equal(t, "---\nname: test-skill\ndescription: A skill\n---\nSkill content", plan.Files[0].Content)
}

func TestClaudeAdapter_RuleProducesPlan(t *testing.T) {
	adapter, err := Get("claude-code")
	require.NoError(t, err)

	rule := &resource.Rule{
		Name:        "test-rule",
		Description: "A rule",
		Content:     "Rule content",
	}

	plan, err := adapter.PlanInstallation(rule, emptyPkg(), "/tmp", "/tmp", &InstallContext{})
	require.NoError(t, err)
	assert.Equal(t, "Rule content", plan.AgentFileContent)
	assert.Empty(t, plan.Files)
}

func TestClaudeAdapter_CommandProducesPlan(t *testing.T) {
	adapter, err := Get("claude-code")
	require.NoError(t, err)

	cmd := &resource.Command{
		Name:        "test-cmd",
		Description: "A command",
		Content:     "Command content",
	}

	plan, err := adapter.PlanInstallation(cmd, emptyPkg(), "/tmp", "/tmp", &InstallContext{})
	require.NoError(t, err)
	require.Len(t, plan.Files, 1)

	assert.Equal(t, ".claude/commands/test-cmd.md", plan.Files[0].Path)
	assert.Equal(t, "---\nname: test-cmd\ndescription: A command\n---\nCommand content", plan.Files[0].Content)
}

func TestClaudeAdapter_AgentProducesPlan(t *testing.T) {
	adapter, err := Get("claude-code")
	require.NoError(t, err)

	agent := &resource.Agent{
		Name:        "explorer",
		Description: "Explores code",
		Content:     "Explore the codebase",
		Claude: &resource.AgentClaudeOverride{
			Model: "sonnet",
			Tools: []string{"Read", "Grep"},
		},
	}

	plan, err := adapter.PlanInstallation(agent, emptyPkg(), "/tmp", "/tmp", &InstallContext{})
	require.NoError(t, err)
	require.Len(t, plan.Files, 1)

	assert.Equal(t, ".claude/agents/explorer.md", plan.Files[0].Path)
	assert.Equal(t, "---\nname: explorer\ndescription: Explores code\nmodel: sonnet\ntools:\n- Read\n- Grep\n---\nExplore the codebase", plan.Files[0].Content)
}

func TestClaudeAdapter_SettingsProducesPlan(t *testing.T) {
	adapter, err := Get("claude-code")
	require.NoError(t, err)

	settings := &resource.Settings{
		Name: "perms",
		Claude: &resource.SettingsClaudeOverride{
			Allow:                      []string{"Bash"},
			EnableAllProjectMCPServers: true,
		},
	}

	plan, err := adapter.PlanInstallation(settings, emptyPkg(), "/tmp", "/tmp", &InstallContext{})
	require.NoError(t, err)
	require.NotNil(t, plan.SettingsEntries)

	// SettingsEntries is a flat map with the settings keys
	allowEntries, ok := plan.SettingsEntries["allow"].([]any)
	require.True(t, ok, "allow should be a []any")
	require.Len(t, allowEntries, 1)
	assert.Equal(t, "Bash", allowEntries[0])
	assert.Equal(t, true, plan.SettingsEntries["enableAllProjectMcpServers"])
}

func TestCopilotAdapter_CommandTranslatesToPrompt(t *testing.T) {
	adapter, err := Get("github-copilot")
	require.NoError(t, err)

	cmd := &resource.Command{
		Name:        "test-cmd",
		Description: "A command",
		Content:     "Command content",
	}

	plan, err := adapter.PlanInstallation(cmd, emptyPkg(), "/tmp", "/tmp", &InstallContext{})
	require.NoError(t, err)
	require.Len(t, plan.Files, 1)

	assert.Equal(t, ".github/prompts/test-cmd.prompt.md", plan.Files[0].Path)
	assert.Equal(t, "---\ndescription: A command\n---\nCommand content", plan.Files[0].Content)
}

func TestCopilotAdapter_RuleTranslatesToInstruction(t *testing.T) {
	adapter, err := Get("github-copilot")
	require.NoError(t, err)

	rule := &resource.Rule{
		Name:        "test-rule",
		Description: "A rule",
		Content:     "Rule content",
	}

	plan, err := adapter.PlanInstallation(rule, emptyPkg(), "/tmp", "/tmp", &InstallContext{})
	require.NoError(t, err)
	assert.Equal(t, "Rule content", plan.AgentFileContent)
	assert.Empty(t, plan.Files)
}

func TestCopilotAdapter_SkillProducesPlan(t *testing.T) {
	adapter, err := Get("github-copilot")
	require.NoError(t, err)

	skill := &resource.Skill{
		Name:        "test-skill",
		Description: "A skill",
		Content:     "Skill content",
	}

	plan, err := adapter.PlanInstallation(skill, emptyPkg(), "/tmp", "/tmp", &InstallContext{})
	require.NoError(t, err)
	require.Len(t, plan.Files, 1)

	assert.Equal(t, ".github/skills/test-skill/SKILL.md", plan.Files[0].Path)
	assert.Equal(t, "---\nname: test-skill\ndescription: A skill\n---\nSkill content", plan.Files[0].Content)
}

func TestCopilotAdapter_AgentProducesPlan(t *testing.T) {
	adapter, err := Get("github-copilot")
	require.NoError(t, err)

	agent := &resource.Agent{
		Name:        "reviewer",
		Description: "Reviews code",
		Content:     "Review the code",
		Copilot: &resource.AgentCopilotOverride{
			Model: "gpt-4",
			Tools: []string{"terminal"},
		},
	}

	plan, err := adapter.PlanInstallation(agent, emptyPkg(), "/tmp", "/tmp", &InstallContext{})
	require.NoError(t, err)
	require.Len(t, plan.Files, 1)

	assert.Equal(t, ".github/agents/reviewer.agent.md", plan.Files[0].Path)
	assert.Equal(t, "---\ndescription: Reviews code\nmodel: gpt-4\ntools:\n- terminal\n---\nReview the code", plan.Files[0].Content)
}
