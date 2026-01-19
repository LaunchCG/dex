"""Tests for dex.registry.local module."""

import json
import tarfile
from pathlib import Path

import pytest

from dex.registry.local import LocalRegistryClient, LocalRegistryError


class TestLocalRegistryClientInit:
    """Tests for LocalRegistryClient initialization."""

    def test_parses_file_url_absolute(self, temp_dir: Path):
        """Parses absolute file:// URL."""
        client = LocalRegistryClient(f"file://{temp_dir}")
        assert client.path == temp_dir

    def test_parses_file_url_relative(self, temp_dir: Path):
        """Parses relative file: URL."""
        # This test uses current working directory context
        client = LocalRegistryClient("file:.")
        assert client.path.exists()

    def test_parses_plain_path(self, temp_dir: Path):
        """Parses plain path."""
        client = LocalRegistryClient(str(temp_dir))
        # Use resolve() to handle macOS /var -> /private/var symlink
        assert client.path.resolve() == temp_dir.resolve()

    def test_default_mode_is_registry(self, temp_registry: Path):
        """Default mode is 'registry'."""
        client = LocalRegistryClient(str(temp_registry))
        assert client.mode == "registry"

    def test_explicit_package_mode(self, temp_plugin_dir: Path):
        """Can set mode to 'package' explicitly."""
        client = LocalRegistryClient(str(temp_plugin_dir), mode="package")
        assert client.mode == "package"

    def test_protocol_property(self, temp_dir: Path):
        """Protocol property returns 'file'."""
        client = LocalRegistryClient(str(temp_dir))
        assert client.protocol == "file"


class TestLocalRegistryClientGetPackageInfo:
    """Tests for LocalRegistryClient.get_package_info()."""

    def test_gets_package_from_registry(self, temp_registry: Path):
        """Gets package info from registry mode."""
        client = LocalRegistryClient(str(temp_registry), mode="registry")

        info = client.get_package_info("test-plugin")

        assert info is not None
        assert info.name == "test-plugin"
        assert "1.0.0" in info.versions
        assert info.latest == "2.0.0"

    def test_returns_none_for_unknown_package(self, temp_registry: Path):
        """Returns None for unknown package in registry."""
        client = LocalRegistryClient(str(temp_registry), mode="registry")

        info = client.get_package_info("nonexistent")

        assert info is None

    def test_gets_package_from_directory(self, temp_plugin_dir: Path):
        """Gets package info from package mode."""
        client = LocalRegistryClient(str(temp_plugin_dir), mode="package")

        info = client.get_package_info("test-plugin")

        assert info is not None
        assert info.name == "test-plugin"
        assert info.versions == ["1.0.0"]
        assert info.latest == "1.0.0"

    def test_returns_none_for_wrong_name_in_directory(self, temp_plugin_dir: Path):
        """Returns None when package name doesn't match manifest."""
        client = LocalRegistryClient(str(temp_plugin_dir), mode="package")

        info = client.get_package_info("wrong-name")

        assert info is None

    def test_raises_for_missing_registry_json(self, temp_dir: Path):
        """Raises error when registry.json is missing in registry mode."""
        client = LocalRegistryClient(str(temp_dir), mode="registry")

        with pytest.raises(LocalRegistryError, match="does not contain registry.json"):
            client.get_package_info("test")

    def test_raises_for_missing_package_json(self, temp_dir: Path):
        """Raises error when package.json is missing in package mode."""
        client = LocalRegistryClient(str(temp_dir), mode="package")

        with pytest.raises(LocalRegistryError, match="package.json"):
            client.get_package_info("test")


class TestLocalRegistryClientResolvePackage:
    """Tests for LocalRegistryClient.resolve_package()."""

    def test_resolves_latest_version(self, temp_registry: Path):
        """Resolves 'latest' to the latest version."""
        # Create a tarball for the latest version
        tarball_path = temp_registry / "test-plugin-2.0.0.tar.gz"
        with tarfile.open(tarball_path, "w:gz"):
            pass  # Empty tarball for test

        client = LocalRegistryClient(str(temp_registry), mode="registry")

        resolved = client.resolve_package("test-plugin", "latest")

        assert resolved is not None
        assert resolved.name == "test-plugin"
        assert resolved.version == "2.0.0"

    def test_resolves_specific_version(self, temp_registry: Path):
        """Resolves a specific version."""
        tarball_path = temp_registry / "test-plugin-1.0.0.tar.gz"
        with tarfile.open(tarball_path, "w:gz"):
            pass

        client = LocalRegistryClient(str(temp_registry), mode="registry")

        resolved = client.resolve_package("test-plugin", "1.0.0")

        assert resolved is not None
        assert resolved.version == "1.0.0"

    def test_resolves_version_range(self, temp_registry: Path):
        """Resolves a version range."""
        # Create tarballs for multiple versions
        for version in ["1.0.0", "1.1.0"]:
            tarball_path = temp_registry / f"test-plugin-{version}.tar.gz"
            with tarfile.open(tarball_path, "w:gz"):
                pass

        client = LocalRegistryClient(str(temp_registry), mode="registry")

        resolved = client.resolve_package("test-plugin", "^1.0.0")

        assert resolved is not None
        assert resolved.version == "1.1.0"  # Highest matching ^1.0.0

    def test_returns_none_for_missing_tarball(self, temp_registry: Path):
        """Returns None when tarball doesn't exist."""
        client = LocalRegistryClient(str(temp_registry), mode="registry")

        resolved = client.resolve_package("test-plugin", "1.0.0")

        assert resolved is None

    def test_resolves_from_directory(self, temp_plugin_dir: Path):
        """Resolves package from package mode."""
        client = LocalRegistryClient(str(temp_plugin_dir), mode="package")

        resolved = client.resolve_package("test-plugin", "latest")

        assert resolved is not None
        assert resolved.name == "test-plugin"
        # Use resolve() to handle macOS /var -> /private/var symlink
        assert resolved.local_path is not None
        assert resolved.local_path.resolve() == temp_plugin_dir.resolve()

    def test_resolved_includes_integrity(self, temp_registry: Path):
        """Resolved package includes integrity hash."""
        tarball_path = temp_registry / "test-plugin-2.0.0.tar.gz"
        with tarfile.open(tarball_path, "w:gz"):
            pass

        client = LocalRegistryClient(str(temp_registry), mode="registry")

        resolved = client.resolve_package("test-plugin", "latest")

        assert resolved is not None
        assert resolved.integrity is not None
        assert resolved.integrity.startswith("sha512-")


