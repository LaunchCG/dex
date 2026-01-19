"""Tests for dex.registry.s3 module."""

import io
import json
from pathlib import Path
from unittest.mock import MagicMock

import pytest

from dex.registry.s3 import S3RegistryClient, S3RegistryError


class TestS3RegistryClientParseUrl:
    """Tests for S3 URL parsing."""

    def test_parses_basic_s3_url(self):
        """Parses basic s3://bucket/path/ URL."""
        bucket, prefix = S3RegistryClient._parse_url("s3://my-bucket/path/to/registry/")

        assert bucket == "my-bucket"
        assert prefix == "path/to/registry/"

    def test_parses_s3_url_without_trailing_slash(self):
        """Adds trailing slash to prefix if missing."""
        bucket, prefix = S3RegistryClient._parse_url("s3://my-bucket/path/to/registry")

        assert bucket == "my-bucket"
        assert prefix == "path/to/registry/"

    def test_parses_s3_url_with_bucket_only(self):
        """Parses s3://bucket with no path."""
        bucket, prefix = S3RegistryClient._parse_url("s3://my-bucket")

        assert bucket == "my-bucket"
        assert prefix == ""

    def test_parses_s3_url_with_single_path_segment(self):
        """Parses s3://bucket/path URL."""
        bucket, prefix = S3RegistryClient._parse_url("s3://my-bucket/plugins")

        assert bucket == "my-bucket"
        assert prefix == "plugins/"

    def test_raises_for_invalid_scheme(self):
        """Raises error for non-s3 scheme."""
        with pytest.raises(S3RegistryError, match="Invalid S3 URL scheme"):
            S3RegistryClient._parse_url("https://my-bucket/path")

    def test_raises_for_missing_bucket(self):
        """Raises error when bucket is missing."""
        with pytest.raises(S3RegistryError, match="must include bucket name"):
            S3RegistryClient._parse_url("s3:///path/only")


class TestS3RegistryClientInit:
    """Tests for S3RegistryClient initialization."""

    def test_parses_url_on_init(self, temp_dir: Path):
        """Parses URL and extracts bucket/prefix."""
        mock_s3 = MagicMock()
        client = S3RegistryClient(
            "s3://my-bucket/registry/",
            cache_dir=temp_dir / "cache",
            s3_client=mock_s3,
        )

        assert client.bucket == "my-bucket"
        assert client.prefix == "registry/"
        assert client.protocol == "s3"

    def test_default_mode_is_registry(self, temp_dir: Path):
        """Default mode is 'registry'."""
        mock_s3 = MagicMock()
        client = S3RegistryClient(
            "s3://my-bucket/",
            cache_dir=temp_dir / "cache",
            s3_client=mock_s3,
        )

        assert client.mode == "registry"

    def test_explicit_package_mode(self, temp_dir: Path):
        """Can set mode to 'package' explicitly."""
        mock_s3 = MagicMock()
        client = S3RegistryClient(
            "s3://my-bucket/plugin/",
            mode="package",
            cache_dir=temp_dir / "cache",
            s3_client=mock_s3,
        )

        assert client.mode == "package"


class TestS3RegistryClientGetPackageInfo:
    """Tests for S3RegistryClient.get_package_info()."""

    def test_gets_package_from_registry_json(self, temp_dir: Path):
        """Gets package info from registry.json."""
        mock_s3 = MagicMock()

        # Mock get_object to return registry.json
        registry_data = {
            "packages": {
                "test-plugin": {
                    "versions": ["1.0.0", "2.0.0"],
                    "latest": "2.0.0",
                }
            }
        }
        mock_s3.get_object.return_value = {"Body": io.BytesIO(json.dumps(registry_data).encode())}

        client = S3RegistryClient(
            "s3://my-bucket/registry/",
            mode="registry",
            cache_dir=temp_dir / "cache",
            s3_client=mock_s3,
        )

        info = client.get_package_info("test-plugin")

        assert info is not None
        assert info.name == "test-plugin"
        assert info.versions == ["1.0.0", "2.0.0"]
        assert info.latest == "2.0.0"

    def test_returns_none_for_unknown_package(self, temp_dir: Path):
        """Returns None for package not in registry."""
        mock_s3 = MagicMock()

        registry_data: dict[str, dict[str, str]] = {"packages": {}}
        mock_s3.get_object.return_value = {"Body": io.BytesIO(json.dumps(registry_data).encode())}

        client = S3RegistryClient(
            "s3://my-bucket/registry/",
            mode="registry",
            cache_dir=temp_dir / "cache",
            s3_client=mock_s3,
        )

        info = client.get_package_info("nonexistent")

        assert info is None

    def test_gets_package_from_package_mode(self, temp_dir: Path):
        """Gets package info from package.json in package mode."""
        mock_s3 = MagicMock()

        # Mock get_object to return package.json
        package_data = {
            "name": "direct-plugin",
            "version": "1.5.0",
            "description": "A direct plugin",
        }
        mock_s3.get_object.return_value = {"Body": io.BytesIO(json.dumps(package_data).encode())}

        client = S3RegistryClient(
            "s3://my-bucket/plugin/",
            mode="package",
            cache_dir=temp_dir / "cache",
            s3_client=mock_s3,
        )

        info = client.get_package_info("direct-plugin")

        assert info is not None
        assert info.name == "direct-plugin"
        assert info.versions == ["1.5.0"]
        assert info.latest == "1.5.0"

    def test_returns_none_for_wrong_name_in_package_mode(self, temp_dir: Path):
        """Returns None when package name doesn't match in package mode."""
        mock_s3 = MagicMock()

        package_data = {"name": "actual-name", "version": "1.0.0"}
        mock_s3.get_object.return_value = {"Body": io.BytesIO(json.dumps(package_data).encode())}

        client = S3RegistryClient(
            "s3://my-bucket/plugin/",
            mode="package",
            cache_dir=temp_dir / "cache",
            s3_client=mock_s3,
        )

        info = client.get_package_info("wrong-name")

        assert info is None


