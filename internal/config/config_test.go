package config

import (
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
	assert.Contains(t, err.Error(), "failed to parse")
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
	assert.Contains(t, err.Error(), "failed to parse")
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
	assert.Contains(t, err.Error(), "project.agentic_platform is required")
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
	assert.Contains(t, err.Error(), "project.agentic_platform is required")
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
	assert.Contains(t, err.Error(), "duplicate registry name: local")
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
	assert.Contains(t, err.Error(), "duplicate plugin name: my-plugin")
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
	assert.Contains(t, err.Error(), "must have either path or url")
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
	assert.Contains(t, err.Error(), "cannot have both path and url")
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
	assert.Contains(t, err.Error(), "must have either source or registry")
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
	assert.Contains(t, err.Error(), "cannot have both source and registry")
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
	assert.Contains(t, err.Error(), "references unknown registry: nonexistent")
}

func TestLoadPackage_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	// Don't create package.hcl

	config, err := LoadPackage(tmpDir)
	assert.Nil(t, config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")
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
	assert.Contains(t, err.Error(), "failed to parse")
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
	assert.Contains(t, err.Error(), "package.name is required")
}

func TestPackageConfig_Validate_MissingVersion(t *testing.T) {
	config := &PackageConfig{
		Package: PackageBlock{
			Name: "test-package",
		},
	}

	err := config.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "package.version is required")
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
	assert.Contains(t, err.Error(), "duplicate variable name: var1")
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
	assert.Contains(t, err.Error(), "is marked required but has a default value")
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
	assert.Contains(t, err.Error(), "required variable")
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
	assert.NotNil(t, ctx.Functions)
	assert.Contains(t, ctx.Functions, "env")

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
	assert.Contains(t, err.Error(), "registry name is required")
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
	assert.Contains(t, err.Error(), "plugin name is required")
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
	assert.Contains(t, err.Error(), "variable name is required")
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
	assert.Contains(t, diags.Error(), "reading template")
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
	assert.Contains(t, diags.Error(), "parsing template")
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
