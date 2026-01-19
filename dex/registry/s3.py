"""S3 registry client for remote packages."""

from __future__ import annotations

import json
import logging
import shutil
import tempfile
from pathlib import Path
from typing import TYPE_CHECKING, Any
from urllib.parse import urlparse

from botocore.exceptions import ClientError

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

if TYPE_CHECKING:
    from mypy_boto3_s3.client import S3Client

logger = logging.getLogger(__name__)


class S3RegistryError(Exception):
    """Error interacting with S3 registry."""

    def __init__(self, message: str, bucket: str | None = None, key: str | None = None):
        self.bucket = bucket
        self.key = key
        super().__init__(message)


class S3RegistryClient(RegistryClient):
    """Registry client for S3-hosted packages.

    Supports two modes (specified explicitly, not auto-detected):
    1. Registry mode: S3 path contains registry.json and .tar.gz files
    2. Package mode: S3 path points to a directory with package.json

    Also handles direct tarball URLs (detected from .tar.gz extension).

    URL formats:
    - s3://bucket-name/path/to/registry/ (registry mode)
    - s3://bucket-name/path/to/plugin/ (package mode)
    - s3://bucket-name/path/to/plugin-1.0.0.tar.gz (direct tarball)

    Uses boto3 for all S3 operations with standard AWS credential chain.
    """

    def __init__(
        self,
        url: str,
        mode: SourceMode = "registry",
        cache_dir: Path | None = None,
        s3_client: S3Client | None = None,
    ):
        """Initialize the S3 registry client.

        Args:
            url: S3 URL (s3://bucket/path/ or s3://bucket/path/file.tar.gz)
            mode: Source mode - "registry" expects registry.json,
                  "package" expects package.json
            cache_dir: Optional directory for caching downloads
            s3_client: Optional boto3 S3 client (for testing)
        """
        self._url = url
        self._mode = mode
        self._s3_client = s3_client
        self._is_direct_tarball: bool = False
        self._tarball_info: dict[str, str] | None = None

        logger.debug("Initializing S3 registry client for %s (mode=%s)", url, mode)

        # Check if this is a direct tarball URL
        if url.endswith(".tar.gz") or url.endswith(".tgz"):
            self._is_direct_tarball = True
            self._bucket, self._prefix = self._parse_url(url, is_tarball=True)
            filename = Path(urlparse(url).path).name
            self._tarball_info = parse_tarball_info(filename)
            logger.debug("Detected direct tarball URL: %s", self._tarball_info)
        else:
            self._bucket, self._prefix = self._parse_url(url)

        logger.debug("S3 config: bucket=%s, prefix=%s", self._bucket, self._prefix)

        # Set up caching
        if cache_dir is None:
            cache_dir = Path(tempfile.gettempdir()) / "dex-cache" / "s3"
        self._cache = RegistryCache(cache_dir)

    def _get_s3_client(self) -> S3Client:
        """Get or create the boto3 S3 client."""
        if self._s3_client is None:
            import boto3

            self._s3_client = boto3.client("s3")
        return self._s3_client

    @staticmethod
    def _parse_url(url: str, is_tarball: bool = False) -> tuple[str, str]:
        """Parse S3 URL into bucket and prefix/key.

        Args:
            url: S3 URL (s3://bucket/path/ or s3://bucket/path/file.tar.gz)
            is_tarball: If True, treat path as a file key (no trailing slash)

        Returns:
            Tuple of (bucket, prefix/key)

        Raises:
            S3RegistryError: If URL is invalid
        """
        parsed = urlparse(url)

        if parsed.scheme != "s3":
            raise S3RegistryError(f"Invalid S3 URL scheme: {parsed.scheme}", bucket=None, key=None)

        bucket = parsed.netloc
        if not bucket:
            raise S3RegistryError("S3 URL must include bucket name", bucket=None, key=None)

        # Remove leading slash
        prefix = parsed.path.lstrip("/")

        # For tarballs, keep the path as-is (it's a file key)
        # For directories, ensure trailing slash
        if not is_tarball and prefix and not prefix.endswith("/"):
            prefix += "/"

        return bucket, prefix

    @property
    def protocol(self) -> str:
        return "s3"

    @property
    def bucket(self) -> str:
        """Get the S3 bucket name."""
        return self._bucket

    @property
    def prefix(self) -> str:
        """Get the S3 key prefix."""
        return self._prefix

    @property
    def mode(self) -> SourceMode:
        """Get the source mode."""
        return self._mode

    def _download_file(self, key: str) -> bytes:
        """Download a file from S3.

        Args:
            key: S3 object key

        Returns:
            File contents as bytes

        Raises:
            S3RegistryError: If download fails (auth, network, not found, etc.)
        """
        try:
            s3 = self._get_s3_client()
            response = s3.get_object(Bucket=self._bucket, Key=key)
            return response["Body"].read()
        except ClientError as e:
            error_code = e.response.get("Error", {}).get("Code", "")
            if error_code == "NoSuchKey":
                raise S3RegistryError(
                    f"File not found: s3://{self._bucket}/{key}",
                    bucket=self._bucket,
                    key=key,
                ) from e
            elif error_code in ("AccessDenied", "403"):
                raise S3RegistryError(
                    f"Access denied to s3://{self._bucket}/{key}",
                    bucket=self._bucket,
                    key=key,
                ) from e
            else:
                raise S3RegistryError(
                    f"Failed to download s3://{self._bucket}/{key}: {e}",
                    bucket=self._bucket,
                    key=key,
                ) from e
        except Exception as e:
            raise S3RegistryError(
                f"Failed to download s3://{self._bucket}/{key}: {e}",
                bucket=self._bucket,
                key=key,
            ) from e

    def _download_to_file(self, key: str, dest: Path) -> Path:
        """Download an S3 object to a local file.

        Args:
            key: S3 object key
            dest: Destination file path

        Returns:
            Path to downloaded file

        Raises:
            S3RegistryError: If download fails
        """
        try:
            s3 = self._get_s3_client()
            dest.parent.mkdir(parents=True, exist_ok=True)
            s3.download_file(self._bucket, key, str(dest))
            return dest
        except ClientError as e:
            error_code = e.response.get("Error", {}).get("Code", "")
            if error_code == "NoSuchKey":
                raise S3RegistryError(
                    f"File not found: s3://{self._bucket}/{key}",
                    bucket=self._bucket,
                    key=key,
                ) from e
            elif error_code in ("AccessDenied", "403"):
                raise S3RegistryError(
                    f"Access denied to s3://{self._bucket}/{key}",
                    bucket=self._bucket,
                    key=key,
                ) from e
            else:
                raise S3RegistryError(
                    f"Failed to download s3://{self._bucket}/{key}: {e}",
                    bucket=self._bucket,
                    key=key,
                ) from e
        except Exception as e:
            raise S3RegistryError(
                f"Failed to download s3://{self._bucket}/{key}: {e}",
                bucket=self._bucket,
                key=key,
            ) from e

    def _get_registry_data(self) -> dict[str, Any]:
        """Load and cache registry.json data.

        Returns:
            Parsed registry.json contents

        Raises:
            S3RegistryError: If registry.json cannot be loaded or is invalid JSON
        """
        registry_url = f"s3://{self._bucket}/{self._prefix}registry.json"

        # Check cache first
        cached_path = self._cache.get(registry_url)
        if cached_path and cached_path.exists():
            try:
                with open(cached_path, encoding="utf-8") as f:
                    result: dict[str, Any] = json.load(f)
                    return result
            except json.JSONDecodeError as e:
                raise S3RegistryError(
                    f"Invalid JSON in cached registry.json: {e}",
                    bucket=self._bucket,
                    key=f"{self._prefix}registry.json",
                ) from e

        # Download and cache
        registry_key = f"{self._prefix}registry.json"
        content = self._download_file(registry_key)
        try:
            data: dict[str, Any] = json.loads(content.decode("utf-8"))
        except json.JSONDecodeError as e:
            raise S3RegistryError(
                f"Invalid JSON in registry.json: {e}",
                bucket=self._bucket,
                key=registry_key,
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
            S3RegistryError: If package.json cannot be loaded or is invalid JSON
        """
        package_url = f"s3://{self._bucket}/{self._prefix}package.json"

        # Check cache first
        cached_path = self._cache.get(package_url)
        if cached_path and cached_path.exists():
            try:
                with open(cached_path, encoding="utf-8") as f:
                    result: dict[str, Any] = json.load(f)
                    return result
            except json.JSONDecodeError as e:
                raise S3RegistryError(
                    f"Invalid JSON in cached package.json: {e}",
                    bucket=self._bucket,
                    key=f"{self._prefix}package.json",
                ) from e

        # Download and cache
        package_key = f"{self._prefix}package.json"
        content = self._download_file(package_key)
        try:
            data: dict[str, Any] = json.loads(content.decode("utf-8"))
        except json.JSONDecodeError as e:
            raise S3RegistryError(
                f"Invalid JSON in package.json: {e}",
                bucket=self._bucket,
                key=package_key,
            ) from e

        # Cache the content
        with tempfile.NamedTemporaryFile(mode="w", suffix=".json", delete=False) as f:
            json.dump(data, f)
            temp_path = Path(f.name)

        self._cache.put(package_url, temp_path)
        temp_path.unlink()

        return data

    def get_package_info(self, name: str) -> PackageInfo | None:
        """Get package information from S3.

        Args:
            name: Package name to look up

        Returns:
            PackageInfo if found, None if package doesn't exist

        Raises:
            S3RegistryError: If there's an auth/network error or invalid JSON
        """
        logger.debug("Getting package info for %s from S3 (mode=%s)", name, self._mode)

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
            S3RegistryError: If tarball cannot be downloaded or extracted
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
            S3RegistryError: If registry.json cannot be downloaded or parsed
        """
        registry_data = self._get_registry_data()
        return extract_package_from_registry_data(registry_data, name)

    def _get_package_from_package(self, name: str) -> PackageInfo | None:
        """Get package info from package.json (package mode).

        Returns:
            PackageInfo if found and name matches, None if name doesn't match

        Raises:
            S3RegistryError: If package.json cannot be downloaded or parsed
        """
        package_data = self._get_package_data()
        return extract_package_from_manifest_data(package_data, name)

    def resolve_package(self, name: str, version: str) -> ResolvedPackage | None:
        """Resolve a package to an S3 location.

        Args:
            name: Package name
            version: Version specifier ('latest', exact, or semver range)

        Returns:
            ResolvedPackage with S3 URL, or None if package/version not found

        Raises:
            S3RegistryError: If there's an auth/network error or invalid JSON
        """
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

        if self._is_direct_tarball:
            # Direct tarball URL
            return ResolvedPackage(
                name=name,
                version=resolved_version,
                resolved_url=self._url,
                local_path=None,
            )
        elif self._mode == "registry":
            # Registry mode - return S3 URL to tarball
            tarball_key = f"{self._prefix}{name}-{resolved_version}.tar.gz"
            resolved_url = f"s3://{self._bucket}/{tarball_key}"

            return ResolvedPackage(
                name=name,
                version=resolved_version,
                resolved_url=resolved_url,
                local_path=None,
            )
        else:
            # Package mode - return S3 URL to directory
            resolved_url = f"s3://{self._bucket}/{self._prefix}"

            return ResolvedPackage(
                name=name,
                version=resolved_version,
                resolved_url=resolved_url,
                local_path=None,
            )

    def fetch_package(self, resolved: ResolvedPackage, dest_dir: Path) -> Path:
        """Fetch a package from S3 to a local directory."""
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

        # Parse URL to get key
        parsed = urlparse(resolved.resolved_url)
        key = parsed.path.lstrip("/")

        if resolved.resolved_url.endswith(".tar.gz"):
            # Download tarball
            with tempfile.NamedTemporaryFile(suffix=".tar.gz", delete=False) as tmp:
                tmp_path = Path(tmp.name)

            try:
                self._download_to_file(key, tmp_path)

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
            return self._download_directory(key, dest_dir / resolved.name)

    def _download_directory(self, prefix: str, dest_dir: Path) -> Path:
        """Download all files under an S3 prefix to a local directory.

        Args:
            prefix: S3 key prefix
            dest_dir: Local destination directory

        Returns:
            Path to downloaded directory
        """
        s3 = self._get_s3_client()
        dest_dir.mkdir(parents=True, exist_ok=True)

        # Ensure prefix ends with /
        if not prefix.endswith("/"):
            prefix += "/"

        # List and download all objects
        paginator = s3.get_paginator("list_objects_v2")

        for page in paginator.paginate(Bucket=self._bucket, Prefix=prefix):
            for obj in page.get("Contents", []):
                key = obj["Key"]
                # Get relative path from prefix
                relative = key[len(prefix) :]
                if not relative:
                    continue

                local_path = dest_dir / relative
                local_path.parent.mkdir(parents=True, exist_ok=True)
                s3.download_file(self._bucket, key, str(local_path))

        # Cache the directory
        self._cache.put(f"s3://{self._bucket}/{prefix}", dest_dir)

        return dest_dir

    def list_packages(self) -> list[str]:
        """List all packages in the S3 source.

        Returns:
            List of package names

        Raises:
            S3RegistryError: If there's an auth/network error or invalid JSON
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
