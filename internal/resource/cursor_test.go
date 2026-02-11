package resource

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// CursorRule Tests
// =============================================================================

func TestCursorRule_ResourceType(t *testing.T) {
	rule := &CursorRule{}
	assert.Equal(t, "cursor_rule", rule.ResourceType())
}

func TestCursorRule_ResourceName(t *testing.T) {
	tests := []struct {
		name string
		rule CursorRule
		want string
	}{
		{
			name: "empty name",
			rule: CursorRule{},
			want: "",
		},
		{
			name: "with name",
			rule: CursorRule{Name: "my-rule"},
			want: "my-rule",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.rule.ResourceName())
		})
	}
}

func TestCursorRule_Platform(t *testing.T) {
	rule := &CursorRule{}
	assert.Equal(t, "cursor", rule.Platform())
}

func TestCursorRule_GetContent(t *testing.T) {
	tests := []struct {
		name string
		rule CursorRule
		want string
	}{
		{
			name: "empty content",
			rule: CursorRule{},
			want: "",
		},
		{
			name: "with content",
			rule: CursorRule{Content: "Rule content here"},
			want: "Rule content here",
		},
		{
			name: "multiline content",
			rule: CursorRule{Content: "Line 1\nLine 2\nLine 3"},
			want: "Line 1\nLine 2\nLine 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.rule.GetContent())
		})
	}
}

func TestCursorRule_GetFiles(t *testing.T) {
	rule := &CursorRule{}
	assert.Nil(t, rule.GetFiles())
}

func TestCursorRule_GetTemplateFiles(t *testing.T) {
	rule := &CursorRule{}
	assert.Nil(t, rule.GetTemplateFiles())
}

