package lockfile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()

	l, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if l.Version != LockFileVersion {
		t.Errorf("Version = %q, want %q", l.Version, LockFileVersion)
	}
	if len(l.Plugins) != 0 {
		t.Errorf("Plugins = %v, want empty map", l.Plugins)
	}
}

func TestLoad_Existing(t *testing.T) {
	tmpDir := t.TempDir()

	lockData := `{
  "version": "1.0",
  "agent": "claude-code",
  "plugins": {
    "test-plugin": {
      "version": "1.2.0",
      "resolved": "git+https://github.com/user/plugin.git#v1.2.0",
      "integrity": "sha256-abc123",
      "dependencies": {
        "other-plugin": "^1.0.0"
      }
    }
  }
}`
	lockPath := filepath.Join(tmpDir, LockFileName)
	if err := os.WriteFile(lockPath, []byte(lockData), 0644); err != nil {
		t.Fatal(err)
	}

	l, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if l.Version != "1.0" {
		t.Errorf("Version = %q, want %q", l.Version, "1.0")
	}
	if l.Agent != "claude-code" {
		t.Errorf("Agent = %q, want %q", l.Agent, "claude-code")
	}

	plugin := l.Get("test-plugin")
	if plugin == nil {
		t.Fatal("Get() returned nil")
	}

	if plugin.Version != "1.2.0" {
		t.Errorf("Version = %q, want %q", plugin.Version, "1.2.0")
	}
	if plugin.Resolved != "git+https://github.com/user/plugin.git#v1.2.0" {
		t.Errorf("Resolved = %q, want %q", plugin.Resolved, "git+https://github.com/user/plugin.git#v1.2.0")
	}
	if plugin.Integrity != "sha256-abc123" {
		t.Errorf("Integrity = %q, want %q", plugin.Integrity, "sha256-abc123")
	}
	if dep, ok := plugin.Dependencies["other-plugin"]; !ok || dep != "^1.0.0" {
		t.Errorf("Dependencies = %v, want map[other-plugin:^1.0.0]", plugin.Dependencies)
	}
}

func TestSave(t *testing.T) {
	tmpDir := t.TempDir()

	l, err := Load(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	l.Agent = "claude-code"
	l.Set("test-plugin", &LockedPlugin{
		Version:   "1.0.0",
		Resolved:  "file:./plugins/test",
		Integrity: "sha256-xyz789",
	})

	if err := l.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file was created
	lockPath := filepath.Join(tmpDir, LockFileName)
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	expected := `{
  "version": "1.0",
  "agent": "claude-code",
  "plugins": {
    "test-plugin": {
      "version": "1.0.0",
      "resolved": "file:./plugins/test",
      "integrity": "sha256-xyz789",
      "dependencies": {}
    }
  }
}`
	if string(data) != expected {
		t.Errorf("Saved content =\n%s\n\nwant:\n%s", string(data), expected)
	}
}

func TestGet_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()

	l, err := Load(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if l.Get("nonexistent") != nil {
		t.Error("Get() returned non-nil for nonexistent plugin")
	}
}

func TestSet(t *testing.T) {
	tmpDir := t.TempDir()

	l, err := Load(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	l.Set("plugin-a", &LockedPlugin{
		Version:   "1.0.0",
		Resolved:  "resolved-a",
		Integrity: "sha256-a",
	})

	// Update existing
	l.Set("plugin-a", &LockedPlugin{
		Version:   "2.0.0",
		Resolved:  "resolved-a-v2",
		Integrity: "sha256-a-v2",
	})

	plugin := l.Get("plugin-a")
	if plugin.Version != "2.0.0" {
		t.Errorf("Version = %q, want %q", plugin.Version, "2.0.0")
	}
}

func TestRemove(t *testing.T) {
	tmpDir := t.TempDir()

	l, err := Load(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	l.Set("plugin-a", &LockedPlugin{Version: "1.0.0"})
	l.Set("plugin-b", &LockedPlugin{Version: "1.0.0"})

	l.Remove("plugin-a")

	if l.Has("plugin-a") {
		t.Error("plugin-a still exists after Remove")
	}
	if !l.Has("plugin-b") {
		t.Error("plugin-b should still exist")
	}
}

func TestHas(t *testing.T) {
	tmpDir := t.TempDir()

	l, err := Load(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	l.Set("plugin-a", &LockedPlugin{Version: "1.0.0"})

	if !l.Has("plugin-a") {
		t.Error("Has() = false for existing plugin")
	}
	if l.Has("plugin-b") {
		t.Error("Has() = true for non-existing plugin")
	}
}

func TestLockedPlugins(t *testing.T) {
	tmpDir := t.TempDir()

	l, err := Load(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	l.Set("zebra", &LockedPlugin{Version: "1.0.0"})
	l.Set("alpha", &LockedPlugin{Version: "1.0.0"})
	l.Set("beta", &LockedPlugin{Version: "1.0.0"})

	plugins := l.LockedPlugins()

	expected := []string{"alpha", "beta", "zebra"}
	if len(plugins) != len(expected) {
		t.Fatalf("LockedPlugins() = %v, want %v", plugins, expected)
	}
	for i, name := range expected {
		if plugins[i] != name {
			t.Errorf("plugins[%d] = %q, want %q", i, plugins[i], name)
		}
	}
}

func TestSet_InitializesDependencies(t *testing.T) {
	tmpDir := t.TempDir()

	l, err := Load(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Set with nil Dependencies
	l.Set("plugin-a", &LockedPlugin{
		Version:      "1.0.0",
		Dependencies: nil,
	})

	plugin := l.Get("plugin-a")
	if plugin.Dependencies == nil {
		t.Error("Dependencies should be initialized to empty map")
	}
}

func TestLockFileFormat(t *testing.T) {
	tmpDir := t.TempDir()

	l, err := Load(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	l.Agent = "claude-code"
	l.Set("plugin-name", &LockedPlugin{
		Version:   "1.2.0",
		Resolved:  "git+https://github.com/user/plugin.git#v1.2.0",
		Integrity: "sha256-abc123",
		Dependencies: map[string]string{
			"dep-plugin": "^1.0.0",
		},
	})

	if err := l.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	lockPath := filepath.Join(tmpDir, LockFileName)
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	// Verify it's valid JSON by parsing
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	// Verify exact format
	expected := `{
  "version": "1.0",
  "agent": "claude-code",
  "plugins": {
    "plugin-name": {
      "version": "1.2.0",
      "resolved": "git+https://github.com/user/plugin.git#v1.2.0",
      "integrity": "sha256-abc123",
      "dependencies": {
        "dep-plugin": "^1.0.0"
      }
    }
  }
}`
	if string(data) != expected {
		t.Errorf("Lock file format mismatch.\nGot:\n%s\n\nWant:\n%s", string(data), expected)
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()

	lockPath := filepath.Join(tmpDir, LockFileName)
	if err := os.WriteFile(lockPath, []byte("invalid json{"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(tmpDir)
	if err == nil {
		t.Error("Load() should return error for invalid JSON")
	}
}
