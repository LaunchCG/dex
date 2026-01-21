"""End-to-end installation integration tests."""

import json
import tarfile
from pathlib import Path

import pytest

from dex.config.schemas import PluginSpec
from dex.core.installer import PluginInstaller
from dex.core.lockfile import LockFileManager
from dex.core.project import Project


@pytest.fixture
def full_test_plugin(temp_dir: Path) -> Path:
    """Create a complete test plugin with all components."""
    plugin_dir = temp_dir / "full-plugin"
    plugin_dir.mkdir()

    # Create package.json
    manifest = {
        "name": "full-plugin",
        "version": "1.0.0",
        "description": "A complete test plugin",
        "skills": [
            {
                "name": "test-skill",
                "description": "A test skill for installation tests",
                "context": "./context/skill.md",
                "files": [{"src": "files/config.json"}],
            }
        ],
        "commands": [
            {
                "name": "test-command",
                "description": "A test command",
                "context": "./context/command.md",
            }
        ],
    }
    (plugin_dir / "package.json").write_text(json.dumps(manifest))

    # Create context files
    context_dir = plugin_dir / "context"
    context_dir.mkdir()
    (context_dir / "skill.md").write_text(
        "# {{ plugin.name }} Skill\n\nProject: {{ env.project.name }}\nPlatform: {{ platform.os }}"
    )
    (context_dir / "command.md").write_text("# Test Command\n\nThis is a command.")

    # Create associated files
    files_dir = plugin_dir / "files"
    files_dir.mkdir()
    (files_dir / "config.json").write_text('{"setting": "value"}')

    return plugin_dir


@pytest.fixture
def test_registry(temp_dir: Path, full_test_plugin: Path) -> Path:
    """Create a test registry with the full plugin."""
    registry_dir = temp_dir / "registry"
    registry_dir.mkdir()

    # Create registry.json
    registry_data = {
        "packages": {
            "full-plugin": {
                "versions": ["1.0.0"],
                "latest": "1.0.0",
            }
        }
    }
    (registry_dir / "registry.json").write_text(json.dumps(registry_data))

    # Create tarball
    tarball_path = registry_dir / "full-plugin-1.0.0.tar.gz"
    with tarfile.open(tarball_path, "w:gz") as tar:
        tar.add(full_test_plugin, arcname="full-plugin")

    return registry_dir


class TestInstallSimplePlugin:
    """Tests for simple plugin installation."""

    def test_install_from_direct_source(self, temp_project: Path, full_test_plugin: Path):
        """Install a plugin from direct source."""
        # Initialize project
        project = Project.init(temp_project, "claude-code", "test-project")
        project._config.registries["local"] = f"file://{full_test_plugin.parent}"

        # Install plugin
        installer = PluginInstaller(project)
        summary = installer.install(
            {"full-plugin": PluginSpec(source=f"file:{full_test_plugin}")},
            use_lockfile=False,
        )

        # Verify installation
        assert summary.all_successful
        assert len(summary.results) == 1
        assert summary.results[0].plugin_name == "full-plugin"

        # Verify skill file created
        skill_path = temp_project / ".claude" / "skills" / "full-plugin-test-skill" / "SKILL.md"
        assert skill_path.exists()

        # Verify content was rendered
        content = skill_path.read_text()
        assert "full-plugin" in content
        assert "test-project" in content

    def test_install_updates_lockfile(self, temp_project: Path, full_test_plugin: Path):
        """Installation updates the lock file."""
        project = Project.init(temp_project, "claude-code")

        installer = PluginInstaller(project)
        installer.install(
            {"full-plugin": PluginSpec(source=f"file:{full_test_plugin}")},
            update_lockfile=True,
        )

        # Verify lock file
        lock_manager = LockFileManager(temp_project, "claude-code")
        lock_manager.load()

        assert lock_manager.is_locked("full-plugin")
        locked = lock_manager.get_locked_plugin("full-plugin")
        assert locked is not None
        assert locked.version == "1.0.0"


class TestInstallPluginWithFiles:
    """Tests for plugin installation with associated files."""

    def test_copies_associated_files(self, temp_project: Path, full_test_plugin: Path):
        """Associated files are copied to skill directory."""
        project = Project.init(temp_project, "claude-code")

        installer = PluginInstaller(project)
        installer.install(
            {"full-plugin": PluginSpec(source=f"file:{full_test_plugin}")},
            use_lockfile=False,
        )

        # Verify config file was copied (dest defaults to basename)
        config_path = temp_project / ".claude" / "skills" / "full-plugin-test-skill" / "config.json"
        assert config_path.exists()
        assert json.loads(config_path.read_text())["setting"] == "value"


