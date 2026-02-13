package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadProject_Valid(t *testing.T) {
	// Create a temp directory with a valid dex.hcl
	tmpDir := t.TempDir()
	hclContent := `
project {
  name = "test-project"
  agentic_platform = "claude-code"
}

registry "local" {
  path = "/path/to/registry"
}

plugin "my-plugin" {
  registry = "local"
  version = "1.0.0"
}
`
	err := os.WriteFile(filepath.Join(tmpDir, "dex.hcl"), []byte(hclContent), 0644)
	require.NoError(t, err)

	config, err := LoadProject(tmpDir)
	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, "test-project", config.Project.Name)
	assert.Equal(t, "claude-code", config.Project.AgenticPlatform)
	assert.Len(t, config.Registries, 1)
	assert.Equal(t, "local", config.Registries[0].Name)
	assert.Equal(t, "/path/to/registry", config.Registries[0].Path)
	assert.Len(t, config.Plugins, 1)
	assert.Equal(t, "my-plugin", config.Plugins[0].Name)
	assert.Equal(t, "local", config.Plugins[0].Registry)
	assert.Equal(t, "1.0.0", config.Plugins[0].Version)
}

func TestLoadProject_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	// Don't create dex.hcl

	config, err := LoadProject(tmpDir)
	assert.Nil(t, config)
	require.Error(t, err)
	expectedPrefix := fmt.Sprintf("failed to parse %s:", filepath.Join(tmpDir, "dex.hcl"))
	assert.Equal(t, expectedPrefix, err.Error()[:len(expectedPrefix)])
}

func TestLoadProject_InvalidHCL(t *testing.T) {
	tmpDir := t.TempDir()
	hclContent := `
project {
  name = "test-project"
  // Missing closing brace
`
	err := os.WriteFile(filepath.Join(tmpDir, "dex.hcl"), []byte(hclContent), 0644)
	require.NoError(t, err)

	config, err := LoadProject(tmpDir)
	assert.Nil(t, config)
	require.Error(t, err)
	expectedPrefix := fmt.Sprintf("failed to parse %s:", filepath.Join(tmpDir, "dex.hcl"))
	assert.Equal(t, expectedPrefix, err.Error()[:len(expectedPrefix)])
}

func TestLoadProject_MissingRequired(t *testing.T) {
	tmpDir := t.TempDir()
	// Empty project block - HCL may require fields at parse time
	hclContent := `
project {
  name = ""
  agentic_platform = ""
}
`
	err := os.WriteFile(filepath.Join(tmpDir, "dex.hcl"), []byte(hclContent), 0644)
	require.NoError(t, err)

	// This should parse successfully but fail validation (missing agentic_platform)
	config, err := LoadProject(tmpDir)
	require.NoError(t, err)
	assert.NotNil(t, config)

	// Validation should fail due to missing agentic_platform
	err = config.Validate()
	require.Error(t, err)
	assert.EqualError(t, err, "project.agentic_platform is required")
}

func TestProjectConfig_Validate(t *testing.T) {
	config := &ProjectConfig{
		Project: ProjectBlock{
			Name:            "test-project",
			AgenticPlatform: "claude-code",
		},
		Registries: []RegistryBlock{
			{Name: "local", Path: "/path/to/registry"},
		},
		Plugins: []PluginBlock{
			{Name: "plugin1", Registry: "local", Version: "1.0.0"},
		},
	}

	err := config.Validate()
	assert.NoError(t, err)
}

func TestProjectConfig_Validate_OptionalProjectName(t *testing.T) {
	// Project name is optional - should validate without error
	config := &ProjectConfig{
		Project: ProjectBlock{
			AgenticPlatform: "claude-code",
		},
	}

	err := config.Validate()
	assert.NoError(t, err, "project name should be optional")
}

func TestProjectConfig_Validate_MissingAgenticPlatform(t *testing.T) {
	config := &ProjectConfig{
		Project: ProjectBlock{
			Name: "test-project",
		},
	}

	err := config.Validate()
	require.Error(t, err)
	assert.EqualError(t, err, "project.agentic_platform is required")
}

func TestProjectConfig_Validate_DuplicateRegistry(t *testing.T) {
	config := &ProjectConfig{
		Project: ProjectBlock{
			Name:            "test-project",
			AgenticPlatform: "claude-code",
		},
		Registries: []RegistryBlock{
			{Name: "local", Path: "/path1"},
			{Name: "local", Path: "/path2"}, // Duplicate name
		},
	}

	err := config.Validate()
	require.Error(t, err)
	assert.EqualError(t, err, "duplicate registry name: local")
}

func TestProjectConfig_Validate_DuplicatePlugin(t *testing.T) {
	config := &ProjectConfig{
		Project: ProjectBlock{
			Name:            "test-project",
			AgenticPlatform: "claude-code",
		},
		Plugins: []PluginBlock{
			{Name: "my-plugin", Source: "file:///path1"},
			{Name: "my-plugin", Source: "file:///path2"}, // Duplicate name
		},
	}

	err := config.Validate()
	require.Error(t, err)
	assert.EqualError(t, err, "duplicate plugin name: my-plugin")
}

func TestProjectConfig_Validate_RegistryMissingPathOrURL(t *testing.T) {
	config := &ProjectConfig{
		Project: ProjectBlock{
			Name:            "test-project",
			AgenticPlatform: "claude-code",
		},
		Registries: []RegistryBlock{
			{Name: "empty-registry"}, // Missing both path and url
		},
	}

	err := config.Validate()
	require.Error(t, err)
	assert.EqualError(t, err, `registry "empty-registry" must have either path or url`)
}

