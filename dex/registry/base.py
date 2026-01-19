"""Abstract base class for registry clients."""

from abc import ABC, abstractmethod
from dataclasses import dataclass
from pathlib import Path


@dataclass
class PackageInfo:
    """Information about a package in a registry."""

    name: str
    versions: list[str]
    latest: str


@dataclass
class ResolvedPackage:
    """A resolved package ready for installation."""

    name: str
    version: str
    resolved_url: str
    local_path: Path | None = None
    integrity: str | None = None


class RegistryClient(ABC):
    """Abstract base class for registry clients.

    Registry clients handle fetching package information and downloading
    packages from various sources (local files, HTTPS, S3, etc.).
    """

    @property
    @abstractmethod
    def protocol(self) -> str:
        """Get the protocol this client handles (e.g., "file", "https")."""
        ...

    @abstractmethod
    def get_package_info(self, name: str) -> PackageInfo | None:
        """Get information about a package.

        Args:
            name: Package name

        Returns:
            PackageInfo if found, None otherwise
        """
        ...

    @abstractmethod
    def resolve_package(self, name: str, version: str) -> ResolvedPackage | None:
        """Resolve a package to a downloadable location.

        Args:
            name: Package name
            version: Version to resolve (can be a specifier)

        Returns:
            ResolvedPackage if found, None otherwise
        """
        ...

    @abstractmethod
    def fetch_package(self, resolved: ResolvedPackage, dest_dir: Path) -> Path:
        """Fetch a package to a local directory.

        Args:
            resolved: Resolved package information
            dest_dir: Directory to download to

        Returns:
            Path to the downloaded/extracted package directory
        """
        ...

    def list_packages(self) -> list[str]:
        """List all available packages in the registry.

        Default implementation returns empty list.

        Returns:
            List of package names
        """
        return []
