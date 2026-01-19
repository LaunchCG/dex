"""Tests for dex.registry.https module."""

import json
from pathlib import Path
from unittest.mock import patch

import pytest

from dex.registry.common import parse_tarball_info
from dex.registry.https import HttpsRegistryClient, HttpsRegistryError


class TestHttpsRegistryClientInit:
    """Tests for HttpsRegistryClient initialization."""

    def test_parses_https_url(self, temp_dir: Path):
        """Parses HTTPS URL correctly."""
        client = HttpsRegistryClient(
            "https://example.com/registry/",
            cache_dir=temp_dir / "cache",
        )

        assert client.protocol == "https"
        assert client.base_url == "https://example.com/registry/"

    def test_adds_trailing_slash_to_url(self, temp_dir: Path):
        """Adds trailing slash to URL if missing."""
        client = HttpsRegistryClient(
            "https://example.com/registry",
            cache_dir=temp_dir / "cache",
        )

        assert client.base_url == "https://example.com/registry/"

    def test_detects_direct_tarball(self, temp_dir: Path):
        """Detects direct tarball URLs."""
        client = HttpsRegistryClient(
            "https://example.com/plugin-1.0.0.tar.gz",
            cache_dir=temp_dir / "cache",
        )

        assert client._is_direct_tarball is True
        assert client._tarball_info is not None
        assert client._tarball_info["name"] == "plugin"
        assert client._tarball_info["version"] == "1.0.0"

    def test_detects_tgz_tarball(self, temp_dir: Path):
        """Detects .tgz tarball URLs."""
        client = HttpsRegistryClient(
            "https://example.com/my-plugin-2.0.0.tgz",
            cache_dir=temp_dir / "cache",
        )

        assert client._is_direct_tarball is True
        assert client._tarball_info is not None
        assert client._tarball_info["name"] == "my-plugin"
        assert client._tarball_info["version"] == "2.0.0"

    def test_raises_for_http_scheme(self, temp_dir: Path):
        """Raises error for HTTP (non-HTTPS) URL."""
        with pytest.raises(HttpsRegistryError, match="expected https"):
            HttpsRegistryClient(
                "http://example.com/registry/",
                cache_dir=temp_dir / "cache",
            )

    def test_default_mode_is_registry(self, temp_dir: Path):
        """Default mode is 'registry'."""
        client = HttpsRegistryClient(
            "https://example.com/registry/",
            cache_dir=temp_dir / "cache",
        )

        assert client.mode == "registry"

    def test_explicit_package_mode(self, temp_dir: Path):
        """Can set mode to 'package' explicitly."""
        client = HttpsRegistryClient(
            "https://example.com/plugin/",
            mode="package",
            cache_dir=temp_dir / "cache",
        )

        assert client.mode == "package"


class TestParseTarballInfo:
    """Tests for tarball URL parsing (centralized in common.py)."""

    def test_parses_standard_format(self):
        """Parses name-version.tar.gz format."""
        result = parse_tarball_info("plugin-1.2.3.tar.gz")

        assert result["name"] == "plugin"
        assert result["version"] == "1.2.3"

    def test_parses_with_v_prefix(self):
        """Parses name-v1.2.3.tar.gz format."""
        result = parse_tarball_info("plugin-v1.2.3.tar.gz")

        assert result["name"] == "plugin"
        assert result["version"] == "1.2.3"

    def test_parses_with_underscore(self):
        """Parses name_version.tar.gz format."""
        result = parse_tarball_info("my_plugin_1.0.0.tar.gz")

        assert result["name"] == "my_plugin"
        assert result["version"] == "1.0.0"

    def test_parses_complex_name(self):
        """Parses complex package names."""
        result = parse_tarball_info("@scope-my-plugin-1.0.0.tar.gz")

        assert result["name"] == "@scope-my-plugin"
        assert result["version"] == "1.0.0"

    def test_handles_prerelease_version(self):
        """Parses version with prerelease suffix."""
        result = parse_tarball_info("plugin-1.0.0-beta.1.tar.gz")

        assert result["name"] == "plugin"
        assert result["version"] == "1.0.0-beta.1"


