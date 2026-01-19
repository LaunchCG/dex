"""Azure Blob Storage registry client for remote packages."""

from __future__ import annotations

import json
import logging
import shutil
import tempfile
from pathlib import Path
from typing import Any
from urllib.parse import urlparse

from azure.core.exceptions import (
    ClientAuthenticationError,
    ResourceNotFoundError,
    ServiceRequestError,
)
from azure.identity import DefaultAzureCredential
from azure.storage.blob import BlobServiceClient, ContainerClient

from dex.registry.base import PackageInfo, RegistryClient, ResolvedPackage
from dex.registry.cache import RegistryCache
from dex.registry.common import (
    SourceMode,
    extract_package_from_manifest_data,
    extract_package_from_registry_data,
    names_match,
    parse_tarball_info,
)
from dex.utils.filesystem import compute_integrity, extract_tarball
from dex.utils.version import find_best_version

logger = logging.getLogger(__name__)


class AzureRegistryError(Exception):
    """Error interacting with Azure Blob registry."""

    def __init__(self, message: str, container: str | None = None, blob: str | None = None):
        self.container = container
        self.blob = blob
        super().__init__(message)


class AzureRegistryClient(RegistryClient):
    """Registry client for Azure Blob Storage-hosted packages.

    Supports two modes (specified explicitly, not auto-detected):
    1. Registry mode: Blob path contains registry.json and .tar.gz files
    2. Package mode: Blob path points to a directory with package.json

    Also handles direct tarball URLs (detected from .tar.gz extension).

    URL formats:
    - az://account/container/path/to/registry/ (registry mode)
    - az://account/container/path/to/plugin/ (package mode)
    - az://account/container/path/to/plugin-1.0.0.tar.gz (direct tarball)

    Uses azure-storage-blob for all operations with standard Azure credential chain.
    """

    def __init__(
        self,
        url: str,
        mode: SourceMode = "registry",
        cache_dir: Path | None = None,
        blob_service_client: BlobServiceClient | None = None,
    ):
        """Initialize the Azure Blob registry client.

        Args:
            url: Azure URL (az://account/container/path/)
            mode: Source mode - "registry" expects registry.json,
                  "package" expects package.json
            cache_dir: Optional directory for caching downloads
            blob_service_client: Optional Azure blob service client (for testing)
        """
        self._url = url
        self._mode = mode
        self._blob_service_client = blob_service_client
        self._is_direct_tarball: bool = False
        self._tarball_info: dict[str, str] | None = None

        logger.debug("Initializing Azure registry client for %s (mode=%s)", url, mode)

        # Check if this is a direct tarball URL
        if url.endswith(".tar.gz") or url.endswith(".tgz"):
            self._is_direct_tarball = True
            self._account, self._container, self._prefix = self._parse_url(url, is_tarball=True)
            # Extract filename from URL path
            parsed = urlparse(url)
            path = parsed.path.lstrip("/")
            parts = path.split("/", 1)
            filename = Path(parts[1]).name if len(parts) > 1 else Path(path).name
            self._tarball_info = parse_tarball_info(filename)
            logger.debug("Detected direct tarball URL: %s", self._tarball_info)
        else:
            self._account, self._container, self._prefix = self._parse_url(url)

        logger.debug(
            "Azure config: account=%s, container=%s, prefix=%s",
            self._account,
            self._container,
            self._prefix,
        )

        # Set up caching
        if cache_dir is None:
            cache_dir = Path(tempfile.gettempdir()) / "dex-cache" / "azure"
        self._cache = RegistryCache(cache_dir)

    def _get_blob_service_client(self) -> BlobServiceClient:
        """Get or create the Azure Blob Service client."""
        if self._blob_service_client is None:
            credential = DefaultAzureCredential()
            account_url = f"https://{self._account}.blob.core.windows.net"
            self._blob_service_client = BlobServiceClient(account_url, credential=credential)
        return self._blob_service_client

    def _get_container_client(self) -> ContainerClient:
        """Get the container client."""
        return self._get_blob_service_client().get_container_client(self._container)

    @staticmethod
    def _parse_url(url: str, is_tarball: bool = False) -> tuple[str, str, str]:
        """Parse Azure URL into account, container, and prefix/key.

        Args:
            url: Azure URL (az://account/container/path/)
            is_tarball: If True, treat path as a blob key (no trailing slash)

        Returns:
            Tuple of (account, container, prefix/key)

        Raises:
            AzureRegistryError: If URL is invalid
        """
        parsed = urlparse(url)

        if parsed.scheme != "az":
            raise AzureRegistryError(
                f"Invalid Azure URL scheme: {parsed.scheme}", container=None, blob=None
            )

        account = parsed.netloc
        if not account:
            raise AzureRegistryError(
                "Azure URL must include storage account name", container=None, blob=None
            )

        # Path format: /container/prefix/path
        path = parsed.path.lstrip("/")
        parts = path.split("/", 1)

        if not parts or not parts[0]:
            raise AzureRegistryError(
                "Azure URL must include container name", container=None, blob=None
            )

        container = parts[0]
        prefix = parts[1] if len(parts) > 1 else ""

        # For tarballs, keep the path as-is (it's a blob key)
        # For directories, ensure trailing slash
        if not is_tarball and prefix and not prefix.endswith("/"):
            prefix += "/"

        return account, container, prefix

    @property
    def protocol(self) -> str:
        return "az"

    @property
    def account(self) -> str:
        """Get the Azure storage account name."""
        return self._account

    @property
    def container(self) -> str:
        """Get the Azure container name."""
        return self._container

    @property
    def prefix(self) -> str:
        """Get the blob prefix."""
        return self._prefix

    @property
    def mode(self) -> SourceMode:
        """Get the source mode."""
        return self._mode

    def _download_blob(self, blob_name: str) -> bytes:
        """Download a blob from Azure.

        Args:
            blob_name: Blob name/key

        Returns:
            Blob contents as bytes

        Raises:
            AzureRegistryError: If download fails
        """
        try:
            container_client = self._get_container_client()
            blob_client = container_client.get_blob_client(blob_name)
            return blob_client.download_blob().readall()
        except ResourceNotFoundError as e:
            raise AzureRegistryError(
                f"File not found: az://{self._account}/{self._container}/{blob_name}",
                container=self._container,
                blob=blob_name,
            ) from e
        except ClientAuthenticationError as e:
            raise AzureRegistryError(
                f"Authentication failed for az://{self._account}/{self._container}/{blob_name}: {e}",
                container=self._container,
                blob=blob_name,
            ) from e
        except ServiceRequestError as e:
            raise AzureRegistryError(
                f"Network error accessing az://{self._account}/{self._container}/{blob_name}: {e}",
                container=self._container,
                blob=blob_name,
            ) from e
        except Exception as e:
            raise AzureRegistryError(
                f"Failed to download az://{self._account}/{self._container}/{blob_name}: {e}",
                container=self._container,
                blob=blob_name,
            ) from e

    def _download_to_file(self, blob_name: str, dest: Path) -> Path:
        """Download an Azure blob to a local file.

        Args:
            blob_name: Blob name/key
            dest: Destination file path

        Returns:
            Path to downloaded file

        Raises:
            AzureRegistryError: If download fails
        """
        try:
            container_client = self._get_container_client()
            blob_client = container_client.get_blob_client(blob_name)
            dest.parent.mkdir(parents=True, exist_ok=True)
            with open(dest, "wb") as f:
                download_stream = blob_client.download_blob()
                f.write(download_stream.readall())
            return dest
        except ResourceNotFoundError as e:
            raise AzureRegistryError(
                f"File not found: az://{self._account}/{self._container}/{blob_name}",
                container=self._container,
                blob=blob_name,
            ) from e
        except ClientAuthenticationError as e:
            raise AzureRegistryError(
                f"Authentication failed for az://{self._account}/{self._container}/{blob_name}: {e}",
                container=self._container,
                blob=blob_name,
            ) from e
        except ServiceRequestError as e:
            raise AzureRegistryError(
                f"Network error accessing az://{self._account}/{self._container}/{blob_name}: {e}",
                container=self._container,
                blob=blob_name,
            ) from e
        except Exception as e:
            raise AzureRegistryError(
                f"Failed to download az://{self._account}/{self._container}/{blob_name}: {e}",
                container=self._container,
                blob=blob_name,
            ) from e

    def _get_registry_data(self) -> dict[str, Any]:
        """Load and cache registry.json data.

        Returns:
            Parsed registry.json contents

        Raises:
            AzureRegistryError: If registry.json cannot be loaded or is invalid JSON
        """
        registry_url = f"az://{self._account}/{self._container}/{self._prefix}registry.json"
        registry_blob = f"{self._prefix}registry.json"

        # Check cache first
        cached_path = self._cache.get(registry_url)
        if cached_path and cached_path.exists():
            try:
                with open(cached_path, encoding="utf-8") as f:
                    result: dict[str, Any] = json.load(f)
                    return result
            except json.JSONDecodeError as e:
                raise AzureRegistryError(
                    f"Invalid JSON in cached registry.json: {e}",
                    container=self._container,
                    blob=registry_blob,
                ) from e

        # Download and cache
        content = self._download_blob(registry_blob)
        try:
            data: dict[str, Any] = json.loads(content.decode("utf-8"))
        except json.JSONDecodeError as e:
            raise AzureRegistryError(
                f"Invalid JSON in registry.json: {e}",
                container=self._container,
                blob=registry_blob,
            ) from e

        # Cache the file
        with tempfile.NamedTemporaryFile(mode="w", suffix=".json", delete=False) as f:
            json.dump(data, f)
            temp_path = Path(f.name)

        self._cache.put(registry_url, temp_path)
        temp_path.unlink()

        return data

    def _get_package_data(self) -> dict[str, Any]:
        """Load and cache package.json data.

        Returns:
            Parsed package.json contents

        Raises:
            AzureRegistryError: If package.json cannot be loaded or is invalid JSON
        """
        package_url = f"az://{self._account}/{self._container}/{self._prefix}package.json"
        package_blob = f"{self._prefix}package.json"

        # Check cache first
        cached_path = self._cache.get(package_url)
        if cached_path and cached_path.exists():
            try:
                with open(cached_path, encoding="utf-8") as f:
                    result: dict[str, Any] = json.load(f)
                    return result
            except json.JSONDecodeError as e:
                raise AzureRegistryError(
                    f"Invalid JSON in cached package.json: {e}",
                    container=self._container,
                    blob=package_blob,
                ) from e

        # Download and cache
        content = self._download_blob(package_blob)
        try:
            data: dict[str, Any] = json.loads(content.decode("utf-8"))
        except json.JSONDecodeError as e:
            raise AzureRegistryError(
                f"Invalid JSON in package.json: {e}",
                container=self._container,
                blob=package_blob,
            ) from e

        # Cache the content
        with tempfile.NamedTemporaryFile(mode="w", suffix=".json", delete=False) as f:
            json.dump(data, f)
            temp_path = Path(f.name)

        self._cache.put(package_url, temp_path)
        temp_path.unlink()

        return data

    def get_package_info(self, name: str) -> PackageInfo | None:
        """Get package information from Azure.

        Args:
            name: Package name to look up

        Returns:
            PackageInfo if found, None if package doesn't exist

        Raises:
            AzureRegistryError: If there's an auth/network error or invalid JSON
        """
        logger.debug("Getting package info for %s from Azure (mode=%s)", name, self._mode)

        if self._is_direct_tarball:
            logger.debug("Using direct tarball mode")
            return self._get_package_from_tarball(name)
        elif self._mode == "registry":
            logger.debug("Using registry mode (registry.json)")
            return self._get_package_from_registry(name)
        else:
            logger.debug("Using package mode (package.json)")
            return self._get_package_from_package(name)

    def _get_package_from_tarball(self, name: str) -> PackageInfo | None:
        """Get package info from direct tarball URL.

        Returns:
            PackageInfo if found and name matches, None if name doesn't match

        Raises:
            AzureRegistryError: If tarball cannot be downloaded or extracted
        """
        if self._tarball_info is None:
            return None

        tarball_name = self._tarball_info.get("name", "")
        tarball_version = self._tarball_info.get("version", "0.0.0")

        # Check if name matches (with normalization)
        if names_match(tarball_name, name):
            return PackageInfo(
                name=name,
                versions=[tarball_version],
                latest=tarball_version,
            )

        # Try to download and inspect the tarball
        with tempfile.NamedTemporaryFile(suffix=".tar.gz", delete=False) as tmp:
            tmp_path = Path(tmp.name)

        try:
            self._download_to_file(self._prefix, tmp_path)

            with tempfile.TemporaryDirectory() as extract_dir:
                extracted = extract_tarball(tmp_path, Path(extract_dir))
                package_json = extracted / "package.json"

                if package_json.exists():
                    with open(package_json, encoding="utf-8") as f:
                        data = json.load(f)

                    return extract_package_from_manifest_data(data, name)
        finally:
            if tmp_path.exists():
                tmp_path.unlink()

        return None

    def _get_package_from_registry(self, name: str) -> PackageInfo | None:
        """Get package info from registry.json.

        Returns:
            PackageInfo if found, None if package not in registry

        Raises:
            AzureRegistryError: If registry.json cannot be downloaded or parsed
        """
        registry_data = self._get_registry_data()
        return extract_package_from_registry_data(registry_data, name)

    def _get_package_from_package(self, name: str) -> PackageInfo | None:
        """Get package info from package.json (package mode).

        Returns:
            PackageInfo if found and name matches, None if name doesn't match

        Raises:
            AzureRegistryError: If package.json cannot be downloaded or parsed
        """
        package_data = self._get_package_data()
        return extract_package_from_manifest_data(package_data, name)

    def resolve_package(self, name: str, version: str) -> ResolvedPackage | None:
        """Resolve a package to an Azure location.

        Args:
            name: Package name
            version: Version specifier ('latest', exact, or semver range)

        Returns:
            ResolvedPackage with Azure URL, or None if package/version not found

        Raises:
            AzureRegistryError: If there's an auth/network error or invalid JSON
        """
        logger.info("Resolving package %s@%s from Azure", name, version)
        info = self.get_package_info(name)
        if info is None:
            logger.debug("Package %s not found in Azure", name)
            return None

        # Find the best matching version
        resolved_version: str | None
        if version == "latest":
            resolved_version = info.latest
        else:
            resolved_version = find_best_version(version, info.versions)

        if resolved_version is None:
            logger.debug("Version %s not found for package %s", version, name)
            return None

        logger.debug("Resolved %s to version %s", name, resolved_version)

        if self._is_direct_tarball:
            # Direct tarball URL
            return ResolvedPackage(
                name=name,
                version=resolved_version,
                resolved_url=self._url,
                local_path=None,
            )
        elif self._mode == "registry":
            # Registry mode - return Azure URL to tarball
            tarball_blob = f"{self._prefix}{name}-{resolved_version}.tar.gz"
            resolved_url = f"az://{self._account}/{self._container}/{tarball_blob}"

            return ResolvedPackage(
                name=name,
                version=resolved_version,
                resolved_url=resolved_url,
                local_path=None,
            )
        else:
            # Package mode - return Azure URL to directory
            resolved_url = f"az://{self._account}/{self._container}/{self._prefix}"

            return ResolvedPackage(
                name=name,
                version=resolved_version,
                resolved_url=resolved_url,
                local_path=None,
            )

    def fetch_package(self, resolved: ResolvedPackage, dest_dir: Path) -> Path:
        """Fetch a package from Azure to a local directory."""
        dest_dir.mkdir(parents=True, exist_ok=True)

        # Check cache first
        cached_path = self._cache.get(resolved.resolved_url)
        if cached_path and cached_path.exists():
            if cached_path.is_dir():
                # Copy cached directory
                plugin_dir = dest_dir / resolved.name
                if plugin_dir.exists():
                    shutil.rmtree(plugin_dir)
                shutil.copytree(cached_path, plugin_dir)
                return plugin_dir
            else:
                # Extract cached tarball
                return extract_tarball(cached_path, dest_dir)

        # Parse URL to get blob name
        parsed = urlparse(resolved.resolved_url)
        path = parsed.path.lstrip("/")
        parts = path.split("/", 1)
        blob_name = parts[1] if len(parts) > 1 else ""

        if resolved.resolved_url.endswith(".tar.gz"):
            # Download tarball
            with tempfile.NamedTemporaryFile(suffix=".tar.gz", delete=False) as tmp:
                tmp_path = Path(tmp.name)

            try:
                self._download_to_file(blob_name, tmp_path)

                # Compute integrity
                integrity = compute_integrity(tmp_path)

                # Cache the tarball
                cached = self._cache.put(
                    resolved.resolved_url, tmp_path, metadata={"integrity": integrity}
                )

                # Extract
                return extract_tarball(cached, dest_dir)
            finally:
                if tmp_path.exists():
                    tmp_path.unlink()
        else:
            # Download directory contents
            return self._download_directory(blob_name, dest_dir / resolved.name)

    def _download_directory(self, prefix: str, dest_dir: Path) -> Path:
        """Download all blobs under an Azure prefix to a local directory.

        Args:
            prefix: Blob name prefix
            dest_dir: Local destination directory

        Returns:
            Path to downloaded directory
        """
        container_client = self._get_container_client()
        dest_dir.mkdir(parents=True, exist_ok=True)

        # Ensure prefix ends with /
        if not prefix.endswith("/"):
            prefix += "/"

        # List and download all blobs
        blobs = container_client.list_blobs(name_starts_with=prefix)

        for blob in blobs:
            blob_name = blob.name
            # Get relative path from prefix
            relative = blob_name[len(prefix) :]
            if not relative:
                continue

            local_path = dest_dir / relative
            local_path.parent.mkdir(parents=True, exist_ok=True)
            self._download_to_file(blob_name, local_path)

        # Cache the directory
        self._cache.put(f"az://{self._account}/{self._container}/{prefix}", dest_dir)

        return dest_dir

    def list_packages(self) -> list[str]:
        """List all packages in the Azure source.

        Returns:
            List of package names

        Raises:
            AzureRegistryError: If there's an auth/network error or invalid JSON
        """
        if self._is_direct_tarball:
            # Direct tarball - return parsed name
            if self._tarball_info:
                name = self._tarball_info.get("name")
                return [name] if name else []
            return []
        elif self._mode == "registry":
            registry_data = self._get_registry_data()
            return list(registry_data.get("packages", {}).keys())
        else:
            # Package mode - single package
            package_data = self._get_package_data()
            name = package_data.get("name")
            return [name] if name else []
