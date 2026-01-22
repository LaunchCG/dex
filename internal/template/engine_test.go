package template

import (
	"os"
	"path/filepath"
	"testing"
)

func TestContext_NewContext(t *testing.T) {
	ctx := NewContext("my-plugin", "1.0.0", "/project", "claude-code")

	if ctx.PluginName != "my-plugin" {
		t.Errorf("PluginName = %q, want %q", ctx.PluginName, "my-plugin")
	}
	if ctx.PluginVersion != "1.0.0" {
		t.Errorf("PluginVersion = %q, want %q", ctx.PluginVersion, "1.0.0")
	}
	if ctx.ProjectRoot != "/project" {
		t.Errorf("ProjectRoot = %q, want %q", ctx.ProjectRoot, "/project")
	}
	if ctx.Platform != "claude-code" {
		t.Errorf("Platform = %q, want %q", ctx.Platform, "claude-code")
	}
	if ctx.Variables == nil {
		t.Error("Variables should be initialized")
	}
	if ctx.ExtraVars == nil {
		t.Error("ExtraVars should be initialized")
	}
}

func TestContext_WithComponentDir(t *testing.T) {
	ctx := NewContext("plugin", "1.0.0", "/project", "claude-code")
	ctx.WithComponentDir("/project/.claude/skills/plugin-test")

	if ctx.ComponentDir != "/project/.claude/skills/plugin-test" {
		t.Errorf("ComponentDir = %q, want %q", ctx.ComponentDir, "/project/.claude/skills/plugin-test")
	}
}

func TestContext_ToMap(t *testing.T) {
	ctx := NewContext("my-plugin", "2.0.0", "/proj", "claude-code")
	ctx.WithComponentDir("/proj/.claude/skills/test")
	ctx.Variables["customVar"] = "customValue"
	ctx.ExtraVars["extraVar"] = "extraValue"

	m := ctx.ToMap()

	tests := map[string]any{
		"PluginName":    "my-plugin",
		"PluginVersion": "2.0.0",
		"ProjectRoot":   "/proj",
		"Platform":      "claude-code",
		"ComponentDir":  "/proj/.claude/skills/test",
		"customVar":     "customValue",
		"extraVar":      "extraValue",
	}

	for key, want := range tests {
		got, ok := m[key]
		if !ok {
			t.Errorf("ToMap() missing key %q", key)
			continue
		}
		if got != want {
			t.Errorf("ToMap()[%q] = %v, want %v", key, got, want)
		}
	}
}

func TestContext_Clone(t *testing.T) {
	ctx := NewContext("plugin", "1.0.0", "/project", "claude-code")
	ctx.Variables["var1"] = "val1"
	ctx.ExtraVars["extra1"] = "extraval1"

	clone := ctx.Clone()

	// Modify original
	ctx.Variables["var1"] = "modified"
	ctx.ExtraVars["extra1"] = "modified"

	// Clone should be unchanged
	if clone.Variables["var1"] != "val1" {
		t.Error("Clone Variables should be independent")
	}
	if clone.ExtraVars["extra1"] != "extraval1" {
		t.Error("Clone ExtraVars should be independent")
	}
}

func TestEngine_Render(t *testing.T) {
	ctx := NewContext("my-plugin", "1.0.0", "/project", "claude-code")
	engine := NewEngine("/tmp", ctx)

	result, err := engine.Render("Plugin: {{ .PluginName }}, Version: {{ .PluginVersion }}")
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	expected := "Plugin: my-plugin, Version: 1.0.0"
	if result != expected {
		t.Errorf("Render() = %q, want %q", result, expected)
	}
}

func TestEngine_Render_AllBuiltinVars(t *testing.T) {
	ctx := NewContext("test-plugin", "2.5.0", "/home/user/project", "claude-code")
	ctx.WithComponentDir("/home/user/project/.claude/skills/test")
	engine := NewEngine("/tmp", ctx)

	template := `Plugin: {{ .PluginName }}
Version: {{ .PluginVersion }}
Project: {{ .ProjectRoot }}
Platform: {{ .Platform }}
Component: {{ .ComponentDir }}`

	result, err := engine.Render(template)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	expected := `Plugin: test-plugin
Version: 2.5.0
Project: /home/user/project
Platform: claude-code
Component: /home/user/project/.claude/skills/test`

	if result != expected {
		t.Errorf("Render() = %q, want %q", result, expected)
	}
}

