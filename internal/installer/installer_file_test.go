package installer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===========================================================================
// File Resource Installation Tests
// ===========================================================================

func TestInstaller_FileResource_WithSrc(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Set up a local plugin with a file resource
	pluginDir := t.TempDir()

	// Create source file in plugin directory
	sourceContent := "# Task Configuration\ntasks:\n  - name: test\n    command: echo hello\n"
	err := os.WriteFile(filepath.Join(pluginDir, "tasks.yaml"), []byte(sourceContent), 0644)
	require.NoError(t, err)

	// Create package.hcl with file resource
	pluginContent := `package {
  name = "file-test"
  version = "1.0.0"
  description = "Plugin with file resource"
}

file "config" {
  src = "tasks.yaml"
  dest = "my_tasks.yaml"
}
`
	err = os.WriteFile(filepath.Join(pluginDir, "package.hcl"), []byte(pluginContent), 0644)
	require.NoError(t, err)

	// Create project config
	createTestProject(t, projectDir, `
plugin "file-test" {
  source = "file:`+pluginDir+`"
}
`)

	// Create installer
	installer, err := NewInstaller(projectDir)
	require.NoError(t, err)

	// Install
	err = installer.InstallAll()
	require.NoError(t, err)

	// Verify file was copied to project root
	destPath := filepath.Join(projectDir, "my_tasks.yaml")
	require.FileExists(t, destPath, "File should be copied to project root")

	// Verify content matches source
	copiedContent, err := os.ReadFile(destPath)
	require.NoError(t, err)
	assert.Equal(t, sourceContent, string(copiedContent), "Copied file content should match source")

	// Verify manifest tracks the file
	manifestPath := filepath.Join(projectDir, ".dex", "manifest.json")
	require.FileExists(t, manifestPath)

	manifestData, err := os.ReadFile(manifestPath)
	require.NoError(t, err)

	var manifest map[string]any
	err = json.Unmarshal(manifestData, &manifest)
	require.NoError(t, err)

	plugins := manifest["plugins"].(map[string]any)
	fileTest := plugins["file-test"].(map[string]any)
	files := fileTest["files"].([]any)

	// Check that my_tasks.yaml is tracked
	assert.Contains(t, files, "my_tasks.yaml", "Manifest should track the copied file")
}

func TestInstaller_FileResource_WithContent(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Set up a local plugin with a file resource that has inline content
	pluginDir := t.TempDir()

	inlineContent := "# Inline Configuration\nkey: value\n"

	// Create package.hcl with file resource containing inline content
	// Using heredoc for multi-line content
	pluginContent := `package {
  name = "file-inline"
  version = "1.0.0"
  description = "Plugin with inline file content"
}

file "config" {
  dest = "inline_config.yaml"
  content = <<EOF
# Inline Configuration
key: value
EOF
}
`
	err := os.WriteFile(filepath.Join(pluginDir, "package.hcl"), []byte(pluginContent), 0644)
	require.NoError(t, err)

	// Create project config
	createTestProject(t, projectDir, `
plugin "file-inline" {
  source = "file:`+pluginDir+`"
}
`)

	// Create installer
	installer, err := NewInstaller(projectDir)
	require.NoError(t, err)

	// Install
	err = installer.InstallAll()
	require.NoError(t, err)

	// Verify file was created with inline content
	destPath := filepath.Join(projectDir, "inline_config.yaml")
	require.FileExists(t, destPath, "File should be created with inline content")

	// Verify content matches inline content
	createdContent, err := os.ReadFile(destPath)
	require.NoError(t, err)
	assert.Equal(t, inlineContent, string(createdContent), "File content should match inline content")
}

