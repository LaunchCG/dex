package lockfile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
	if len(l.Packages) != 0 {
		t.Errorf("Packages = %v, want empty map", l.Packages)
	}
}

func TestLoad_Existing(t *testing.T) {
	tmpDir := t.TempDir()

	lockData := `{
  "version": "1.0",
  "agent": "claude-code",
  "packages": {
    "test-pkg": {
      "version": "1.2.0",
      "resolved": "git+https://github.com/user/pkg.git#v1.2.0",
      "integrity": "sha256-abc123",
      "dependencies": {
        "other-pkg": "^1.0.0"
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

	pkg := l.Get("test-pkg")
	if pkg == nil {
		t.Fatal("Get() returned nil")
	}

	if pkg.Version != "1.2.0" {
		t.Errorf("Version = %q, want %q", pkg.Version, "1.2.0")
	}
	if pkg.Resolved != "git+https://github.com/user/pkg.git#v1.2.0" {
		t.Errorf("Resolved = %q, want %q", pkg.Resolved, "git+https://github.com/user/pkg.git#v1.2.0")
	}
	if pkg.Integrity != "sha256-abc123" {
		t.Errorf("Integrity = %q, want %q", pkg.Integrity, "sha256-abc123")
	}
	if dep, ok := pkg.Dependencies["other-pkg"]; !ok || dep != "^1.0.0" {
		t.Errorf("Dependencies = %v, want map[other-pkg:^1.0.0]", pkg.Dependencies)
	}
}

func TestSave(t *testing.T) {
	tmpDir := t.TempDir()

	l, err := Load(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	l.Agent = "claude-code"
	l.Set("test-pkg", &LockedPackage{
		Version:   "1.0.0",
		Resolved:  "file:./pkgs/test",
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
  "packages": {
    "test-pkg": {
      "version": "1.0.0",
      "resolved": "file:./pkgs/test",
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
		t.Error("Get() returned non-nil for nonexistent pkg")
	}
}

func TestSet(t *testing.T) {
	tmpDir := t.TempDir()

	l, err := Load(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	l.Set("pkg-a", &LockedPackage{
		Version:   "1.0.0",
		Resolved:  "resolved-a",
		Integrity: "sha256-a",
	})

	// Update existing
	l.Set("pkg-a", &LockedPackage{
		Version:   "2.0.0",
		Resolved:  "resolved-a-v2",
		Integrity: "sha256-a-v2",
	})

	pkg := l.Get("pkg-a")
	if pkg.Version != "2.0.0" {
		t.Errorf("Version = %q, want %q", pkg.Version, "2.0.0")
	}
}

func TestRemove(t *testing.T) {
	tmpDir := t.TempDir()

	l, err := Load(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	l.Set("pkg-a", &LockedPackage{Version: "1.0.0"})
	l.Set("pkg-b", &LockedPackage{Version: "1.0.0"})

	l.Remove("pkg-a")

	if l.Has("pkg-a") {
		t.Error("pkg-a still exists after Remove")
	}
	if !l.Has("pkg-b") {
		t.Error("pkg-b should still exist")
	}
}

func TestHas(t *testing.T) {
	tmpDir := t.TempDir()

	l, err := Load(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	l.Set("pkg-a", &LockedPackage{Version: "1.0.0"})

	if !l.Has("pkg-a") {
		t.Error("Has() = false for existing pkg")
	}
	if l.Has("pkg-b") {
		t.Error("Has() = true for non-existing pkg")
	}
}

func TestLockedPackages(t *testing.T) {
	tmpDir := t.TempDir()

	l, err := Load(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	l.Set("zebra", &LockedPackage{Version: "1.0.0"})
	l.Set("alpha", &LockedPackage{Version: "1.0.0"})
	l.Set("beta", &LockedPackage{Version: "1.0.0"})

	pkgs := l.LockedPackages()

	expected := []string{"alpha", "beta", "zebra"}
	if len(pkgs) != len(expected) {
		t.Fatalf("LockedPackages() = %v, want %v", pkgs, expected)
	}
	for i, name := range expected {
		if pkgs[i] != name {
			t.Errorf("pkgs[%d] = %q, want %q", i, pkgs[i], name)
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
	l.Set("pkg-a", &LockedPackage{
		Version:      "1.0.0",
		Dependencies: nil,
	})

	pkg := l.Get("pkg-a")
	if pkg.Dependencies == nil {
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
	l.Set("pkg-name", &LockedPackage{
		Version:   "1.2.0",
		Resolved:  "git+https://github.com/user/pkg.git#v1.2.0",
		Integrity: "sha256-abc123",
		Dependencies: map[string]string{
			"dep-pkg": "^1.0.0",
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
  "packages": {
    "pkg-name": {
      "version": "1.2.0",
      "resolved": "git+https://github.com/user/pkg.git#v1.2.0",
      "integrity": "sha256-abc123",
      "dependencies": {
        "dep-pkg": "^1.0.0"
      }
    }
  }
}`
	if string(data) != expected {
		t.Errorf("Lock file format mismatch.\nGot:\n%s\n\nWant:\n%s", string(data), expected)
	}
}

func TestSave_VersionConstraintsNotHTMLEscaped(t *testing.T) {
	tmpDir := t.TempDir()

	l, err := Load(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	l.Agent = "claude-code"
	l.Set("my-plugin", &LockedPackage{
		Version:   "2.0.0",
		Resolved:  "https://example.com/my-plugin/2.0.0.tar.gz",
		Integrity: "sha256-abc",
		Dependencies: map[string]string{
			"dep-a": ">=1.0.0",
			"dep-b": "<2.0.0",
			"dep-c": ">=1.0.0 & <3.0.0",
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

	raw := string(data)

	// Must contain literal >= and <, not unicode escapes
	if !strings.Contains(raw, ">=1.0.0") {
		t.Errorf("lock file should contain literal >=1.0.0, got:\n%s", raw)
	}
	if !strings.Contains(raw, "<2.0.0") {
		t.Errorf("lock file should contain literal <2.0.0, got:\n%s", raw)
	}

	// Must NOT contain Go's HTML-escaped versions
	if strings.Contains(raw, `\u003e`) {
		t.Errorf("lock file contains escaped \\u003e (>), should use literal >:\n%s", raw)
	}
	if strings.Contains(raw, `\u003c`) {
		t.Errorf("lock file contains escaped \\u003c (<), should use literal <:\n%s", raw)
	}
	if strings.Contains(raw, `\u0026`) {
		t.Errorf("lock file contains escaped \\u0026 (&), should use literal &:\n%s", raw)
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
