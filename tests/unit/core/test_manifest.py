"""Tests for dex.core.manifest module."""

import json
from pathlib import Path

import pytest

from dex.config.schemas import DexManifest, PluginFiles
from dex.core.manifest import ManifestManager


class TestDexManifest:
    """Tests for DexManifest schema."""

    def test_creates_empty_manifest(self):
        """Creates an empty manifest."""
        manifest = DexManifest()
        assert manifest.version == "1.0"
        assert manifest.plugins == {}

    def test_add_file(self):
        """Records files by plugin."""
        manifest = DexManifest()
        manifest.add_file("test-plugin", ".claude/skills/test/SKILL.md")

        assert "test-plugin" in manifest.plugins
        assert ".claude/skills/test/SKILL.md" in manifest.plugins["test-plugin"].files

    def test_add_file_creates_plugin_entry(self):
        """Creates plugin entry if it doesn't exist."""
        manifest = DexManifest()
        manifest.add_file("new-plugin", "file.md")

        assert "new-plugin" in manifest.plugins
        assert isinstance(manifest.plugins["new-plugin"], PluginFiles)

    def test_add_file_no_duplicates(self):
        """Does not add duplicate files."""
        manifest = DexManifest()
        manifest.add_file("test-plugin", "file.md")
        manifest.add_file("test-plugin", "file.md")

        assert manifest.plugins["test-plugin"].files.count("file.md") == 1

    def test_add_directory(self):
        """Records directories by plugin."""
        manifest = DexManifest()
        manifest.add_directory("test-plugin", ".claude/skills/test")

        assert ".claude/skills/test" in manifest.plugins["test-plugin"].directories

    def test_add_mcp_server(self):
        """Records MCP servers by plugin."""
        manifest = DexManifest()
        manifest.add_mcp_server("test-plugin", "test-server")

        assert "test-server" in manifest.plugins["test-plugin"].mcp_servers

    def test_get_plugin_files(self):
        """Returns plugin files or None."""
        manifest = DexManifest()
        manifest.add_file("test-plugin", "file.md")

        result = manifest.get_plugin_files("test-plugin")
        assert result is not None
        assert "file.md" in result.files

        assert manifest.get_plugin_files("nonexistent") is None

    def test_remove_plugin(self):
        """Removes plugin and returns its files."""
        manifest = DexManifest()
        manifest.add_file("test-plugin", "file.md")
        manifest.add_mcp_server("test-plugin", "server")

        result = manifest.remove_plugin("test-plugin")

        assert result is not None
        assert "file.md" in result.files
        assert "test-plugin" not in manifest.plugins

    def test_remove_nonexistent_plugin(self):
        """Returns None when removing nonexistent plugin."""
        manifest = DexManifest()
        result = manifest.remove_plugin("nonexistent")
        assert result is None

    def test_get_mcp_servers_to_remove_returns_orphaned_servers(self):
        """Returns servers only used by the plugin being removed."""
        manifest = DexManifest()
        manifest.add_mcp_server("plugin-a", "shared-server")
        manifest.add_mcp_server("plugin-a", "exclusive-server")
        manifest.add_mcp_server("plugin-b", "shared-server")

        # exclusive-server should be removed, shared-server should not
        result = manifest.get_mcp_servers_to_remove("plugin-a")

        assert "exclusive-server" in result
        assert "shared-server" not in result

    def test_get_mcp_servers_to_remove_empty_for_all_shared(self):
        """Returns empty list when all servers are shared."""
        manifest = DexManifest()
        manifest.add_mcp_server("plugin-a", "shared-server")
        manifest.add_mcp_server("plugin-b", "shared-server")

        result = manifest.get_mcp_servers_to_remove("plugin-a")
        assert result == []

    def test_add_claude_settings_allow(self):
        """Records permission allow patterns by plugin."""
        manifest = DexManifest()
        manifest.add_claude_settings_allow("test-plugin", "mcp__serena")

        assert "test-plugin" in manifest.plugins
        assert "mcp__serena" in manifest.plugins["test-plugin"].claude_settings_allow

    def test_add_claude_settings_allow_creates_plugin_entry(self):
        """Creates plugin entry if it doesn't exist."""
        manifest = DexManifest()
        manifest.add_claude_settings_allow("new-plugin", "mcp__github")

        assert "new-plugin" in manifest.plugins
        assert isinstance(manifest.plugins["new-plugin"], PluginFiles)

    def test_add_claude_settings_allow_no_duplicates(self):
        """Does not add duplicate allow patterns."""
        manifest = DexManifest()
        manifest.add_claude_settings_allow("test-plugin", "mcp__serena")
        manifest.add_claude_settings_allow("test-plugin", "mcp__serena")

        assert manifest.plugins["test-plugin"].claude_settings_allow.count("mcp__serena") == 1

    def test_add_claude_settings_deny(self):
        """Records permission deny patterns by plugin."""
        manifest = DexManifest()
        manifest.add_claude_settings_deny("test-plugin", "Bash(curl:*)")

        assert "test-plugin" in manifest.plugins
        assert "Bash(curl:*)" in manifest.plugins["test-plugin"].claude_settings_deny

    def test_add_claude_settings_deny_no_duplicates(self):
        """Does not add duplicate deny patterns."""
        manifest = DexManifest()
        manifest.add_claude_settings_deny("test-plugin", "Bash(rm:*)")
        manifest.add_claude_settings_deny("test-plugin", "Bash(rm:*)")

        assert manifest.plugins["test-plugin"].claude_settings_deny.count("Bash(rm:*)") == 1

    def test_get_claude_settings_to_remove_returns_orphaned(self):
        """Returns settings only used by the plugin being removed."""
        manifest = DexManifest()
        manifest.add_claude_settings_allow("plugin-a", "mcp__shared")
        manifest.add_claude_settings_allow("plugin-a", "mcp__exclusive")
        manifest.add_claude_settings_allow("plugin-b", "mcp__shared")
        manifest.add_claude_settings_deny("plugin-a", "Bash(rm:*)")

        result = manifest.get_claude_settings_to_remove("plugin-a")

        # mcp__exclusive should be removed, mcp__shared should not
        assert "mcp__exclusive" in result["allow"]
        assert "mcp__shared" not in result["allow"]
        assert "Bash(rm:*)" in result["deny"]

    def test_get_claude_settings_to_remove_preserves_shared(self):
        """Preserves settings used by other plugins."""
        manifest = DexManifest()
        manifest.add_claude_settings_allow("plugin-a", "mcp__shared")
        manifest.add_claude_settings_allow("plugin-b", "mcp__shared")
        manifest.add_claude_settings_deny("plugin-a", "shared_deny")
        manifest.add_claude_settings_deny("plugin-b", "shared_deny")

        result = manifest.get_claude_settings_to_remove("plugin-a")

        assert result == {"allow": [], "deny": []}

    def test_get_claude_settings_to_remove_empty_for_nonexistent(self):
        """Returns empty lists for nonexistent plugin."""
        manifest = DexManifest()

        result = manifest.get_claude_settings_to_remove("nonexistent")

        assert result == {"allow": [], "deny": []}