func TestInstaller_FileResource_WithMCPServer(t *testing.T) {
	// THIS TEST REPRODUCES THE REPORTED BUG
	// Set up the project directory
	projectDir := t.TempDir()

	// Set up a local plugin similar to docker-compose package
	pluginDir := t.TempDir()

	// Create source tasks.yaml file
	tasksContent := `manifest:
  version: 1.0.0
tasks:
  - name: build
    command: docker compose build
  - name: up
    command: docker compose up -d
`
	err := os.WriteFile(filepath.Join(pluginDir, "tasks.yaml"), []byte(tasksContent), 0644)
	require.NoError(t, err)

	// Create package.hcl with file resource AND mcp_server
	// This mirrors the docker-compose package structure
	pluginContent := `package {
  name = "docker-compose"
  version = "0.1.0"
  description = "Docker Compose task automation"
}

file "tasks" {
  src = "tasks.yaml"
  dest = "docker_compose_tasks.yaml"
}

mcp_server "docker-compose-tasks" {
  description = "Docker Compose task automation"
  command = "dev-toolkit-mcp"
  args = ["-config", "docker_compose_tasks.yaml"]
}
`
	err = os.WriteFile(filepath.Join(pluginDir, "package.hcl"), []byte(pluginContent), 0644)
	require.NoError(t, err)

	// Create project config
	createTestProject(t, projectDir, `
plugin "docker-compose" {
  source = "file:`+pluginDir+`"
}
`)

	// Create installer
	installer, err := NewInstaller(projectDir)
	require.NoError(t, err)

	// Install
	err = installer.InstallAll()
	require.NoError(t, err)

	// ===========================================================
	// CRITICAL BUG CHECKS
	// ===========================================================

	// 1. Verify file was copied to project root
	tasksFilePath := filepath.Join(projectDir, "docker_compose_tasks.yaml")
	require.FileExists(t, tasksFilePath, "BUG: docker_compose_tasks.yaml should be copied to project root")

	// Verify content matches source
	copiedContent, err := os.ReadFile(tasksFilePath)
	require.NoError(t, err)
	assert.Equal(t, tasksContent, string(copiedContent), "Copied file content should match source")

	// 2. Verify .mcp.json has correct args (NOT .dex/mcp_dev_tasks/manifest.yaml)
	mcpPath := filepath.Join(projectDir, ".mcp.json")
	require.FileExists(t, mcpPath)

	mcpData, err := os.ReadFile(mcpPath)
	require.NoError(t, err)

	var mcpConfig map[string]any
	err = json.Unmarshal(mcpData, &mcpConfig)
	require.NoError(t, err)

	mcpServers := mcpConfig["mcpServers"].(map[string]any)
	assert.Contains(t, mcpServers, "docker-compose-tasks")

	server := mcpServers["docker-compose-tasks"].(map[string]any)
	assert.Equal(t, "dev-toolkit-mcp", server["command"])

	// CRITICAL: Args should reference docker_compose_tasks.yaml, NOT .dex/mcp_dev_tasks/manifest.yaml
	expectedArgs := []any{"-config", "docker_compose_tasks.yaml"}
	assert.Equal(t, expectedArgs, server["args"],
		"BUG: MCP args are being overwritten with incorrect paths. Should be ['-config', 'docker_compose_tasks.yaml']")

	// 3. Verify manifest tracks the file
	manifestPath := filepath.Join(projectDir, ".dex", "manifest.json")
	require.FileExists(t, manifestPath)

	manifestData, err := os.ReadFile(manifestPath)
	require.NoError(t, err)

	var manifest map[string]any
	err = json.Unmarshal(manifestData, &manifest)
	require.NoError(t, err)

	plugins := manifest["plugins"].(map[string]any)
	dockerCompose := plugins["docker-compose"].(map[string]any)
	files := dockerCompose["files"].([]any)

	assert.Contains(t, files, "docker_compose_tasks.yaml", "Manifest should track docker_compose_tasks.yaml")
}

