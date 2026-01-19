"""Tests for dex.config.parser module."""

import json
from pathlib import Path

import pytest
import yaml

from dex.config.parser import (
    ConfigError,
    find_project_root,
    load_json,
    load_lockfile,
    load_plugin_manifest,
    load_project_config,
    load_yaml,
    save_json,
    save_lockfile,
    save_project_config,
    save_yaml,
)
from dex.config.schemas import LockFile, PluginManifest, ProjectConfig


class TestLoadJson:
    """Tests for load_json function."""

    def test_loads_valid_json(self, temp_dir: Path):
        """Loads valid JSON file."""
        file_path = temp_dir / "test.json"
        file_path.write_text('{"key": "value"}')

        result = load_json(file_path)

        assert result == {"key": "value"}

    def test_raises_for_missing_file(self, temp_dir: Path):
        """Raises ConfigError for missing file."""
        file_path = temp_dir / "nonexistent.json"

        with pytest.raises(ConfigError, match="File not found"):
            load_json(file_path)

    def test_raises_for_invalid_json(self, temp_dir: Path):
        """Raises ConfigError for invalid JSON."""
        file_path = temp_dir / "invalid.json"
        file_path.write_text("not valid json {")

        with pytest.raises(ConfigError, match="Invalid JSON"):
            load_json(file_path)


class TestSaveJson:
    """Tests for save_json function."""

    def test_saves_json(self, temp_dir: Path):
        """Saves data to JSON file."""
        file_path = temp_dir / "output.json"
        data = {"key": "value", "number": 42}

        save_json(file_path, data)

        result = json.loads(file_path.read_text())
        assert result == data

    def test_creates_parent_directories(self, temp_dir: Path):
        """Creates parent directories if needed."""
        file_path = temp_dir / "nested" / "dir" / "output.json"

        save_json(file_path, {"key": "value"})

        assert file_path.exists()

    def test_uses_indent(self, temp_dir: Path):
        """Uses specified indentation."""
        file_path = temp_dir / "output.json"

        save_json(file_path, {"key": "value"}, indent=4)

        content = file_path.read_text()
        assert "    " in content  # 4-space indent


class TestLoadYaml:
    """Tests for load_yaml function."""

    def test_loads_valid_yaml(self, temp_dir: Path):
        """Loads valid YAML file."""
        file_path = temp_dir / "test.yaml"
        file_path.write_text("key: value\nnumber: 42")

        result = load_yaml(file_path)

        assert result == {"key": "value", "number": 42}

    def test_loads_empty_file_as_empty_dict(self, temp_dir: Path):
        """Loads empty YAML file as empty dict."""
        file_path = temp_dir / "empty.yaml"
        file_path.write_text("")

        result = load_yaml(file_path)

        assert result == {}

    def test_raises_for_missing_file(self, temp_dir: Path):
        """Raises ConfigError for missing file."""
        file_path = temp_dir / "nonexistent.yaml"

        with pytest.raises(ConfigError, match="File not found"):
            load_yaml(file_path)

    def test_raises_for_invalid_yaml(self, temp_dir: Path):
        """Raises ConfigError for invalid YAML."""
        file_path = temp_dir / "invalid.yaml"
        file_path.write_text("key: [invalid")

        with pytest.raises(ConfigError, match="Invalid YAML"):
            load_yaml(file_path)

    def test_raises_for_non_dict_yaml(self, temp_dir: Path):
        """Raises ConfigError for YAML that is not a mapping."""
        file_path = temp_dir / "list.yaml"
        file_path.write_text("- item1\n- item2")

        with pytest.raises(ConfigError, match="YAML file must contain a mapping"):
            load_yaml(file_path)


class TestSaveYaml:
    """Tests for save_yaml function."""

    def test_saves_yaml(self, temp_dir: Path):
        """Saves data to YAML file."""
        file_path = temp_dir / "output.yaml"
        data = {"key": "value", "number": 42}

        save_yaml(file_path, data)

        result = yaml.safe_load(file_path.read_text())
        assert result == data

    def test_creates_parent_directories(self, temp_dir: Path):
        """Creates parent directories if needed."""
        file_path = temp_dir / "nested" / "dir" / "output.yaml"

        save_yaml(file_path, {"key": "value"})

        assert file_path.exists()


class TestLoadProjectConfig:
    """Tests for load_project_config function."""

    def test_loads_valid_config(self, temp_dir: Path):
        """Loads valid project configuration."""
        config_content = """\
agent: claude-code
project_name: test-project
plugins: {}
"""
        (temp_dir / "dex.yaml").write_text(config_content)

        result = load_project_config(temp_dir)

        assert isinstance(result, ProjectConfig)
        assert result.agent == "claude-code"
        assert result.project_name == "test-project"

    def test_raises_for_missing_config(self, temp_dir: Path):
        """Raises ConfigError for missing dex.yaml."""
        with pytest.raises(ConfigError, match="File not found"):
            load_project_config(temp_dir)

    def test_raises_for_invalid_config(self, temp_dir: Path):
        """Raises ConfigError for invalid configuration."""
        # Missing required 'agent' field
        (temp_dir / "dex.yaml").write_text("project_name: test")

        with pytest.raises(ConfigError, match="Invalid project config"):
            load_project_config(temp_dir)