func TestProjectConfig_Validate_RegistryBothPathAndURL(t *testing.T) {
	config := &ProjectConfig{
		Project: ProjectBlock{
			Name:            "test-project",
			AgenticPlatform: "claude-code",
		},
		Registries: []RegistryBlock{
			{Name: "both-registry", Path: "/path", URL: "https://example.com"}, // Has both
		},
	}

	err := config.Validate()
	require.Error(t, err)
	assert.EqualError(t, err, `registry "both-registry" cannot have both path and url`)
}

func TestProjectConfig_Validate_PluginMissingSourceOrRegistry(t *testing.T) {
	config := &ProjectConfig{
		Project: ProjectBlock{
			Name:            "test-project",
			AgenticPlatform: "claude-code",
		},
		Plugins: []PluginBlock{
			{Name: "orphan-plugin"}, // Missing both source and registry
		},
	}

	err := config.Validate()
	require.Error(t, err)
	assert.EqualError(t, err, `plugin "orphan-plugin" must have either source or registry`)
}

func TestProjectConfig_Validate_PluginBothSourceAndRegistry(t *testing.T) {
	config := &ProjectConfig{
		Project: ProjectBlock{
			Name:            "test-project",
			AgenticPlatform: "claude-code",
		},
		Registries: []RegistryBlock{
			{Name: "local", Path: "/path"},
		},
		Plugins: []PluginBlock{
			{Name: "confused-plugin", Source: "file:///path", Registry: "local"}, // Has both
		},
	}

	err := config.Validate()
	require.Error(t, err)
	assert.EqualError(t, err, `plugin "confused-plugin" cannot have both source and registry`)
}

func TestProjectConfig_Validate_PluginUnknownRegistry(t *testing.T) {
	config := &ProjectConfig{
		Project: ProjectBlock{
			Name:            "test-project",
			AgenticPlatform: "claude-code",
		},
		Plugins: []PluginBlock{
			{Name: "my-plugin", Registry: "nonexistent"}, // Registry doesn't exist
		},
	}

	err := config.Validate()
	require.Error(t, err)
	assert.EqualError(t, err, `plugin "my-plugin" references unknown registry: nonexistent`)
}

func TestLoadPackage_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	// Don't create package.hcl

	config, err := LoadPackage(tmpDir)
	assert.Nil(t, config)
	require.Error(t, err)
	expectedPrefix := fmt.Sprintf("failed to parse %s:", filepath.Join(tmpDir, "package.hcl"))
	assert.Equal(t, expectedPrefix, err.Error()[:len(expectedPrefix)])
}

func TestLoadPackage_InvalidHCL(t *testing.T) {
	tmpDir := t.TempDir()
	hclContent := `
package {
  name = "test
  // Invalid string
}
`
	err := os.WriteFile(filepath.Join(tmpDir, "package.hcl"), []byte(hclContent), 0644)
	require.NoError(t, err)

	config, err := LoadPackage(tmpDir)
	assert.Nil(t, config)
	require.Error(t, err)
	expectedPrefix := fmt.Sprintf("failed to parse %s:", filepath.Join(tmpDir, "package.hcl"))
	assert.Equal(t, expectedPrefix, err.Error()[:len(expectedPrefix)])
}

func TestPackageConfig_Validate(t *testing.T) {
	config := &PackageConfig{
		Package: PackageBlock{
			Name:    "test-package",
			Version: "1.0.0",
		},
	}

	err := config.Validate()
	assert.NoError(t, err)
}

func TestPackageConfig_Validate_MissingName(t *testing.T) {
	config := &PackageConfig{
		Package: PackageBlock{
			Version: "1.0.0",
		},
	}

	err := config.Validate()
	require.Error(t, err)
	assert.EqualError(t, err, "package.name is required")
}

func TestPackageConfig_Validate_MissingVersion(t *testing.T) {
	config := &PackageConfig{
		Package: PackageBlock{
			Name: "test-package",
		},
	}

	err := config.Validate()
	require.Error(t, err)
	assert.EqualError(t, err, "package.version is required")
}

func TestPackageConfig_Validate_DuplicateVariable(t *testing.T) {
	config := &PackageConfig{
		Package: PackageBlock{
			Name:    "test-package",
			Version: "1.0.0",
		},
		Variables: []VariableBlock{
			{Name: "var1", Default: "value1"},
			{Name: "var1", Default: "value2"}, // Duplicate
		},
	}

	err := config.Validate()
	require.Error(t, err)
	assert.EqualError(t, err, "duplicate variable name: var1")
}

func TestPackageConfig_Validate_RequiredWithDefault(t *testing.T) {
	config := &PackageConfig{
		Package: PackageBlock{
			Name:    "test-package",
			Version: "1.0.0",
		},
		Variables: []VariableBlock{
			{Name: "var1", Required: true, Default: "value"}, // Required with default
		},
	}

	err := config.Validate()
	require.Error(t, err)
	assert.EqualError(t, err, `variable "var1" is marked required but has a default value`)
}

func TestPackageConfig_GetVariable(t *testing.T) {
	config := &PackageConfig{
		Variables: []VariableBlock{
			{Name: "var1", Default: "value1"},
			{Name: "var2", Default: "value2"},
		},
	}

	v := config.GetVariable("var1")
	require.NotNil(t, v)
	assert.Equal(t, "var1", v.Name)
	assert.Equal(t, "value1", v.Default)

	v = config.GetVariable("var2")
	require.NotNil(t, v)
	assert.Equal(t, "var2", v.Name)

	v = config.GetVariable("nonexistent")
	assert.Nil(t, v)
}

func TestVariableBlock_ResolveValue_FromEnv(t *testing.T) {
	// Save original and restore after test
	original := lookupEnv
	defer func() { lookupEnv = original }()

	lookupEnv = func(key string) (string, bool) {
		if key == "TEST_VAR" {
			return "env_value", true
		}
		return "", false
	}

	v := &VariableBlock{
		Name:    "test",
		Env:     "TEST_VAR",
		Default: "default_value",
	}

	value, err := v.ResolveValue(nil)
	require.NoError(t, err)
	assert.Equal(t, "env_value", value)
}