func TestInstaller_MultipleFileResources(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Set up a local plugin with multiple file resources
	pluginDir := t.TempDir()

	// Create multiple source files
	file1Content := "# Config 1\nvalue: 1\n"
	file2Content := "# Config 2\nvalue: 2\n"

	err := os.WriteFile(filepath.Join(pluginDir, "config1.yaml"), []byte(file1Content), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(pluginDir, "config2.yaml"), []byte(file2Content), 0644)
	require.NoError(t, err)

	// Create package.hcl with multiple file resources
	pluginContent := `package {
  name = "multi-files"
  version = "1.0.0"
  description = "Plugin with multiple file resources"
}

file "config1" {
  src = "config1.yaml"
  dest = "my_config1.yaml"
}

file "config2" {
  src = "config2.yaml"
  dest = "my_config2.yaml"
}
`
	err = os.WriteFile(filepath.Join(pluginDir, "package.hcl"), []byte(pluginContent), 0644)
	require.NoError(t, err)

	// Create project config
	createTestProject(t, projectDir, `
plugin "multi-files" {
  source = "file:`+pluginDir+`"
}
`)

	// Create installer
	installer, err := NewInstaller(projectDir)
	require.NoError(t, err)

	// Install
	err = installer.InstallAll()
	require.NoError(t, err)

	// Verify both files were copied
	file1Path := filepath.Join(projectDir, "my_config1.yaml")
	require.FileExists(t, file1Path, "First file should be copied")

	file2Path := filepath.Join(projectDir, "my_config2.yaml")
	require.FileExists(t, file2Path, "Second file should be copied")

	// Verify content
	content1, err := os.ReadFile(file1Path)
	require.NoError(t, err)
	assert.Equal(t, file1Content, string(content1))

	content2, err := os.ReadFile(file2Path)
	require.NoError(t, err)
	assert.Equal(t, file2Content, string(content2))

	// Verify manifest tracks both files
	manifestPath := filepath.Join(projectDir, ".dex", "manifest.json")
	require.FileExists(t, manifestPath)

	manifestData, err := os.ReadFile(manifestPath)
	require.NoError(t, err)

	var manifest map[string]any
	err = json.Unmarshal(manifestData, &manifest)
	require.NoError(t, err)

	plugins := manifest["plugins"].(map[string]any)
	multiFiles := plugins["multi-files"].(map[string]any)
	files := multiFiles["files"].([]any)

	assert.Contains(t, files, "my_config1.yaml", "Manifest should track first file")
	assert.Contains(t, files, "my_config2.yaml", "Manifest should track second file")
}

func TestInstaller_FileResource_WithChmod(t *testing.T) {
	// Set up the project directory
	projectDir := t.TempDir()

	// Set up a local plugin with a file resource that has chmod
	pluginDir := t.TempDir()

	// Create source script file
	scriptContent := "#!/bin/bash\necho 'Hello from script'\n"
	err := os.WriteFile(filepath.Join(pluginDir, "script.sh"), []byte(scriptContent), 0644)
	require.NoError(t, err)

	// Create package.hcl with file resource with chmod
	pluginContent := `package {
  name = "file-chmod"
  version = "1.0.0"
  description = "Plugin with executable file"
}

file "script" {
  src = "script.sh"
  dest = "my_script.sh"
  chmod = "755"
}
`
	err = os.WriteFile(filepath.Join(pluginDir, "package.hcl"), []byte(pluginContent), 0644)
	require.NoError(t, err)

	// Create project config
	createTestProject(t, projectDir, `
plugin "file-chmod" {
  source = "file:`+pluginDir+`"
}
`)

	// Create installer
	installer, err := NewInstaller(projectDir)
	require.NoError(t, err)

	// Install
	err = installer.InstallAll()
	require.NoError(t, err)

	// Verify file was copied with correct permissions
	scriptPath := filepath.Join(projectDir, "my_script.sh")
	require.FileExists(t, scriptPath, "Script file should be copied")

	// Verify content
	copiedContent, err := os.ReadFile(scriptPath)
	require.NoError(t, err)
	assert.Equal(t, scriptContent, string(copiedContent))

	// Verify permissions
	info, err := os.Stat(scriptPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0755), info.Mode().Perm(), "File should have 0755 permissions")
}
