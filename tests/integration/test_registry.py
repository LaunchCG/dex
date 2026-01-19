"""Integration tests for registry operations."""

import json
import tarfile
from pathlib import Path

import pytest

from dex.registry.factory import create_registry_client
from dex.registry.local import LocalRegistryClient


@pytest.fixture
def multi_version_registry(temp_dir: Path) -> Path:
    """Create a registry with multiple versions of a plugin."""
    registry_dir = temp_dir / "registry"
    registry_dir.mkdir()

    # Create registry.json
    registry_data = {
        "packages": {
            "versioned-plugin": {
                "versions": ["1.0.0", "1.1.0", "1.2.0", "2.0.0"],
                "latest": "2.0.0",
            }
        }
    }
    (registry_dir / "registry.json").write_text(json.dumps(registry_data))

    # Create tarballs for each version
    for version in ["1.0.0", "1.1.0", "1.2.0", "2.0.0"]:
        # Create plugin directory
        plugin_dir = temp_dir / f"plugin-{version}"
        plugin_dir.mkdir()
        manifest = {
            "name": "versioned-plugin",
            "version": version,
            "description": f"Version {version}",
            "skills": [
                {"name": "skill", "description": "Version skill", "context": "./context.md"}
            ],
        }
        (plugin_dir / "package.json").write_text(json.dumps(manifest))
        (plugin_dir / "context.md").write_text(f"# Version {version}")

        # Create tarball
        tarball_path = registry_dir / f"versioned-plugin-{version}.tar.gz"
        with tarfile.open(tarball_path, "w:gz") as tar:
            tar.add(plugin_dir, arcname="versioned-plugin")

    return registry_dir


class TestLocalRegistryTarball:
    """Tests for local registry tarball operations."""

    def test_resolve_and_fetch_tarball(self, multi_version_registry: Path, temp_dir: Path):
        """Resolve and fetch a package from tarball."""
        client = LocalRegistryClient(str(multi_version_registry))

        # Resolve specific version
        resolved = client.resolve_package("versioned-plugin", "1.1.0")

        assert resolved is not None
        assert resolved.version == "1.1.0"
        assert resolved.integrity is not None

        # Fetch the package
        dest_dir = temp_dir / "installed"
        dest_dir.mkdir()

        plugin_path = client.fetch_package(resolved, dest_dir)

        assert plugin_path.exists()
        assert (plugin_path / "package.json").exists()

        # Verify correct version
        manifest = json.loads((plugin_path / "package.json").read_text())
        assert manifest["version"] == "1.1.0"


class TestLocalRegistryDirect:
    """Tests for direct local plugin operations."""

    def test_resolve_from_directory(self, temp_plugin_dir: Path):
        """Resolve plugin from direct directory."""
        client = LocalRegistryClient(str(temp_plugin_dir))

        resolved = client.resolve_package("test-plugin", "latest")

        assert resolved is not None
        assert resolved.name == "test-plugin"
        # Use resolve() to handle macOS /var -> /private/var symlink
        assert resolved.local_path is not None
        assert resolved.local_path.resolve() == temp_plugin_dir.resolve()


class TestVersionResolution:
    """Tests for version resolution in registries."""

    def test_resolves_latest(self, multi_version_registry: Path):
        """Resolves 'latest' to the latest version."""
        client = LocalRegistryClient(str(multi_version_registry))

        resolved = client.resolve_package("versioned-plugin", "latest")

        assert resolved is not None
        assert resolved.version == "2.0.0"

    def test_resolves_caret_range(self, multi_version_registry: Path):
        """Resolves caret version range."""
        client = LocalRegistryClient(str(multi_version_registry))

        resolved = client.resolve_package("versioned-plugin", "^1.0.0")

        assert resolved is not None
        assert resolved.version == "1.2.0"  # Highest 1.x.x

    def test_resolves_tilde_range(self, multi_version_registry: Path):
        """Resolves tilde version range."""
        client = LocalRegistryClient(str(multi_version_registry))

        resolved = client.resolve_package("versioned-plugin", "~1.1.0")

        assert resolved is not None
        assert resolved.version == "1.1.0"  # Only 1.1.x

    def test_resolves_exact_version(self, multi_version_registry: Path):
        """Resolves exact version."""
        client = LocalRegistryClient(str(multi_version_registry))

        resolved = client.resolve_package("versioned-plugin", "1.0.0")

        assert resolved is not None
        assert resolved.version == "1.0.0"


class TestRegistryFactory:
    """Tests for registry client factory."""

    def test_creates_local_client_for_file_url(self, temp_dir: Path):
        """Creates local client for file:// URL."""
        client = create_registry_client(f"file://{temp_dir}")

        assert isinstance(client, LocalRegistryClient)

    def test_creates_local_client_for_relative_path(self, temp_dir: Path):
        """Creates local client for file: relative path."""
        client = create_registry_client("file:./path")

        assert isinstance(client, LocalRegistryClient)


class TestRegistryPackageListing:
    """Tests for listing packages in registries."""

    def test_list_packages_in_registry(self, multi_version_registry: Path):
        """Lists all packages in registry."""
        client = LocalRegistryClient(str(multi_version_registry))

        packages = client.list_packages()

        assert "versioned-plugin" in packages

    def test_list_single_package_in_direct_mode(self, temp_plugin_dir: Path):
        """Lists single package in direct mode."""
        client = LocalRegistryClient(str(temp_plugin_dir))

        packages = client.list_packages()

        assert packages == ["test-plugin"]


class TestPackageInfo:
    """Tests for getting package information."""

    def test_get_package_info(self, multi_version_registry: Path):
        """Gets package information from registry."""
        client = LocalRegistryClient(str(multi_version_registry))

        info = client.get_package_info("versioned-plugin")

        assert info is not None
        assert info.name == "versioned-plugin"
        assert "1.0.0" in info.versions
        assert "2.0.0" in info.versions
        assert info.latest == "2.0.0"

    def test_returns_none_for_unknown_package(self, multi_version_registry: Path):
        """Returns None for unknown package."""
        client = LocalRegistryClient(str(multi_version_registry))

        info = client.get_package_info("nonexistent-plugin")

        assert info is None
