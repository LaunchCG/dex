"""Tests for dex.core.project module."""

from pathlib import Path

import pytest
import yaml

from dex.config.schemas import PluginSpec, ProjectConfig
from dex.core.project import Project


class TestProjectLoad:
    """Tests for Project.load() class method."""

    def test_loads_existing_project(self, temp_dir: Path):
        """Loads an existing project."""
        config_content = """\
agent: claude-code
project_name: test-project
plugins: {}
"""
        (temp_dir / "dex.yaml").write_text(config_content)

        project = Project.load(temp_dir)

        assert project.agent == "claude-code"
        assert project.project_name == "test-project"

    def test_raises_for_missing_config(self, temp_dir: Path):
        """Raises FileNotFoundError for missing dex.yaml."""
        with pytest.raises(FileNotFoundError, match="No dex.yaml found"):
            Project.load(temp_dir)

    def test_loads_from_cwd_when_none(self, temp_dir: Path, monkeypatch):
        """Loads from current directory when path is None."""
        (temp_dir / "dex.yaml").write_text("agent: claude-code")
        monkeypatch.chdir(temp_dir)

        project = Project.load(None)

        assert project.agent == "claude-code"

    def test_searches_parent_directories(self, temp_dir: Path, monkeypatch):
        """Searches parent directories for dex.yaml."""
        (temp_dir / "dex.yaml").write_text("agent: claude-code")
        subdir = temp_dir / "src" / "deep"
        subdir.mkdir(parents=True)
        monkeypatch.chdir(subdir)

        project = Project.load(None)

        # Use resolve() to handle macOS /var -> /private/var symlink
        assert project.root.resolve() == temp_dir.resolve()


class TestProjectInit:
    """Tests for Project.init() class method."""

    def test_creates_new_project(self, temp_dir: Path):
        """Creates a new project."""
        project = Project.init(temp_dir, "claude-code", project_name="my-project")

        assert project.agent == "claude-code"
        assert project.project_name == "my-project"
        assert (temp_dir / "dex.yaml").exists()

    def test_uses_directory_name_as_default(self, temp_dir: Path):
        """Uses directory name when project_name not specified."""
        project = Project.init(temp_dir, "claude-code")

        assert project.project_name == temp_dir.name

    def test_raises_for_existing_project(self, temp_dir: Path):
        """Raises FileExistsError for existing project."""
        (temp_dir / "dex.yaml").write_text("agent: claude-code")

        with pytest.raises(FileExistsError, match="already initialized"):
            Project.init(temp_dir, "claude-code")


class TestProjectSave:
    """Tests for Project.save() method."""

    def test_saves_configuration(self, temp_dir: Path):
        """Saves configuration to disk."""
        project = Project.init(temp_dir, "claude-code")
        project.add_plugin("test-plugin", "^1.0.0")

        project.save()

        # Reload and verify
        saved_data = yaml.safe_load((temp_dir / "dex.yaml").read_text())
        assert "test-plugin" in saved_data["plugins"]


class TestProjectProperties:
    """Tests for Project properties."""

    def test_root_property(self, initialized_project: Project, temp_project: Path):
        """root property returns project root."""
        # Use resolve() to handle macOS /var -> /private/var symlink
        assert initialized_project.root.resolve() == temp_project.resolve()

    def test_agent_property(self, initialized_project: Project):
        """agent property returns agent type."""
        assert initialized_project.agent == "claude-code"

    def test_project_name_property(self, initialized_project: Project):
        """project_name property returns project name."""
        assert initialized_project.project_name == "test-project"

    def test_config_property(self, initialized_project: Project):
        """config property returns ProjectConfig."""
        assert isinstance(initialized_project.config, ProjectConfig)

    def test_plugins_property(self, initialized_project: Project):
        """plugins property returns plugins dict."""
        assert isinstance(initialized_project.plugins, dict)


class TestProjectAddPlugin:
    """Tests for Project.add_plugin() method."""

    def test_adds_plugin_with_version(self, initialized_project: Project):
        """Adds plugin with version string."""
        initialized_project.add_plugin("test-plugin", "^1.0.0")

        assert "test-plugin" in initialized_project.plugins
        assert initialized_project.plugins["test-plugin"] == "^1.0.0"

    def test_adds_plugin_with_spec(self, initialized_project: Project):
        """Adds plugin with PluginSpec."""
        spec = PluginSpec(source="file:./local-plugin")
        initialized_project.add_plugin("local-plugin", spec)

        assert "local-plugin" in initialized_project.plugins

    def test_updates_existing_plugin(self, initialized_project: Project):
        """Updates existing plugin specification."""
        initialized_project.add_plugin("test-plugin", "^1.0.0")
        initialized_project.add_plugin("test-plugin", "^2.0.0")

        assert initialized_project.plugins["test-plugin"] == "^2.0.0"


class TestProjectRemovePlugin:
    """Tests for Project.remove_plugin() method."""

    def test_removes_plugin(self, initialized_project: Project):
        """Removes an existing plugin."""
        initialized_project.add_plugin("test-plugin", "^1.0.0")

        result = initialized_project.remove_plugin("test-plugin")

        assert result is True
        assert "test-plugin" not in initialized_project.plugins

    def test_returns_false_for_nonexistent(self, initialized_project: Project):
        """Returns False for nonexistent plugin."""
        result = initialized_project.remove_plugin("nonexistent")

        assert result is False


class TestProjectGetPluginSpec:
    """Tests for Project.get_plugin_spec() method."""

    def test_gets_spec_from_version_string(self, initialized_project: Project):
        """Gets PluginSpec from version string."""
        initialized_project.add_plugin("test-plugin", "^1.0.0")

        spec = initialized_project.get_plugin_spec("test-plugin")

        assert spec is not None
        assert spec.version == "^1.0.0"

    def test_gets_spec_from_plugin_spec(self, initialized_project: Project):
        """Gets PluginSpec when plugin already has PluginSpec."""
        original_spec = PluginSpec(source="file:./local")
        initialized_project.add_plugin("local-plugin", original_spec)

        spec = initialized_project.get_plugin_spec("local-plugin")

        assert spec is original_spec

    def test_returns_none_for_nonexistent(self, initialized_project: Project):
        """Returns None for nonexistent plugin."""
        spec = initialized_project.get_plugin_spec("nonexistent")

        assert spec is None


class TestProjectGetRegistryUrl:
    """Tests for Project.get_registry_url() method."""

    def test_gets_named_registry(self, initialized_project: Project):
        """Gets URL for named registry."""
        initialized_project._config.registries["local"] = "file:./registry"

        url = initialized_project.get_registry_url("local")

        assert url == "file:./registry"

    def test_gets_default_registry(self, initialized_project: Project):
        """Gets URL for default registry."""
        initialized_project._config.registries["local"] = "file:./registry"
        initialized_project._config.default_registry = "local"

        url = initialized_project.get_registry_url(None)

        assert url == "file:./registry"

    def test_returns_none_for_unknown_registry(self, initialized_project: Project):
        """Returns None for unknown registry."""
        url = initialized_project.get_registry_url("nonexistent")

        assert url is None


class TestProjectRepr:
    """Tests for Project.__repr__() method."""

    def test_repr(self, initialized_project: Project, temp_project: Path):
        """Returns useful repr string."""
        repr_str = repr(initialized_project)

        assert "Project" in repr_str
        assert str(temp_project) in repr_str
        assert "claude-code" in repr_str
