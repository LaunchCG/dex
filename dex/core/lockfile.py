"""Lock file management for Dex."""

from pathlib import Path

from dex.config.parser import load_lockfile, save_lockfile
from dex.config.schemas import AgentType, LockedPlugin, LockFile


class LockFileManager:
    """Manages the dex.lock file for deterministic installations."""

    def __init__(self, project_root: Path, agent: AgentType):
        """Initialize the lock file manager.

        Args:
            project_root: Path to the project root
            agent: Target agent platform
        """
        self._project_root = project_root
        self._agent = agent
        self._lockfile: LockFile | None = None
        self._modified = False

    def load(self) -> LockFile:
        """Load the lock file from disk.

        Creates a new empty lock file if one doesn't exist.

        Returns:
            The loaded or new lock file
        """
        self._lockfile = load_lockfile(self._project_root)
        if self._lockfile is None:
            self._lockfile = LockFile(agent=self._agent)
            self._modified = True
        return self._lockfile

    def save(self) -> None:
        """Save the lock file to disk if modified."""
        if self._lockfile is not None and self._modified:
            save_lockfile(self._project_root, self._lockfile)
            self._modified = False

    @property
    def lockfile(self) -> LockFile:
        """Get the current lock file, loading if necessary."""
        if self._lockfile is None:
            self.load()
        assert self._lockfile is not None
        return self._lockfile

    def get_locked_version(self, plugin_name: str) -> str | None:
        """Get the locked version for a plugin.

        Args:
            plugin_name: Name of the plugin

        Returns:
            Locked version string, or None if not locked
        """
        locked = self.lockfile.plugins.get(plugin_name)
        return locked.version if locked else None

    def get_locked_plugin(self, plugin_name: str) -> LockedPlugin | None:
        """Get the full locked plugin entry.

        Args:
            plugin_name: Name of the plugin

        Returns:
            LockedPlugin entry, or None if not locked
        """
        return self.lockfile.plugins.get(plugin_name)

    def lock_plugin(
        self,
        name: str,
        version: str,
        resolved_url: str,
        integrity: str | None = None,
        dependencies: dict[str, str] | None = None,
    ) -> None:
        """Lock a plugin to a specific version.

        Args:
            name: Plugin name
            version: Resolved version
            resolved_url: URL the package was resolved from
            integrity: Optional integrity hash
            dependencies: Optional dictionary of dependency versions
        """
        self.lockfile.plugins[name] = LockedPlugin(
            version=version,
            resolved=resolved_url,
            integrity=integrity or "",
            dependencies=dependencies or {},
        )
        self._modified = True

    def unlock_plugin(self, name: str) -> bool:
        """Remove a plugin from the lock file.

        Args:
            name: Plugin name to remove

        Returns:
            True if the plugin was removed, False if it wasn't locked
        """
        if name in self.lockfile.plugins:
            del self.lockfile.plugins[name]
            self._modified = True
            return True
        return False

    def is_locked(self, plugin_name: str) -> bool:
        """Check if a plugin is locked.

        Args:
            plugin_name: Name of the plugin

        Returns:
            True if the plugin has a lock entry
        """
        return plugin_name in self.lockfile.plugins

    def list_locked(self) -> list[str]:
        """List all locked plugin names.

        Returns:
            List of plugin names with lock entries
        """
        return list(self.lockfile.plugins.keys())

    def clear(self) -> None:
        """Clear all lock entries."""
        self.lockfile.plugins.clear()
        self._modified = True
