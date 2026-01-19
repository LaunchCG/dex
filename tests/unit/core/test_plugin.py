"""Tests for dex.core.plugin module."""

import json
from pathlib import Path

import pytest

from dex.core.plugin import Plugin


class TestPluginLoad:
    """Tests for Plugin.load() class method."""

    def test_loads_plugin(self, temp_plugin_dir: Path):
        """Loads a plugin from directory."""
        plugin = Plugin.load(temp_plugin_dir)

        assert plugin.name == "test-plugin"
        assert plugin.version == "1.0.0"

    def test_raises_for_missing_manifest(self, temp_dir: Path):
        """Raises FileNotFoundError for missing package.json."""
        with pytest.raises(FileNotFoundError, match="No package.json found"):
            Plugin.load(temp_dir)


class TestPluginProperties:
    """Tests for Plugin properties."""

    def test_path_property(self, temp_plugin_dir: Path):
        """path property returns plugin directory."""
        plugin = Plugin.load(temp_plugin_dir)
        # Use resolve() to handle macOS /var -> /private/var symlink
        assert plugin.path.resolve() == temp_plugin_dir.resolve()

    def test_manifest_property(self, temp_plugin_dir: Path):
        """manifest property returns PluginManifest."""
        plugin = Plugin.load(temp_plugin_dir)
        assert plugin.manifest.name == "test-plugin"

    def test_name_property(self, temp_plugin_dir: Path):
        """name property returns plugin name."""
        plugin = Plugin.load(temp_plugin_dir)
        assert plugin.name == "test-plugin"

    def test_version_property(self, temp_plugin_dir: Path):
        """version property returns plugin version."""
        plugin = Plugin.load(temp_plugin_dir)
        assert plugin.version == "1.0.0"

    def test_description_property(self, temp_dir: Path):
        """description property returns plugin description."""
        manifest = {
            "name": "test-plugin",
            "version": "1.0.0",
            "description": "A test plugin",
        }
        (temp_dir / "package.json").write_text(json.dumps(manifest))

        plugin = Plugin.load(temp_dir)
        assert plugin.description == "A test plugin"

    def test_skills_property(self, temp_dir: Path):
        """skills property returns skill list."""
        manifest = {
            "name": "test-plugin",
            "version": "1.0.0",
            "description": "Test",
            "skills": [
                {"name": "skill1", "description": "Skill 1", "context": "./s1.md"},
                {"name": "skill2", "description": "Skill 2", "context": "./s2.md"},
            ],
        }
        (temp_dir / "package.json").write_text(json.dumps(manifest))

        plugin = Plugin.load(temp_dir)
        assert len(plugin.skills) == 2
        assert plugin.skills[0].name == "skill1"

    def test_commands_property(self, temp_dir: Path):
        """commands property returns command list."""
        manifest = {
            "name": "test-plugin",
            "version": "1.0.0",
            "description": "Test",
            "commands": [{"name": "cmd1", "description": "Command 1", "context": "./c1.md"}],
        }
        (temp_dir / "package.json").write_text(json.dumps(manifest))

        plugin = Plugin.load(temp_dir)
        assert len(plugin.commands) == 1

    def test_sub_agents_property(self, temp_dir: Path):
        """sub_agents property returns sub-agent list."""
        manifest = {
            "name": "test-plugin",
            "version": "1.0.0",
            "description": "Test",
            "sub_agents": [{"name": "agent1", "description": "Agent 1", "context": "./a1.md"}],
        }
        (temp_dir / "package.json").write_text(json.dumps(manifest))

        plugin = Plugin.load(temp_dir)
        assert len(plugin.sub_agents) == 1

    def test_mcp_servers_property(self, temp_dir: Path):
        """mcp_servers property returns MCP server list."""
        manifest = {
            "name": "test-plugin",
            "version": "1.0.0",
            "description": "Test",
            "mcp_servers": [{"name": "server1", "type": "bundled", "path": "./server.js"}],
        }
        (temp_dir / "package.json").write_text(json.dumps(manifest))

        plugin = Plugin.load(temp_dir)
        assert len(plugin.mcp_servers) == 1

    def test_dependencies_property(self, temp_dir: Path):
        """dependencies property returns dependencies dict."""
        manifest = {
            "name": "test-plugin",
            "version": "1.0.0",
            "description": "Test",
            "dependencies": {"dep1": "^1.0.0"},
        }
        (temp_dir / "package.json").write_text(json.dumps(manifest))

        plugin = Plugin.load(temp_dir)
        assert plugin.dependencies["dep1"] == "^1.0.0"


