"""Tests for dex.registry.azure module."""

import json
from pathlib import Path
from unittest.mock import MagicMock

import pytest

from dex.registry.azure import AzureRegistryClient, AzureRegistryError
from dex.registry.common import parse_tarball_info


class TestAzureRegistryClientParseUrl:
    """Tests for URL parsing."""

    def test_parses_basic_azure_url(self):
        """Parses basic Azure URL correctly."""
        account, container, prefix = AzureRegistryClient._parse_url(
            "az://myaccount/mycontainer/path/to/registry/"
        )

        assert account == "myaccount"
        assert container == "mycontainer"
        assert prefix == "path/to/registry/"

    def test_parses_azure_url_without_trailing_slash(self):
        """Adds trailing slash to prefix."""
        account, container, prefix = AzureRegistryClient._parse_url(
            "az://myaccount/mycontainer/registry"
        )

        assert prefix == "registry/"

    def test_parses_azure_url_with_container_only(self):
        """Handles URL with just account and container."""
        account, container, prefix = AzureRegistryClient._parse_url("az://myaccount/mycontainer")

        assert account == "myaccount"
        assert container == "mycontainer"
        assert prefix == ""

    def test_parses_azure_url_with_single_path_segment(self):
        """Handles URL with single path segment."""
        account, container, prefix = AzureRegistryClient._parse_url(
            "az://myaccount/mycontainer/plugins"
        )

        assert prefix == "plugins/"

    def test_raises_for_invalid_scheme(self):
        """Raises error for non-az:// URL."""
        with pytest.raises(AzureRegistryError, match="Invalid Azure URL scheme"):
            AzureRegistryClient._parse_url("s3://bucket/path/")

    def test_raises_for_missing_account(self):
        """Raises error for URL without account."""
        with pytest.raises(AzureRegistryError, match="must include storage account"):
            AzureRegistryClient._parse_url("az:///container/path/")


class TestAzureRegistryClientInit:
    """Tests for client initialization."""

    def test_parses_url_on_init(self, temp_dir: Path):
        """Parses URL during initialization."""
        mock_client = MagicMock()
        client = AzureRegistryClient(
            "az://myaccount/mycontainer/registry/",
            cache_dir=temp_dir / "cache",
            blob_service_client=mock_client,
        )

        assert client.account == "myaccount"
        assert client.container == "mycontainer"
        assert client.prefix == "registry/"
        assert client.protocol == "az"

    def test_detects_direct_tarball(self, temp_dir: Path):
        """Detects direct tarball URL."""
        mock_client = MagicMock()
        client = AzureRegistryClient(
            "az://myaccount/mycontainer/plugin-1.0.0.tar.gz",
            cache_dir=temp_dir / "cache",
            blob_service_client=mock_client,
        )

        assert client._is_direct_tarball is True
        assert client._tarball_info is not None
        assert client._tarball_info["name"] == "plugin"
        assert client._tarball_info["version"] == "1.0.0"

    def test_default_mode_is_registry(self, temp_dir: Path):
        """Default mode is 'registry'."""
        mock_client = MagicMock()
        client = AzureRegistryClient(
            "az://myaccount/mycontainer/registry/",
            cache_dir=temp_dir / "cache",
            blob_service_client=mock_client,
        )

        assert client.mode == "registry"

    def test_explicit_package_mode(self, temp_dir: Path):
        """Can set mode to 'package' explicitly."""
        mock_client = MagicMock()
        client = AzureRegistryClient(
            "az://myaccount/mycontainer/plugin/",
            mode="package",
            cache_dir=temp_dir / "cache",
            blob_service_client=mock_client,
        )

        assert client.mode == "package"


