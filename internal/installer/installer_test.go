package installer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dex-tools/dex/internal/adapter"
	"github.com/dex-tools/dex/internal/manifest"
	"github.com/dex-tools/dex/internal/resource"
)

// Helper function to create a test manifest
func newTestManifest(t *testing.T, projectRoot string) *manifest.Manifest {
	t.Helper()
	m, err := manifest.Load(projectRoot)
	require.NoError(t, err)
	return m
}

// Test Executor

func TestExecutor_CreateDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	m := newTestManifest(t, tmpDir)
	executor := NewExecutor(tmpDir, m, false)

	plan := &adapter.Plan{
		PluginName:  "test-plugin",
		Directories: []string{"dir1", "dir2/nested"},
	}

	err := executor.Execute(plan, nil)
	require.NoError(t, err)

	// Verify directories created
	_, err = os.Stat(filepath.Join(tmpDir, "dir1"))
	assert.NoError(t, err, "dir1 should exist")

	_, err = os.Stat(filepath.Join(tmpDir, "dir2/nested"))
	assert.NoError(t, err, "dir2/nested should exist")
}

func TestExecutor_WriteFiles(t *testing.T) {
	tmpDir := t.TempDir()
	m := newTestManifest(t, tmpDir)
	executor := NewExecutor(tmpDir, m, false)

	plan := &adapter.Plan{
		PluginName: "test-plugin",
		Files: []adapter.FileWrite{
			{Path: "test.txt", Content: "hello world", Chmod: ""},
			{Path: "subdir/file.txt", Content: "nested content", Chmod: ""},
		},
	}

	err := executor.Execute(plan, nil)
	require.NoError(t, err)

	// Verify file content
	content, err := os.ReadFile(filepath.Join(tmpDir, "test.txt"))
	require.NoError(t, err)
	assert.Equal(t, "hello world", string(content))

	content, err = os.ReadFile(filepath.Join(tmpDir, "subdir/file.txt"))
	require.NoError(t, err)
	assert.Equal(t, "nested content", string(content))
}

func TestExecutor_WriteFiles_WithPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	m := newTestManifest(t, tmpDir)
	executor := NewExecutor(tmpDir, m, false)

	plan := &adapter.Plan{
		PluginName: "test-plugin",
		Files: []adapter.FileWrite{
			{Path: "script.sh", Content: "#!/bin/bash\necho hello", Chmod: "755"},
		},
	}

	err := executor.Execute(plan, nil)
	require.NoError(t, err)

	// Verify file permissions
	info, err := os.Stat(filepath.Join(tmpDir, "script.sh"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0755), info.Mode().Perm())
}

func TestExecutor_WriteFiles_WithTemplateVars(t *testing.T) {
	tmpDir := t.TempDir()
	m := newTestManifest(t, tmpDir)
	executor := NewExecutor(tmpDir, m, false)

	plan := &adapter.Plan{
		PluginName: "test-plugin",
		Files: []adapter.FileWrite{
			{Path: "config.txt", Content: "name: ${NAME}\nversion: {{VERSION}}", Chmod: ""},
		},
	}

	vars := map[string]string{
		"NAME":    "test-app",
		"VERSION": "1.0.0",
	}

	err := executor.Execute(plan, vars)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(tmpDir, "config.txt"))
	require.NoError(t, err)
	assert.Equal(t, "name: test-app\nversion: 1.0.0", string(content))
}

