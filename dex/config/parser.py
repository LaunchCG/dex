"""Configuration file parsing utilities."""

import json
from pathlib import Path
from typing import Any

import yaml
from pydantic import ValidationError

from dex.config.schemas import LockFile, PluginManifest, ProjectConfig


class ConfigError(Exception):
    """Error loading or parsing configuration."""

    def __init__(self, message: str, path: Path | None = None):
        self.path = path
        super().__init__(message)


def load_json(path: Path) -> dict[str, Any]:
    """Load and parse a JSON file.

    Args:
        path: Path to the JSON file

    Returns:
        Parsed JSON as a dictionary

    Raises:
        ConfigError: If the file cannot be read or parsed
    """
    if not path.exists():
        raise ConfigError(f"File not found: {path}", path)

    try:
        with open(path, encoding="utf-8") as f:
            result: dict[str, Any] = json.load(f)
            return result
    except json.JSONDecodeError as e:
        raise ConfigError(f"Invalid JSON in {path}: {e}", path) from e
    except OSError as e:
        raise ConfigError(f"Cannot read {path}: {e}", path) from e


def save_json(path: Path, data: dict[str, Any], indent: int = 2) -> None:
    """Save data to a JSON file.

    Args:
        path: Path to write to
        data: Data to serialize
        indent: JSON indentation level
    """
    path.parent.mkdir(parents=True, exist_ok=True)
    with open(path, "w", encoding="utf-8") as f:
        json.dump(data, f, indent=indent, default=str)
        f.write("\n")


def load_yaml(path: Path) -> dict[str, Any]:
    """Load and parse a YAML file.

    Args:
        path: Path to the YAML file

    Returns:
        Parsed YAML as a dictionary

    Raises:
        ConfigError: If the file cannot be read or parsed
    """
    if not path.exists():
        raise ConfigError(f"File not found: {path}", path)

    try:
        with open(path, encoding="utf-8") as f:
            result = yaml.safe_load(f)
            if result is None:
                return {}
            if not isinstance(result, dict):
                raise ConfigError(f"YAML file must contain a mapping: {path}", path)
            return result
    except yaml.YAMLError as e:
        raise ConfigError(f"Invalid YAML in {path}: {e}", path) from e
    except OSError as e:
        raise ConfigError(f"Cannot read {path}: {e}", path) from e


def save_yaml(path: Path, data: dict[str, Any]) -> None:
    """Save data to a YAML file.

    Args:
        path: Path to write to
        data: Data to serialize
    """
    path.parent.mkdir(parents=True, exist_ok=True)
    with open(path, "w", encoding="utf-8") as f:
        yaml.safe_dump(data, f, default_flow_style=False, sort_keys=False, allow_unicode=True)


def load_project_config(project_root: Path) -> ProjectConfig:
    """Load project configuration from dex.yaml.

    Args:
        project_root: Path to the project root directory

    Returns:
        Parsed ProjectConfig

    Raises:
        ConfigError: If the file is missing or invalid
    """
    config_path = project_root / "dex.yaml"
    data = load_yaml(config_path)

    try:
        return ProjectConfig.model_validate(data)
    except ValidationError as e:
        raise ConfigError(f"Invalid project config: {e}", config_path) from e


def save_project_config(project_root: Path, config: ProjectConfig) -> None:
    """Save project configuration to dex.yaml.

    Args:
        project_root: Path to the project root directory
        config: ProjectConfig to save
    """
    config_path = project_root / "dex.yaml"
    save_yaml(config_path, config.model_dump(exclude_none=True, exclude_unset=True))


def load_lockfile(project_root: Path) -> LockFile | None:
    """Load lock file from dex.lock if it exists.

    Args:
        project_root: Path to the project root directory

    Returns:
        Parsed LockFile or None if it doesn't exist

    Raises:
        ConfigError: If the file exists but is invalid
    """
    lock_path = project_root / "dex.lock"
    if not lock_path.exists():
        return None

    data = load_yaml(lock_path)

    try:
        return LockFile.model_validate(data)
    except ValidationError as e:
        raise ConfigError(f"Invalid lock file: {e}", lock_path) from e


def save_lockfile(project_root: Path, lockfile: LockFile) -> None:
    """Save lock file to dex.lock.

    Args:
        project_root: Path to the project root directory
        lockfile: LockFile to save
    """
    lock_path = project_root / "dex.lock"
    save_yaml(lock_path, lockfile.model_dump())


def load_plugin_manifest(plugin_path: Path) -> PluginManifest:
    """Load plugin manifest from package.json.

    Args:
        plugin_path: Path to the plugin directory

    Returns:
        Parsed PluginManifest

    Raises:
        ConfigError: If the file is missing or invalid
    """
    manifest_path = plugin_path / "package.json"
    data = load_json(manifest_path)

    try:
        return PluginManifest.model_validate(data)
    except ValidationError as e:
        raise ConfigError(f"Invalid plugin manifest: {e}", manifest_path) from e


def find_project_root(start_path: Path | None = None) -> Path | None:
    """Find the project root by looking for dex.yaml.

    Args:
        start_path: Directory to start searching from (defaults to cwd)

    Returns:
        Path to project root, or None if not found
    """
    if start_path is None:
        start_path = Path.cwd()

    current = start_path.resolve()
    while current != current.parent:
        if (current / "dex.yaml").exists():
            return current
        current = current.parent

    # Check root
    if (current / "dex.yaml").exists():
        return current

    return None