class TestAzureRegistryClientGetPackageInfo:
    """Tests for get_package_info()."""

    def test_gets_package_from_registry_json(self, temp_dir: Path):
        """Gets package info from registry.json."""
        # Mock blob service client
        mock_blob_service = MagicMock()
        mock_container = MagicMock()
        mock_blob = MagicMock()

        mock_blob_service.get_container_client.return_value = mock_container
        mock_container.get_blob_client.return_value = mock_blob

        # Download registry.json
        registry_data = {
            "packages": {
                "test-plugin": {
                    "versions": ["1.0.0", "2.0.0"],
                    "latest": "2.0.0",
                }
            }
        }
        mock_download = MagicMock()
        mock_download.readall.return_value = json.dumps(registry_data).encode()
        mock_blob.download_blob.return_value = mock_download

        client = AzureRegistryClient(
            "az://myaccount/mycontainer/registry/",
            mode="registry",
            cache_dir=temp_dir / "cache",
            blob_service_client=mock_blob_service,
        )

        info = client.get_package_info("test-plugin")

        assert info is not None
        assert info.name == "test-plugin"
        assert info.versions == ["1.0.0", "2.0.0"]
        assert info.latest == "2.0.0"

    def test_returns_none_for_unknown_package(self, temp_dir: Path):
        """Returns None for package not in registry."""
        mock_blob_service = MagicMock()
        mock_container = MagicMock()
        mock_blob = MagicMock()

        mock_blob_service.get_container_client.return_value = mock_container
        mock_container.get_blob_client.return_value = mock_blob

        registry_data: dict[str, dict[str, str]] = {"packages": {}}
        mock_download = MagicMock()
        mock_download.readall.return_value = json.dumps(registry_data).encode()
        mock_blob.download_blob.return_value = mock_download

        client = AzureRegistryClient(
            "az://myaccount/mycontainer/registry/",
            mode="registry",
            cache_dir=temp_dir / "cache",
            blob_service_client=mock_blob_service,
        )

        info = client.get_package_info("nonexistent")

        assert info is None

    def test_gets_package_from_direct_tarball(self, temp_dir: Path):
        """Gets package info from direct tarball URL."""
        mock_client = MagicMock()
        client = AzureRegistryClient(
            "az://myaccount/mycontainer/my-plugin-1.0.0.tar.gz",
            cache_dir=temp_dir / "cache",
            blob_service_client=mock_client,
        )

        info = client.get_package_info("my-plugin")

        assert info is not None
        assert info.name == "my-plugin"
        assert info.versions == ["1.0.0"]
        assert info.latest == "1.0.0"

    def test_gets_package_from_package_mode(self, temp_dir: Path):
        """Gets package info from package.json in package mode."""
        mock_blob_service = MagicMock()
        mock_container = MagicMock()
        mock_blob = MagicMock()

        mock_blob_service.get_container_client.return_value = mock_container
        mock_container.get_blob_client.return_value = mock_blob

        package_data = {
            "name": "direct-plugin",
            "version": "1.5.0",
            "description": "A direct plugin",
        }
        mock_download = MagicMock()
        mock_download.readall.return_value = json.dumps(package_data).encode()
        mock_blob.download_blob.return_value = mock_download

        client = AzureRegistryClient(
            "az://myaccount/mycontainer/plugin/",
            mode="package",
            cache_dir=temp_dir / "cache",
            blob_service_client=mock_blob_service,
        )

        info = client.get_package_info("direct-plugin")

        assert info is not None
        assert info.name == "direct-plugin"
        assert info.versions == ["1.5.0"]
        assert info.latest == "1.5.0"


class TestAzureRegistryClientResolvePackage:
    """Tests for resolve_package()."""

    def test_resolves_latest_version(self, temp_dir: Path):
        """Resolves 'latest' to the latest version."""
        mock_blob_service = MagicMock()
        mock_container = MagicMock()
        mock_blob = MagicMock()

        mock_blob_service.get_container_client.return_value = mock_container
        mock_container.get_blob_client.return_value = mock_blob

        registry_data = {
            "packages": {"test-plugin": {"versions": ["1.0.0", "2.0.0"], "latest": "2.0.0"}}
        }
        mock_download = MagicMock()
        mock_download.readall.return_value = json.dumps(registry_data).encode()
        mock_blob.download_blob.return_value = mock_download

        client = AzureRegistryClient(
            "az://myaccount/mycontainer/registry/",
            mode="registry",
            cache_dir=temp_dir / "cache",
            blob_service_client=mock_blob_service,
        )

        resolved = client.resolve_package("test-plugin", "latest")

        assert resolved is not None
        assert resolved.name == "test-plugin"
        assert resolved.version == "2.0.0"
        assert (
            resolved.resolved_url == "az://myaccount/mycontainer/registry/test-plugin-2.0.0.tar.gz"
        )

    def test_resolves_specific_version(self, temp_dir: Path):
        """Resolves specific version."""
        mock_blob_service = MagicMock()
        mock_container = MagicMock()
        mock_blob = MagicMock()

        mock_blob_service.get_container_client.return_value = mock_container
        mock_container.get_blob_client.return_value = mock_blob

        registry_data = {
            "packages": {"test-plugin": {"versions": ["1.0.0", "2.0.0"], "latest": "2.0.0"}}
        }
        mock_download = MagicMock()
        mock_download.readall.return_value = json.dumps(registry_data).encode()
        mock_blob.download_blob.return_value = mock_download

        client = AzureRegistryClient(
            "az://myaccount/mycontainer/registry/",
            mode="registry",
            cache_dir=temp_dir / "cache",
            blob_service_client=mock_blob_service,
        )

        resolved = client.resolve_package("test-plugin", "1.0.0")

        assert resolved is not None
        assert resolved.version == "1.0.0"

    def test_resolves_direct_tarball(self, temp_dir: Path):
        """Resolves direct tarball URL."""
        mock_client = MagicMock()
        client = AzureRegistryClient(
            "az://myaccount/mycontainer/plugin-1.0.0.tar.gz",
            cache_dir=temp_dir / "cache",
            blob_service_client=mock_client,
        )

        resolved = client.resolve_package("plugin", "latest")

        assert resolved is not None
        assert resolved.name == "plugin"
        assert resolved.version == "1.0.0"
        assert resolved.resolved_url == "az://myaccount/mycontainer/plugin-1.0.0.tar.gz"