func TestVariableBlock_ResolveValue_FromConfig(t *testing.T) {
	// Save original and restore after test
	original := lookupEnv
	defer func() { lookupEnv = original }()

	lookupEnv = func(key string) (string, bool) {
		return "", false
	}

	v := &VariableBlock{
		Name:    "test",
		Default: "default_value",
	}

	value, err := v.ResolveValue(map[string]string{"test": "config_value"})
	require.NoError(t, err)
	assert.Equal(t, "config_value", value)
}

func TestVariableBlock_ResolveValue_FromDefault(t *testing.T) {
	// Save original and restore after test
	original := lookupEnv
	defer func() { lookupEnv = original }()

	lookupEnv = func(key string) (string, bool) {
		return "", false
	}

	v := &VariableBlock{
		Name:    "test",
		Default: "default_value",
	}

	value, err := v.ResolveValue(nil)
	require.NoError(t, err)
	assert.Equal(t, "default_value", value)
}

func TestVariableBlock_ResolveValue_RequiredNotSet(t *testing.T) {
	// Save original and restore after test
	original := lookupEnv
	defer func() { lookupEnv = original }()

	lookupEnv = func(key string) (string, bool) {
		return "", false
	}

	v := &VariableBlock{
		Name:     "test",
		Required: true,
	}

	value, err := v.ResolveValue(nil)
	require.Error(t, err)
	assert.EqualError(t, err, `required variable "test" has no value`)
	assert.Equal(t, "", value)
}

func TestVariableBlock_ResolveValue_OptionalNotSet(t *testing.T) {
	// Save original and restore after test
	original := lookupEnv
	defer func() { lookupEnv = original }()

	lookupEnv = func(key string) (string, bool) {
		return "", false
	}

	v := &VariableBlock{
		Name:     "test",
		Required: false,
	}

	value, err := v.ResolveValue(nil)
	require.NoError(t, err)
	assert.Equal(t, "", value)
}

func TestParser_ParseFile(t *testing.T) {
	tmpDir := t.TempDir()
	hclContent := `
foo = "bar"
`
	filename := filepath.Join(tmpDir, "test.hcl")
	err := os.WriteFile(filename, []byte(hclContent), 0644)
	require.NoError(t, err)

	parser := NewParser()
	file, diags := parser.ParseFile(filename)

	assert.False(t, diags.HasErrors())
	assert.NotNil(t, file)
}

func TestParser_ParseFile_Invalid(t *testing.T) {
	tmpDir := t.TempDir()
	hclContent := `
foo = "unterminated string
`
	filename := filepath.Join(tmpDir, "test.hcl")
	err := os.WriteFile(filename, []byte(hclContent), 0644)
	require.NoError(t, err)

	parser := NewParser()
	_, diags := parser.ParseFile(filename)

	assert.True(t, diags.HasErrors())
}

func TestParser_ParseFile_NotExists(t *testing.T) {
	parser := NewParser()
	_, diags := parser.ParseFile("/nonexistent/path/file.hcl")

	assert.True(t, diags.HasErrors())
}

func TestNewEvalContext_EnvFunction(t *testing.T) {
	// Set an environment variable for testing
	os.Setenv("TEST_ENV_VAR", "test_value")
	defer os.Unsetenv("TEST_ENV_VAR")

	tmpDir := t.TempDir()
	hclContent := `
value = env("TEST_ENV_VAR")
`
	filename := filepath.Join(tmpDir, "test.hcl")
	err := os.WriteFile(filename, []byte(hclContent), 0644)
	require.NoError(t, err)

	parser := NewParser()
	file, diags := parser.ParseFile(filename)
	require.False(t, diags.HasErrors())

	ctx := NewEvalContext()
	assert.NotNil(t, ctx)
	require.NotNil(t, ctx.Functions)
	assert.Len(t, ctx.Functions, 1)
	assert.NotNil(t, ctx.Functions["env"])

	// Decode to verify env function works
	var result struct {
		Value string `hcl:"value,attr"`
	}
	diags = DecodeBody(file.Body, ctx, &result)
	require.False(t, diags.HasErrors())
	assert.Equal(t, "test_value", result.Value)
}

func TestNewEvalContext_EnvFunction_WithDefault(t *testing.T) {
	// Make sure the env var doesn't exist
	os.Unsetenv("NONEXISTENT_VAR")

	tmpDir := t.TempDir()
	hclContent := `
value = env("NONEXISTENT_VAR", "default_val")
`
	filename := filepath.Join(tmpDir, "test.hcl")
	err := os.WriteFile(filename, []byte(hclContent), 0644)
	require.NoError(t, err)

	parser := NewParser()
	file, diags := parser.ParseFile(filename)
	require.False(t, diags.HasErrors())

	ctx := NewEvalContext()

	var result struct {
		Value string `hcl:"value,attr"`
	}
	diags = DecodeBody(file.Body, ctx, &result)
	require.False(t, diags.HasErrors())
	assert.Equal(t, "default_val", result.Value)
}

func TestNewEvalContext_EnvFunction_NotSet(t *testing.T) {
	// Make sure the env var doesn't exist
	os.Unsetenv("NONEXISTENT_VAR")

	tmpDir := t.TempDir()
	hclContent := `
value = env("NONEXISTENT_VAR")
`
	filename := filepath.Join(tmpDir, "test.hcl")
	err := os.WriteFile(filename, []byte(hclContent), 0644)
	require.NoError(t, err)

	parser := NewParser()
	file, diags := parser.ParseFile(filename)
	require.False(t, diags.HasErrors())

	ctx := NewEvalContext()

	var result struct {
		Value string `hcl:"value,attr"`
	}
	diags = DecodeBody(file.Body, ctx, &result)
	require.False(t, diags.HasErrors())
	assert.Equal(t, "", result.Value) // Returns empty string when not set
}