func TestEngine_Render_UserVariables(t *testing.T) {
	ctx := NewContext("plugin", "1.0.0", "/project", "claude-code")
	ctx.Variables["apiUrl"] = "https://api.example.com"
	ctx.Variables["timeout"] = "30"
	engine := NewEngine("/tmp", ctx)

	result, err := engine.Render("API: {{ .apiUrl }}, Timeout: {{ .timeout }}s")
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	expected := "API: https://api.example.com, Timeout: 30s"
	if result != expected {
		t.Errorf("Render() = %q, want %q", result, expected)
	}
}

func TestEngine_Render_InvalidTemplate(t *testing.T) {
	ctx := NewContext("plugin", "1.0.0", "/project", "claude-code")
	engine := NewEngine("/tmp", ctx)

	_, err := engine.Render("{{ .Invalid")
	if err == nil {
		t.Error("Render() should return error for invalid template")
	}
}

func TestEngine_RenderFile(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "test.tmpl"), []byte("Hello {{ .PluginName }}"), 0644); err != nil {
		t.Fatal(err)
	}

	ctx := NewContext("test-plugin", "1.0.0", "/project", "claude-code")
	engine := NewEngine(tmpDir, ctx)

	result, err := engine.RenderFile("test.tmpl")
	if err != nil {
		t.Fatalf("RenderFile() error = %v", err)
	}

	expected := "Hello test-plugin"
	if result != expected {
		t.Errorf("RenderFile() = %q, want %q", result, expected)
	}
}

func TestEngine_RenderFile_NotFound(t *testing.T) {
	ctx := NewContext("plugin", "1.0.0", "/project", "claude-code")
	engine := NewEngine(t.TempDir(), ctx)

	_, err := engine.RenderFile("nonexistent.tmpl")
	if err == nil {
		t.Error("RenderFile() should return error for missing file")
	}
}

func TestEngine_RenderFileWithVars(t *testing.T) {
	tmpDir := t.TempDir()
	content := "Plugin: {{ .PluginName }}, Env: {{ .environment }}, Region: {{ .region }}"
	if err := os.WriteFile(filepath.Join(tmpDir, "config.tmpl"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	ctx := NewContext("my-plugin", "1.0.0", "/project", "claude-code")
	engine := NewEngine(tmpDir, ctx)

	vars := map[string]any{
		"environment": "production",
		"region":      "us-east-1",
	}

	result, err := engine.RenderFileWithVars("config.tmpl", vars)
	if err != nil {
		t.Fatalf("RenderFileWithVars() error = %v", err)
	}

	expected := "Plugin: my-plugin, Env: production, Region: us-east-1"
	if result != expected {
		t.Errorf("RenderFileWithVars() = %q, want %q", result, expected)
	}
}

func TestEngine_RenderWithVars(t *testing.T) {
	ctx := NewContext("plugin", "1.0.0", "/project", "claude-code")
	engine := NewEngine(t.TempDir(), ctx)

	vars := map[string]any{
		"customKey": "customValue",
	}

	result, err := engine.RenderWithVars("Key: {{ .customKey }}", vars)
	if err != nil {
		t.Fatalf("RenderWithVars() error = %v", err)
	}

	expected := "Key: customValue"
	if result != expected {
		t.Errorf("RenderWithVars() = %q, want %q", result, expected)
	}
}

func TestEngine_FileFunction(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "include.txt"), []byte("INCLUDED CONTENT"), 0644); err != nil {
		t.Fatal(err)
	}

	ctx := NewContext("plugin", "1.0.0", "/project", "claude-code")
	engine := NewEngine(tmpDir, ctx)

	result, err := engine.Render(`Content: {{ file "include.txt" }}`)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	expected := "Content: INCLUDED CONTENT"
	if result != expected {
		t.Errorf("Render() = %q, want %q", result, expected)
	}
}

func TestEngine_FileFunction_NotFound(t *testing.T) {
	ctx := NewContext("plugin", "1.0.0", "/project", "claude-code")
	engine := NewEngine(t.TempDir(), ctx)

	_, err := engine.Render(`{{ file "nonexistent.txt" }}`)
	if err == nil {
		t.Error("file() should return error for missing file")
	}
}

func TestEngine_EnvFunction(t *testing.T) {
	os.Setenv("TEST_TEMPLATE_VAR", "test_value")
	defer os.Unsetenv("TEST_TEMPLATE_VAR")

	ctx := NewContext("plugin", "1.0.0", "/project", "claude-code")
	engine := NewEngine(t.TempDir(), ctx)

	result, err := engine.Render(`Value: {{ env "TEST_TEMPLATE_VAR" }}`)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	expected := "Value: test_value"
	if result != expected {
		t.Errorf("Render() = %q, want %q", result, expected)
	}
}

