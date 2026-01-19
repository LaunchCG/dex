"""Local file system registry client."""

import json
from pathlib import Path
from urllib.parse import urlparse

from dex.config.parser import load_plugin_manifest
from dex.registry.base import PackageInfo, RegistryClient, ResolvedPackage
from dex.utils.filesystem import compute_integrity, copy_directory, extract_tarball
from dex.utils.version import find_best_version


class LocalRegistryClient(RegistryClient):
    """Registry client for local file system sources.

    Supports two modes:
    1. Registry mode: Points to a directory with registry.json and .tar.gz files
    2. Direct mode: Points directly to a plugin directory

    URL format:
    - file:///path/to/registry (registry with registry.json)
    - file:../relative/path (relative path, can be registry or plugin)
    """

    def __init__(self, url: str):
        """Initialize the local registry client.

        Args:
            url: Local file URL (file:// or file:)
        """
        self._url = url
        self._path = self._parse_url(url)
        self._is_registry = (self._path / "registry.json").exists()

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

    def get_package_info(self, name: str) -> PackageInfo | None:
        """Get package information from local registry."""
        if self._is_registry:
            return self._get_package_from_registry(name)
        else:
            # Direct plugin directory
            return self._get_package_from_directory(name)

    def _get_package_from_registry(self, name: str) -> PackageInfo | None:
        """Get package info from a registry.json file."""
        registry_file = self._path / "registry.json"
        if not registry_file.exists():
            return None

        with open(registry_file, encoding="utf-8") as f:
            registry_data = json.load(f)

        packages = registry_data.get("packages", {})
        if name not in packages:
            return None

        pkg_data = packages[name]
        return PackageInfo(
            name=name,
            versions=pkg_data.get("versions", []),
            latest=pkg_data.get("latest", pkg_data.get("versions", ["0.0.0"])[-1]),
        )

    def _get_package_from_directory(self, name: str) -> PackageInfo | None:
        """Get package info from a direct plugin directory."""
        manifest_path = self._path / "package.json"
        if not manifest_path.exists():
            return None

        try:
            manifest = load_plugin_manifest(self._path)
            if manifest.name != name:
                return None
            return PackageInfo(
                name=manifest.name,
                versions=[manifest.version],
                latest=manifest.version,
            )
        except Exception:
            return None

    def resolve_package(self, name: str, version: str) -> ResolvedPackage | None:
        """Resolve a package to a local path."""
        info = self.get_package_info(name)
        if info is None:
            return None

        # Find the best matching version
        resolved_version: str | None
        if version == "latest":
            resolved_version = info.latest
        else:
            resolved_version = find_best_version(version, info.versions)

        if resolved_version is None:
            return None

        if self._is_registry:
            # Look for tarball
            tarball_name = f"{name}-{resolved_version}.tar.gz"
            tarball_path = self._path / tarball_name
            if not tarball_path.exists():
                return None

            return ResolvedPackage(
                name=name,
                version=resolved_version,
                resolved_url=f"file://{tarball_path}",
                local_path=tarball_path,
                integrity=compute_integrity(tarball_path),
            )
        else:
            # Direct directory
            return ResolvedPackage(
                name=name,
                version=resolved_version,
                resolved_url=f"file://{self._path}",
                local_path=self._path,
            )

    def fetch_package(self, resolved: ResolvedPackage, dest_dir: Path) -> Path:
        """Fetch a package to a local directory."""
        if resolved.local_path is None:
            raise ValueError("Resolved package has no local path")

        local_path = resolved.local_path

        if local_path.is_dir():
            # Copy directory
            plugin_dir = dest_dir / resolved.name
            return copy_directory(local_path, plugin_dir)
        elif local_path.suffix == ".gz" or str(local_path).endswith(".tar.gz"):
            # Extract tarball
            return extract_tarball(local_path, dest_dir)
        else:
            raise ValueError(f"Unknown package format: {local_path}")

    def list_packages(self) -> list[str]:
        """List all packages in the registry."""
        if self._is_registry:
            registry_file = self._path / "registry.json"
            if not registry_file.exists():
                return []

            with open(registry_file, encoding="utf-8") as f:
                registry_data = json.load(f)

            return list(registry_data.get("packages", {}).keys())
        else:
            # Single plugin
            manifest_path = self._path / "package.json"
            if manifest_path.exists():
                try:
                    manifest = load_plugin_manifest(self._path)
                    return [manifest.name]
                except Exception:
                    pass
            return []