class TestLocalRegistryClientFetchPackage:
    """Tests for LocalRegistryClient.fetch_package()."""

    def test_fetches_directory(self, temp_dir: Path, temp_plugin_dir: Path):
        """Fetches package from directory (copies it)."""
        client = LocalRegistryClient(str(temp_plugin_dir), mode="package")
        resolved = client.resolve_package("test-plugin", "latest")
        assert resolved is not None
        dest_dir = temp_dir / "installed"
        dest_dir.mkdir()

        result = client.fetch_package(resolved, dest_dir)

        assert result.exists()
        assert (result / "package.json").exists()

    def test_fetches_tarball(self, temp_dir: Path, temp_registry: Path):
        """Fetches and extracts tarball."""
        # Create a valid tarball with plugin content
        plugin_dir = temp_dir / "plugin-source"
        plugin_dir.mkdir()
        (plugin_dir / "package.json").write_text(
            json.dumps({"name": "test-plugin", "version": "2.0.0", "description": "Test"})
        )

        tarball_path = temp_registry / "test-plugin-2.0.0.tar.gz"
        with tarfile.open(tarball_path, "w:gz") as tar:
            tar.add(plugin_dir, arcname="test-plugin")

        client = LocalRegistryClient(str(temp_registry), mode="registry")
        resolved = client.resolve_package("test-plugin", "latest")
        assert resolved is not None
        dest_dir = temp_dir / "installed"
        dest_dir.mkdir()

        result = client.fetch_package(resolved, dest_dir)

        assert result.exists()
        assert (result / "package.json").exists()

    def test_raises_for_missing_local_path(self, temp_registry: Path, temp_dir: Path):
        """Raises error when resolved package has no local path."""
        from dex.registry.base import ResolvedPackage

        client = LocalRegistryClient(str(temp_registry), mode="registry")
        resolved = ResolvedPackage(
            name="test",
            version="1.0.0",
            resolved_url="file:///nonexistent",
            local_path=None,
        )
        dest_dir = temp_dir / "dest"
        dest_dir.mkdir()

        with pytest.raises(ValueError, match="no local path"):
            client.fetch_package(resolved, dest_dir)


class TestLocalRegistryClientListPackages:
    """Tests for LocalRegistryClient.list_packages()."""

    def test_lists_packages_from_registry(self, temp_registry: Path):
        """Lists all packages in registry."""
        client = LocalRegistryClient(str(temp_registry), mode="registry")

        packages = client.list_packages()

        assert "test-plugin" in packages
        assert "other-plugin" in packages

    def test_lists_single_package_from_directory(self, temp_plugin_dir: Path):
        """Lists single package from package mode."""
        client = LocalRegistryClient(str(temp_plugin_dir), mode="package")

        packages = client.list_packages()

        assert packages == ["test-plugin"]

    def test_raises_for_invalid_directory_in_registry_mode(self, temp_dir: Path):
        """Raises error for directory without registry.json in registry mode."""
        client = LocalRegistryClient(str(temp_dir), mode="registry")

        with pytest.raises(LocalRegistryError, match="does not contain registry.json"):
            client.list_packages()

    def test_raises_for_invalid_directory_in_package_mode(self, temp_dir: Path):
        """Raises error for directory without package.json in package mode."""
        client = LocalRegistryClient(str(temp_dir), mode="package")

        with pytest.raises(LocalRegistryError, match="package.json"):
            client.list_packages()


class TestLocalRegistryError:
    """Tests for LocalRegistryError."""

    def test_stores_path(self):
        """Error stores path attribute."""
        error = LocalRegistryError("message", path="/some/path")
        assert error.path == "/some/path"

    def test_handles_none_path(self):
        """Error handles None path."""
        error = LocalRegistryError("message")
        assert error.path is None