class TestInstallFromRegistry:
    """Tests for installation from a registry."""

    def test_install_from_registry_tarball(self, temp_project: Path, test_registry: Path):
        """Install a plugin from registry tarball."""
        project = Project.init(temp_project, "claude-code", "test-project")
        project._config.registries["local"] = f"file://{test_registry}"
        project._config.default_registry = "local"
        project.save()

        installer = PluginInstaller(project)
        summary = installer.install(
            {"full-plugin": PluginSpec(version="^1.0.0")},
            use_lockfile=False,
        )

        assert summary.all_successful

        # Verify skill was installed
        skill_path = temp_project / ".claude" / "skills" / "full-plugin-test-skill" / "SKILL.md"
        assert skill_path.exists()


class TestInstallFromLockfile:
    """Tests for installation using lock file."""

    def test_uses_locked_versions(self, temp_project: Path, full_test_plugin: Path):
        """Second install uses locked versions."""
        project = Project.init(temp_project, "claude-code")

        # First install
        installer = PluginInstaller(project)
        installer.install(
            {"full-plugin": PluginSpec(source=f"file:{full_test_plugin}")},
            update_lockfile=True,
        )

        # Verify lock was created
        lock_manager = LockFileManager(temp_project, "claude-code")
        lock_manager.load()
        assert lock_manager.is_locked("full-plugin")

        # Second install should use lockfile
        installer2 = PluginInstaller(project)
        summary = installer2.install(
            {"full-plugin": PluginSpec(source=f"file:{full_test_plugin}")},
            use_lockfile=True,
        )

        assert summary.all_successful


class TestInstallMultiplePlugins:
    """Tests for installing multiple plugins."""

    def test_install_multiple(self, temp_project: Path, temp_dir: Path):
        """Install multiple plugins in one operation."""
        # Create two plugins
        for i in range(2):
            plugin_dir = temp_dir / f"plugin-{i}"
            plugin_dir.mkdir()
            manifest = {
                "name": f"plugin-{i}",
                "version": "1.0.0",
                "description": f"Plugin {i}",
                "skills": [
                    {
                        "name": f"skill-{i}",
                        "description": f"Skill {i}",
                        "context": "./context.md",
                    }
                ],
            }
            (plugin_dir / "package.json").write_text(json.dumps(manifest))
            context_dir = plugin_dir / "context.md"
            context_dir.write_text(f"# Skill {i}")

        project = Project.init(temp_project, "claude-code")
        installer = PluginInstaller(project)

        summary = installer.install(
            {
                "plugin-0": PluginSpec(source=f"file:{temp_dir / 'plugin-0'}"),
                "plugin-1": PluginSpec(source=f"file:{temp_dir / 'plugin-1'}"),
            },
            use_lockfile=False,
        )

        assert summary.all_successful
        assert summary.success_count == 2

        # Verify both installed
        for i in range(2):
            skill_path = temp_project / ".claude" / "skills" / f"plugin-{i}-skill-{i}" / "SKILL.md"
            assert skill_path.exists()


class TestTemplateRendering:
    """Tests for template rendering during installation."""

    def test_renders_platform_variables(self, temp_project: Path, temp_dir: Path):
        """Platform variables are rendered in templates."""
        # Create plugin with platform conditionals
        plugin_dir = temp_dir / "platform-plugin"
        plugin_dir.mkdir()
        manifest = {
            "name": "platform-plugin",
            "version": "1.0.0",
            "description": "Test",
            "skills": [
                {
                    "name": "platform-skill",
                    "description": "Platform-aware skill",
                    "context": "./context.md",
                }
            ],
        }
        (plugin_dir / "package.json").write_text(json.dumps(manifest))
        (plugin_dir / "context.md").write_text(
            "OS: {{ platform.os }}\n"
            "Arch: {{ platform.arch }}\n"
            "{% if platform.os is unix %}Unix system{% endif %}"
        )

        project = Project.init(temp_project, "claude-code")
        installer = PluginInstaller(project)
        installer.install(
            {"platform-plugin": PluginSpec(source=f"file:{plugin_dir}")},
            use_lockfile=False,
        )

        skill_path = (
            temp_project / ".claude" / "skills" / "platform-plugin-platform-skill" / "SKILL.md"
        )
        content = skill_path.read_text()

        # Should have OS value (linux, macos, or windows)
        assert "OS: " in content
        # Should have arch value
        assert "Arch: " in content