func TestCursorRule_Validate(t *testing.T) {
	tests := []struct {
		name    string
		rule    CursorRule
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid rule",
			rule: CursorRule{
				Name:        "test-rule",
				Description: "A test rule",
				Content:     "Rule content",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			rule: CursorRule{
				Description: "A test rule",
				Content:     "Rule content",
			},
			wantErr: true,
			errMsg:  "cursor_rule: name is required",
		},
		{
			name: "missing description",
			rule: CursorRule{
				Name:    "test-rule",
				Content: "Rule content",
			},
			wantErr: true,
			errMsg:  `cursor_rule "test-rule": description is required`,
		},
		{
			name: "missing content",
			rule: CursorRule{
				Name:        "test-rule",
				Description: "A test rule",
			},
			wantErr: true,
			errMsg:  `cursor_rule "test-rule": content is required`,
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

// =============================================================================
// CursorMCPServer Tests
// =============================================================================

func TestCursorMCPServer_ResourceType(t *testing.T) {
	server := &CursorMCPServer{}
	assert.Equal(t, "cursor_mcp_server", server.ResourceType())
}

func TestCursorMCPServer_ResourceName(t *testing.T) {
	server := &CursorMCPServer{Name: "my-server"}
	assert.Equal(t, "my-server", server.ResourceName())
}

func TestCursorMCPServer_Platform(t *testing.T) {
	server := &CursorMCPServer{}
	assert.Equal(t, "cursor", server.Platform())
}

func TestCursorMCPServer_GetContent(t *testing.T) {
	server := &CursorMCPServer{Name: "test"}
	assert.Equal(t, "", server.GetContent())
}

func TestCursorMCPServer_GetFiles(t *testing.T) {
	server := &CursorMCPServer{Name: "test"}
	assert.Nil(t, server.GetFiles())
}

func TestCursorMCPServer_GetTemplateFiles(t *testing.T) {
	server := &CursorMCPServer{Name: "test"}
	assert.Nil(t, server.GetTemplateFiles())
}

func TestCursorMCPServer_Validate(t *testing.T) {
	tests := []struct {
		name    string
		server  CursorMCPServer
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid stdio server",
			server: CursorMCPServer{
				Name:    "test-server",
				Type:    "stdio",
				Command: "npx",
				Args:    []string{"-y", "@mcp/server"},
			},
			wantErr: false,
		},
		{
			name: "valid http server",
			server: CursorMCPServer{
				Name: "test-server",
				Type: "http",
				URL:  "https://example.com/mcp",
			},
			wantErr: false,
		},
		{
			name: "valid sse server",
			server: CursorMCPServer{
				Name: "test-server",
				Type: "sse",
				URL:  "https://example.com/sse",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			server: CursorMCPServer{
				Type:    "stdio",
				Command: "npx",
			},
			wantErr: true,
			errMsg:  "cursor_mcp_server: name is required",
		},
		{
			name: "missing type",
			server: CursorMCPServer{
				Name:    "test-server",
				Command: "npx",
			},
			wantErr: true,
			errMsg:  `cursor_mcp_server "test-server": type is required`,
		},
		{
			name: "invalid type",
			server: CursorMCPServer{
				Name: "test-server",
				Type: "invalid",
			},
			wantErr: true,
			errMsg:  `cursor_mcp_server "test-server": type must be 'stdio', 'http', or 'sse', got "invalid"`,
		},
		{
			name: "stdio missing command",
			server: CursorMCPServer{
				Name: "test-server",
				Type: "stdio",
			},
			wantErr: true,
			errMsg:  `cursor_mcp_server "test-server": command is required for type 'stdio'`,
		},
		{
			name: "http missing url",
			server: CursorMCPServer{
				Name: "test-server",
				Type: "http",
			},
			wantErr: true,
			errMsg:  `cursor_mcp_server "test-server": url is required for type "http"`,
		},
		{
			name: "sse missing url",
			server: CursorMCPServer{
				Name: "test-server",
				Type: "sse",
			},
			wantErr: true,
			errMsg:  `cursor_mcp_server "test-server": url is required for type "sse"`,
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

// =============================================================================
// CursorRules Tests
// =============================================================================

func TestCursorRules_ResourceType(t *testing.T) {
	rules := &CursorRules{}
	assert.Equal(t, "cursor_rules", rules.ResourceType())
}

func TestCursorRules_ResourceName(t *testing.T) {
	rules := &CursorRules{Name: "my-rules"}
	assert.Equal(t, "my-rules", rules.ResourceName())
}

func TestCursorRules_Platform(t *testing.T) {
	rules := &CursorRules{}
	assert.Equal(t, "cursor", rules.Platform())
}

func TestCursorRules_GetContent(t *testing.T) {
	tests := []struct {
		name  string
		rules CursorRules
		want  string
	}{
		{
			name:  "empty content",
			rules: CursorRules{},
			want:  "",
		},
		{
			name:  "with content",
			rules: CursorRules{Content: "Rules content"},
			want:  "Rules content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.rules.GetContent())
		})
	}
}

func TestCursorRules_GetFiles(t *testing.T) {
	tests := []struct {
		name  string
		rules CursorRules
		want  []FileBlock
	}{
		{
			name:  "nil files",
			rules: CursorRules{},
			want:  nil,
		},
		{
			name: "with files",
			rules: CursorRules{
				Files: []FileBlock{{Src: "file.txt"}},
			},
			want: []FileBlock{{Src: "file.txt"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.rules.GetFiles())
		})
	}
}

func TestCursorRules_GetTemplateFiles(t *testing.T) {
	tests := []struct {
		name  string
		rules CursorRules
		want  []TemplateFileBlock
	}{
		{
			name:  "nil template files",
			rules: CursorRules{},
			want:  nil,
		},
		{
			name: "with template files",
			rules: CursorRules{
				TemplateFiles: []TemplateFileBlock{{Src: "config.tmpl"}},
			},
			want: []TemplateFileBlock{{Src: "config.tmpl"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.rules.GetTemplateFiles())
		})
	}
}

func TestCursorRules_Validate(t *testing.T) {
	tests := []struct {
		name    string
		rules   CursorRules
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid rules",
			rules: CursorRules{
				Name:        "test-rules",
				Description: "Test rules",
				Content:     "Rules content",
			},
			wantErr: false,
		},
		{
			name: "valid rules with globs",
			rules: CursorRules{
				Name:        "test-rules",
				Description: "Test rules",
				Content:     "Rules content",
				Globs:       []string{"**/*.ts", "**/*.tsx"},
			},
			wantErr: false,
		},
		{
			name: "valid rules with always_apply",
			rules: CursorRules{
				Name:        "test-rules",
				Description: "Test rules",
				Content:     "Rules content",
				AlwaysApply: boolPtr(true),
			},
			wantErr: false,
		},
		{
			name: "missing name",
			rules: CursorRules{
				Description: "Test rules",
				Content:     "Rules content",
			},
			wantErr: true,
			errMsg:  "cursor_rules: name is required",
		},
		{
			name: "missing description",
			rules: CursorRules{
				Name:    "test-rules",
				Content: "Rules content",
			},
			wantErr: true,
			errMsg:  `cursor_rules "test-rules": description is required`,
		},
		{
			name: "missing content",
			rules: CursorRules{
				Name:        "test-rules",
				Description: "Test rules",
			},
			wantErr: true,
			errMsg:  `cursor_rules "test-rules": content is required`,
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

// =============================================================================
// CursorCommand Tests
// =============================================================================

func TestCursorCommand_ResourceType(t *testing.T) {
	cmd := &CursorCommand{}
	assert.Equal(t, "cursor_command", cmd.ResourceType())
}

func TestCursorCommand_ResourceName(t *testing.T) {
	cmd := &CursorCommand{Name: "my-command"}
	assert.Equal(t, "my-command", cmd.ResourceName())
}

func TestCursorCommand_Platform(t *testing.T) {
	cmd := &CursorCommand{}
	assert.Equal(t, "cursor", cmd.Platform())
}

func TestCursorCommand_GetContent(t *testing.T) {
	tests := []struct {
		name string
		cmd  CursorCommand
		want string
	}{
		{
			name: "empty content",
			cmd:  CursorCommand{},
			want: "",
		},
		{
			name: "with content",
			cmd:  CursorCommand{Content: "Command content"},
			want: "Command content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.cmd.GetContent())
		})
	}
}

func TestCursorCommand_GetFiles(t *testing.T) {
	tests := []struct {
		name string
		cmd  CursorCommand
		want []FileBlock
	}{
		{
			name: "nil files",
			cmd:  CursorCommand{},
			want: nil,
		},
		{
			name: "with files",
			cmd: CursorCommand{
				Files: []FileBlock{{Src: "script.sh"}},
			},
			want: []FileBlock{{Src: "script.sh"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.cmd.GetFiles())
		})
	}
}

func TestCursorCommand_GetTemplateFiles(t *testing.T) {
	tests := []struct {
		name string
		cmd  CursorCommand
		want []TemplateFileBlock
	}{
		{
			name: "nil template files",
			cmd:  CursorCommand{},
			want: nil,
		},
		{
			name: "with template files",
			cmd: CursorCommand{
				TemplateFiles: []TemplateFileBlock{{Src: "cmd.tmpl"}},
			},
			want: []TemplateFileBlock{{Src: "cmd.tmpl"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.cmd.GetTemplateFiles())
		})
	}
}

func TestCursorCommand_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cmd     CursorCommand
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid command",
			cmd: CursorCommand{
				Name:        "test-command",
				Description: "A test command",
				Content:     "Command content",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			cmd: CursorCommand{
				Description: "A test command",
				Content:     "Command content",
			},
			wantErr: true,
			errMsg:  "cursor_command: name is required",
		},
		{
			name: "missing description",
			cmd: CursorCommand{
				Name:    "test-command",
				Content: "Command content",
			},
			wantErr: true,
			errMsg:  `cursor_command "test-command": description is required`,
		},
		{
			name: "missing content",
			cmd: CursorCommand{
				Name:        "test-command",
				Description: "A test command",
			},
			wantErr: true,
			errMsg:  `cursor_command "test-command": content is required`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cmd.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.EqualError(t, err, tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// Registry Tests for Cursor Resources
// =============================================================================

func TestNewResource_CursorTypes(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		wantType Resource
	}{
		{
			name:     "cursor_rule",
			typeName: "cursor_rule",
			wantType: &CursorRule{},
		},
		{
			name:     "cursor_mcp_server",
			typeName: "cursor_mcp_server",
			wantType: &CursorMCPServer{},
		},
		{
			name:     "cursor_rules",
			typeName: "cursor_rules",
			wantType: &CursorRules{},
		},
		{
			name:     "cursor_command",
			typeName: "cursor_command",
			wantType: &CursorCommand{},
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

func TestRegisteredTypes_IncludesCursor(t *testing.T) {
	types := RegisteredTypes()

	// Full expected list is validated in TestRegisteredTypes; verify cursor subset here
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

func TestIsRegisteredType_CursorTypes(t *testing.T) {
	tests := []struct {
		typeName string
		want     bool
	}{
		{"cursor_rule", true},
		{"cursor_mcp_server", true},
		{"cursor_rules", true},
		{"cursor_command", true},
		{"cursor_unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			got := IsRegisteredType(tt.typeName)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Helper function
func boolPtr(b bool) *bool {
	return &b
}
