"""Project model representing a Dex-managed project."""

from pathlib import Path

from dex.config.parser import (
    find_project_root,
    load_project_config,
    save_project_config,
)
from dex.config.schemas import AgentType, PluginSpec, ProjectConfig


class Project:
    """Represents a Dex-managed project.

    A project is defined by its dex.yaml configuration file.
    """

    def __init__(self, root: Path, config: ProjectConfig):
        """Initialize a Project.

        Args:
            root: Path to the project root directory
            config: Parsed project configuration
        """
        self._root = root.resolve()
        self._config = config

    @classmethod
    def load(cls, path: Path | None = None) -> "Project":
        """Load a project from disk.

        Args:
            path: Path to the project root, or None to search from cwd

        Returns:
            Loaded Project instance

        Raises:
            FileNotFoundError: If no project is found
        """
        if path is None:
            path = find_project_root()
            if path is None:
                raise FileNotFoundError(
                    "No dex.yaml found in current directory or any parent directory"
                )
        else:
            path = path.resolve()
            if not (path / "dex.yaml").exists():
                raise FileNotFoundError(f"No dex.yaml found in {path}")

        config = load_project_config(path)
        return cls(path, config)

    @classmethod
    def init(
        cls,
        path: Path,
        agent: AgentType,
        project_name: str | None = None,
    ) -> "Project":
        """Initialize a new project.

        Args:
            path: Path to the project root directory
            agent: Target AI agent platform
            project_name: Optional project name (defaults to directory name)

        Returns:
            New Project instance

        Raises:
            FileExistsError: If dex.yaml already exists
        """
        path = path.resolve()
        config_path = path / "dex.yaml"

        if config_path.exists():
            raise FileExistsError(f"Project already initialized: {config_path}")

        # Use directory name if project_name not specified
        if project_name is None:
            project_name = path.name

        config = ProjectConfig(
            agent=agent,
            project_name=project_name,
            plugins={},
            registries={},
        )

        project = cls(path, config)
        project.save()
        return project

    def save(self) -> None:
        """Save the project configuration to disk."""
        save_project_config(self._root, self._config)

    @property
    def root(self) -> Path:
        """Get the project root directory."""
        return self._root

    @property
    def agent(self) -> AgentType:
        """Get the target AI agent platform."""
        return self._config.agent

    @property
    def project_name(self) -> str:
        """Get the project name."""
        return self._config.project_name or self._root.name

    @property
    def config(self) -> ProjectConfig:
        """Get the underlying configuration."""
        return self._config

    @property
    def plugins(self) -> dict[str, str | PluginSpec]:
        """Get the plugin specifications."""
        return self._config.plugins

    def add_plugin(self, name: str, spec: str | PluginSpec) -> None:
        """Add or update a plugin specification.

        Args:
            name: Plugin name
            spec: Version string or PluginSpec
        """
        self._config.plugins[name] = spec

    def remove_plugin(self, name: str) -> bool:
        """Remove a plugin specification.

        Args:
            name: Plugin name to remove

        Returns:
            True if the plugin was removed, False if it wasn't present
        """
        if name in self._config.plugins:
            del self._config.plugins[name]
            return True
        return False

    def get_plugin_spec(self, name: str) -> PluginSpec | None:
        """Get the specification for a plugin.

        Args:
            name: Plugin name

        Returns:
            PluginSpec or None if not found
        """
        spec = self._config.plugins.get(name)
        if spec is None:
            return None
        if isinstance(spec, str):
            return PluginSpec(version=spec)
        return spec

    def get_registry_url(self, name: str | None = None) -> str | None:
        """Get a registry URL by name.

        Args:
            name: Registry name, or None for default registry

        Returns:
            Registry URL or None if not found
        """
        if name is None:
            name = self._config.default_registry
        if name is None:
            return None
        return self._config.registries.get(name)

    def __repr__(self) -> str:
        return f"Project(root={self._root!r}, agent={self.agent!r})"