class TestAzureRegistryClientListPackages:
    """Tests for list_packages()."""

    def test_lists_packages_from_registry(self, temp_dir: Path):
        """Lists all packages in registry."""
        mock_blob_service = MagicMock()
        mock_container = MagicMock()
        mock_blob = MagicMock()

        mock_blob_service.get_container_client.return_value = mock_container
        mock_container.get_blob_client.return_value = mock_blob

        registry_data = {
            "packages": {
                "plugin-a": {"versions": ["1.0.0"], "latest": "1.0.0"},
                "plugin-b": {"versions": ["2.0.0"], "latest": "2.0.0"},
            }
        }
        mock_download = MagicMock()
        mock_download.readall.return_value = json.dumps(registry_data).encode()
        mock_blob.download_blob.return_value = mock_download

        client = AzureRegistryClient(
            "az://myaccount/mycontainer/registry/",
            mode="registry",
            cache_dir=temp_dir / "cache",
            blob_service_client=mock_blob_service,
        )

        packages = client.list_packages()

        assert "plugin-a" in packages
        assert "plugin-b" in packages

    def test_lists_single_package_from_tarball(self, temp_dir: Path):
        """Lists single package from direct tarball."""
        mock_client = MagicMock()
        client = AzureRegistryClient(
            "az://myaccount/mycontainer/my-plugin-1.0.0.tar.gz",
            cache_dir=temp_dir / "cache",
            blob_service_client=mock_client,
        )

        packages = client.list_packages()

        assert packages == ["my-plugin"]


class TestAzureRegistryError:
    """Tests for AzureRegistryError."""

    def test_stores_container_and_blob(self):
        """Error stores container and blob."""
        error = AzureRegistryError(
            "Not found",
            container="mycontainer",
            blob="path/to/blob",
        )

        assert error.container == "mycontainer"
        assert error.blob == "path/to/blob"
        assert "Not found" in str(error)

    def test_handles_none_values(self):
        """Error handles None container and blob."""
        error = AzureRegistryError("Test error")

        assert error.container is None
        assert error.blob is None


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

    def test_parses_from_path(self):
        """Parses tarball in nested path (extracts filename)."""
        # Note: parse_tarball_info takes just filename, not full path
        # So extract filename first
        from pathlib import PurePosixPath

        full_path = "path/to/plugins/my-plugin-1.0.0.tar.gz"
        filename = PurePosixPath(full_path).name
        result = parse_tarball_info(filename)

        assert result["name"] == "my-plugin"
        assert result["version"] == "1.0.0"