func TestEngine_EnvFunction_WithDefault(t *testing.T) {
	// Ensure the variable is not set
	os.Unsetenv("NONEXISTENT_VAR_FOR_TEST")

	ctx := NewContext("plugin", "1.0.0", "/project", "claude-code")
	engine := NewEngine(t.TempDir(), ctx)

	result, err := engine.Render(`Value: {{ env "NONEXISTENT_VAR_FOR_TEST" "default_value" }}`)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	expected := "Value: default_value"
	if result != expected {
		t.Errorf("Render() = %q, want %q", result, expected)
	}
}

func TestEngine_DictFunction(t *testing.T) {
	ctx := NewContext("plugin", "1.0.0", "/project", "claude-code")
	engine := NewEngine(t.TempDir(), ctx)

	result, err := engine.Render(`{{ $m := dict "name" "Alice" "age" 30 }}Name: {{ $m.name }}, Age: {{ $m.age }}`)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	expected := "Name: Alice, Age: 30"
	if result != expected {
		t.Errorf("Render() = %q, want %q", result, expected)
	}
}

func TestEngine_DictFunction_Empty(t *testing.T) {
	ctx := NewContext("plugin", "1.0.0", "/project", "claude-code")
	engine := NewEngine(t.TempDir(), ctx)

	result, err := engine.Render(`{{ $m := dict }}{{ len $m }}`)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if result != "0" {
		t.Errorf("Render() = %q, want %q", result, "0")
	}
}

func TestEngine_DictFunction_OddArgs(t *testing.T) {
	ctx := NewContext("plugin", "1.0.0", "/project", "claude-code")
	engine := NewEngine(t.TempDir(), ctx)

	// Odd number of args - last value is ignored
	result, err := engine.Render(`{{ $m := dict "key1" "val1" "key2" }}{{ $m.key1 }}`)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if result != "val1" {
		t.Errorf("Render() = %q, want %q", result, "val1")
	}
}

func TestEngine_TemplatefileWithDict(t *testing.T) {
	tmpDir := t.TempDir()
	nested := "Hello {{ .name }}, welcome to {{ .project }}!"
	if err := os.WriteFile(filepath.Join(tmpDir, "greeting.tmpl"), []byte(nested), 0644); err != nil {
		t.Fatal(err)
	}

	ctx := NewContext("my-plugin", "1.0.0", "/project", "claude-code")
	engine := NewEngine(tmpDir, ctx)

	result, err := engine.Render(`{{ templatefile "greeting.tmpl" (dict "name" "Bob" "project" "Dex") }}`)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	expected := "Hello Bob, welcome to Dex!"
	if result != expected {
		t.Errorf("Render() = %q, want %q", result, expected)
	}
}

func TestEngine_TemplatefileWithContextVars(t *testing.T) {
	tmpDir := t.TempDir()
	nested := "Plugin: {{ .PluginName }}, Custom: {{ .customVar }}"
	if err := os.WriteFile(filepath.Join(tmpDir, "nested.tmpl"), []byte(nested), 0644); err != nil {
		t.Fatal(err)
	}

	ctx := NewContext("my-plugin", "1.0.0", "/project", "claude-code")
	engine := NewEngine(tmpDir, ctx)

	result, err := engine.Render(`{{ templatefile "nested.tmpl" (dict "customVar" "custom_value") }}`)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	expected := "Plugin: my-plugin, Custom: custom_value"
	if result != expected {
		t.Errorf("Render() = %q, want %q", result, expected)
	}
}

func TestEngine_ComplexTemplate(t *testing.T) {
	tmpDir := t.TempDir()

	// Create include file (file function includes raw content, not rendered)
	if err := os.WriteFile(filepath.Join(tmpDir, "header.txt"), []byte("# Header for complex-plugin"), 0644); err != nil {
		t.Fatal(err)
	}

	ctx := NewContext("complex-plugin", "3.0.0", "/home/project", "claude-code")
	ctx.WithComponentDir("/home/project/.claude/skills/complex")
	ctx.Variables["author"] = "Test Author"
	engine := NewEngine(tmpDir, ctx)

	template := `{{ file "header.txt" }}

## Plugin Info
- Name: {{ .PluginName }}
- Version: {{ .PluginVersion }}
- Author: {{ .author }}
- Installed to: {{ .ComponentDir }}
- Platform: {{ .Platform }}`

	result, err := engine.Render(template)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	expected := `# Header for complex-plugin

## Plugin Info
- Name: complex-plugin
- Version: 3.0.0
- Author: Test Author
- Installed to: /home/project/.claude/skills/complex
- Platform: claude-code`

	if result != expected {
		t.Errorf("Render() =\n%s\n\nwant:\n%s", result, expected)
	}
}