func TestProjectConfig_Validate_RegistryEmptyName(t *testing.T) {
	config := &ProjectConfig{
		Project: ProjectBlock{
			Name:            "test-project",
			AgenticPlatform: "claude-code",
		},
		Registries: []RegistryBlock{
			{Path: "/path"}, // Missing name
		},
	}

	err := config.Validate()
	require.Error(t, err)
	assert.EqualError(t, err, "registry name is required")
}

func TestProjectConfig_Validate_PluginEmptyName(t *testing.T) {
	config := &ProjectConfig{
		Project: ProjectBlock{
			Name:            "test-project",
			AgenticPlatform: "claude-code",
		},
		Plugins: []PluginBlock{
			{Source: "file:///path"}, // Missing name
		},
	}

	err := config.Validate()
	require.Error(t, err)
	assert.EqualError(t, err, "plugin name is required")
}

func TestPackageConfig_Validate_VariableEmptyName(t *testing.T) {
	config := &PackageConfig{
		Package: PackageBlock{
			Name:    "test-package",
			Version: "1.0.0",
		},
		Variables: []VariableBlock{
			{Default: "value"}, // Missing name
		},
	}

	err := config.Validate()
	require.Error(t, err)
	assert.EqualError(t, err, "variable name is required")
}

func TestPackageBlock_OptionalFields(t *testing.T) {
	// Test PackageBlock struct directly instead of parsing HCL
	pkg := &PackageConfig{
		Package: PackageBlock{
			Name:       "test-package",
			Version:    "1.0.0",
			Platforms:  []string{"claude-code", "cursor"},
			Repository: "https://github.com/example/repo",
		},
	}

	assert.Equal(t, []string{"claude-code", "cursor"}, pkg.Package.Platforms)
	assert.Equal(t, "https://github.com/example/repo", pkg.Package.Repository)

	// Validation should pass
	err := pkg.Validate()
	assert.NoError(t, err)
}

func TestProjectConfig_WithPluginConfig(t *testing.T) {
	tmpDir := t.TempDir()
	hclContent := `
project {
  name = "test-project"
  agentic_platform = "claude-code"
}

plugin "my-plugin" {
  source = "file:///path/to/plugin"
  config = {
    api_key = "secret"
    endpoint = "https://api.example.com"
  }
}
`
	err := os.WriteFile(filepath.Join(tmpDir, "dex.hcl"), []byte(hclContent), 0644)
	require.NoError(t, err)

	config, err := LoadProject(tmpDir)
	require.NoError(t, err)
	assert.Len(t, config.Plugins, 1)
	assert.Equal(t, "secret", config.Plugins[0].Config["api_key"])
	assert.Equal(t, "https://api.example.com", config.Plugins[0].Config["endpoint"])
}

func TestNewPackageEvalContext_FileFunction(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file to be included
	includeContent := "This is included content"
	err := os.WriteFile(filepath.Join(tmpDir, "include.txt"), []byte(includeContent), 0644)
	require.NoError(t, err)

	// Create HCL that uses file() function
	hclContent := `
value = file("include.txt")
`
	filename := filepath.Join(tmpDir, "test.hcl")
	err = os.WriteFile(filename, []byte(hclContent), 0644)
	require.NoError(t, err)

	parser := NewParser()
	file, diags := parser.ParseFile(filename)
	require.False(t, diags.HasErrors())

	ctx := NewPackageEvalContext(tmpDir)

	var result struct {
		Value string `hcl:"value,attr"`
	}
	diags = DecodeBody(file.Body, ctx, &result)
	require.False(t, diags.HasErrors())
	assert.Equal(t, "This is included content", result.Value)
}

func TestNewPackageEvalContext_TemplatefileFunction(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a template file
	templateContent := "Hello {{ .name }}, welcome to {{ .project }}!"
	err := os.WriteFile(filepath.Join(tmpDir, "greeting.tmpl"), []byte(templateContent), 0644)
	require.NoError(t, err)

	// Create HCL that uses templatefile() function
	hclContent := `
value = templatefile("greeting.tmpl", {
  name = "Alice"
  project = "Dex"
})
`
	filename := filepath.Join(tmpDir, "test.hcl")
	err = os.WriteFile(filename, []byte(hclContent), 0644)
	require.NoError(t, err)

	parser := NewParser()
	file, diags := parser.ParseFile(filename)
	require.False(t, diags.HasErrors())

	ctx := NewPackageEvalContext(tmpDir)

	var result struct {
		Value string `hcl:"value,attr"`
	}
	diags = DecodeBody(file.Body, ctx, &result)
	require.False(t, diags.HasErrors())
	assert.Equal(t, "Hello Alice, welcome to Dex!", result.Value)
}

func TestNewPackageEvalContext_TemplatefileFunction_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	// Create HCL that uses templatefile() with non-existent file
	hclContent := `
value = templatefile("nonexistent.tmpl", {})
`
	filename := filepath.Join(tmpDir, "test.hcl")
	err := os.WriteFile(filename, []byte(hclContent), 0644)
	require.NoError(t, err)

	parser := NewParser()
	file, diags := parser.ParseFile(filename)
	require.False(t, diags.HasErrors())

	ctx := NewPackageEvalContext(tmpDir)

	var result struct {
		Value string `hcl:"value,attr"`
	}
	diags = DecodeBody(file.Body, ctx, &result)
	require.True(t, diags.HasErrors())
	expectedDiag := fmt.Sprintf(
		`%s:2,9-22: Error in function call; Call to function "templatefile" failed: reading template nonexistent.tmpl: open %s: no such file or directory., and 1 other diagnostic(s)`,
		filename, filepath.Join(tmpDir, "nonexistent.tmpl"))
	assert.Equal(t, expectedDiag, diags.Error())
}

