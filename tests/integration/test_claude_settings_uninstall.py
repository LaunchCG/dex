"""Integration tests for Claude Code settings uninstall behavior.

Tests that shared permissions are preserved when uninstalling plugins,
and orphaned permissions are removed.
"""

import json
from pathlib import Path

import pytest

from dex.adapters import get_adapter
from dex.config.schemas import (
    ClaudeSettingsConfig,
    PluginManifest,
    PluginSpec,
    SkillConfig,
)
from dex.core.installer import PluginInstaller
from dex.core.manifest import ManifestManager
from dex.core.project import Project


@pytest.fixture
def project_with_plugins(tmp_path: Path) -> tuple[Project, Path, Path]:
    """Create a project and two plugin directories with claude_settings."""
    # Initialize project
    project = Project.init(tmp_path, "claude-code")

    # Create plugin-a with unique and shared permissions
    plugin_a_dir = tmp_path / "plugins" / "plugin-a"
    plugin_a_dir.mkdir(parents=True)
    (plugin_a_dir / "package.json").write_text(
        json.dumps(
            {
                "name": "plugin-a",
                "version": "1.0.0",
                "description": "Test plugin A",
                "skills": [
                    {
                        "name": "skill-a",
                        "description": "A skill",
                        "context": "./context.md",
                    }
                ],
                "claude_settings": {
                    "allow": ["mcp__serena", "mcp__shared"],
                    "deny": ["Bash(rm:*)"],
                },
            }
        )
    )
    (plugin_a_dir / "context.md").write_text("# Skill A")

    # Create plugin-b with shared permissions
    plugin_b_dir = tmp_path / "plugins" / "plugin-b"
    plugin_b_dir.mkdir(parents=True)
    (plugin_b_dir / "package.json").write_text(
        json.dumps(
            {
                "name": "plugin-b",
                "version": "1.0.0",
                "description": "Test plugin B",
                "skills": [
                    {
                        "name": "skill-b",
                        "description": "A skill",
                        "context": "./context.md",
                    }
                ],
                "claude_settings": {
                    "allow": ["mcp__github", "mcp__shared"],
                },
            }
        )
    )
    (plugin_b_dir / "context.md").write_text("# Skill B")

    return project, plugin_a_dir, plugin_b_dir