func TestExecutor_WriteFile_Conflict(t *testing.T) {
	tmpDir := t.TempDir()
	m := newTestManifest(t, tmpDir)
	executor := NewExecutor(tmpDir, m, false)

	// Create existing non-managed file
	existingFile := filepath.Join(tmpDir, "existing.txt")
	err := os.WriteFile(existingFile, []byte("original content"), 0644)
	require.NoError(t, err)

	plan := &adapter.Plan{
		PluginName: "test-plugin",
		Files: []adapter.FileWrite{
			{Path: "existing.txt", Content: "new content", Chmod: ""},
		},
	}

	// Attempt to write over it (should error)
	err = executor.Execute(plan, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists and is not managed by dex")

	// Verify original content unchanged
	content, err := os.ReadFile(existingFile)
	require.NoError(t, err)
	assert.Equal(t, "original content", string(content))
}

func TestExecutor_WriteFile_Force(t *testing.T) {
	tmpDir := t.TempDir()
	m := newTestManifest(t, tmpDir)
	executor := NewExecutor(tmpDir, m, true) // force=true

	// Create existing non-managed file
	existingFile := filepath.Join(tmpDir, "existing.txt")
	err := os.WriteFile(existingFile, []byte("original content"), 0644)
	require.NoError(t, err)

	plan := &adapter.Plan{
		PluginName: "test-plugin",
		Files: []adapter.FileWrite{
			{Path: "existing.txt", Content: "new content", Chmod: ""},
		},
	}

	// Attempt to write with force=true (should succeed)
	err = executor.Execute(plan, nil)
	require.NoError(t, err)

	// Verify content overwritten
	content, err := os.ReadFile(existingFile)
	require.NoError(t, err)
	assert.Equal(t, "new content", string(content))
}

func TestExecutor_WriteFile_Tracked(t *testing.T) {
	tmpDir := t.TempDir()
	m := newTestManifest(t, tmpDir)

	// Pre-track the file as managed
	m.Track("test-plugin", []string{"existing.txt"}, nil)

	executor := NewExecutor(tmpDir, m, false)

	// Create existing managed file
	existingFile := filepath.Join(tmpDir, "existing.txt")
	err := os.WriteFile(existingFile, []byte("original content"), 0644)
	require.NoError(t, err)

	plan := &adapter.Plan{
		PluginName: "test-plugin",
		Files: []adapter.FileWrite{
			{Path: "existing.txt", Content: "updated content", Chmod: ""},
		},
	}

	// Should succeed because file is tracked
	err = executor.Execute(plan, nil)
	require.NoError(t, err)

	content, err := os.ReadFile(existingFile)
	require.NoError(t, err)
	assert.Equal(t, "updated content", string(content))
}

func TestExecutor_ApplyMCPConfig_New(t *testing.T) {
	tmpDir := t.TempDir()
	m := newTestManifest(t, tmpDir)
	executor := NewExecutor(tmpDir, m, false)

	plan := &adapter.Plan{
		PluginName: "test-plugin",
		MCPEntries: map[string]any{
			"mcpServers": map[string]any{
				"test-server": map[string]any{
					"command": "test-cmd",
					"args":    []any{"--flag"},
				},
			},
		},
	}

	err := executor.Execute(plan, nil)
	require.NoError(t, err)

	// Verify .mcp.json created with correct content
	mcpPath := filepath.Join(tmpDir, ".mcp.json")
	content, err := os.ReadFile(mcpPath)
	require.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(content, &result)
	require.NoError(t, err)

	servers, ok := result["mcpServers"].(map[string]any)
	require.True(t, ok)

	server, ok := servers["test-server"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "test-cmd", server["command"])
}

func TestExecutor_ApplyMCPConfig_Merge(t *testing.T) {
	tmpDir := t.TempDir()
	m := newTestManifest(t, tmpDir)

	// Create existing .mcp.json
	existingMCP := map[string]any{
		"mcpServers": map[string]any{
			"existing-server": map[string]any{
				"command": "existing-cmd",
			},
		},
	}
	err := WriteJSONFile(filepath.Join(tmpDir, ".mcp.json"), existingMCP)
	require.NoError(t, err)

	executor := NewExecutor(tmpDir, m, false)

	plan := &adapter.Plan{
		PluginName: "test-plugin",
		MCPEntries: map[string]any{
			"mcpServers": map[string]any{
				"new-server": map[string]any{
					"command": "new-cmd",
				},
			},
		},
	}

	err = executor.Execute(plan, nil)
	require.NoError(t, err)

	// Verify both servers exist
	mcpPath := filepath.Join(tmpDir, ".mcp.json")
	content, err := os.ReadFile(mcpPath)
	require.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(content, &result)
	require.NoError(t, err)

	servers, ok := result["mcpServers"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, servers, "existing-server")
	assert.Contains(t, servers, "new-server")
}

func TestExecutor_ApplySettingsConfig_New(t *testing.T) {
	tmpDir := t.TempDir()
	m := newTestManifest(t, tmpDir)
	executor := NewExecutor(tmpDir, m, false)

	plan := &adapter.Plan{
		PluginName: "test-plugin",
		SettingsEntries: map[string]any{
			"allow": []any{"Bash(npm:*)"},
		},
	}

	err := executor.Execute(plan, nil)
	require.NoError(t, err)

	// Verify settings.json created
	settingsPath := filepath.Join(tmpDir, ".claude", "settings.json")
	content, err := os.ReadFile(settingsPath)
	require.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(content, &result)
	require.NoError(t, err)

	allow, ok := result["allow"].([]any)
	require.True(t, ok)
	assert.Contains(t, allow, "Bash(npm:*)")
}

func TestExecutor_ApplySettingsConfig_Merge(t *testing.T) {
	tmpDir := t.TempDir()
	m := newTestManifest(t, tmpDir)

	// Create existing settings
	claudeDir := filepath.Join(tmpDir, ".claude")
	err := os.MkdirAll(claudeDir, 0755)
	require.NoError(t, err)

	existingSettings := map[string]any{
		"allow": []any{"Bash(git:*)"},
	}
	err = WriteJSONFile(filepath.Join(claudeDir, "settings.json"), existingSettings)
	require.NoError(t, err)

	executor := NewExecutor(tmpDir, m, false)

	plan := &adapter.Plan{
		PluginName: "test-plugin",
		SettingsEntries: map[string]any{
			"allow": []any{"Bash(npm:*)"},
		},
	}

	err = executor.Execute(plan, nil)
	require.NoError(t, err)

	// Verify merged correctly
	content, err := os.ReadFile(filepath.Join(claudeDir, "settings.json"))
	require.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(content, &result)
	require.NoError(t, err)

	allow, ok := result["allow"].([]any)
	require.True(t, ok)
	assert.Contains(t, allow, "Bash(git:*)")
	assert.Contains(t, allow, "Bash(npm:*)")
}

func TestExecutor_ApplyAgentFileContent_New(t *testing.T) {
	tmpDir := t.TempDir()
	m := newTestManifest(t, tmpDir)
	executor := NewExecutor(tmpDir, m, false)

	plan := &adapter.Plan{
		PluginName:       "test-plugin",
		AgentFileContent: "Test content here",
	}

	err := executor.Execute(plan, nil)
	require.NoError(t, err)

	// Verify CLAUDE.md created with markers
	content, err := os.ReadFile(filepath.Join(tmpDir, "CLAUDE.md"))
	require.NoError(t, err)

	expected := "<!-- dex:test-plugin -->\nTest content here\n<!-- /dex:test-plugin -->"
	assert.Equal(t, expected, string(content))
}

func TestExecutor_ApplyAgentFileContent_Merge(t *testing.T) {
	tmpDir := t.TempDir()
	m := newTestManifest(t, tmpDir)

	// Create existing CLAUDE.md with other content
	existingContent := "# My Rules\n\nSome rules here."
	err := os.WriteFile(filepath.Join(tmpDir, "CLAUDE.md"), []byte(existingContent), 0644)
	require.NoError(t, err)

	executor := NewExecutor(tmpDir, m, false)

	plan := &adapter.Plan{
		PluginName:       "test-plugin",
		AgentFileContent: "Test content",
	}

	err = executor.Execute(plan, nil)
	require.NoError(t, err)

	// Verify content added with markers, existing content preserved
	content, err := os.ReadFile(filepath.Join(tmpDir, "CLAUDE.md"))
	require.NoError(t, err)

	expected := "# My Rules\n\nSome rules here.\n\n<!-- dex:test-plugin -->\nTest content\n<!-- /dex:test-plugin -->"
	assert.Equal(t, expected, string(content))
}

func TestExecutor_ApplyAgentFileContent_Update(t *testing.T) {
	tmpDir := t.TempDir()
	m := newTestManifest(t, tmpDir)

	// Create existing CLAUDE.md with this plugin's markers
	existingContent := "<!-- dex:test-plugin -->\nOld content\n<!-- /dex:test-plugin -->"
	err := os.WriteFile(filepath.Join(tmpDir, "CLAUDE.md"), []byte(existingContent), 0644)
	require.NoError(t, err)

	executor := NewExecutor(tmpDir, m, false)

	plan := &adapter.Plan{
		PluginName:       "test-plugin",
		AgentFileContent: "New content",
	}

	err = executor.Execute(plan, nil)
	require.NoError(t, err)

	// Verify content replaced
	content, err := os.ReadFile(filepath.Join(tmpDir, "CLAUDE.md"))
	require.NoError(t, err)

	expected := "<!-- dex:test-plugin -->\nNew content\n<!-- /dex:test-plugin -->"
	assert.Equal(t, expected, string(content))
}

func TestExecutor_EmptyPlan(t *testing.T) {
	tmpDir := t.TempDir()
	m := newTestManifest(t, tmpDir)
	executor := NewExecutor(tmpDir, m, false)

	// Empty plan should do nothing
	err := executor.Execute(nil, nil)
	require.NoError(t, err)

	// Empty plan with no operations
	emptyPlan := &adapter.Plan{PluginName: "test"}
	err = executor.Execute(emptyPlan, nil)
	require.NoError(t, err)
}

// Test Merger

func TestMergeJSON(t *testing.T) {
	tests := []struct {
		name     string
		base     map[string]any
		overlay  map[string]any
		expected map[string]any
	}{
		{
			name:     "simple merge",
			base:     map[string]any{"a": 1},
			overlay:  map[string]any{"b": 2},
			expected: map[string]any{"a": 1, "b": 2},
		},
		{
			name:     "nested merge",
			base:     map[string]any{"nested": map[string]any{"a": 1}},
			overlay:  map[string]any{"nested": map[string]any{"b": 2}},
			expected: map[string]any{"nested": map[string]any{"a": 1, "b": 2}},
		},
		{
			name:     "array merge",
			base:     map[string]any{"arr": []any{1, 2}},
			overlay:  map[string]any{"arr": []any{2, 3}},
			expected: map[string]any{"arr": []any{1, 2, 3}},
		},
		{
			name:     "overlay takes precedence for simple values",
			base:     map[string]any{"key": "old"},
			overlay:  map[string]any{"key": "new"},
			expected: map[string]any{"key": "new"},
		},
		{
			name:     "nil base",
			base:     nil,
			overlay:  map[string]any{"a": 1},
			expected: map[string]any{"a": 1},
		},
		{
			name:     "nil overlay",
			base:     map[string]any{"a": 1},
			overlay:  nil,
			expected: map[string]any{"a": 1},
		},
		{
			name:     "both nil",
			base:     nil,
			overlay:  nil,
			expected: map[string]any{},
		},
		{
			name: "deeply nested merge",
			base: map[string]any{
				"level1": map[string]any{
					"level2": map[string]any{
						"a": 1,
					},
				},
			},
			overlay: map[string]any{
				"level1": map[string]any{
					"level2": map[string]any{
						"b": 2,
					},
				},
			},
			expected: map[string]any{
				"level1": map[string]any{
					"level2": map[string]any{
						"a": 1,
						"b": 2,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeJSON(tt.base, tt.overlay)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMergeJSONArrays(t *testing.T) {
	tests := []struct {
		name     string
		base     []any
		overlay  []any
		expected []any
	}{
		{
			name:     "basic merge",
			base:     []any{1, 2},
			overlay:  []any{3, 4},
			expected: []any{1, 2, 3, 4},
		},
		{
			name:     "deduplicate",
			base:     []any{1, 2},
			overlay:  []any{2, 3},
			expected: []any{1, 2, 3},
		},
		{
			name:     "string values",
			base:     []any{"a", "b"},
			overlay:  []any{"b", "c"},
			expected: []any{"a", "b", "c"},
		},
		{
			name:     "nil base",
			base:     nil,
			overlay:  []any{1, 2},
			expected: []any{1, 2},
		},
		{
			name:     "nil overlay",
			base:     []any{1, 2},
			overlay:  nil,
			expected: []any{1, 2},
		},
		{
			name:     "empty arrays",
			base:     []any{},
			overlay:  []any{},
			expected: []any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeJSONArrays(tt.base, tt.overlay)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMergeMCPServers(t *testing.T) {
	tests := []struct {
		name     string
		base     map[string]any
		overlay  map[string]any
		expected map[string]any
	}{
		{
			name: "no existing servers",
			base: map[string]any{},
			overlay: map[string]any{
				"mcpServers": map[string]any{
					"server1": map[string]any{"command": "cmd1"},
				},
			},
			expected: map[string]any{
				"mcpServers": map[string]any{
					"server1": map[string]any{"command": "cmd1"},
				},
			},
		},
		{
			name: "add new server",
			base: map[string]any{
				"mcpServers": map[string]any{
					"server1": map[string]any{"command": "cmd1"},
				},
			},
			overlay: map[string]any{
				"mcpServers": map[string]any{
					"server2": map[string]any{"command": "cmd2"},
				},
			},
			expected: map[string]any{
				"mcpServers": map[string]any{
					"server1": map[string]any{"command": "cmd1"},
					"server2": map[string]any{"command": "cmd2"},
				},
			},
		},
		{
			name: "replace existing server",
			base: map[string]any{
				"mcpServers": map[string]any{
					"server1": map[string]any{"command": "old-cmd"},
				},
			},
			overlay: map[string]any{
				"mcpServers": map[string]any{
					"server1": map[string]any{"command": "new-cmd"},
				},
			},
			expected: map[string]any{
				"mcpServers": map[string]any{
					"server1": map[string]any{"command": "new-cmd"},
				},
			},
		},
		{
			name: "multiple servers",
			base: map[string]any{
				"mcpServers": map[string]any{
					"server1": map[string]any{"command": "cmd1"},
					"server2": map[string]any{"command": "cmd2"},
				},
			},
			overlay: map[string]any{
				"mcpServers": map[string]any{
					"server2": map[string]any{"command": "updated-cmd2"},
					"server3": map[string]any{"command": "cmd3"},
				},
			},
			expected: map[string]any{
				"mcpServers": map[string]any{
					"server1": map[string]any{"command": "cmd1"},
					"server2": map[string]any{"command": "updated-cmd2"},
					"server3": map[string]any{"command": "cmd3"},
				},
			},
		},
		{
			name:     "nil base",
			base:     nil,
			overlay:  map[string]any{"mcpServers": map[string]any{"s": map[string]any{}}},
			expected: map[string]any{"mcpServers": map[string]any{"s": map[string]any{}}},
		},
		{
			name:     "nil overlay",
			base:     map[string]any{"mcpServers": map[string]any{"s": map[string]any{}}},
			overlay:  nil,
			expected: map[string]any{"mcpServers": map[string]any{"s": map[string]any{}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeMCPServers(tt.base, tt.overlay)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRemoveMCPServers(t *testing.T) {
	tests := []struct {
		name     string
		config   map[string]any
		names    []string
		expected map[string]any
	}{
		{
			name: "remove single server",
			config: map[string]any{
				"mcpServers": map[string]any{
					"server1": map[string]any{"command": "cmd1"},
					"server2": map[string]any{"command": "cmd2"},
				},
			},
			names: []string{"server1"},
			expected: map[string]any{
				"mcpServers": map[string]any{
					"server2": map[string]any{"command": "cmd2"},
				},
			},
		},
		{
			name: "remove multiple servers",
			config: map[string]any{
				"mcpServers": map[string]any{
					"server1": map[string]any{"command": "cmd1"},
					"server2": map[string]any{"command": "cmd2"},
					"server3": map[string]any{"command": "cmd3"},
				},
			},
			names: []string{"server1", "server3"},
			expected: map[string]any{
				"mcpServers": map[string]any{
					"server2": map[string]any{"command": "cmd2"},
				},
			},
		},
		{
			name: "remove non-existent server",
			config: map[string]any{
				"mcpServers": map[string]any{
					"server1": map[string]any{"command": "cmd1"},
				},
			},
			names: []string{"non-existent"},
			expected: map[string]any{
				"mcpServers": map[string]any{
					"server1": map[string]any{"command": "cmd1"},
				},
			},
		},
		{
			name:     "nil config",
			config:   nil,
			names:    []string{"server1"},
			expected: nil,
		},
		{
			name: "no mcpServers key",
			config: map[string]any{
				"other": "value",
			},
			names: []string{"server1"},
			expected: map[string]any{
				"other": "value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RemoveMCPServers(tt.config, tt.names)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMergeAgentContent(t *testing.T) {
	tests := []struct {
		name       string
		existing   string
		pluginName string
		content    string
		expected   string
	}{
		{
			name:       "new plugin to empty file",
			existing:   "",
			pluginName: "test-plugin",
			content:    "Test content",
			expected:   "<!-- dex:test-plugin -->\nTest content\n<!-- /dex:test-plugin -->",
		},
		{
			name:       "new plugin to file with content",
			existing:   "# My Rules\n\nSome rules here.",
			pluginName: "test-plugin",
			content:    "Test content",
			expected:   "# My Rules\n\nSome rules here.\n\n<!-- dex:test-plugin -->\nTest content\n<!-- /dex:test-plugin -->",
		},
		{
			name:       "new plugin to file with trailing newline",
			existing:   "# My Rules\n\nSome rules here.\n",
			pluginName: "test-plugin",
			content:    "Test content",
			expected:   "# My Rules\n\nSome rules here.\n\n<!-- dex:test-plugin -->\nTest content\n<!-- /dex:test-plugin -->",
		},
		{
			name:       "update existing plugin",
			existing:   "<!-- dex:test-plugin -->\nOld content\n<!-- /dex:test-plugin -->",
			pluginName: "test-plugin",
			content:    "New content",
			expected:   "<!-- dex:test-plugin -->\nNew content\n<!-- /dex:test-plugin -->",
		},
		{
			name:       "update plugin preserving other content",
			existing:   "# Rules\n\n<!-- dex:test-plugin -->\nOld content\n<!-- /dex:test-plugin -->\n\nMore rules.",
			pluginName: "test-plugin",
			content:    "New content",
			expected:   "# Rules\n\n<!-- dex:test-plugin -->\nNew content\n<!-- /dex:test-plugin -->\n\nMore rules.",
		},
		{
			name:       "multiple plugins",
			existing:   "<!-- dex:other-plugin -->\nOther content\n<!-- /dex:other-plugin -->",
			pluginName: "test-plugin",
			content:    "Test content",
			expected:   "<!-- dex:other-plugin -->\nOther content\n<!-- /dex:other-plugin -->\n\n<!-- dex:test-plugin -->\nTest content\n<!-- /dex:test-plugin -->",
		},
		{
			name:       "multiline content",
			existing:   "",
			pluginName: "test-plugin",
			content:    "Line 1\nLine 2\nLine 3",
			expected:   "<!-- dex:test-plugin -->\nLine 1\nLine 2\nLine 3\n<!-- /dex:test-plugin -->",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeAgentContent(tt.existing, tt.pluginName, tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRemoveAgentContent(t *testing.T) {
	tests := []struct {
		name       string
		existing   string
		pluginName string
		expected   string
	}{
		{
			name:       "remove only content",
			existing:   "<!-- dex:test-plugin -->\nTest content\n<!-- /dex:test-plugin -->",
			pluginName: "test-plugin",
			expected:   "",
		},
		{
			name:       "remove preserving other content before",
			existing:   "# Rules\n\n<!-- dex:test-plugin -->\nTest\n<!-- /dex:test-plugin -->",
			pluginName: "test-plugin",
			expected:   "# Rules\n",
		},
		{
			name:       "remove preserving other content after",
			existing:   "<!-- dex:test-plugin -->\nTest\n<!-- /dex:test-plugin -->\n\nMore rules.",
			pluginName: "test-plugin",
			expected:   "More rules.\n",
		},
		{
			name:       "remove preserving other content before and after",
			existing:   "# Rules\n\n<!-- dex:test-plugin -->\nTest\n<!-- /dex:test-plugin -->\n\nMore rules.",
			pluginName: "test-plugin",
			expected:   "# Rules\nMore rules.\n", // TrimSpace collapses consecutive newlines
		},
		{
			name:       "remove non-existent does nothing",
			existing:   "# Rules\n",
			pluginName: "test-plugin",
			expected:   "# Rules\n",
		},
		{
			name:       "remove preserves other plugin markers",
			existing:   "<!-- dex:other-plugin -->\nOther\n<!-- /dex:other-plugin -->\n\n<!-- dex:test-plugin -->\nTest\n<!-- /dex:test-plugin -->",
			pluginName: "test-plugin",
			expected:   "<!-- dex:other-plugin -->\nOther\n<!-- /dex:other-plugin -->\n",
		},
		{
			name:       "remove empty existing",
			existing:   "",
			pluginName: "test-plugin",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RemoveAgentContent(tt.existing, tt.pluginName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMergeSettingsArrays(t *testing.T) {
	tests := []struct {
		name     string
		base     []string
		overlay  []string
		expected []string
	}{
		{
			name:     "basic merge",
			base:     []string{"a", "b"},
			overlay:  []string{"c", "d"},
			expected: []string{"a", "b", "c", "d"},
		},
		{
			name:     "deduplicate",
			base:     []string{"a", "b"},
			overlay:  []string{"b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "nil base",
			base:     nil,
			overlay:  []string{"a"},
			expected: []string{"a"},
		},
		{
			name:     "nil overlay",
			base:     []string{"a"},
			overlay:  nil,
			expected: []string{"a"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeSettingsArrays(tt.base, tt.overlay)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMergeEnvMaps(t *testing.T) {
	tests := []struct {
		name     string
		base     map[string]string
		overlay  map[string]string
		expected map[string]string
	}{
		{
			name:     "basic merge",
			base:     map[string]string{"A": "1"},
			overlay:  map[string]string{"B": "2"},
			expected: map[string]string{"A": "1", "B": "2"},
		},
		{
			name:     "overlay overwrites",
			base:     map[string]string{"A": "1"},
			overlay:  map[string]string{"A": "2"},
			expected: map[string]string{"A": "2"},
		},
		{
			name:     "nil base",
			base:     nil,
			overlay:  map[string]string{"A": "1"},
			expected: map[string]string{"A": "1"},
		},
		{
			name:     "nil overlay",
			base:     map[string]string{"A": "1"},
			overlay:  nil,
			expected: map[string]string{"A": "1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeEnvMaps(tt.base, tt.overlay)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReadJSONFile_NotExist(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistent := filepath.Join(tmpDir, "non-existent.json")

	result, err := ReadJSONFile(nonExistent)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{}, result)
}

func TestReadJSONFile_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "test.json")

	content := `{"key": "value", "number": 42}`
	err := os.WriteFile(jsonFile, []byte(content), 0644)
	require.NoError(t, err)

	result, err := ReadJSONFile(jsonFile)
	require.NoError(t, err)
	assert.Equal(t, "value", result["key"])
	assert.Equal(t, float64(42), result["number"])
}

func TestReadJSONFile_Invalid(t *testing.T) {
	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "invalid.json")

	content := `{invalid json}`
	err := os.WriteFile(jsonFile, []byte(content), 0644)
	require.NoError(t, err)

	_, err = ReadJSONFile(jsonFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing JSON file")
}

func TestReadJSONFile_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "empty.json")

	err := os.WriteFile(jsonFile, []byte("null"), 0644)
	require.NoError(t, err)

	result, err := ReadJSONFile(jsonFile)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{}, result)
}

func TestWriteJSONFile(t *testing.T) {
	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "output.json")

	data := map[string]any{
		"key":    "value",
		"number": 42,
	}

	err := WriteJSONFile(jsonFile, data)
	require.NoError(t, err)

	// Read back and verify
	content, err := os.ReadFile(jsonFile)
	require.NoError(t, err)

	// Should be properly formatted with indentation and trailing newline
	expected := `{
  "key": "value",
  "number": 42
}
`
	assert.Equal(t, expected, string(content))
}

func TestWriteJSONFile_CreatesDir(t *testing.T) {
	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "nested", "dir", "output.json")

	data := map[string]any{"key": "value"}

	err := WriteJSONFile(jsonFile, data)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(jsonFile)
	require.NoError(t, err)
}

func TestProcessTemplate(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		vars     map[string]string
		expected string
	}{
		{
			name:     "dollar brace syntax",
			content:  "Hello ${NAME}!",
			vars:     map[string]string{"NAME": "World"},
			expected: "Hello World!",
		},
		{
			name:     "double brace syntax",
			content:  "Hello {{NAME}}!",
			vars:     map[string]string{"NAME": "World"},
			expected: "Hello World!",
		},
		{
			name:     "multiple variables",
			content:  "${GREETING} ${NAME}!",
			vars:     map[string]string{"GREETING": "Hi", "NAME": "User"},
			expected: "Hi User!",
		},
		{
			name:     "mixed syntax",
			content:  "${VAR1} and {{VAR2}}",
			vars:     map[string]string{"VAR1": "A", "VAR2": "B"},
			expected: "A and B",
		},
		{
			name:     "no vars",
			content:  "Static content",
			vars:     map[string]string{},
			expected: "Static content",
		},
		{
			name:     "unmatched variable",
			content:  "Hello ${UNKNOWN}!",
			vars:     map[string]string{"NAME": "World"},
			expected: "Hello ${UNKNOWN}!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processTemplate(tt.content, tt.vars)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseChmod(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected os.FileMode
		hasError bool
	}{
		{name: "755", input: "755", expected: os.FileMode(0755), hasError: false},
		{name: "644", input: "644", expected: os.FileMode(0644), hasError: false},
		{name: "600", input: "600", expected: os.FileMode(0600), hasError: false},
		{name: "777", input: "777", expected: os.FileMode(0777), hasError: false},
		{name: "invalid", input: "invalid", expected: 0, hasError: true},
		{name: "empty", input: "", expected: 0, hasError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseChmod(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// Integration-style tests

func TestExecutor_FullPlan(t *testing.T) {
	tmpDir := t.TempDir()
	m := newTestManifest(t, tmpDir)
	executor := NewExecutor(tmpDir, m, false)

	// Create a comprehensive plan
	plan := &adapter.Plan{
		PluginName:  "full-test-plugin",
		Directories: []string{".claude/commands"},
		Files: []adapter.FileWrite{
			{Path: ".claude/commands/test.md", Content: "---\nname: test\n---\nContent", Chmod: ""},
		},
		MCPEntries: map[string]any{
			"mcpServers": map[string]any{
				"test-server": map[string]any{
					"command": "npx",
					"args":    []any{"-y", "test-mcp"},
				},
			},
		},
		SettingsEntries: map[string]any{
			"allow": []any{"Bash(npm:*)"},
		},
		AgentFileContent: "# Test Plugin Rules\n\nFollow these rules.",
	}

	err := executor.Execute(plan, nil)
	require.NoError(t, err)

	// Verify directory created
	_, err = os.Stat(filepath.Join(tmpDir, ".claude/commands"))
	assert.NoError(t, err)

	// Verify file created
	content, err := os.ReadFile(filepath.Join(tmpDir, ".claude/commands/test.md"))
	require.NoError(t, err)
	assert.Equal(t, "---\nname: test\n---\nContent", string(content))

	// Verify MCP config
	mcpContent, err := os.ReadFile(filepath.Join(tmpDir, ".mcp.json"))
	require.NoError(t, err)
	var mcpResult map[string]any
	err = json.Unmarshal(mcpContent, &mcpResult)
	require.NoError(t, err)
	servers := mcpResult["mcpServers"].(map[string]any)
	assert.Contains(t, servers, "test-server")

	// Verify settings config
	settingsContent, err := os.ReadFile(filepath.Join(tmpDir, ".claude/settings.json"))
	require.NoError(t, err)
	var settingsResult map[string]any
	err = json.Unmarshal(settingsContent, &settingsResult)
	require.NoError(t, err)
	allow := settingsResult["allow"].([]any)
	assert.Contains(t, allow, "Bash(npm:*)")

	// Verify CLAUDE.md
	agentContent, err := os.ReadFile(filepath.Join(tmpDir, "CLAUDE.md"))
	require.NoError(t, err)
	assert.Contains(t, string(agentContent), "<!-- dex:full-test-plugin -->")
	assert.Contains(t, string(agentContent), "# Test Plugin Rules")
}

func TestManifestTracking(t *testing.T) {
	tmpDir := t.TempDir()
	m := newTestManifest(t, tmpDir)
	executor := NewExecutor(tmpDir, m, false)

	plan := &adapter.Plan{
		PluginName:  "tracked-plugin",
		Directories: []string{"test-dir"},
		Files: []adapter.FileWrite{
			{Path: "test-dir/file.txt", Content: "content", Chmod: ""},
		},
		MCPEntries: map[string]any{
			"mcpServers": map[string]any{
				"tracked-server": map[string]any{"command": "cmd"},
			},
		},
		AgentFileContent: "Agent content",
	}

	err := executor.Execute(plan, nil)
	require.NoError(t, err)

	// Verify manifest tracking
	plugin := m.GetPlugin("tracked-plugin")
	require.NotNil(t, plugin)
	assert.Contains(t, plugin.Files, "test-dir/file.txt")
	assert.Contains(t, plugin.Directories, "test-dir")
	assert.True(t, plugin.HasAgentContent)
}

func TestSettingsDeduplicationOnUninstall(t *testing.T) {
	// This test verifies that when two plugins share a settings value,
	// uninstalling one plugin keeps the shared value if the other still needs it.
	tmpDir := t.TempDir()

	m := newTestManifest(t, tmpDir)
	executor := NewExecutor(tmpDir, m, false)

	// Install plugin A with settings
	planA := &adapter.Plan{
		PluginName: "plugin-a",
		SettingsEntries: map[string]any{
			"allow": []any{"bash:npm run *", "write:*.ts"},
		},
	}
	err := executor.Execute(planA, nil)
	require.NoError(t, err)

	// Install plugin B with overlapping settings
	planB := &adapter.Plan{
		PluginName: "plugin-b",
		SettingsEntries: map[string]any{
			"allow": []any{"bash:npm run *", "bash:yarn *"}, // "bash:npm run *" is shared
		},
	}
	err = executor.Execute(planB, nil)
	require.NoError(t, err)

	// Save manifest so we can read it properly
	err = m.Save()
	require.NoError(t, err)

	// Verify merged settings
	settingsPath := filepath.Join(tmpDir, ".claude", "settings.json")
	content, err := os.ReadFile(settingsPath)
	require.NoError(t, err)

	var settings map[string]any
	err = json.Unmarshal(content, &settings)
	require.NoError(t, err)

	allow := settings["allow"].([]any)
	assert.Len(t, allow, 3) // bash:npm run *, write:*.ts, bash:yarn *

	// Verify manifest tracks each plugin's contributions
	pluginA := m.GetPlugin("plugin-a")
	require.NotNil(t, pluginA)
	assert.Contains(t, pluginA.SettingsValues["allow"], "bash:npm run *")
	assert.Contains(t, pluginA.SettingsValues["allow"], "write:*.ts")

	pluginB := m.GetPlugin("plugin-b")
	require.NotNil(t, pluginB)
	assert.Contains(t, pluginB.SettingsValues["allow"], "bash:npm run *")
	assert.Contains(t, pluginB.SettingsValues["allow"], "bash:yarn *")

	// Verify IsSettingsValueUsedByOthers works correctly
	assert.True(t, m.IsSettingsValueUsedByOthers("plugin-a", "allow", "bash:npm run *"),
		"bash:npm run * should be used by plugin-b")
	assert.False(t, m.IsSettingsValueUsedByOthers("plugin-a", "allow", "write:*.ts"),
		"write:*.ts should NOT be used by others")
}

// Test Resource Platform() method

func TestClaudeResources_Platform(t *testing.T) {
	tests := []struct {
		name     string
		resource resource.Resource
		expected string
	}{
		{
			name:     "ClaudeSkill",
			resource: &resource.ClaudeSkill{Name: "test", Description: "test", Content: "test"},
			expected: "claude-code",
		},
		{
			name:     "ClaudeCommand",
			resource: &resource.ClaudeCommand{Name: "test", Description: "test", Content: "test"},
			expected: "claude-code",
		},
		{
			name:     "ClaudeSubagent",
			resource: &resource.ClaudeSubagent{Name: "test", Description: "test", Content: "test"},
			expected: "claude-code",
		},
		{
			name:     "ClaudeRule",
			resource: &resource.ClaudeRule{Name: "test", Description: "test", Content: "test"},
			expected: "claude-code",
		},
		{
			name:     "ClaudeRules",
			resource: &resource.ClaudeRules{Name: "test", Description: "test", Content: "test"},
			expected: "claude-code",
		},
		{
			name:     "ClaudeSettings",
			resource: &resource.ClaudeSettings{Name: "test"},
			expected: "claude-code",
		},
		{
			name:     "ClaudeMCPServer",
			resource: &resource.ClaudeMCPServer{Name: "test", Type: "command", Command: "test"},
			expected: "claude-code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.resource.Platform())
		})
	}
}

func TestCopilotResources_Platform(t *testing.T) {
	tests := []struct {
		name     string
		resource resource.Resource
		expected string
	}{
		{
			name:     "CopilotInstruction",
			resource: &resource.CopilotInstruction{Name: "test", Description: "test", Content: "test"},
			expected: "github-copilot",
		},
		{
			name:     "CopilotMCPServer",
			resource: &resource.CopilotMCPServer{Name: "test", Type: "stdio", Command: "test"},
			expected: "github-copilot",
		},
		{
			name:     "CopilotInstructions",
			resource: &resource.CopilotInstructions{Name: "test", Description: "test", Content: "test"},
			expected: "github-copilot",
		},
		{
			name:     "CopilotPrompt",
			resource: &resource.CopilotPrompt{Name: "test", Description: "test", Content: "test"},
			expected: "github-copilot",
		},
		{
			name:     "CopilotAgent",
			resource: &resource.CopilotAgent{Name: "test", Description: "test", Content: "test"},
			expected: "github-copilot",
		},
		{
			name:     "CopilotSkill",
			resource: &resource.CopilotSkill{Name: "test", Description: "test", Content: "test"},
			expected: "github-copilot",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.resource.Platform())
		})
	}
}

func TestPlatformFiltering_ClaudeResourcesSkippedForCopilot(t *testing.T) {
	// Create a list of mixed resources
	resources := []resource.Resource{
		&resource.CopilotInstruction{Name: "copilot-inst", Description: "test", Content: "test"},
		&resource.ClaudeRule{Name: "claude-rule", Description: "test", Content: "test"},
		&resource.CopilotPrompt{Name: "copilot-prompt", Description: "test", Content: "test"},
		&resource.ClaudeMCPServer{Name: "claude-mcp", Type: "command", Command: "test"},
	}

	// Filter for github-copilot platform
	targetPlatform := "github-copilot"
	var filtered []resource.Resource
	for _, res := range resources {
		if res.Platform() == targetPlatform {
			filtered = append(filtered, res)
		}
	}

	// Should only have 2 Copilot resources
	assert.Len(t, filtered, 2)
	assert.Equal(t, "copilot_instruction", filtered[0].ResourceType())
	assert.Equal(t, "copilot_prompt", filtered[1].ResourceType())
}

func TestPlatformFiltering_CopilotResourcesSkippedForClaude(t *testing.T) {
	// Create a list of mixed resources
	resources := []resource.Resource{
		&resource.CopilotInstruction{Name: "copilot-inst", Description: "test", Content: "test"},
		&resource.ClaudeRule{Name: "claude-rule", Description: "test", Content: "test"},
		&resource.CopilotPrompt{Name: "copilot-prompt", Description: "test", Content: "test"},
		&resource.ClaudeMCPServer{Name: "claude-mcp", Type: "command", Command: "test"},
	}

	// Filter for claude-code platform
	targetPlatform := "claude-code"
	var filtered []resource.Resource
	for _, res := range resources {
		if res.Platform() == targetPlatform {
			filtered = append(filtered, res)
		}
	}

	// Should only have 2 Claude resources
	assert.Len(t, filtered, 2)
	assert.Equal(t, "claude_rule", filtered[0].ResourceType())
	assert.Equal(t, "claude_mcp_server", filtered[1].ResourceType())
}

func TestPlatformFiltering_AllResourcesMatchingPlatform(t *testing.T) {
	// All Claude resources
	resources := []resource.Resource{
		&resource.ClaudeSkill{Name: "skill", Description: "test", Content: "test"},
		&resource.ClaudeCommand{Name: "cmd", Description: "test", Content: "test"},
		&resource.ClaudeRule{Name: "rule", Description: "test", Content: "test"},
	}

	// Filter for claude-code platform
	targetPlatform := "claude-code"
	var filtered []resource.Resource
	for _, res := range resources {
		if res.Platform() == targetPlatform {
			filtered = append(filtered, res)
		}
	}

	// All should be included
	assert.Len(t, filtered, 3)
}

func TestPlatformFiltering_NoResourcesMatchingPlatform(t *testing.T) {
	// All Copilot resources
	resources := []resource.Resource{
		&resource.CopilotInstruction{Name: "inst", Description: "test", Content: "test"},
		&resource.CopilotPrompt{Name: "prompt", Description: "test", Content: "test"},
	}

	// Filter for claude-code platform
	targetPlatform := "claude-code"
	var filtered []resource.Resource
	for _, res := range resources {
		if res.Platform() == targetPlatform {
			filtered = append(filtered, res)
		}
	}

	// None should be included
	assert.Len(t, filtered, 0)
}

// Test Copilot MCP config with custom path and key

func TestExecutor_ApplyMCPConfig_CopilotPath(t *testing.T) {
	tmpDir := t.TempDir()
	m := newTestManifest(t, tmpDir)
	executor := NewExecutor(tmpDir, m, false)

	// Copilot uses .vscode/mcp.json with "servers" key
	plan := &adapter.Plan{
		PluginName: "test-plugin",
		MCPPath:    ".vscode/mcp.json",
		MCPKey:     "servers",
		MCPEntries: map[string]any{
			"servers": map[string]any{
				"context7": map[string]any{
					"type":    "stdio",
					"command": "npx",
					"args":    []any{"-y", "@anthropic/mcp-server-context7"},
				},
			},
		},
	}

	err := executor.Execute(plan, nil)
	require.NoError(t, err)

	// Verify .vscode/mcp.json created with exact content
	mcpPath := filepath.Join(tmpDir, ".vscode", "mcp.json")
	content, err := os.ReadFile(mcpPath)
	require.NoError(t, err)

	expectedJSON := `{
  "servers": {
    "context7": {
      "args": [
        "-y",
        "@anthropic/mcp-server-context7"
      ],
      "command": "npx",
      "type": "stdio"
    }
  }
}
`
	assert.Equal(t, expectedJSON, string(content))

	// Verify .mcp.json was NOT created
	_, err = os.Stat(filepath.Join(tmpDir, ".mcp.json"))
	assert.True(t, os.IsNotExist(err), ".mcp.json should not exist for Copilot")
}

func TestExecutor_ApplyMCPConfig_CopilotMerge(t *testing.T) {
	tmpDir := t.TempDir()
	m := newTestManifest(t, tmpDir)

	// Create existing .vscode/mcp.json with servers key
	vscodeDir := filepath.Join(tmpDir, ".vscode")
	err := os.MkdirAll(vscodeDir, 0755)
	require.NoError(t, err)

	existingMCP := map[string]any{
		"servers": map[string]any{
			"existing-server": map[string]any{
				"type":    "stdio",
				"command": "existing-cmd",
			},
		},
	}
	err = WriteJSONFile(filepath.Join(vscodeDir, "mcp.json"), existingMCP)
	require.NoError(t, err)

	executor := NewExecutor(tmpDir, m, false)

	plan := &adapter.Plan{
		PluginName: "test-plugin",
		MCPPath:    ".vscode/mcp.json",
		MCPKey:     "servers",
		MCPEntries: map[string]any{
			"servers": map[string]any{
				"new-server": map[string]any{
					"type": "http",
					"url":  "https://example.com/mcp",
				},
			},
		},
	}

	err = executor.Execute(plan, nil)
	require.NoError(t, err)

	// Verify exact merged content
	content, err := os.ReadFile(filepath.Join(vscodeDir, "mcp.json"))
	require.NoError(t, err)

	expectedJSON := `{
  "servers": {
    "existing-server": {
      "command": "existing-cmd",
      "type": "stdio"
    },
    "new-server": {
      "type": "http",
      "url": "https://example.com/mcp"
    }
  }
}
`
	assert.Equal(t, expectedJSON, string(content))
}

func TestExecutor_ApplyAgentFileContent_CopilotPath(t *testing.T) {
	tmpDir := t.TempDir()
	m := newTestManifest(t, tmpDir)
	executor := NewExecutor(tmpDir, m, false)

	// Copilot uses .github/copilot-instructions.md
	plan := &adapter.Plan{
		PluginName:       "test-plugin",
		AgentFilePath:    ".github/copilot-instructions.md",
		AgentFileContent: "Always use TypeScript strict mode.",
	}

	err := executor.Execute(plan, nil)
	require.NoError(t, err)

	// Verify .github/copilot-instructions.md created (not CLAUDE.md)
	agentPath := filepath.Join(tmpDir, ".github", "copilot-instructions.md")
	content, err := os.ReadFile(agentPath)
	require.NoError(t, err)

	expected := "<!-- dex:test-plugin -->\nAlways use TypeScript strict mode.\n<!-- /dex:test-plugin -->"
	assert.Equal(t, expected, string(content))

	// Verify CLAUDE.md was NOT created
	_, err = os.Stat(filepath.Join(tmpDir, "CLAUDE.md"))
	assert.True(t, os.IsNotExist(err), "CLAUDE.md should not exist for Copilot")
}