func TestNewPackageEvalContext_TemplatefileFunction_InvalidTemplate(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an invalid template file
	templateContent := "Hello {{ .name"
	err := os.WriteFile(filepath.Join(tmpDir, "invalid.tmpl"), []byte(templateContent), 0644)
	require.NoError(t, err)

	// Create HCL that uses templatefile()
	hclContent := `
value = templatefile("invalid.tmpl", { name = "Alice" })
`
	filename := filepath.Join(tmpDir, "test.hcl")
	err = os.WriteFile(filename, []byte(hclContent), 0644)
	require.NoError(t, err)

	parser := NewParser()
	file, diags := parser.ParseFile(filename)
	require.False(t, diags.HasErrors())

	ctx := NewPackageEvalContext(tmpDir)

	var result struct {
		Value string `hcl:"value,attr"`
	}
	diags = DecodeBody(file.Body, ctx, &result)
	require.True(t, diags.HasErrors())
	expectedDiag := fmt.Sprintf(
		`%s:2,9-22: Error in function call; Call to function "templatefile" failed: parsing template invalid.tmpl: template: hcl:1: unclosed action., and 1 other diagnostic(s)`,
		filename)
	assert.Equal(t, expectedDiag, diags.Error())
}

func TestNewPackageEvalContext_TemplatefileFunction_ComplexVars(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a template that uses various types
	templateContent := `Name: {{ .name }}
Count: {{ .count }}
Active: {{ .active }}
Tags: {{ range .tags }}{{ . }} {{ end }}`
	err := os.WriteFile(filepath.Join(tmpDir, "complex.tmpl"), []byte(templateContent), 0644)
	require.NoError(t, err)

	// Create HCL that uses templatefile() with complex vars
	hclContent := `
value = templatefile("complex.tmpl", {
  name = "Test"
  count = 42
  active = true
  tags = ["go", "hcl", "template"]
})
`
	filename := filepath.Join(tmpDir, "test.hcl")
	err = os.WriteFile(filename, []byte(hclContent), 0644)
	require.NoError(t, err)

	parser := NewParser()
	file, diags := parser.ParseFile(filename)
	require.False(t, diags.HasErrors())

	ctx := NewPackageEvalContext(tmpDir)

	var result struct {
		Value string `hcl:"value,attr"`
	}
	diags = DecodeBody(file.Body, ctx, &result)
	require.False(t, diags.HasErrors())

	expected := `Name: Test
Count: 42
Active: true
Tags: go hcl template `
	assert.Equal(t, expected, result.Value)
}

// Tests for project variables (var.X syntax)

func TestLoadProject_WithVariables(t *testing.T) {
	tmpDir := t.TempDir()
	hclContent := `
variable "ORG_NAME" {
  description = "Organization name"
  default     = "my-org"
}

variable "API_KEY" {
  description = "API key"
  default     = "default-key"
}

project {
  name             = "test-project"
  agentic_platform = "claude-code"
}

plugin "test-plugin" {
  source = "file:///path/to/plugin"
  config = {
    org = var.ORG_NAME
    key = var.API_KEY
  }
}
`
	err := os.WriteFile(filepath.Join(tmpDir, "dex.hcl"), []byte(hclContent), 0644)
	require.NoError(t, err)

	config, err := LoadProject(tmpDir)
	require.NoError(t, err)
	assert.NotNil(t, config)

	// Check variables were parsed
	assert.Len(t, config.Variables, 2)
	assert.Equal(t, "ORG_NAME", config.Variables[0].Name)
	assert.Equal(t, "my-org", config.Variables[0].Default)
	assert.Equal(t, "API_KEY", config.Variables[1].Name)

	// Check resolved vars
	assert.Equal(t, "my-org", config.ResolvedVars["ORG_NAME"])
	assert.Equal(t, "default-key", config.ResolvedVars["API_KEY"])

	// Check plugin config uses interpolated values
	assert.Len(t, config.Plugins, 1)
	assert.Equal(t, "my-org", config.Plugins[0].Config["org"])
	assert.Equal(t, "default-key", config.Plugins[0].Config["key"])
}

func TestLoadProject_VariableFromEnv(t *testing.T) {
	// Set environment variable
	os.Setenv("TEST_PROJECT_VAR", "env-value")
	defer os.Unsetenv("TEST_PROJECT_VAR")

	tmpDir := t.TempDir()
	hclContent := `
variable "MY_VAR" {
  description = "Test variable"
  env         = "TEST_PROJECT_VAR"
  default     = "default-value"
}

project {
  name             = "test-project"
  agentic_platform = "claude-code"
}

plugin "test-plugin" {
  source = "file:///path/to/plugin"
  config = {
    value = var.MY_VAR
  }
}
`
	err := os.WriteFile(filepath.Join(tmpDir, "dex.hcl"), []byte(hclContent), 0644)
	require.NoError(t, err)

	config, err := LoadProject(tmpDir)
	require.NoError(t, err)

	// Environment variable should take precedence over default
	assert.Equal(t, "env-value", config.ResolvedVars["MY_VAR"])
	assert.Equal(t, "env-value", config.Plugins[0].Config["value"])
}

func TestLoadProject_RequiredVariableWithEnv(t *testing.T) {
	// Set environment variable
	os.Setenv("TEST_REQUIRED_VAR", "required-value")
	defer os.Unsetenv("TEST_REQUIRED_VAR")

	tmpDir := t.TempDir()
	hclContent := `
variable "REQUIRED_VAR" {
  description = "Required variable"
  env         = "TEST_REQUIRED_VAR"
  required    = true
}

project {
  name             = "test-project"
  agentic_platform = "claude-code"
}
`
	err := os.WriteFile(filepath.Join(tmpDir, "dex.hcl"), []byte(hclContent), 0644)
	require.NoError(t, err)

	config, err := LoadProject(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, "required-value", config.ResolvedVars["REQUIRED_VAR"])
}