class TestAzureRegistryClientFetchPackage:
    """Tests for fetch_package()."""

    def test_fetches_tarball_from_azure(self, temp_dir: Path):
        """Fetches and extracts tarball from Azure."""
        import tarfile

        # Create a test tarball
        tarball_dir = temp_dir / "tarball_source"
        tarball_dir.mkdir()
        plugin_dir = tarball_dir / "test-plugin"
        plugin_dir.mkdir()
        (plugin_dir / "package.json").write_text('{"name": "test-plugin", "version": "1.0.0"}')
        (plugin_dir / "context.md").write_text("# Test Plugin")

        tarball_path = temp_dir / "test-plugin-1.0.0.tar.gz"
        with tarfile.open(tarball_path, "w:gz") as tar:
            tar.add(plugin_dir, arcname="test-plugin")

        tarball_content = tarball_path.read_bytes()

        # Mock blob service
        mock_blob_service = MagicMock()
        mock_container = MagicMock()
        mock_blob = MagicMock()

        mock_blob_service.get_container_client.return_value = mock_container
        mock_container.get_blob_client.return_value = mock_blob

        # Mock download
        mock_download = MagicMock()
        mock_download.readall.return_value = tarball_content
        mock_blob.download_blob.return_value = mock_download

        client = AzureRegistryClient(
            "az://myaccount/mycontainer/registry/",
            mode="registry",
            cache_dir=temp_dir / "cache",
            blob_service_client=mock_blob_service,
        )

        # Create resolved package
        from dex.registry.base import ResolvedPackage

        resolved = ResolvedPackage(
            name="test-plugin",
            version="1.0.0",
            resolved_url="az://myaccount/mycontainer/registry/test-plugin-1.0.0.tar.gz",
            local_path=None,
        )

        dest_dir = temp_dir / "dest"
        result = client.fetch_package(resolved, dest_dir)

        assert result.exists()
        assert (result / "package.json").exists()
        assert (result / "context.md").exists()

    def test_uses_cache_on_second_fetch(self, temp_dir: Path):
        """Uses cached tarball on second fetch."""
        import tarfile

        # Create a test tarball
        tarball_dir = temp_dir / "tarball_source"
        tarball_dir.mkdir()
        plugin_dir = tarball_dir / "test-plugin"
        plugin_dir.mkdir()
        (plugin_dir / "package.json").write_text('{"name": "test-plugin", "version": "1.0.0"}')

        tarball_path = temp_dir / "test-plugin-1.0.0.tar.gz"
        with tarfile.open(tarball_path, "w:gz") as tar:
            tar.add(plugin_dir, arcname="test-plugin")

        tarball_content = tarball_path.read_bytes()

        # Mock blob service
        mock_blob_service = MagicMock()
        mock_container = MagicMock()
        mock_blob = MagicMock()

        mock_blob_service.get_container_client.return_value = mock_container
        mock_container.get_blob_client.return_value = mock_blob

        mock_download = MagicMock()
        mock_download.readall.return_value = tarball_content
        mock_blob.download_blob.return_value = mock_download

        client = AzureRegistryClient(
            "az://myaccount/mycontainer/registry/",
            mode="registry",
            cache_dir=temp_dir / "cache",
            blob_service_client=mock_blob_service,
        )

        from dex.registry.base import ResolvedPackage

        resolved = ResolvedPackage(
            name="test-plugin",
            version="1.0.0",
            resolved_url="az://myaccount/mycontainer/registry/test-plugin-1.0.0.tar.gz",
            local_path=None,
        )

        # First fetch
        dest1 = temp_dir / "dest1"
        client.fetch_package(resolved, dest1)

        # Second fetch should use cache
        dest2 = temp_dir / "dest2"
        client.fetch_package(resolved, dest2)

        # Both should exist
        assert (dest1 / "test-plugin" / "package.json").exists()
        assert (dest2 / "test-plugin" / "package.json").exists()

        # download_blob should only be called once (for first fetch)
        assert mock_blob.download_blob.call_count == 1