class TestPluginResolveContextPath:
    """Tests for Plugin.resolve_context_path() method."""

    def test_resolves_relative_path(self, temp_plugin_dir: Path):
        """Resolves relative context path."""
        plugin = Plugin.load(temp_plugin_dir)

        path = plugin.resolve_context_path("./context/skill.md")

        # Use resolve() to handle macOS /var -> /private/var symlink
        assert path.resolve() == (temp_plugin_dir / "context" / "skill.md").resolve()

    def test_resolves_path_without_dot_slash(self, temp_plugin_dir: Path):
        """Resolves path without leading ./"""
        plugin = Plugin.load(temp_plugin_dir)

        path = plugin.resolve_context_path("context/skill.md")

        # Use resolve() to handle macOS /var -> /private/var symlink
        assert path.resolve() == (temp_plugin_dir / "context" / "skill.md").resolve()


class TestPluginResolveFilePath:
    """Tests for Plugin.resolve_file_path() method."""

    def test_resolves_file_path(self, temp_plugin_dir: Path):
        """Resolves file path relative to plugin."""
        plugin = Plugin.load(temp_plugin_dir)

        path = plugin.resolve_file_path("./config/settings.json")

        # Use resolve() to handle macOS /var -> /private/var symlink
        assert path.resolve() == (temp_plugin_dir / "config" / "settings.json").resolve()


class TestPluginHasComponent:
    """Tests for Plugin.has_component() method."""

    def test_finds_existing_skill(self, temp_plugin_dir: Path):
        """Finds existing skill."""
        plugin = Plugin.load(temp_plugin_dir)

        assert plugin.has_component("skill", "test-skill") is True

    def test_not_finds_nonexistent_skill(self, temp_plugin_dir: Path):
        """Doesn't find nonexistent skill."""
        plugin = Plugin.load(temp_plugin_dir)

        assert plugin.has_component("skill", "nonexistent") is False

    def test_handles_different_component_types(self, temp_dir: Path):
        """Handles different component types."""
        manifest = {
            "name": "test-plugin",
            "version": "1.0.0",
            "description": "Test",
            "skills": [{"name": "skill1", "description": "Skill 1", "context": "./s.md"}],
            "commands": [{"name": "cmd1", "description": "Command 1", "context": "./c.md"}],
            "sub_agents": [{"name": "agent1", "description": "Agent 1", "context": "./a.md"}],
            "mcp_servers": [{"name": "server1", "type": "bundled", "path": "./s.js"}],
        }
        (temp_dir / "package.json").write_text(json.dumps(manifest))

        plugin = Plugin.load(temp_dir)

        assert plugin.has_component("skill", "skill1") is True
        assert plugin.has_component("command", "cmd1") is True
        assert plugin.has_component("sub_agent", "agent1") is True
        assert plugin.has_component("mcp_server", "server1") is True


class TestPluginRepr:
    """Tests for Plugin.__repr__() method."""

    def test_repr(self, temp_plugin_dir: Path):
        """Returns useful repr string."""
        plugin = Plugin.load(temp_plugin_dir)

        repr_str = repr(plugin)

        assert "Plugin" in repr_str
        assert "test-plugin" in repr_str
        assert "1.0.0" in repr_str
