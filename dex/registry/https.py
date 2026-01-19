"""HTTPS registry client for remote packages."""

from __future__ import annotations

import json
import logging
import shutil
import ssl
import tempfile
from pathlib import Path
from typing import Any
from urllib.error import HTTPError, URLError
from urllib.parse import urlparse
from urllib.request import Request, urlopen

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


class HttpsRegistryError(Exception):
    """Error interacting with HTTPS registry."""

    def __init__(self, message: str, url: str | None = None, status_code: int | None = None):
        self.url = url
        self.status_code = status_code
        super().__init__(message)


class HttpsRegistryClient(RegistryClient):
    """Registry client for HTTPS-hosted packages.

    Supports two modes (specified explicitly, not auto-detected):
    1. Registry mode: URL points to a directory with registry.json and .tar.gz files
    2. Package mode: URL points to a directory with package.json

    Also handles direct tarball URLs (detected from .tar.gz extension).

    URL formats:
    - https://example.com/path/to/registry/ (registry mode)
    - https://example.com/path/to/package/ (package mode)
    - https://example.com/path/to/package.tar.gz (direct tarball)

    Supports optional authentication via headers (Bearer tokens, Basic auth, etc.).
    """

    DEFAULT_TIMEOUT = 30  # seconds

    def __init__(
        self,
        url: str,
        mode: SourceMode = "registry",
        cache_dir: Path | None = None,
        headers: dict[str, str] | None = None,
        timeout: int | None = None,
    ):
        """Initialize the HTTPS registry client.

        Args:
            url: HTTPS URL (https://example.com/path/)
            mode: Source mode - "registry" expects registry.json,
                  "package" expects package.json
            cache_dir: Optional directory for caching downloads
            headers: Optional HTTP headers (for authentication, etc.)
            timeout: Request timeout in seconds (default: 30)
        """
        self._original_url = url
        self._mode = mode
        self._headers = headers or {}
        self._timeout = timeout or self.DEFAULT_TIMEOUT
        self._is_direct_tarball: bool = False

        logger.info("Initializing HTTPS registry client for %s (mode=%s)", url, mode)

        # Parse URL to determine mode
        self._parsed = urlparse(url)

        # Validate URL scheme
        if self._parsed.scheme != "https":
            raise HttpsRegistryError(
                f"Invalid URL scheme: {self._parsed.scheme} (expected https)",
                url=url,
            )

        # Check if this is a direct tarball URL
        self._tarball_info: dict[str, str] | None = None
        if url.endswith(".tar.gz") or url.endswith(".tgz"):
            self._is_direct_tarball = True
            self._url = url
            # Extract package info from tarball name
            filename = Path(self._parsed.path).name
            self._tarball_info = parse_tarball_info(filename)
            logger.debug("Detected direct tarball URL: %s", self._tarball_info)
        else:
            self._url = url.rstrip("/") + "/"

        # Set up caching
        if cache_dir is None:
            cache_dir = Path(tempfile.gettempdir()) / "dex-cache" / "https"
        self._cache = RegistryCache(cache_dir)
        logger.debug("Using cache directory: %s", cache_dir)

        # Create SSL context
        self._ssl_context = ssl.create_default_context()

    @property
    def protocol(self) -> str:
        return "https"

    @property
    def base_url(self) -> str:
        """Get the base URL."""
        return self._url

    @property
    def mode(self) -> SourceMode:
        """Get the source mode."""
        return self._mode

    def _make_request(self, url: str, method: str = "GET") -> bytes:
        """Make an HTTP request.

        Args:
            url: URL to request
            method: HTTP method (default: GET)

        Returns:
            Response body as bytes

        Raises:
            HttpsRegistryError: If request fails
        """
        logger.debug("Making %s request to %s", method, url)
        try:
            request = Request(url, method=method)
            for key, value in self._headers.items():
                request.add_header(key, value)

            with urlopen(request, timeout=self._timeout, context=self._ssl_context) as response:
                result: bytes = response.read()
                logger.debug("Request successful, received %d bytes", len(result))
                return result
        except HTTPError as e:
            logger.error("HTTP error %d: %s for %s", e.code, e.reason, url)
            raise HttpsRegistryError(
                f"HTTP {e.code}: {e.reason} for {url}",
                url=url,
                status_code=e.code,
            ) from e
        except URLError as e:
            logger.error("Failed to connect to %s: %s", url, e.reason)
            raise HttpsRegistryError(
                f"Failed to connect to {url}: {e.reason}",
                url=url,
            ) from e
        except TimeoutError as e:
            logger.error("Request timed out for %s", url)
            raise HttpsRegistryError(
                f"Request timed out for {url}",
                url=url,
            ) from e

    def _head_request(self, url: str) -> bool:
        """Check if a URL exists using HEAD request.

        Args:
            url: URL to check

        Returns:
            True if URL exists (2xx response), False otherwise
        """
        try:
            request = Request(url, method="HEAD")
            for key, value in self._headers.items():
                request.add_header(key, value)

            with urlopen(request, timeout=self._timeout, context=self._ssl_context) as response:
                status: int = response.status
                return 200 <= status < 300
        except (HTTPError, URLError, TimeoutError):
            return False

    def _download_to_file(self, url: str, dest: Path) -> Path:
        """Download a URL to a local file.

        Args:
            url: URL to download
            dest: Destination file path

        Returns:
            Path to downloaded file

        Raises:
            HttpsRegistryError: If download fails
        """
        content = self._make_request(url)
        dest.parent.mkdir(parents=True, exist_ok=True)
        dest.write_bytes(content)
        return dest

    def _get_registry_data(self) -> dict[str, Any]:
        """Load and cache registry.json data.

        Returns:
            Parsed registry.json contents

        Raises:
            HttpsRegistryError: If registry.json cannot be loaded or is invalid JSON
        """
        registry_url = f"{self._url}registry.json"

        # Check cache first
        cached_path = self._cache.get(registry_url)
        if cached_path and cached_path.exists():
            try:
                with open(cached_path, encoding="utf-8") as f:
                    result: dict[str, Any] = json.load(f)
                    return result
            except json.JSONDecodeError as e:
                raise HttpsRegistryError(
                    f"Invalid JSON in cached registry.json: {e}",
                    url=registry_url,
                ) from e

        # Download and cache
        content = self._make_request(registry_url)
        try:
            data: dict[str, Any] = json.loads(content.decode("utf-8"))
        except json.JSONDecodeError as e:
            raise HttpsRegistryError(
                f"Invalid JSON in registry.json: {e}",
                url=registry_url,
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
            HttpsRegistryError: If package.json cannot be loaded or is invalid JSON
        """
        package_url = f"{self._url}package.json"

        # Check cache first
        cached_path = self._cache.get(package_url)
        if cached_path and cached_path.exists():
            try:
                with open(cached_path, encoding="utf-8") as f:
                    result: dict[str, Any] = json.load(f)
                    return result
            except json.JSONDecodeError as e:
                raise HttpsRegistryError(
                    f"Invalid JSON in cached package.json: {e}",
                    url=package_url,
                ) from e

        # Download and cache
        content = self._make_request(package_url)
        try:
            data: dict[str, Any] = json.loads(content.decode("utf-8"))
        except json.JSONDecodeError as e:
            raise HttpsRegistryError(
                f"Invalid JSON in package.json: {e}",
                url=package_url,
            ) from e

        # Cache the content
        with tempfile.NamedTemporaryFile(mode="w", suffix=".json", delete=False) as f:
            json.dump(data, f)
            temp_path = Path(f.name)

        self._cache.put(package_url, temp_path)
        temp_path.unlink()

        return data

    def get_package_info(self, name: str) -> PackageInfo | None:
        """Get package information from HTTPS.

        Args:
            name: Package name to look up

        Returns:
            PackageInfo if found, None if package doesn't exist

        Raises:
            HttpsRegistryError: If there's a network error or invalid JSON
        """
        logger.debug("Getting package info for '%s' from HTTPS (mode=%s)", name, self._mode)

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
            HttpsRegistryError: If tarball cannot be downloaded or extracted
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
            self._download_to_file(self._url, tmp_path)

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
            HttpsRegistryError: If registry.json cannot be downloaded or parsed
        """
        registry_data = self._get_registry_data()
        return extract_package_from_registry_data(registry_data, name)

    def _get_package_from_package(self, name: str) -> PackageInfo | None:
        """Get package info from package.json (package mode).

        Returns:
            PackageInfo if found and name matches, None if name doesn't match

        Raises:
            HttpsRegistryError: If package.json cannot be downloaded or parsed
        """
        package_data = self._get_package_data()
        return extract_package_from_manifest_data(package_data, name)

    def resolve_package(self, name: str, version: str) -> ResolvedPackage | None:
        """Resolve a package to an HTTPS location.

        Args:
            name: Package name
            version: Version specifier ('latest', exact, or semver range)

        Returns:
            ResolvedPackage with HTTPS URL, or None if package/version not found

        Raises:
            HttpsRegistryError: If there's a network error or invalid JSON
        """
        logger.info("Resolving package '%s' version '%s' from HTTPS", name, version)
        info = self.get_package_info(name)
        if info is None:
            logger.warning("Package '%s' not found in HTTPS registry", name)
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

        if self._is_direct_tarball:
            # Direct tarball URL - return the URL as-is
            logger.info(
                "Resolved package '%s' to version %s (direct tarball)", name, resolved_version
            )
            return ResolvedPackage(
                name=name,
                version=resolved_version,
                resolved_url=self._url,
                local_path=None,
            )
        elif self._mode == "registry":
            # Registry mode - return URL to versioned tarball
            resolved_url = f"{self._url}{name}-{resolved_version}.tar.gz"
            logger.info(
                "Resolved package '%s' to version %s at %s", name, resolved_version, resolved_url
            )

            return ResolvedPackage(
                name=name,
                version=resolved_version,
                resolved_url=resolved_url,
                local_path=None,
            )
        else:
            # Package mode
            logger.info(
                "Resolved package '%s' to version %s (package mode)", name, resolved_version
            )
            return ResolvedPackage(
                name=name,
                version=resolved_version,
                resolved_url=self._url,
                local_path=None,
            )

    def fetch_package(self, resolved: ResolvedPackage, dest_dir: Path) -> Path:
        """Fetch a package from HTTPS to a local directory."""
        logger.info(
            "Fetching package '%s' v%s from HTTPS to %s", resolved.name, resolved.version, dest_dir
        )
        dest_dir.mkdir(parents=True, exist_ok=True)

        # Check cache first
        cached_path = self._cache.get(resolved.resolved_url)
        if cached_path and cached_path.exists():
            logger.debug("Using cached download from %s", cached_path)
            if cached_path.is_dir():
                # Copy cached directory
                plugin_dir = dest_dir / resolved.name
                if plugin_dir.exists():
                    shutil.rmtree(plugin_dir)
                shutil.copytree(cached_path, plugin_dir)
                logger.info("Package '%s' fetched from cache", resolved.name)
                return plugin_dir
            else:
                # Extract cached tarball
                logger.info("Package '%s' extracted from cached tarball", resolved.name)
                return extract_tarball(cached_path, dest_dir)

        # Check if URL is a tarball
        is_tarball = resolved.resolved_url.endswith(".tar.gz") or resolved.resolved_url.endswith(
            ".tgz"
        )

        if is_tarball:
            # Download tarball
            logger.debug("Downloading tarball from %s", resolved.resolved_url)
            with tempfile.NamedTemporaryFile(suffix=".tar.gz", delete=False) as tmp:
                tmp_path = Path(tmp.name)

            try:
                self._download_to_file(resolved.resolved_url, tmp_path)

                # Compute integrity
                integrity = compute_integrity(tmp_path)
                logger.debug("Computed integrity: %s", integrity)

                # Cache the tarball
                cached = self._cache.put(
                    resolved.resolved_url, tmp_path, metadata={"integrity": integrity}
                )

                # Extract
                logger.info("Package '%s' downloaded and cached", resolved.name)
                return extract_tarball(cached, dest_dir)
            finally:
                if tmp_path.exists():
                    tmp_path.unlink()
        else:
            # Direct directory mode - HTTP doesn't support directory listing
            # Try to find a tarball at a conventional location
            tarball_url = f"{resolved.resolved_url.rstrip('/')}.tar.gz"
            alt_tarball_url = f"{resolved.resolved_url}{resolved.name}-{resolved.version}.tar.gz"

            logger.debug("Looking for tarball at conventional locations")
            for url in [tarball_url, alt_tarball_url]:
                if self._head_request(url):
                    logger.debug("Found tarball at %s", url)
                    with tempfile.NamedTemporaryFile(suffix=".tar.gz", delete=False) as tmp:
                        tmp_path = Path(tmp.name)

                    try:
                        self._download_to_file(url, tmp_path)
                        integrity = compute_integrity(tmp_path)
                        cached = self._cache.put(url, tmp_path, metadata={"integrity": integrity})
                        logger.info("Package '%s' downloaded and cached", resolved.name)
                        return extract_tarball(cached, dest_dir)
                    finally:
                        if tmp_path.exists():
                            tmp_path.unlink()

            logger.error("Cannot fetch directory from HTTPS, no tarball found")
            raise HttpsRegistryError(
                f"Cannot fetch directory from HTTPS. Provide a tarball URL or "
                f"ensure a tarball exists at {tarball_url}",
                url=resolved.resolved_url,
            )

    def list_packages(self) -> list[str]:
        """List all packages in the HTTPS source.

        Returns:
            List of package names

        Raises:
            HttpsRegistryError: If there's a network error or invalid JSON
        """
        if self._is_direct_tarball:
            # Direct tarball - return the parsed name
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