class TestClaudeSettingsUninstall:
    """Tests for Claude settings cleanup during uninstall."""

    def test_uninstall_preserves_shared_permissions(
        self, project_with_plugins: tuple[Project, Path, Path]
    ):
        """Uninstalling a plugin preserves permissions used by other plugins."""
        project, plugin_a_dir, plugin_b_dir = project_with_plugins

        # Install both plugins
        installer = PluginInstaller(project)
        installer.install(
            {
                "plugin-a": PluginSpec(source=f"file:{plugin_a_dir}"),
                "plugin-b": PluginSpec(source=f"file:{plugin_b_dir}"),
            },
            use_lockfile=False,
            update_lockfile=False,
        )

        # Verify settings were written
        settings_path = project.root / ".claude" / "settings.json"
        assert settings_path.exists()
        settings_data = json.loads(settings_path.read_text())
        assert "mcp__serena" in settings_data["permissions"]["allow"]
        assert "mcp__github" in settings_data["permissions"]["allow"]
        assert "mcp__shared" in settings_data["permissions"]["allow"]

        # Get manifest manager for uninstall operations
        adapter = get_adapter(project.agent)
        manifest_manager = ManifestManager(project.root)

        # Get claude_settings to remove for plugin-a
        claude_settings_to_remove = manifest_manager.get_claude_settings_to_remove("plugin-a")

        # Verify that shared permission is NOT in the removal list
        assert "mcp__shared" not in claude_settings_to_remove["allow"]
        # Verify that unique permissions ARE in the removal list
        assert "mcp__serena" in claude_settings_to_remove["allow"]
        assert "Bash(rm:*)" in claude_settings_to_remove["deny"]

    def test_uninstall_removes_orphaned_permissions(
        self, project_with_plugins: tuple[Project, Path, Path]
    ):
        """Uninstalling a plugin removes permissions only used by that plugin."""
        project, plugin_a_dir, plugin_b_dir = project_with_plugins

        # Install both plugins
        installer = PluginInstaller(project)
        installer.install(
            {
                "plugin-a": PluginSpec(source=f"file:{plugin_a_dir}"),
                "plugin-b": PluginSpec(source=f"file:{plugin_b_dir}"),
            },
            use_lockfile=False,
            update_lockfile=False,
        )

        # Verify initial state
        settings_path = project.root / ".claude" / "settings.json"
        settings_data = json.loads(settings_path.read_text())
        assert "mcp__serena" in settings_data["permissions"]["allow"]
        assert "Bash(rm:*)" in settings_data["permissions"]["deny"]

        # Simulate uninstall of plugin-a (doing the settings cleanup part)
        adapter = get_adapter(project.agent)
        manifest_manager = ManifestManager(project.root)

        claude_settings_to_remove = manifest_manager.get_claude_settings_to_remove("plugin-a")

        # Remove permissions from settings file
        if any(claude_settings_to_remove.values()):
            for pattern in claude_settings_to_remove.get("allow", []):
                if pattern in settings_data["permissions"].get("allow", []):
                    settings_data["permissions"]["allow"].remove(pattern)
            for pattern in claude_settings_to_remove.get("deny", []):
                if pattern in settings_data["permissions"].get("deny", []):
                    settings_data["permissions"]["deny"].remove(pattern)
            settings_path.write_text(json.dumps(settings_data, indent=2) + "\n")

        # Verify permissions were removed correctly
        settings_data = json.loads(settings_path.read_text())

        # mcp__serena should be removed (only used by plugin-a)
        assert "mcp__serena" not in settings_data["permissions"]["allow"]

        # Bash(rm:*) should be removed (only used by plugin-a)
        deny_list = settings_data["permissions"].get("deny", [])
        assert "Bash(rm:*)" not in deny_list

        # mcp__shared should be preserved (used by both plugins)
        assert "mcp__shared" in settings_data["permissions"]["allow"]

        # mcp__github should be preserved (plugin-b still installed)
        assert "mcp__github" in settings_data["permissions"]["allow"]

    def test_uninstall_all_plugins_removes_all_permissions(
        self, project_with_plugins: tuple[Project, Path, Path]
    ):
        """Uninstalling all plugins removes all plugin-contributed permissions."""
        project, plugin_a_dir, plugin_b_dir = project_with_plugins

        # Install both plugins
        installer = PluginInstaller(project)
        installer.install(
            {
                "plugin-a": PluginSpec(source=f"file:{plugin_a_dir}"),
                "plugin-b": PluginSpec(source=f"file:{plugin_b_dir}"),
            },
            use_lockfile=False,
            update_lockfile=False,
        )

        settings_path = project.root / ".claude" / "settings.json"
        adapter = get_adapter(project.agent)
        manifest_manager = ManifestManager(project.root)

        # Uninstall plugin-a first
        claude_settings_to_remove = manifest_manager.get_claude_settings_to_remove("plugin-a")
        settings_data = json.loads(settings_path.read_text())
        for pattern in claude_settings_to_remove.get("allow", []):
            if pattern in settings_data["permissions"].get("allow", []):
                settings_data["permissions"]["allow"].remove(pattern)
        for pattern in claude_settings_to_remove.get("deny", []):
            if pattern in settings_data["permissions"].get("deny", []):
                settings_data["permissions"]["deny"].remove(pattern)
        settings_path.write_text(json.dumps(settings_data, indent=2) + "\n")
        manifest_manager.remove_plugin("plugin-a")
        manifest_manager.save()

        # Reload manifest
        manifest_manager = ManifestManager(project.root)

        # Uninstall plugin-b
        claude_settings_to_remove = manifest_manager.get_claude_settings_to_remove("plugin-b")
        settings_data = json.loads(settings_path.read_text())
        for pattern in claude_settings_to_remove.get("allow", []):
            if pattern in settings_data["permissions"].get("allow", []):
                settings_data["permissions"]["allow"].remove(pattern)
        for pattern in claude_settings_to_remove.get("deny", []):
            if pattern in settings_data["permissions"].get("deny", []):
                settings_data["permissions"]["deny"].remove(pattern)
        settings_path.write_text(json.dumps(settings_data, indent=2) + "\n")

        # Verify all plugin permissions are removed
        settings_data = json.loads(settings_path.read_text())

        # All allow permissions should be removed
        allow_list = settings_data["permissions"].get("allow", [])
        assert "mcp__serena" not in allow_list
        assert "mcp__github" not in allow_list
        assert "mcp__shared" not in allow_list

        # All deny permissions should be removed
        deny_list = settings_data["permissions"].get("deny", [])
        assert "Bash(rm:*)" not in deny_list