class TestSaveProjectConfig:
    """Tests for save_project_config function."""

    def test_saves_config(self, temp_dir: Path):
        """Saves project configuration."""
        config = ProjectConfig(
            agent="claude-code",
            project_name="test-project",
        )

        save_project_config(temp_dir, config)

        config_path = temp_dir / "dex.yaml"
        assert config_path.exists()

        saved_data = yaml.safe_load(config_path.read_text())
        assert saved_data["agent"] == "claude-code"


class TestLoadLockfile:
    """Tests for load_lockfile function."""

    def test_loads_valid_lockfile(self, temp_dir: Path):
        """Loads valid lock file."""
        lock_content = """\
version: '1.0'
agent: claude-code
plugins: {}
"""
        (temp_dir / "dex.lock").write_text(lock_content)

        result = load_lockfile(temp_dir)

        assert isinstance(result, LockFile)
        assert result.version == "1.0"
        assert result.agent == "claude-code"

    def test_returns_none_for_missing_lockfile(self, temp_dir: Path):
        """Returns None for missing lock file."""
        result = load_lockfile(temp_dir)
        assert result is None

    def test_raises_for_invalid_lockfile(self, temp_dir: Path):
        """Raises ConfigError for invalid lock file."""
        (temp_dir / "dex.lock").write_text("invalid: true")

        with pytest.raises(ConfigError, match="Invalid lock file"):
            load_lockfile(temp_dir)


class TestSaveLockfile:
    """Tests for save_lockfile function."""

    def test_saves_lockfile(self, temp_dir: Path):
        """Saves lock file."""
        lockfile = LockFile(agent="claude-code")

        save_lockfile(temp_dir, lockfile)

        lock_path = temp_dir / "dex.lock"
        assert lock_path.exists()

        saved_data = yaml.safe_load(lock_path.read_text())
        assert saved_data["agent"] == "claude-code"


class TestLoadPluginManifest:
    """Tests for load_plugin_manifest function."""

    def test_loads_valid_manifest(self, temp_dir: Path):
        """Loads valid plugin manifest."""
        manifest_data = {
            "name": "test-plugin",
            "version": "1.0.0",
            "description": "A test plugin",
        }
        (temp_dir / "package.json").write_text(json.dumps(manifest_data))

        result = load_plugin_manifest(temp_dir)

        assert isinstance(result, PluginManifest)
        assert result.name == "test-plugin"
        assert result.version == "1.0.0"

    def test_raises_for_missing_manifest(self, temp_dir: Path):
        """Raises ConfigError for missing package.json."""
        with pytest.raises(ConfigError, match="File not found"):
            load_plugin_manifest(temp_dir)

    def test_raises_for_invalid_manifest(self, temp_dir: Path):
        """Raises ConfigError for invalid manifest."""
        # Invalid version format
        manifest_data = {
            "name": "test-plugin",
            "version": "invalid",
            "description": "Test",
        }
        (temp_dir / "package.json").write_text(json.dumps(manifest_data))

        with pytest.raises(ConfigError, match="Invalid plugin manifest"):
            load_plugin_manifest(temp_dir)


class TestFindProjectRoot:
    """Tests for find_project_root function."""

    def test_finds_root_in_current_dir(self, temp_dir: Path):
        """Finds project root in current directory."""
        (temp_dir / "dex.yaml").write_text("agent: claude-code")

        result = find_project_root(temp_dir)

        # Use resolve() to handle macOS /var -> /private/var symlink
        assert result is not None
        assert result.resolve() == temp_dir.resolve()

    def test_finds_root_in_parent_dir(self, temp_dir: Path):
        """Finds project root in parent directory."""
        (temp_dir / "dex.yaml").write_text("agent: claude-code")
        subdir = temp_dir / "src" / "deep" / "nested"
        subdir.mkdir(parents=True)

        result = find_project_root(subdir)

        # Use resolve() to handle macOS /var -> /private/var symlink
        assert result is not None
        assert result.resolve() == temp_dir.resolve()

    def test_returns_none_when_not_found(self, temp_dir: Path):
        """Returns None when no project root is found."""
        subdir = temp_dir / "no_project"
        subdir.mkdir()

        result = find_project_root(subdir)

        assert result is None


class TestConfigError:
    """Tests for ConfigError exception."""

    def test_error_message(self):
        """Error has message."""
        error = ConfigError("Test error message")
        assert str(error) == "Test error message"

    def test_error_with_path(self):
        """Error includes path."""
        error = ConfigError("File not found", path=Path("/tmp/test.json"))
        assert error.path == Path("/tmp/test.json")