class TestAzureRegistryClientErrorHandling:
    """Tests for error handling."""

    def test_raises_for_missing_registry_json_in_registry_mode(self, temp_dir: Path):
        """Raises error when registry.json doesn't exist in registry mode."""
        from azure.core.exceptions import ResourceNotFoundError

        mock_blob_service = MagicMock()
        mock_container = MagicMock()
        mock_blob = MagicMock()

        mock_blob_service.get_container_client.return_value = mock_container
        mock_container.get_blob_client.return_value = mock_blob

        # Simulate missing registry.json
        mock_blob.download_blob.side_effect = ResourceNotFoundError("Not found")

        client = AzureRegistryClient(
            "az://myaccount/mycontainer/registry/",
            mode="registry",
            cache_dir=temp_dir / "cache",
            blob_service_client=mock_blob_service,
        )

        with pytest.raises(AzureRegistryError, match="(?i)not found"):
            client.get_package_info("test-plugin")

    def test_raises_for_missing_package_json_in_package_mode(self, temp_dir: Path):
        """Raises error when package.json doesn't exist in package mode."""
        from azure.core.exceptions import ResourceNotFoundError

        mock_blob_service = MagicMock()
        mock_container = MagicMock()
        mock_blob = MagicMock()

        mock_blob_service.get_container_client.return_value = mock_container
        mock_container.get_blob_client.return_value = mock_blob

        # Simulate missing package.json
        mock_blob.download_blob.side_effect = ResourceNotFoundError("Not found")

        client = AzureRegistryClient(
            "az://myaccount/mycontainer/plugin/",
            mode="package",
            cache_dir=temp_dir / "cache",
            blob_service_client=mock_blob_service,
        )

        with pytest.raises(AzureRegistryError, match="(?i)not found"):
            client.get_package_info("test-plugin")

    def test_raises_for_invalid_registry_json(self, temp_dir: Path):
        """Raises AzureRegistryError when registry.json is invalid."""
        mock_blob_service = MagicMock()
        mock_container = MagicMock()
        mock_blob = MagicMock()

        mock_blob_service.get_container_client.return_value = mock_container
        mock_container.get_blob_client.return_value = mock_blob

        # Return invalid JSON
        mock_download = MagicMock()
        mock_download.readall.return_value = b"not valid json"
        mock_blob.download_blob.return_value = mock_download

        client = AzureRegistryClient(
            "az://myaccount/mycontainer/registry/",
            mode="registry",
            cache_dir=temp_dir / "cache",
            blob_service_client=mock_blob_service,
        )

        with pytest.raises(AzureRegistryError, match="Invalid JSON"):
            client.get_package_info("test-plugin")

    def test_raises_on_auth_error_during_get_package_info(self, temp_dir: Path):
        """Raises AzureRegistryError when auth fails during get_package_info."""
        from azure.core.exceptions import ClientAuthenticationError

        mock_blob_service = MagicMock()
        mock_container = MagicMock()
        mock_blob = MagicMock()

        mock_blob_service.get_container_client.return_value = mock_container
        mock_container.get_blob_client.return_value = mock_blob

        # Download fails with auth error
        mock_blob.download_blob.side_effect = ClientAuthenticationError("Authentication failed")

        client = AzureRegistryClient(
            "az://myaccount/mycontainer/registry/",
            mode="registry",
            cache_dir=temp_dir / "cache",
            blob_service_client=mock_blob_service,
        )

        with pytest.raises(AzureRegistryError, match="Authentication failed"):
            client.get_package_info("test-plugin")

    def test_raises_on_network_error_during_get_package_info(self, temp_dir: Path):
        """Raises AzureRegistryError when network fails during get_package_info."""
        from azure.core.exceptions import ServiceRequestError

        mock_blob_service = MagicMock()
        mock_container = MagicMock()
        mock_blob = MagicMock()

        mock_blob_service.get_container_client.return_value = mock_container
        mock_container.get_blob_client.return_value = mock_blob

        # Download fails with network error
        mock_blob.download_blob.side_effect = ServiceRequestError("Network error")

        client = AzureRegistryClient(
            "az://myaccount/mycontainer/registry/",
            mode="registry",
            cache_dir=temp_dir / "cache",
            blob_service_client=mock_blob_service,
        )

        with pytest.raises(AzureRegistryError, match="Network error"):
            client.get_package_info("test-plugin")

    def test_handles_missing_tarball_on_fetch(self, temp_dir: Path):
        """Raises AzureRegistryError when tarball not found during fetch."""
        from azure.core.exceptions import ResourceNotFoundError

        mock_blob_service = MagicMock()
        mock_container = MagicMock()
        mock_blob = MagicMock()

        mock_blob_service.get_container_client.return_value = mock_container
        mock_container.get_blob_client.return_value = mock_blob

        # Simulate missing tarball
        mock_blob.download_blob.side_effect = ResourceNotFoundError("Not found")

        client = AzureRegistryClient(
            "az://myaccount/mycontainer/registry/",
            mode="registry",
            cache_dir=temp_dir / "cache",
            blob_service_client=mock_blob_service,
        )

        from dex.registry.base import ResolvedPackage

        resolved = ResolvedPackage(
            name="test-plugin",
            version="1.0.0",
            resolved_url="az://myaccount/mycontainer/registry/test-plugin-1.0.0.tar.gz",
            local_path=None,
        )

        dest_dir = temp_dir / "dest"
        # Case-insensitive match for "Not found"
        with pytest.raises(AzureRegistryError, match="(?i)not found"):
            client.fetch_package(resolved, dest_dir)

    def test_handles_auth_error_on_fetch(self, temp_dir: Path):
        """Raises AzureRegistryError when auth fails during fetch."""
        from azure.core.exceptions import ClientAuthenticationError

        mock_blob_service = MagicMock()
        mock_container = MagicMock()
        mock_blob = MagicMock()

        mock_blob_service.get_container_client.return_value = mock_container
        mock_container.get_blob_client.return_value = mock_blob

        # Simulate auth error during download
        mock_blob.download_blob.side_effect = ClientAuthenticationError("Authentication failed")

        client = AzureRegistryClient(
            "az://myaccount/mycontainer/registry/",
            mode="registry",
            cache_dir=temp_dir / "cache",
            blob_service_client=mock_blob_service,
        )

        from dex.registry.base import ResolvedPackage

        resolved = ResolvedPackage(
            name="test-plugin",
            version="1.0.0",
            resolved_url="az://myaccount/mycontainer/registry/test-plugin-1.0.0.tar.gz",
            local_path=None,
        )

        dest_dir = temp_dir / "dest"
        with pytest.raises(AzureRegistryError, match="Authentication failed"):
            client.fetch_package(resolved, dest_dir)