class TestHttpsRegistryClientGetPackageInfo:
    """Tests for HttpsRegistryClient.get_package_info()."""

    def test_gets_package_from_registry_json(self, temp_dir: Path):
        """Gets package info from registry.json."""
        client = HttpsRegistryClient(
            "https://example.com/registry/",
            mode="registry",
            cache_dir=temp_dir / "cache",
        )

        registry_data = {
            "packages": {
                "test-plugin": {
                    "versions": ["1.0.0", "2.0.0"],
                    "latest": "2.0.0",
                }
            }
        }

        with patch.object(client, "_make_request", return_value=json.dumps(registry_data).encode()):
            info = client.get_package_info("test-plugin")

        assert info is not None
        assert info.name == "test-plugin"
        assert info.versions == ["1.0.0", "2.0.0"]
        assert info.latest == "2.0.0"

    def test_returns_none_for_unknown_package(self, temp_dir: Path):
        """Returns None for package not in registry."""
        client = HttpsRegistryClient(
            "https://example.com/registry/",
            mode="registry",
            cache_dir=temp_dir / "cache",
        )

        registry_data: dict[str, dict[str, str]] = {"packages": {}}

        with patch.object(client, "_make_request", return_value=json.dumps(registry_data).encode()):
            info = client.get_package_info("nonexistent")

        assert info is None

    def test_gets_package_from_package_mode(self, temp_dir: Path):
        """Gets package info from package.json in package mode."""
        client = HttpsRegistryClient(
            "https://example.com/plugin/",
            mode="package",
            cache_dir=temp_dir / "cache",
        )

        package_data = {
            "name": "my-plugin",
            "version": "1.5.0",
            "description": "A plugin",
        }

        with patch.object(client, "_make_request", return_value=json.dumps(package_data).encode()):
            info = client.get_package_info("my-plugin")

        assert info is not None
        assert info.name == "my-plugin"
        assert info.versions == ["1.5.0"]
        assert info.latest == "1.5.0"

    def test_gets_package_from_direct_tarball(self, temp_dir: Path):
        """Gets package info from direct tarball URL."""
        client = HttpsRegistryClient(
            "https://example.com/my-plugin-1.0.0.tar.gz",
            cache_dir=temp_dir / "cache",
        )

        info = client.get_package_info("my-plugin")

        assert info is not None
        assert info.name == "my-plugin"
        assert info.versions == ["1.0.0"]
        assert info.latest == "1.0.0"


class TestHttpsRegistryClientResolvePackage:
    """Tests for HttpsRegistryClient.resolve_package()."""

    def test_resolves_latest_version(self, temp_dir: Path):
        """Resolves 'latest' to the latest version."""
        client = HttpsRegistryClient(
            "https://example.com/registry/",
            mode="registry",
            cache_dir=temp_dir / "cache",
        )

        registry_data = {
            "packages": {"test-plugin": {"versions": ["1.0.0", "2.0.0"], "latest": "2.0.0"}}
        }

        with patch.object(client, "_make_request", return_value=json.dumps(registry_data).encode()):
            resolved = client.resolve_package("test-plugin", "latest")

        assert resolved is not None
        assert resolved.name == "test-plugin"
        assert resolved.version == "2.0.0"
        assert resolved.resolved_url == "https://example.com/registry/test-plugin-2.0.0.tar.gz"

    def test_resolves_specific_version(self, temp_dir: Path):
        """Resolves specific version."""
        client = HttpsRegistryClient(
            "https://example.com/registry/",
            mode="registry",
            cache_dir=temp_dir / "cache",
        )

        registry_data = {
            "packages": {"test-plugin": {"versions": ["1.0.0", "2.0.0"], "latest": "2.0.0"}}
        }

        with patch.object(client, "_make_request", return_value=json.dumps(registry_data).encode()):
            resolved = client.resolve_package("test-plugin", "1.0.0")

        assert resolved is not None
        assert resolved.version == "1.0.0"

    def test_resolves_direct_tarball(self, temp_dir: Path):
        """Resolves direct tarball URL."""
        client = HttpsRegistryClient(
            "https://example.com/plugin-1.0.0.tar.gz",
            cache_dir=temp_dir / "cache",
        )

        resolved = client.resolve_package("plugin", "latest")

        assert resolved is not None
        assert resolved.name == "plugin"
        assert resolved.version == "1.0.0"
        assert resolved.resolved_url == "https://example.com/plugin-1.0.0.tar.gz"


class TestHttpsRegistryClientListPackages:
    """Tests for HttpsRegistryClient.list_packages()."""

    def test_lists_packages_from_registry(self, temp_dir: Path):
        """Lists all packages in registry."""
        client = HttpsRegistryClient(
            "https://example.com/registry/",
            mode="registry",
            cache_dir=temp_dir / "cache",
        )

        registry_data = {
            "packages": {
                "plugin-a": {"versions": ["1.0.0"], "latest": "1.0.0"},
                "plugin-b": {"versions": ["2.0.0"], "latest": "2.0.0"},
            }
        }

        with patch.object(client, "_make_request", return_value=json.dumps(registry_data).encode()):
            packages = client.list_packages()

        assert "plugin-a" in packages
        assert "plugin-b" in packages

    def test_lists_single_package_from_tarball(self, temp_dir: Path):
        """Lists single package from direct tarball."""
        client = HttpsRegistryClient(
            "https://example.com/my-plugin-1.0.0.tar.gz",
            cache_dir=temp_dir / "cache",
        )

        packages = client.list_packages()

        assert packages == ["my-plugin"]


class TestHttpsRegistryError:
    """Tests for HttpsRegistryError."""

    def test_stores_url_and_status_code(self):
        """Error stores URL and status code."""
        error = HttpsRegistryError(
            "Not found",
            url="https://example.com/test",
            status_code=404,
        )

        assert error.url == "https://example.com/test"
        assert error.status_code == 404
        assert "Not found" in str(error)

    def test_handles_none_values(self):
        """Error handles None URL and status code."""
        error = HttpsRegistryError("Test error")

        assert error.url is None
        assert error.status_code is None