func TestLoadProject_RequiredVariableMissing(t *testing.T) {
	// Ensure the env var is not set
	os.Unsetenv("NONEXISTENT_REQUIRED_VAR")

	tmpDir := t.TempDir()
	hclContent := `
variable "MISSING_VAR" {
  description = "Required variable without value"
  env         = "NONEXISTENT_REQUIRED_VAR"
  required    = true
}

project {
  name             = "test-project"
  agentic_platform = "claude-code"
}
`
	err := os.WriteFile(filepath.Join(tmpDir, "dex.hcl"), []byte(hclContent), 0644)
	require.NoError(t, err)

	config, err := LoadProject(tmpDir)
	assert.Nil(t, config)
	require.Error(t, err)
	expectedErr := fmt.Sprintf("failed to resolve variables in %s: required variable %q has no value (set via env var %q or default)",
		filepath.Join(tmpDir, "dex.hcl"), "MISSING_VAR", "NONEXISTENT_REQUIRED_VAR")
	assert.EqualError(t, err, expectedErr)
}

func TestProjectConfig_Validate_DuplicateVariableName(t *testing.T) {
	config := &ProjectConfig{
		Project: ProjectBlock{
			Name:            "test-project",
			AgenticPlatform: "claude-code",
		},
		Variables: []ProjectVariableBlock{
			{Name: "VAR1", Default: "value1"},
			{Name: "VAR1", Default: "value2"}, // Duplicate
		},
	}

	err := config.Validate()
	require.Error(t, err)
	assert.EqualError(t, err, "duplicate variable name: VAR1")
}

func TestProjectConfig_Validate_RequiredWithDefault(t *testing.T) {
	config := &ProjectConfig{
		Project: ProjectBlock{
			Name:            "test-project",
			AgenticPlatform: "claude-code",
		},
		Variables: []ProjectVariableBlock{
			{Name: "VAR1", Required: true, Default: "value"}, // Required with default
		},
	}

	err := config.Validate()
	require.Error(t, err)
	assert.EqualError(t, err, `variable "VAR1" is marked required but has a default value`)
}

func TestProjectConfig_Validate_EmptyVariableName(t *testing.T) {
	config := &ProjectConfig{
		Project: ProjectBlock{
			Name:            "test-project",
			AgenticPlatform: "claude-code",
		},
		Variables: []ProjectVariableBlock{
			{Default: "value"}, // Missing name
		},
	}

	err := config.Validate()
	require.Error(t, err)
	assert.EqualError(t, err, "variable name is required")
}

func TestLoadProject_VariableOptionalNoValue(t *testing.T) {
	// Ensure the env var is not set
	os.Unsetenv("NONEXISTENT_OPTIONAL_VAR")

	tmpDir := t.TempDir()
	hclContent := `
variable "OPTIONAL_VAR" {
  description = "Optional variable"
  env         = "NONEXISTENT_OPTIONAL_VAR"
}

project {
  name             = "test-project"
  agentic_platform = "claude-code"
}

plugin "test-plugin" {
  source = "file:///path/to/plugin"
  config = {
    value = var.OPTIONAL_VAR
  }
}
`
	err := os.WriteFile(filepath.Join(tmpDir, "dex.hcl"), []byte(hclContent), 0644)
	require.NoError(t, err)

	config, err := LoadProject(tmpDir)
	require.NoError(t, err)

	// Optional variable with no value should be empty string
	assert.Equal(t, "", config.ResolvedVars["OPTIONAL_VAR"])
	assert.Equal(t, "", config.Plugins[0].Config["value"])
}

func TestNewProjectEvalContext(t *testing.T) {
	resolvedVars := map[string]string{
		"VAR1": "value1",
		"VAR2": "value2",
	}

	projectDir := t.TempDir()
	ctx := NewProjectEvalContext(projectDir, resolvedVars)
	assert.NotNil(t, ctx)
	require.NotNil(t, ctx.Functions)
	assert.Len(t, ctx.Functions, 3)
	assert.NotNil(t, ctx.Functions["env"])
	assert.NotNil(t, ctx.Functions["file"])
	assert.NotNil(t, ctx.Functions["templatefile"])
	require.NotNil(t, ctx.Variables)
	assert.Len(t, ctx.Variables, 1)
	assert.True(t, ctx.Variables["var"].IsKnown())

	// Verify the var object contains the expected values
	varObj := ctx.Variables["var"]
	assert.True(t, varObj.Type().IsObjectType())
	val1 := varObj.GetAttr("VAR1")
	assert.Equal(t, "value1", val1.AsString())
	val2 := varObj.GetAttr("VAR2")
	assert.Equal(t, "value2", val2.AsString())
}