class TestAzureRegistryClientVersionResolution:
    """Tests for version resolution."""

    def test_resolves_caret_version(self, temp_dir: Path):
        """Resolves caret version range."""
        mock_blob_service = MagicMock()
        mock_container = MagicMock()
        mock_blob = MagicMock()

        mock_blob_service.get_container_client.return_value = mock_container
        mock_container.get_blob_client.return_value = mock_blob

        registry_data = {
            "packages": {
                "test-plugin": {
                    "versions": ["1.0.0", "1.1.0", "1.2.0", "2.0.0"],
                    "latest": "2.0.0",
                }
            }
        }
        mock_download = MagicMock()
        mock_download.readall.return_value = json.dumps(registry_data).encode()
        mock_blob.download_blob.return_value = mock_download

        client = AzureRegistryClient(
            "az://myaccount/mycontainer/registry/",
            mode="registry",
            cache_dir=temp_dir / "cache",
            blob_service_client=mock_blob_service,
        )

        # ^1.0.0 should resolve to highest 1.x.x
        resolved = client.resolve_package("test-plugin", "^1.0.0")

        assert resolved is not None
        assert resolved.version == "1.2.0"

    def test_resolves_tilde_version(self, temp_dir: Path):
        """Resolves tilde version range."""
        mock_blob_service = MagicMock()
        mock_container = MagicMock()
        mock_blob = MagicMock()

        mock_blob_service.get_container_client.return_value = mock_container
        mock_container.get_blob_client.return_value = mock_blob

        registry_data = {
            "packages": {
                "test-plugin": {
                    "versions": ["1.0.0", "1.0.5", "1.1.0", "1.2.0"],
                    "latest": "1.2.0",
                }
            }
        }
        mock_download = MagicMock()
        mock_download.readall.return_value = json.dumps(registry_data).encode()
        mock_blob.download_blob.return_value = mock_download

        client = AzureRegistryClient(
            "az://myaccount/mycontainer/registry/",
            mode="registry",
            cache_dir=temp_dir / "cache",
            blob_service_client=mock_blob_service,
        )

        # ~1.0.0 should resolve to highest 1.0.x
        resolved = client.resolve_package("test-plugin", "~1.0.0")

        assert resolved is not None
        assert resolved.version == "1.0.5"

    def test_returns_none_for_unmatched_version(self, temp_dir: Path):
        """Returns None when no version matches."""
        mock_blob_service = MagicMock()
        mock_container = MagicMock()
        mock_blob = MagicMock()

        mock_blob_service.get_container_client.return_value = mock_container
        mock_container.get_blob_client.return_value = mock_blob

        registry_data = {"packages": {"test-plugin": {"versions": ["1.0.0"], "latest": "1.0.0"}}}
        mock_download = MagicMock()
        mock_download.readall.return_value = json.dumps(registry_data).encode()
        mock_blob.download_blob.return_value = mock_download

        client = AzureRegistryClient(
            "az://myaccount/mycontainer/registry/",
            mode="registry",
            cache_dir=temp_dir / "cache",
            blob_service_client=mock_blob_service,
        )

        # No 2.x.x versions available
        resolved = client.resolve_package("test-plugin", "^2.0.0")

        assert resolved is None
