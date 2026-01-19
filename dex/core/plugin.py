"""Plugin model representing an installed or resolved plugin."""

from pathlib import Path

from dex.config.parser import load_plugin_manifest
from dex.config.schemas import (
    CommandConfig,
    MCPServerConfig,
    PluginManifest,
    SkillConfig,
    SubAgentConfig,
)

# Type alias for component lists
ComponentList = (
    list[SkillConfig] | list[CommandConfig] | list[SubAgentConfig] | list[MCPServerConfig]
)


class Plugin:
    """Represents a plugin that can be installed.

    A plugin is loaded from a local directory containing a package.json manifest.
    """

    def __init__(self, path: Path, manifest: PluginManifest):
        """Initialize a Plugin.

        Args:
            path: Path to the plugin directory
            manifest: Parsed plugin manifest
        """
        self._path = path.resolve()
        self._manifest = manifest

    @classmethod
    def load(cls, path: Path) -> "Plugin":
        """Load a plugin from disk.

        Args:
            path: Path to the plugin directory

        Returns:
            Loaded Plugin instance

        Raises:
            FileNotFoundError: If package.json is not found
        """
        path = path.resolve()
        if not (path / "package.json").exists():
            raise FileNotFoundError(f"No package.json found in {path}")

        manifest = load_plugin_manifest(path)
        return cls(path, manifest)

    @property
    def path(self) -> Path:
        """Get the plugin directory path."""
        return self._path

    @property
    def manifest(self) -> PluginManifest:
        """Get the plugin manifest."""
        return self._manifest

    @property
    def name(self) -> str:
        """Get the plugin name."""
        return self._manifest.name

    @property
    def version(self) -> str:
        """Get the plugin version."""
        return self._manifest.version

    @property
    def description(self) -> str:
        """Get the plugin description."""
        return self._manifest.description

    @property
    def skills(self) -> list[SkillConfig]:
        """Get the plugin's skill definitions."""
        return self._manifest.skills

    @property
    def commands(self) -> list[CommandConfig]:
        """Get the plugin's command definitions."""
        return self._manifest.commands

    @property
    def sub_agents(self) -> list[SubAgentConfig]:
        """Get the plugin's sub-agent definitions."""
        return self._manifest.sub_agents

    @property
    def mcp_servers(self) -> list[MCPServerConfig]:
        """Get the plugin's MCP server definitions."""
        return self._manifest.mcp_servers

    @property
    def dependencies(self) -> dict[str, str]:
        """Get the plugin's dependencies."""
        return self._manifest.dependencies

    def resolve_context_path(self, context_path: str) -> Path:
        """Resolve a context path relative to the plugin directory.

        Args:
            context_path: Path from the manifest (e.g., "./skills/linting.md")

        Returns:
            Absolute path to the context file
        """
        # Remove leading "./" if present
        if context_path.startswith("./"):
            context_path = context_path[2:]
        return self._path / context_path

    def resolve_file_path(self, file_path: str) -> Path:
        """Resolve a file path relative to the plugin directory.

        Args:
            file_path: Path from the manifest

        Returns:
            Absolute path to the file
        """
        if file_path.startswith("./"):
            file_path = file_path[2:]
        return self._path / file_path

    def has_component(self, component_type: str, name: str) -> bool:
        """Check if the plugin has a component of the given type and name.

        Args:
            component_type: One of "skill", "command", "sub_agent", "mcp_server"
            name: Component name

        Returns:
            True if the component exists
        """
        components: dict[str, ComponentList] = {
            "skill": self.skills,
            "command": self.commands,
            "sub_agent": self.sub_agents,
            "mcp_server": self.mcp_servers,
        }
        component_list = components.get(component_type, [])
        return any(c.name == name for c in component_list)

    def __repr__(self) -> str:
        return f"Plugin(name={self.name!r}, version={self.version!r})"