class TestS3RegistryClientResolvePackage:
    """Tests for S3RegistryClient.resolve_package()."""

    def test_resolves_latest_version(self, temp_dir: Path):
        """Resolves 'latest' to the latest version."""
        mock_s3 = MagicMock()

        registry_data = {
            "packages": {"test-plugin": {"versions": ["1.0.0", "2.0.0"], "latest": "2.0.0"}}
        }
        mock_s3.get_object.return_value = {"Body": io.BytesIO(json.dumps(registry_data).encode())}

        client = S3RegistryClient(
            "s3://my-bucket/registry/",
            mode="registry",
            cache_dir=temp_dir / "cache",
            s3_client=mock_s3,
        )

        resolved = client.resolve_package("test-plugin", "latest")

        assert resolved is not None
        assert resolved.name == "test-plugin"
        assert resolved.version == "2.0.0"
        assert resolved.resolved_url == "s3://my-bucket/registry/test-plugin-2.0.0.tar.gz"

    def test_resolves_specific_version(self, temp_dir: Path):
        """Resolves specific version."""
        mock_s3 = MagicMock()

        registry_data = {
            "packages": {"test-plugin": {"versions": ["1.0.0", "2.0.0"], "latest": "2.0.0"}}
        }
        mock_s3.get_object.return_value = {"Body": io.BytesIO(json.dumps(registry_data).encode())}

        client = S3RegistryClient(
            "s3://my-bucket/registry/",
            mode="registry",
            cache_dir=temp_dir / "cache",
            s3_client=mock_s3,
        )

        resolved = client.resolve_package("test-plugin", "1.0.0")

        assert resolved is not None
        assert resolved.version == "1.0.0"
        assert "1.0.0" in resolved.resolved_url

    def test_resolves_version_range(self, temp_dir: Path):
        """Resolves version range to best match."""
        mock_s3 = MagicMock()

        registry_data = {
            "packages": {
                "test-plugin": {"versions": ["1.0.0", "1.5.0", "2.0.0"], "latest": "2.0.0"}
            }
        }
        mock_s3.get_object.return_value = {"Body": io.BytesIO(json.dumps(registry_data).encode())}

        client = S3RegistryClient(
            "s3://my-bucket/registry/",
            mode="registry",
            cache_dir=temp_dir / "cache",
            s3_client=mock_s3,
        )

        resolved = client.resolve_package("test-plugin", "^1.0.0")

        assert resolved is not None
        assert resolved.version == "1.5.0"

    def test_returns_none_for_unknown_package(self, temp_dir: Path):
        """Returns None for unknown package."""
        mock_s3 = MagicMock()

        registry_data: dict[str, dict[str, str]] = {"packages": {}}
        mock_s3.get_object.return_value = {"Body": io.BytesIO(json.dumps(registry_data).encode())}

        client = S3RegistryClient(
            "s3://my-bucket/registry/",
            mode="registry",
            cache_dir=temp_dir / "cache",
            s3_client=mock_s3,
        )

        resolved = client.resolve_package("nonexistent", "latest")

        assert resolved is None


class TestS3RegistryClientListPackages:
    """Tests for S3RegistryClient.list_packages()."""

    def test_lists_packages_from_registry(self, temp_dir: Path):
        """Lists all packages in registry."""
        mock_s3 = MagicMock()

        registry_data = {
            "packages": {
                "plugin-a": {"versions": ["1.0.0"], "latest": "1.0.0"},
                "plugin-b": {"versions": ["2.0.0"], "latest": "2.0.0"},
            }
        }
        mock_s3.get_object.return_value = {"Body": io.BytesIO(json.dumps(registry_data).encode())}

        client = S3RegistryClient(
            "s3://my-bucket/registry/",
            mode="registry",
            cache_dir=temp_dir / "cache",
            s3_client=mock_s3,
        )

        packages = client.list_packages()

        assert "plugin-a" in packages
        assert "plugin-b" in packages

    def test_lists_single_package_from_package_mode(self, temp_dir: Path):
        """Lists single package from package mode."""
        mock_s3 = MagicMock()

        package_data = {"name": "single-plugin", "version": "1.0.0"}
        mock_s3.get_object.return_value = {"Body": io.BytesIO(json.dumps(package_data).encode())}

        client = S3RegistryClient(
            "s3://my-bucket/plugin/",
            mode="package",
            cache_dir=temp_dir / "cache",
            s3_client=mock_s3,
        )

        packages = client.list_packages()

        assert packages == ["single-plugin"]


class TestS3RegistryError:
    """Tests for S3RegistryError."""

    def test_stores_bucket_and_key(self):
        """Error stores bucket and key."""
        error = S3RegistryError("Test error", bucket="my-bucket", key="path/to/file")

        assert error.bucket == "my-bucket"
        assert error.key == "path/to/file"
        assert "Test error" in str(error)

    def test_handles_none_bucket_and_key(self):
        """Error handles None bucket and key."""
        error = S3RegistryError("Test error")

        assert error.bucket is None
        assert error.key is None