func TestLoadProject_EndToEnd_VariableInterpolation(t *testing.T) {
	// Set environment variable for one of the vars
	os.Setenv("E2E_PAT_VAR", "secret-pat-from-env")
	defer os.Unsetenv("E2E_PAT_VAR")

	tmpDir := t.TempDir()
	hclContent := `
variable "ORG_NAME" {
  description = "Organization name"
  default     = "my-default-org"
}

variable "PAT" {
  description = "Personal access token"
  env         = "E2E_PAT_VAR"
  default     = "default-pat"
}

variable "EMPTY_VAR" {
  description = "Variable without default or env"
}

project {
  name             = "test-project"
  agentic_platform = "claude-code"
}

plugin "azure-devops" {
  source = "file:///tmp/azure-devops"
  config = {
    org       = var.ORG_NAME
    pat       = var.PAT
    empty_val = var.EMPTY_VAR
  }
}
`
	err := os.WriteFile(filepath.Join(tmpDir, "dex.hcl"), []byte(hclContent), 0644)
	require.NoError(t, err)

	config, err := LoadProject(tmpDir)
	require.NoError(t, err)

	// Verify project metadata
	assert.Equal(t, "test-project", config.Project.Name)
	assert.Equal(t, "claude-code", config.Project.AgenticPlatform)

	// Verify variables were extracted
	assert.Len(t, config.Variables, 3)

	// Find ORG_NAME variable
	var orgVar *ProjectVariableBlock
	for i := range config.Variables {
		if config.Variables[i].Name == "ORG_NAME" {
			orgVar = &config.Variables[i]
			break
		}
	}
	require.NotNil(t, orgVar)
	assert.Equal(t, "my-default-org", orgVar.Default)
	assert.Equal(t, "Organization name", orgVar.Description)

	// Verify resolved values
	assert.Equal(t, "my-default-org", config.ResolvedVars["ORG_NAME"], "ORG_NAME should resolve to default")
	assert.Equal(t, "secret-pat-from-env", config.ResolvedVars["PAT"], "PAT should resolve from env var")
	assert.Equal(t, "", config.ResolvedVars["EMPTY_VAR"], "EMPTY_VAR should be empty string")

	// Verify plugin config was interpolated correctly
	assert.Len(t, config.Plugins, 1)
	plugin := config.Plugins[0]
	assert.Equal(t, "azure-devops", plugin.Name)
	assert.Equal(t, "my-default-org", plugin.Config["org"], "Plugin org should be interpolated from var.ORG_NAME")
	assert.Equal(t, "secret-pat-from-env", plugin.Config["pat"], "Plugin pat should be interpolated from var.PAT")
	assert.Equal(t, "", plugin.Config["empty_val"], "Plugin empty_val should be empty string")

	// Verify validation passes
	err = config.Validate()
	assert.NoError(t, err)
}

func TestLoadProject_WithResources(t *testing.T) {
	tmpDir := t.TempDir()
	hclContent := `
variable "MCP_COMMAND" {
  default = "npx"
}

variable "MCP_ARG" {
  env     = "TEST_MCP_ARG"
  default = "default-arg"
}

project {
  name             = "resource-test"
  agentic_platform = "claude-code"
}

claude_mcp_server "test-server" {
  type    = "command"
  command = var.MCP_COMMAND
  args    = ["--arg", var.MCP_ARG]
}
`
	err := os.WriteFile(filepath.Join(tmpDir, "dex.hcl"), []byte(hclContent), 0644)
	require.NoError(t, err)

	config, err := LoadProject(tmpDir)
	require.NoError(t, err)

	// Check variables were parsed
	assert.Len(t, config.Variables, 2)
	assert.Equal(t, "default-arg", config.ResolvedVars["MCP_ARG"])

	// Check resources were parsed with interpolated values
	assert.Len(t, config.Resources, 1)
	assert.Len(t, config.MCPServers, 1)
	assert.Equal(t, "test-server", config.MCPServers[0].Name)
	assert.Equal(t, "npx", config.MCPServers[0].Command)
	assert.Equal(t, []string{"--arg", "default-arg"}, config.MCPServers[0].Args)
}

func TestAddRegistry_WithURL(t *testing.T) {
	tmpDir := t.TempDir()
	initialContent := `project {
  name             = "test-project"
  agentic_platform = "claude-code"
}
`
	err := os.WriteFile(filepath.Join(tmpDir, "dex.hcl"), []byte(initialContent), 0644)
	require.NoError(t, err)

	err = AddRegistry(tmpDir, "my-registry", "https://example.com/registry", "")
	require.NoError(t, err)

	// Read back and assert exact content
	content, err := os.ReadFile(filepath.Join(tmpDir, "dex.hcl"))
	require.NoError(t, err)
	expected := `project {
  name             = "test-project"
  agentic_platform = "claude-code"
}

registry "my-registry" {
  url = "https://example.com/registry"
}
`
	assert.Equal(t, expected, string(content))
}

func TestAddRegistry_WithPath(t *testing.T) {
	tmpDir := t.TempDir()
	initialContent := `project {
  name             = "test-project"
  agentic_platform = "claude-code"
}
`
	err := os.WriteFile(filepath.Join(tmpDir, "dex.hcl"), []byte(initialContent), 0644)
	require.NoError(t, err)

	err = AddRegistry(tmpDir, "local-registry", "", "/path/to/local/registry")
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(tmpDir, "dex.hcl"))
	require.NoError(t, err)
	expected := `project {
  name             = "test-project"
  agentic_platform = "claude-code"
}

registry "local-registry" {
  path = "/path/to/local/registry"
}
`
	assert.Equal(t, expected, string(content))
}

func TestAddRegistry_DuplicateSkipped(t *testing.T) {
	tmpDir := t.TempDir()
	initialContent := `project {
  name             = "test-project"
  agentic_platform = "claude-code"
}

registry "existing" {
  url = "https://example.com/registry"
}
`
	err := os.WriteFile(filepath.Join(tmpDir, "dex.hcl"), []byte(initialContent), 0644)
	require.NoError(t, err)

	err = AddRegistry(tmpDir, "existing", "https://other.com/registry", "")
	require.NoError(t, err)

	// File should be unchanged
	content, err := os.ReadFile(filepath.Join(tmpDir, "dex.hcl"))
	require.NoError(t, err)
	assert.Equal(t, initialContent, string(content))
}