class TestManifestManager:
    """Tests for ManifestManager."""

    @pytest.fixture
    def temp_project(self, tmp_path: Path) -> Path:
        """Create a temporary project directory."""
        return tmp_path

    def test_manifest_path(self, temp_project: Path):
        """Returns correct manifest path."""
        manager = ManifestManager(temp_project)
        assert manager.manifest_path == temp_project / ".dex" / "manifest.json"

    def test_load_creates_new_manifest(self, temp_project: Path):
        """Creates new manifest if none exists."""
        manager = ManifestManager(temp_project)
        manifest = manager.load()

        assert isinstance(manifest, DexManifest)
        assert manifest.version == "1.0"

    def test_load_reads_existing_manifest(self, temp_project: Path):
        """Loads existing manifest from disk."""
        # Create manifest file
        manifest_dir = temp_project / ".dex"
        manifest_dir.mkdir(parents=True)
        manifest_file = manifest_dir / "manifest.json"
        manifest_file.write_text(
            json.dumps(
                {
                    "version": "1.0",
                    "plugins": {
                        "test-plugin": {
                            "files": ["file1.md", "file2.md"],
                            "directories": [".claude/skills"],
                            "mcp_servers": ["test-server"],
                        }
                    },
                }
            )
        )

        manager = ManifestManager(temp_project)
        manifest = manager.load()

        assert "test-plugin" in manifest.plugins
        assert "file1.md" in manifest.plugins["test-plugin"].files

    def test_save_creates_manifest_file(self, temp_project: Path):
        """Saves manifest to disk."""
        manager = ManifestManager(temp_project)
        manager.load()
        manager.add_file("test-plugin", temp_project / "file.md")
        manager.save()

        assert manager.manifest_path.exists()

        # Verify contents
        data = json.loads(manager.manifest_path.read_text())
        assert "test-plugin" in data["plugins"]

    def test_add_file_converts_to_relative_path(self, temp_project: Path):
        """Converts absolute paths to relative paths."""
        manager = ManifestManager(temp_project)
        absolute_path = temp_project / ".claude" / "skills" / "SKILL.md"

        manager.add_file("test-plugin", absolute_path)

        files = manager.get_plugin_files("test-plugin")
        assert files is not None
        # Should be stored as relative path
        assert ".claude/skills/SKILL.md" in files.files

    def test_add_directory_converts_to_relative_path(self, temp_project: Path):
        """Converts absolute directory paths to relative."""
        manager = ManifestManager(temp_project)
        absolute_path = temp_project / ".claude" / "skills"

        manager.add_directory("test-plugin", absolute_path)

        files = manager.get_plugin_files("test-plugin")
        assert files is not None
        assert ".claude/skills" in files.directories

    def test_get_mcp_servers_to_remove(self, temp_project: Path):
        """Delegates to manifest's get_mcp_servers_to_remove."""
        manager = ManifestManager(temp_project)
        manager.add_mcp_server("plugin-a", "exclusive-server")
        manager.add_mcp_server("plugin-a", "shared-server")
        manager.add_mcp_server("plugin-b", "shared-server")

        result = manager.get_mcp_servers_to_remove("plugin-a")

        assert "exclusive-server" in result
        assert "shared-server" not in result

    def test_remove_plugin(self, temp_project: Path):
        """Removes plugin from manifest."""
        manager = ManifestManager(temp_project)
        manager.add_file("test-plugin", temp_project / "file.md")

        result = manager.remove_plugin("test-plugin")

        assert result is not None
        assert manager.get_plugin_files("test-plugin") is None

    def test_load_caches_manifest(self, temp_project: Path):
        """Caches loaded manifest to avoid repeated disk reads."""
        manager = ManifestManager(temp_project)

        manifest1 = manager.load()
        manifest2 = manager.load()

        # Should be the same object (cached)
        assert manifest1 is manifest2

    def test_add_claude_settings_allow(self, temp_project: Path):
        """Records a permission allow pattern."""
        manager = ManifestManager(temp_project)
        manager.add_claude_settings_allow("test-plugin", "mcp__serena")

        files = manager.get_plugin_files("test-plugin")
        assert files is not None
        assert "mcp__serena" in files.claude_settings_allow

    def test_add_claude_settings_deny(self, temp_project: Path):
        """Records a permission deny pattern."""
        manager = ManifestManager(temp_project)
        manager.add_claude_settings_deny("test-plugin", "Bash(rm:*)")

        files = manager.get_plugin_files("test-plugin")
        assert files is not None
        assert "Bash(rm:*)" in files.claude_settings_deny

    def test_get_claude_settings_to_remove(self, temp_project: Path):
        """Delegates to manifest's get_claude_settings_to_remove."""
        manager = ManifestManager(temp_project)
        manager.add_claude_settings_allow("plugin-a", "mcp__exclusive")
        manager.add_claude_settings_allow("plugin-a", "mcp__shared")
        manager.add_claude_settings_allow("plugin-b", "mcp__shared")

        result = manager.get_claude_settings_to_remove("plugin-a")

        assert "mcp__exclusive" in result["allow"]
        assert "mcp__shared" not in result["allow"]
