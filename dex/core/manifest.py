"""Dex manifest manager for tracking managed files.

The manifest tracks all files, directories, and MCP servers managed by dex,
organized by plugin. This enables clean uninstallation.
"""

import json
from pathlib import Path

from dex.config.schemas import DexManifest, PluginFiles


class ManifestManager:
    """Manages the dex manifest file.

    The manifest is stored at .dex/manifest.json in the project root.
    """

    MANIFEST_DIR = ".dex"
    MANIFEST_FILE = "manifest.json"

    def __init__(self, project_root: Path) -> None:
        """Initialize the manifest manager.

        Args:
            project_root: Path to the project root directory
        """
        self.project_root = project_root
        self._manifest: DexManifest | None = None

    @property
    def manifest_dir(self) -> Path:
        """Get the manifest directory path."""
        return self.project_root / self.MANIFEST_DIR

    @property
    def manifest_path(self) -> Path:
        """Get the manifest file path."""
        return self.manifest_dir / self.MANIFEST_FILE

    def load(self) -> DexManifest:
        """Load the manifest from disk, or create a new one."""
        if self._manifest is not None:
            return self._manifest

        if self.manifest_path.exists():
            try:
                with open(self.manifest_path, encoding="utf-8") as f:
                    data = json.load(f)
                self._manifest = DexManifest.model_validate(data)
            except (json.JSONDecodeError, OSError, ValueError):
                self._manifest = DexManifest()
        else:
            self._manifest = DexManifest()

        return self._manifest

    def save(self) -> None:
        """Save the manifest to disk."""
        if self._manifest is None:
            return

        # Ensure directory exists
        self.manifest_dir.mkdir(parents=True, exist_ok=True)

        # Write manifest
        with open(self.manifest_path, "w", encoding="utf-8") as f:
            json.dump(self._manifest.model_dump(), f, indent=2)
            f.write("\n")

    def add_file(self, plugin_name: str, file_path: Path) -> None:
        """Record a file as managed by a plugin.

        Args:
            plugin_name: Name of the plugin
            file_path: Absolute path to the file
        """
        manifest = self.load()
        # Store relative path
        try:
            rel_path = str(file_path.relative_to(self.project_root))
        except ValueError:
            rel_path = str(file_path)
        manifest.add_file(plugin_name, rel_path)

    def add_directory(self, plugin_name: str, dir_path: Path) -> None:
        """Record a directory as managed by a plugin.

        Args:
            plugin_name: Name of the plugin
            dir_path: Absolute path to the directory
        """
        manifest = self.load()
        try:
            rel_path = str(dir_path.relative_to(self.project_root))
        except ValueError:
            rel_path = str(dir_path)
        manifest.add_directory(plugin_name, rel_path)

    def add_mcp_server(self, plugin_name: str, server_name: str) -> None:
        """Record an MCP server as added by a plugin.

        Args:
            plugin_name: Name of the plugin
            server_name: Name of the MCP server
        """
        manifest = self.load()
        manifest.add_mcp_server(plugin_name, server_name)

    def get_plugin_files(self, plugin_name: str) -> PluginFiles | None:
        """Get all files managed by a plugin.

        Args:
            plugin_name: Name of the plugin

        Returns:
            PluginFiles object or None if plugin not found
        """
        manifest = self.load()
        return manifest.get_plugin_files(plugin_name)

    def get_mcp_servers_to_remove(self, plugin_name: str) -> list[str]:
        """Get MCP servers that should be removed when uninstalling a plugin.

        Only returns servers not used by any other installed plugin.

        Args:
            plugin_name: Name of the plugin being uninstalled

        Returns:
            List of server names to remove
        """
        manifest = self.load()
        return manifest.get_mcp_servers_to_remove(plugin_name)

    def remove_plugin(self, plugin_name: str) -> PluginFiles | None:
        """Remove a plugin from the manifest.

        Args:
            plugin_name: Name of the plugin to remove

        Returns:
            PluginFiles that were managed by the plugin, or None
        """
        manifest = self.load()
        return manifest.remove_plugin(plugin_name)

    def is_file_managed(self, file_path: Path) -> bool:
        """Check if a file is managed by any plugin.

        Args:
            file_path: Absolute path to the file

        Returns:
            True if the file is managed by dex
        """
        manifest = self.load()
        try:
            rel_path = str(file_path.relative_to(self.project_root))
        except ValueError:
            rel_path = str(file_path)

        return any(rel_path in plugin_files.files for plugin_files in manifest.plugins.values())

    def get_all_managed_files(self) -> set[str]:
        """Get all files managed by any plugin.

        Returns:
            Set of relative file paths
        """
        manifest = self.load()
        all_files: set[str] = set()
        for plugin_files in manifest.plugins.values():
            all_files.update(plugin_files.files)
        return all_files