func TestAddRegistry_ErrorBothURLAndPath(t *testing.T) {
	tmpDir := t.TempDir()
	initialContent := `project {
  name             = "test-project"
  agentic_platform = "claude-code"
}
`
	err := os.WriteFile(filepath.Join(tmpDir, "dex.hcl"), []byte(initialContent), 0644)
	require.NoError(t, err)

	err = AddRegistry(tmpDir, "bad-registry", "https://example.com", "/local/path")
	require.Error(t, err)
	assert.EqualError(t, err, "cannot specify both --url and --local")
}

func TestAddRegistry_ErrorNeitherURLNorPath(t *testing.T) {
	tmpDir := t.TempDir()
	initialContent := `project {
  name             = "test-project"
  agentic_platform = "claude-code"
}
`
	err := os.WriteFile(filepath.Join(tmpDir, "dex.hcl"), []byte(initialContent), 0644)
	require.NoError(t, err)

	err = AddRegistry(tmpDir, "bad-registry", "", "")
	require.Error(t, err)
	assert.EqualError(t, err, "exactly one of --url or --local must be provided")
}

func TestAddRegistry_ErrorNoDexHCL(t *testing.T) {
	tmpDir := t.TempDir()
	// Don't create dex.hcl

	err := AddRegistry(tmpDir, "my-registry", "https://example.com", "")
	require.Error(t, err)
	expectedPrefix := fmt.Sprintf("failed to read %s:", filepath.Join(tmpDir, "dex.hcl"))
	assert.Equal(t, expectedPrefix, err.Error()[:len(expectedPrefix)])
}

func TestLoadProject_ResourcesWithEnvVarInterpolation(t *testing.T) {
	// Set env var
	os.Setenv("TEST_RESOURCE_VAR", "env-value")
	defer os.Unsetenv("TEST_RESOURCE_VAR")

	tmpDir := t.TempDir()
	hclContent := `
variable "SERVER_ARG" {
  env     = "TEST_RESOURCE_VAR"
  default = "default-value"
}

project {
  name             = "env-resource-test"
  agentic_platform = "claude-code"
}

claude_mcp_server "env-server" {
  type    = "command"
  command = "node"
  args    = [var.SERVER_ARG]
}
`
	err := os.WriteFile(filepath.Join(tmpDir, "dex.hcl"), []byte(hclContent), 0644)
	require.NoError(t, err)

	config, err := LoadProject(tmpDir)
	require.NoError(t, err)

	// Env var should override default
	assert.Equal(t, "env-value", config.ResolvedVars["SERVER_ARG"])
	assert.Len(t, config.MCPServers, 1)
	assert.Equal(t, []string{"env-value"}, config.MCPServers[0].Args)
}

// Tests for dependency block parsing

func TestLoadPackage_WithDependencies(t *testing.T) {
	tmpDir := t.TempDir()
	hclContent := `
package {
  name    = "my-plugin"
  version = "1.0.0"
}

dependency "core-lib" {
  version = "^2.0.0"
}

dependency "utils" {
  version  = ">=1.0.0"
  registry = "internal"
}

dependency "external" {
  version = "~1.5.0"
  source  = "git+https://github.com/example/external"
}
`
	err := os.WriteFile(filepath.Join(tmpDir, "package.hcl"), []byte(hclContent), 0644)
	require.NoError(t, err)

	config, err := LoadPackage(tmpDir)
	require.NoError(t, err)
	assert.NotNil(t, config)

	// Check dependencies were parsed
	assert.Len(t, config.Dependencies, 3)

	// Check first dependency
	assert.Equal(t, "core-lib", config.Dependencies[0].Name)
	assert.Equal(t, "^2.0.0", config.Dependencies[0].Version)
	assert.Empty(t, config.Dependencies[0].Registry)
	assert.Empty(t, config.Dependencies[0].Source)

	// Check second dependency
	assert.Equal(t, "utils", config.Dependencies[1].Name)
	assert.Equal(t, ">=1.0.0", config.Dependencies[1].Version)
	assert.Equal(t, "internal", config.Dependencies[1].Registry)
	assert.Empty(t, config.Dependencies[1].Source)

	// Check third dependency
	assert.Equal(t, "external", config.Dependencies[2].Name)
	assert.Equal(t, "~1.5.0", config.Dependencies[2].Version)
	assert.Empty(t, config.Dependencies[2].Registry)
	assert.Equal(t, "git+https://github.com/example/external", config.Dependencies[2].Source)
}

func TestLoadPackage_NoDependencies(t *testing.T) {
	tmpDir := t.TempDir()
	hclContent := `
package {
  name    = "simple-plugin"
  version = "1.0.0"
}
`
	err := os.WriteFile(filepath.Join(tmpDir, "package.hcl"), []byte(hclContent), 0644)
	require.NoError(t, err)

	config, err := LoadPackage(tmpDir)
	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.Empty(t, config.Dependencies)
}

func TestLoadPackage_DependenciesFromPkgHCL(t *testing.T) {
	tmpDir := t.TempDir()

	// Main package.hcl
	mainHCL := `
package {
  name    = "main-plugin"
  version = "1.0.0"
}

dependency "core" {
  version = "^1.0.0"
}
`
	err := os.WriteFile(filepath.Join(tmpDir, "package.hcl"), []byte(mainHCL), 0644)
	require.NoError(t, err)

	// Additional deps.pkg.hcl
	depsHCL := `
dependency "extra" {
  version = "^2.0.0"
}
`
	err = os.WriteFile(filepath.Join(tmpDir, "deps.pkg.hcl"), []byte(depsHCL), 0644)
	require.NoError(t, err)

	config, err := LoadPackage(tmpDir)
	require.NoError(t, err)
	assert.NotNil(t, config)

	// Should have dependencies from both files
	assert.Len(t, config.Dependencies, 2)

	// Check both dependencies are present
	depNames := make(map[string]bool)
	for _, dep := range config.Dependencies {
		depNames[dep.Name] = true
	}
	assert.True(t, depNames["core"])
	assert.True(t, depNames["extra"])
}
