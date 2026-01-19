"""Local file system registry client."""

from __future__ import annotations

import json
import logging
from pathlib import Path
from typing import Any
from urllib.parse import urlparse

from dex.config.parser import load_plugin_manifest
from dex.registry.base import PackageInfo, RegistryClient, ResolvedPackage
from dex.registry.common import (
    SourceMode,
    extract_package_from_registry_data,
)
from dex.utils.filesystem import compute_integrity, copy_directory, extract_tarball
from dex.utils.version import find_best_version

logger = logging.getLogger(__name__)


class LocalRegistryError(Exception):
    """Error interacting with local registry."""

    def __init__(self, message: str, path: str | None = None):
        self.path = path
        super().__init__(message)


class LocalRegistryClient(RegistryClient):
    """Registry client for local file system sources.

    Supports two modes (specified explicitly, not auto-detected):
    1. Registry mode: Points to a directory with registry.json and .tar.gz files
    2. Package mode: Points directly to a plugin directory with package.json

    URL format:
    - file:///path/to/registry (registry mode)
    - file:../relative/path (relative path)
    """

    def __init__(self, url: str, mode: SourceMode = "registry"):
        """Initialize the local registry client.

        Args:
            url: Local file URL (file:// or file:)
            mode: Source mode - "registry" expects registry.json,
                  "package" expects package.json
        """
        self._url = url
        self._mode = mode
        self._path = self._parse_url(url)

        logger.info("Initializing local registry client for %s (mode=%s)", self._path, mode)

    def _parse_url(self, url: str) -> Path:
        """Parse a file URL to a Path."""
        if url.startswith("file://"):
            # Absolute path
            parsed = urlparse(url)
            return Path(parsed.path)
        elif url.startswith("file:"):
            # Relative path (file:../path or file:./path)
            return Path(url[5:]).resolve()
        else:
            # Assume it's a path
            return Path(url).resolve()

    @property
    def protocol(self) -> str:
        return "file"

    @property
    def path(self) -> Path:
        """Get the local path this client points to."""
        return self._path

    @property
    def mode(self) -> SourceMode:
        """Get the source mode."""
        return self._mode

    def _get_registry_data(self) -> dict[str, Any]:
        """Load registry.json data.

        Returns:
            Parsed registry.json contents

        Raises:
            LocalRegistryError: If registry.json cannot be loaded or is invalid JSON
        """
        registry_file = self._path / "registry.json"
        if not registry_file.exists():
            raise LocalRegistryError(
                f"Registry not found: {self._path} does not contain registry.json",
                path=str(self._path),
            )

        try:
            with open(registry_file, encoding="utf-8") as f:
                result: dict[str, Any] = json.load(f)
                return result
        except json.JSONDecodeError as e:
            raise LocalRegistryError(
                f"Invalid JSON in registry.json: {e}",
                path=str(registry_file),
            ) from e

    def _get_package_data(self) -> dict[str, Any]:
        """Load package.json data.

        Returns:
            Parsed package.json contents

        Raises:
            LocalRegistryError: If package.json cannot be loaded or is invalid JSON
        """
        package_file = self._path / "package.json"
        if not package_file.exists():
            raise LocalRegistryError(
                f"Package not found: {self._path} does not contain package.json",
                path=str(self._path),
            )

        try:
            with open(package_file, encoding="utf-8") as f:
                result: dict[str, Any] = json.load(f)
                return result
        except json.JSONDecodeError as e:
            raise LocalRegistryError(
                f"Invalid JSON in package.json: {e}",
                path=str(package_file),
            ) from e

    def get_package_info(self, name: str) -> PackageInfo | None:
        """Get package information from local source.

        Args:
            name: Package name to look up

        Returns:
            PackageInfo if found, None if package doesn't exist

        Raises:
            LocalRegistryError: If there's an error reading files
        """
        logger.debug("Getting package info for '%s' from local (mode=%s)", name, self._mode)

        if self._mode == "registry":
            return self._get_package_from_registry(name)
        else:
            return self._get_package_from_package(name)

    def _get_package_from_registry(self, name: str) -> PackageInfo | None:
        """Get package info from registry.json.

        Returns:
            PackageInfo if found, None if package not in registry

        Raises:
            LocalRegistryError: If registry.json cannot be read or parsed
        """
        registry_data = self._get_registry_data()
        return extract_package_from_registry_data(registry_data, name)

    def _get_package_from_package(self, name: str) -> PackageInfo | None:
        """Get package info from package.json (package mode).

        Returns:
            PackageInfo if found and name matches, None if name doesn't match

        Raises:
            LocalRegistryError: If package.json cannot be read or parsed
        """
        try:
            manifest = load_plugin_manifest(self._path)
            if manifest.name != name:
                return None
            return PackageInfo(
                name=manifest.name,
                versions=[manifest.version],
                latest=manifest.version,
            )
        except FileNotFoundError as e:
            raise LocalRegistryError(
                f"Package not found: {self._path} does not contain package.json",
                path=str(self._path),
            ) from e
        except Exception as e:
            raise LocalRegistryError(
                f"Failed to load package.json: {e}",
                path=str(self._path),
            ) from e

    def resolve_package(self, name: str, version: str) -> ResolvedPackage | None:
        """Resolve a package to a local path.

        Args:
            name: Package name
            version: Version specifier ('latest', exact, or semver range)

        Returns:
            ResolvedPackage with local path, or None if package/version not found

        Raises:
            LocalRegistryError: If there's an error reading files
        """
        logger.info("Resolving package '%s' version '%s' from local", name, version)
        info = self.get_package_info(name)
        if info is None:
            logger.warning("Package '%s' not found in local registry", name)
            return None

        # Find the best matching version
        resolved_version: str | None
        if version == "latest":
            resolved_version = info.latest
        else:
            resolved_version = find_best_version(version, info.versions)

        if resolved_version is None:
            logger.warning("Version '%s' not found for package '%s'", version, name)
            return None

        if self._mode == "registry":
            # Look for tarball
            tarball_name = f"{name}-{resolved_version}.tar.gz"
            tarball_path = self._path / tarball_name
            if not tarball_path.exists():
                logger.warning("Tarball not found: %s", tarball_path)
                return None

            logger.info(
                "Resolved package '%s' to version %s at %s", name, resolved_version, tarball_path
            )
            return ResolvedPackage(
                name=name,
                version=resolved_version,
                resolved_url=f"file://{tarball_path}",
                local_path=tarball_path,
                integrity=compute_integrity(tarball_path),
            )
        else:
            # Direct directory
            logger.info(
                "Resolved package '%s' to version %s (direct directory)", name, resolved_version
            )
            return ResolvedPackage(
                name=name,
                version=resolved_version,
                resolved_url=f"file://{self._path}",
                local_path=self._path,
            )

    def fetch_package(self, resolved: ResolvedPackage, dest_dir: Path) -> Path:
        """Fetch a package to a local directory."""
        logger.info("Fetching package '%s' v%s to %s", resolved.name, resolved.version, dest_dir)
        if resolved.local_path is None:
            raise ValueError("Resolved package has no local path")

        local_path = resolved.local_path

        if local_path.is_dir():
            # Copy directory
            logger.debug("Copying directory from %s", local_path)
            plugin_dir = dest_dir / resolved.name
            result = copy_directory(local_path, plugin_dir)
            logger.info("Package '%s' copied to %s", resolved.name, result)
            return result
        elif local_path.suffix == ".gz" or str(local_path).endswith(".tar.gz"):
            # Extract tarball
            logger.debug("Extracting tarball from %s", local_path)
            result = extract_tarball(local_path, dest_dir)
            logger.info("Package '%s' extracted to %s", resolved.name, result)
            return result
        else:
            logger.error("Unknown package format: %s", local_path)
            raise ValueError(f"Unknown package format: {local_path}")

    def list_packages(self) -> list[str]:
        """List all packages in the local source.

        Returns:
            List of package names

        Raises:
            LocalRegistryError: If there's an error reading files
        """
        if self._mode == "registry":
            registry_data = self._get_registry_data()
            return list(registry_data.get("packages", {}).keys())
        else:
            # Single plugin
            try:
                manifest = load_plugin_manifest(self._path)
                return [manifest.name]
            except FileNotFoundError as e:
                raise LocalRegistryError(
                    f"Package not found: {self._path} does not contain package.json",
                    path=str(self._path),
                ) from e
            except Exception as e:
                raise LocalRegistryError(
                    f"Failed to load package.json: {e}",
                    path=str(self._path),
                ) from e
