package resource

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClaudeSkill_ResourceType(t *testing.T) {
	skill := &ClaudeSkill{}
	assert.Equal(t, "claude_skill", skill.ResourceType())
}

func TestClaudeSkill_ResourceName(t *testing.T) {
	tests := []struct {
		name  string
		skill ClaudeSkill
		want  string
	}{
		{
			name:  "empty name",
			skill: ClaudeSkill{},
			want:  "",
		},
		{
			name: "with name",
			skill: ClaudeSkill{
				Name: "my-skill",
			},
			want: "my-skill",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.skill.ResourceName())
		})
	}
}

func TestClaudeSkill_GetContent(t *testing.T) {
	tests := []struct {
		name  string
		skill ClaudeSkill
		want  string
	}{
		{
			name:  "empty content",
			skill: ClaudeSkill{},
			want:  "",
		},
		{
			name: "with content",
			skill: ClaudeSkill{
				Content: "This is the content",
			},
			want: "This is the content",
		},
		{
			name: "multiline content",
			skill: ClaudeSkill{
				Content: "Line 1\nLine 2\nLine 3",
			},
			want: "Line 1\nLine 2\nLine 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.skill.GetContent()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClaudeSkill_GetFiles(t *testing.T) {
	tests := []struct {
		name  string
		skill ClaudeSkill
		want  []FileBlock
	}{
		{
			name:  "nil files",
			skill: ClaudeSkill{},
			want:  nil,
		},
		{
			name: "empty files",
			skill: ClaudeSkill{
				Files: []FileBlock{},
			},
			want: []FileBlock{},
		},
		{
			name: "single file",
			skill: ClaudeSkill{
				Files: []FileBlock{
					{Src: "src/file.txt", Dest: "dest/file.txt", Chmod: "644"},
				},
			},
			want: []FileBlock{
				{Src: "src/file.txt", Dest: "dest/file.txt", Chmod: "644"},
			},
		},
		{
			name: "multiple files",
			skill: ClaudeSkill{
				Files: []FileBlock{
					{Src: "file1.txt"},
					{Src: "file2.txt", Dest: "renamed.txt"},
					{Src: "script.sh", Chmod: "755"},
				},
			},
			want: []FileBlock{
				{Src: "file1.txt"},
				{Src: "file2.txt", Dest: "renamed.txt"},
				{Src: "script.sh", Chmod: "755"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.skill.GetFiles()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClaudeSkill_GetTemplateFiles(t *testing.T) {
	tests := []struct {
		name  string
		skill ClaudeSkill
		want  []TemplateFileBlock
	}{
		{
			name:  "nil template files",
			skill: ClaudeSkill{},
			want:  nil,
		},
		{
			name: "empty template files",
			skill: ClaudeSkill{
				TemplateFiles: []TemplateFileBlock{},
			},
			want: []TemplateFileBlock{},
		},
		{
			name: "single template file",
			skill: ClaudeSkill{
				TemplateFiles: []TemplateFileBlock{
					{Src: "config.tmpl", Dest: "config.yaml", Vars: map[string]string{"key": "value"}},
				},
			},
			want: []TemplateFileBlock{
				{Src: "config.tmpl", Dest: "config.yaml", Vars: map[string]string{"key": "value"}},
			},
		},
		{
			name: "multiple template files",
			skill: ClaudeSkill{
				TemplateFiles: []TemplateFileBlock{
					{Src: "config.tmpl"},
					{Src: "settings.tmpl", Dest: "settings.json"},
					{Src: "script.tmpl", Chmod: "755", Vars: map[string]string{"env": "prod"}},
				},
			},
			want: []TemplateFileBlock{
				{Src: "config.tmpl"},
				{Src: "settings.tmpl", Dest: "settings.json"},
				{Src: "script.tmpl", Chmod: "755", Vars: map[string]string{"env": "prod"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.skill.GetTemplateFiles()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClaudeSkill_Validate(t *testing.T) {
	validSkill := &ClaudeSkill{
		Name:        "test-skill",
		Description: "A test skill",
		Content:     "Skill content here",
	}

	err := validSkill.Validate()
	assert.NoError(t, err)
}

func TestClaudeSkill_Validate_MissingName(t *testing.T) {
	skill := &ClaudeSkill{
		Description: "A test skill",
		Content:     "Skill content",
	}

	err := skill.Validate()
	require.Error(t, err)
	assert.EqualError(t, err, "claude_skill: name is required")
}

func TestClaudeSkill_Validate_MissingDescription(t *testing.T) {
	skill := &ClaudeSkill{
		Name:    "test-skill",
		Content: "Skill content",
	}

	err := skill.Validate()
	require.Error(t, err)
	assert.EqualError(t, err, `claude_skill "test-skill": description is required`)
}

func TestClaudeSkill_Validate_MissingContent(t *testing.T) {
	skill := &ClaudeSkill{
		Name:        "test-skill",
		Description: "A test skill",
	}

	err := skill.Validate()
	require.Error(t, err)
	assert.EqualError(t, err, `claude_skill "test-skill": content is required`)
}

func TestClaudeCommand_ResourceType(t *testing.T) {
	cmd := &ClaudeCommand{}
	assert.Equal(t, "claude_command", cmd.ResourceType())
}

func TestClaudeCommand_ResourceName(t *testing.T) {
	cmd := &ClaudeCommand{
		Name: "my-command",
	}
	assert.Equal(t, "my-command", cmd.ResourceName())
}

func TestClaudeCommand_Validate(t *testing.T) {
	tests := []struct {
		name    string
		command ClaudeCommand
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid command",
			command: ClaudeCommand{
				Name:        "test-command",
				Description: "A test command",
				Content:     "Command instructions",
			},
			wantErr: false,
		},
		{
			name: "valid command with optional fields",
			command: ClaudeCommand{
				Name:         "test-command",
				Description:  "A test command",
				Content:      "Command instructions",
				ArgumentHint: "[env]",
				AllowedTools: []string{"Read", "Write"},
				Model:        "sonnet",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			command: ClaudeCommand{
				Description: "A test command",
				Content:     "Command instructions",
			},
			wantErr: true,
			errMsg:  "claude_command: name is required",
		},
		{
			name: "missing description",
			command: ClaudeCommand{
				Name:    "test-command",
				Content: "Command instructions",
			},
			wantErr: true,
			errMsg:  `claude_command "test-command": description is required`,
		},
		{
			name: "missing content",
			command: ClaudeCommand{
				Name:        "test-command",
				Description: "A test command",
			},
			wantErr: true,
			errMsg:  `claude_command "test-command": content is required`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.command.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.EqualError(t, err, tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClaudeSubagent_ResourceType(t *testing.T) {
	agent := &ClaudeSubagent{}
	assert.Equal(t, "claude_subagent", agent.ResourceType())
}

func TestClaudeSubagent_ResourceName(t *testing.T) {
	agent := &ClaudeSubagent{
		Name: "my-agent",
	}
	assert.Equal(t, "my-agent", agent.ResourceName())
}

func TestClaudeSubagent_Validate(t *testing.T) {
	tests := []struct {
		name     string
		subagent ClaudeSubagent
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid subagent",
			subagent: ClaudeSubagent{
				Name:        "test-agent",
				Description: "A test agent",
				Content:     "Agent instructions",
			},
			wantErr: false,
		},
		{
			name: "valid subagent with optional fields",
			subagent: ClaudeSubagent{
				Name:        "test-agent",
				Description: "A test agent",
				Content:     "Agent instructions",
				Model:       "opus",
				Color:       "blue",
				Tools:       []string{"Read", "Write", "Bash"},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			subagent: ClaudeSubagent{
				Description: "A test agent",
				Content:     "Agent instructions",
			},
			wantErr: true,
			errMsg:  "claude_subagent: name is required",
		},
		{
			name: "missing description",
			subagent: ClaudeSubagent{
				Name:    "test-agent",
				Content: "Agent instructions",
			},
			wantErr: true,
			errMsg:  `claude_subagent "test-agent": description is required`,
		},
		{
			name: "missing content",
			subagent: ClaudeSubagent{
				Name:        "test-agent",
				Description: "A test agent",
			},
			wantErr: true,
			errMsg:  `claude_subagent "test-agent": content is required`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.subagent.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.EqualError(t, err, tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClaudeRule_ResourceType(t *testing.T) {
	rule := &ClaudeRule{}
	assert.Equal(t, "claude_rule", rule.ResourceType())
}

func TestClaudeRule_ResourceName(t *testing.T) {
	rule := &ClaudeRule{
		Name: "my-rule",
	}
	assert.Equal(t, "my-rule", rule.ResourceName())
}

func TestClaudeRule_Validate(t *testing.T) {
	tests := []struct {
		name    string
		rule    ClaudeRule
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid rule",
			rule: ClaudeRule{
				Name:        "test-rule",
				Description: "A test rule",
				Content:     "Rule content",
			},
			wantErr: false,
		},
		{
			name: "valid rule with paths",
			rule: ClaudeRule{
				Name:        "test-rule",
				Description: "A test rule",
				Content:     "Rule content",
				Paths:       []string{"*.go", "**/*.ts"},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			rule: ClaudeRule{
				Description: "A test rule",
				Content:     "Rule content",
			},
			wantErr: true,
			errMsg:  "claude_rule: name is required",
		},
		{
			name: "missing description",
			rule: ClaudeRule{
				Name:    "test-rule",
				Content: "Rule content",
			},
			wantErr: true,
			errMsg:  `claude_rule "test-rule": description is required`,
		},
		{
			name: "missing content",
			rule: ClaudeRule{
				Name:        "test-rule",
				Description: "A test rule",
			},
			wantErr: true,
			errMsg:  `claude_rule "test-rule": content is required`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rule.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.EqualError(t, err, tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClaudeRules_ResourceType(t *testing.T) {
	rules := &ClaudeRules{}
	assert.Equal(t, "claude_rules", rules.ResourceType())
}

func TestClaudeRules_ResourceName(t *testing.T) {
	rules := &ClaudeRules{
		Name: "my-rules",
	}
	assert.Equal(t, "my-rules", rules.ResourceName())
}

func TestClaudeRules_Validate(t *testing.T) {
	tests := []struct {
		name    string
		rules   ClaudeRules
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid rules",
			rules: ClaudeRules{
				Name:        "test-rules",
				Description: "Test rules",
				Content:     "Rules content",
			},
			wantErr: false,
		},
		{
			name: "valid rules with paths",
			rules: ClaudeRules{
				Name:        "test-rules",
				Description: "Test rules",
				Content:     "Rules content",
				Paths:       []string{"src/**", "lib/**"},
			},
			wantErr: false,
		},
		{
			name: "valid rules with only file blocks",
			rules: ClaudeRules{
				Name:        "test-rules",
				Description: "Test rules",
				Content:     "", // No content
				Files: []FileBlock{
					{Src: "file1.md"},
					{Src: "file2.md"},
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			rules: ClaudeRules{
				Description: "Test rules",
				Content:     "Rules content",
			},
			wantErr: true,
			errMsg:  "claude_rules: name is required",
		},
		{
			name: "missing description",
			rules: ClaudeRules{
				Name:    "test-rules",
				Content: "Rules content",
			},
			wantErr: true,
			errMsg:  `claude_rules "test-rules": description is required`,
		},
		{
			name: "missing content and files",
			rules: ClaudeRules{
				Name:        "test-rules",
				Description: "Test rules",
			},
			wantErr: true,
			errMsg:  `claude_rules "test-rules": must specify either content or file blocks`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rules.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.EqualError(t, err, tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClaudeSettings_ResourceType(t *testing.T) {
	settings := &ClaudeSettings{}
	assert.Equal(t, "claude_settings", settings.ResourceType())
}

func TestClaudeSettings_ResourceName(t *testing.T) {
	settings := &ClaudeSettings{Name: "my-settings"}
	assert.Equal(t, "my-settings", settings.ResourceName())
}

func TestClaudeSettings_Validate(t *testing.T) {
	// Settings only require a name
	tests := []struct {
		name     string
		settings ClaudeSettings
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid settings with only name",
			settings: ClaudeSettings{
				Name: "test-settings",
			},
			wantErr: false,
		},
		{
			name: "valid settings with all fields",
			settings: ClaudeSettings{
				Name:                       "test-settings",
				Allow:                      []string{"Bash(*)"},
				Ask:                        []string{"Write(*)"},
				Deny:                       []string{"Delete(*)"},
				Env:                        map[string]string{"DEBUG": "true"},
				EnableAllProjectMCPServers: true,
				EnabledMCPServers:          []string{"server1"},
				DisabledMCPServers:         []string{"server2"},
				RespectGitignore:           true,
				IncludeCoAuthoredBy:        true,
				Model:                      "opus",
				OutputStyle:                "concise",
				AlwaysThinkingEnabled:      true,
				PlansDirectory:             ".plans",
			},
			wantErr: false,
		},
		{
			name:     "missing name",
			settings: ClaudeSettings{},
			wantErr:  true,
			errMsg:   "claude_settings: name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.settings.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.EqualError(t, err, tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClaudeSettings_GetContent(t *testing.T) {
	settings := &ClaudeSettings{Name: "test"}
	assert.Equal(t, "", settings.GetContent())
}

func TestClaudeSettings_GetFiles(t *testing.T) {
	settings := &ClaudeSettings{Name: "test"}
	assert.Nil(t, settings.GetFiles())
}

func TestClaudeSettings_GetTemplateFiles(t *testing.T) {
	settings := &ClaudeSettings{Name: "test"}
	assert.Nil(t, settings.GetTemplateFiles())
}

func TestClaudeMCPServer_ResourceType(t *testing.T) {
	server := &ClaudeMCPServer{}
	assert.Equal(t, "claude_mcp_server", server.ResourceType())
}

func TestClaudeMCPServer_ResourceName(t *testing.T) {
	server := &ClaudeMCPServer{Name: "my-server"}
	assert.Equal(t, "my-server", server.ResourceName())
}

func TestClaudeMCPServer_Validate(t *testing.T) {
	tests := []struct {
		name    string
		server  ClaudeMCPServer
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid command server",
			server: ClaudeMCPServer{
				Name:    "test-server",
				Type:    "command",
				Command: "npx",
				Args:    []string{"-y", "@modelcontextprotocol/server"},
			},
			wantErr: false,
		},
		{
			name: "valid command server with source",
			server: ClaudeMCPServer{
				Name:   "test-server",
				Type:   "command",
				Source: "npm:@mcp/server",
			},
			wantErr: false,
		},
		{
			name: "valid http server",
			server: ClaudeMCPServer{
				Name: "test-server",
				Type: "http",
				URL:  "https://example.com/mcp",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			server: ClaudeMCPServer{
				Type:    "command",
				Command: "npx",
			},
			wantErr: true,
			errMsg:  "claude_mcp_server: name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.server.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.EqualError(t, err, tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClaudeMCPServer_Validate_InvalidType(t *testing.T) {
	server := &ClaudeMCPServer{
		Name: "test-server",
		Type: "invalid",
	}

	err := server.Validate()
	require.Error(t, err)
	assert.EqualError(t, err, `claude_mcp_server "test-server": type must be 'command' or 'http', got "invalid"`)
}

func TestClaudeMCPServer_Validate_MissingCommand(t *testing.T) {
	server := &ClaudeMCPServer{
		Name: "test-server",
		Type: "command",
		// Missing both Command and Source
	}

	err := server.Validate()
	require.Error(t, err)
	assert.EqualError(t, err, `claude_mcp_server "test-server": command or source is required for type 'command'`)
}

func TestClaudeMCPServer_Validate_MissingURL(t *testing.T) {
	server := &ClaudeMCPServer{
		Name: "test-server",
		Type: "http",
		// Missing URL
	}

	err := server.Validate()
	require.Error(t, err)
	assert.EqualError(t, err, `claude_mcp_server "test-server": url is required for type 'http'`)
}

func TestClaudeMCPServer_Validate_MissingType(t *testing.T) {
	server := &ClaudeMCPServer{
		Name:    "test-server",
		Command: "npx",
	}

	err := server.Validate()
	require.Error(t, err)
	assert.EqualError(t, err, `claude_mcp_server "test-server": type is required`)
}

func TestClaudeMCPServer_GetContent(t *testing.T) {
	server := &ClaudeMCPServer{Name: "test"}
	assert.Equal(t, "", server.GetContent())
}

func TestClaudeMCPServer_GetFiles(t *testing.T) {
	server := &ClaudeMCPServer{Name: "test"}
	assert.Nil(t, server.GetFiles())
}

func TestClaudeMCPServer_GetTemplateFiles(t *testing.T) {
	server := &ClaudeMCPServer{Name: "test"}
	assert.Nil(t, server.GetTemplateFiles())
}

func TestNewResource(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		wantType Resource
	}{
		{
			name:     "claude_skill",
			typeName: "claude_skill",
			wantType: &ClaudeSkill{},
		},
		{
			name:     "claude_command",
			typeName: "claude_command",
			wantType: &ClaudeCommand{},
		},
		{
			name:     "claude_subagent",
			typeName: "claude_subagent",
			wantType: &ClaudeSubagent{},
		},
		{
			name:     "claude_rule",
			typeName: "claude_rule",
			wantType: &ClaudeRule{},
		},
		{
			name:     "claude_rules",
			typeName: "claude_rules",
			wantType: &ClaudeRules{},
		},
		{
			name:     "claude_settings",
			typeName: "claude_settings",
			wantType: &ClaudeSettings{},
		},
		{
			name:     "claude_mcp_server",
			typeName: "claude_mcp_server",
			wantType: &ClaudeMCPServer{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := NewResource(tt.typeName)
			require.NoError(t, err)
			assert.IsType(t, tt.wantType, res)
			assert.Equal(t, tt.typeName, res.ResourceType())
		})
	}
}

func TestNewResource_Unknown(t *testing.T) {
	res, err := NewResource("unknown_type")
	assert.Nil(t, res)
	require.Error(t, err)
	assert.EqualError(t, err, `unknown resource type: "unknown_type"`)
}

func TestRegisteredTypes(t *testing.T) {
	types := RegisteredTypes()

	// Should return all registered types in sorted order
	expected := []string{
		"claude_command",
		"claude_mcp_server",
		"claude_rule",
		"claude_rules",
		"claude_settings",
		"claude_skill",
		"claude_subagent",
		"copilot_agent",
		"copilot_instruction",
		"copilot_instructions",
		"copilot_mcp_server",
		"copilot_prompt",
		"copilot_skill",
		"cursor_command",
		"cursor_mcp_server",
		"cursor_rule",
		"cursor_rules",
		"directory",
		"file",
	}
	assert.Equal(t, expected, types)
}

func TestIsRegisteredType(t *testing.T) {
	tests := []struct {
		typeName string
		want     bool
	}{
		{"claude_skill", true},
		{"claude_command", true},
		{"claude_subagent", true},
		{"claude_rule", true},
		{"claude_rules", true},
		{"claude_settings", true},
		{"claude_mcp_server", true},
		{"unknown_type", false},
		{"", false},
		{"CLAUDE_SKILL", false}, // case-sensitive
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			got := IsRegisteredType(tt.typeName)
			assert.Equal(t, tt.want, got)
		})
	}
}