class TestClaudeSettingsInstall:
    """Tests for Claude settings installation behavior."""

    def test_install_creates_settings_with_permissions(
        self, project_with_plugins: tuple[Project, Path, Path]
    ):
        """Installing a plugin creates settings.json with permissions."""
        project, plugin_a_dir, _ = project_with_plugins

        installer = PluginInstaller(project)
        installer.install(
            {"plugin-a": PluginSpec(source=f"file:{plugin_a_dir}")},
            use_lockfile=False,
            update_lockfile=False,
        )

        settings_path = project.root / ".claude" / "settings.json"
        assert settings_path.exists()

        settings_data = json.loads(settings_path.read_text())
        assert "permissions" in settings_data
        assert "mcp__serena" in settings_data["permissions"]["allow"]
        assert "mcp__shared" in settings_data["permissions"]["allow"]
        assert "Bash(rm:*)" in settings_data["permissions"]["deny"]

    def test_install_merges_permissions_from_multiple_plugins(
        self, project_with_plugins: tuple[Project, Path, Path]
    ):
        """Installing multiple plugins merges their permissions."""
        project, plugin_a_dir, plugin_b_dir = project_with_plugins

        installer = PluginInstaller(project)
        installer.install(
            {
                "plugin-a": PluginSpec(source=f"file:{plugin_a_dir}"),
                "plugin-b": PluginSpec(source=f"file:{plugin_b_dir}"),
            },
            use_lockfile=False,
            update_lockfile=False,
        )

        settings_path = project.root / ".claude" / "settings.json"
        settings_data = json.loads(settings_path.read_text())

        # Should have permissions from both plugins
        assert "mcp__serena" in settings_data["permissions"]["allow"]
        assert "mcp__github" in settings_data["permissions"]["allow"]
        assert "mcp__shared" in settings_data["permissions"]["allow"]

        # Shared permission should only appear once (de-duplicated)
        assert settings_data["permissions"]["allow"].count("mcp__shared") == 1

    def test_install_deduplicates_permissions(
        self, project_with_plugins: tuple[Project, Path, Path]
    ):
        """Installing plugins with shared permissions de-duplicates them."""
        project, plugin_a_dir, plugin_b_dir = project_with_plugins

        # Install plugin-a first
        installer = PluginInstaller(project)
        installer.install(
            {"plugin-a": PluginSpec(source=f"file:{plugin_a_dir}")},
            use_lockfile=False,
            update_lockfile=False,
        )

        # Then install plugin-b
        installer.install(
            {"plugin-b": PluginSpec(source=f"file:{plugin_b_dir}")},
            use_lockfile=False,
            update_lockfile=False,
        )

        settings_path = project.root / ".claude" / "settings.json"
        settings_data = json.loads(settings_path.read_text())

        # Shared permission should only appear once
        assert settings_data["permissions"]["allow"].count("mcp__shared") == 1
