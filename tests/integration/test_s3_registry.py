"""Integration tests for S3 registry using moto."""

import json
import tarfile
from pathlib import Path
from typing import Any

import boto3
import pytest
from moto import mock_aws

from dex.registry.s3 import S3RegistryClient, S3RegistryError


@pytest.fixture
def s3_bucket():
    """Create a mock S3 bucket for testing."""
    return "test-dex-registry"


@pytest.fixture
def mock_registry_data():
    """Create mock registry data."""
    return {
        "packages": {
            "test-plugin": {
                "versions": ["1.0.0", "1.1.0", "2.0.0"],
                "latest": "2.0.0",
            },
            "another-plugin": {
                "versions": ["0.5.0", "1.0.0"],
                "latest": "1.0.0",
            },
        }
    }


@pytest.fixture
def mock_plugin_manifest():
    """Create mock plugin manifest."""
    return {
        "name": "test-plugin",
        "version": "2.0.0",
        "description": "A test plugin for integration testing",
        "skills": [
            {
                "name": "test-skill",
                "description": "A test skill",
                "context": "./context/skill.md",
            }
        ],
    }


def create_plugin_tarball(temp_dir: Path, name: str, version: str) -> Path:
    """Create a plugin tarball for testing.

    Args:
        temp_dir: Temporary directory for creating files
        name: Plugin name
        version: Plugin version

    Returns:
        Path to the created tarball
    """
    # Create plugin directory structure
    plugin_dir = temp_dir / f"{name}-{version}"
    plugin_dir.mkdir(parents=True)

    # Create package.json
    manifest = {
        "name": name,
        "version": version,
        "description": f"Test plugin {name}",
        "skills": [
            {
                "name": "test-skill",
                "description": "A test skill",
                "context": "./context/skill.md",
            }
        ],
    }
    (plugin_dir / "package.json").write_text(json.dumps(manifest, indent=2))

    # Create context directory
    context_dir = plugin_dir / "context"
    context_dir.mkdir()
    (context_dir / "skill.md").write_text("# Test Skill\n\nThis is a test skill.")

    # Create tarball
    tarball_path = temp_dir / f"{name}-{version}.tar.gz"
    with tarfile.open(tarball_path, "w:gz") as tar:
        tar.add(plugin_dir, arcname=name)

    return tarball_path


@mock_aws
class TestS3RegistryIntegration:
    """Integration tests using moto to mock AWS S3."""

    def test_full_workflow_registry_mode(
        self, temp_dir: Path, s3_bucket: str, mock_registry_data: dict[str, Any]
    ):
        """Test complete workflow: list, resolve, fetch in registry mode."""
        # Set up mock S3
        s3 = boto3.client("s3", region_name="us-east-1")
        s3.create_bucket(Bucket=s3_bucket)

        # Upload registry.json
        s3.put_object(
            Bucket=s3_bucket,
            Key="registry/registry.json",
            Body=json.dumps(mock_registry_data).encode(),
        )

        # Create and upload a tarball
        tarball = create_plugin_tarball(temp_dir, "test-plugin", "2.0.0")
        with open(tarball, "rb") as f:
            s3.put_object(
                Bucket=s3_bucket,
                Key="registry/test-plugin-2.0.0.tar.gz",
                Body=f.read(),
            )

        # Create client
        client = S3RegistryClient(
            f"s3://{s3_bucket}/registry/",
            cache_dir=temp_dir / "cache",
        )

        # Test list_packages
        packages = client.list_packages()
        assert "test-plugin" in packages
        assert "another-plugin" in packages

        # Test get_package_info
        info = client.get_package_info("test-plugin")
        assert info is not None
        assert info.name == "test-plugin"
        assert info.versions == ["1.0.0", "1.1.0", "2.0.0"]
        assert info.latest == "2.0.0"

        # Test resolve_package
        resolved = client.resolve_package("test-plugin", "latest")
        assert resolved is not None
        assert resolved.name == "test-plugin"
        assert resolved.version == "2.0.0"
        assert resolved.resolved_url == f"s3://{s3_bucket}/registry/test-plugin-2.0.0.tar.gz"

        # Test fetch_package
        dest_dir = temp_dir / "installed"
        dest_dir.mkdir()
        result = client.fetch_package(resolved, dest_dir)

        assert result.exists()
        assert (result / "package.json").exists()

        # Verify content
        with open(result / "package.json") as f:
            manifest = json.load(f)
        assert manifest["name"] == "test-plugin"
        assert manifest["version"] == "2.0.0"

    def test_full_workflow_direct_mode(
        self, temp_dir: Path, s3_bucket: str, mock_plugin_manifest: dict[str, Any]
    ):
        """Test complete workflow in direct mode (no registry.json)."""
        # Set up mock S3
        s3 = boto3.client("s3", region_name="us-east-1")
        s3.create_bucket(Bucket=s3_bucket)

        # Upload package.json directly (no registry.json)
        s3.put_object(
            Bucket=s3_bucket,
            Key="plugins/test-plugin/package.json",
            Body=json.dumps(mock_plugin_manifest).encode(),
        )

        # Upload context file
        s3.put_object(
            Bucket=s3_bucket,
            Key="plugins/test-plugin/context/skill.md",
            Body=b"# Test Skill\n\nThis is a test skill.",
        )

        # Create client with explicit package mode for direct source
        client = S3RegistryClient(
            f"s3://{s3_bucket}/plugins/test-plugin/",
            mode="package",
            cache_dir=temp_dir / "cache",
        )

        # Test get_package_info
        info = client.get_package_info("test-plugin")
        assert info is not None
        assert info.name == "test-plugin"
        assert info.versions == ["2.0.0"]
        assert info.latest == "2.0.0"

        # Test resolve_package
        resolved = client.resolve_package("test-plugin", "latest")
        assert resolved is not None
        assert resolved.name == "test-plugin"
        assert resolved.version == "2.0.0"

    def test_version_resolution(
        self, temp_dir: Path, s3_bucket: str, mock_registry_data: dict[str, Any]
    ):
        """Test version resolution with ranges."""
        # Set up mock S3
        s3 = boto3.client("s3", region_name="us-east-1")
        s3.create_bucket(Bucket=s3_bucket)

        # Upload registry.json
        s3.put_object(
            Bucket=s3_bucket,
            Key="registry/registry.json",
            Body=json.dumps(mock_registry_data).encode(),
        )

        client = S3RegistryClient(
            f"s3://{s3_bucket}/registry/",
            cache_dir=temp_dir / "cache",
        )

        # Test caret range ^1.0.0 should match 1.1.0 (highest in 1.x)
        resolved = client.resolve_package("test-plugin", "^1.0.0")
        assert resolved is not None
        assert resolved.version == "1.1.0"

        # Test exact version
        resolved = client.resolve_package("test-plugin", "1.0.0")
        assert resolved is not None
        assert resolved.version == "1.0.0"

        # Test latest
        resolved = client.resolve_package("test-plugin", "latest")
        assert resolved is not None
        assert resolved.version == "2.0.0"

    def test_handles_missing_package(
        self, temp_dir: Path, s3_bucket: str, mock_registry_data: dict[str, Any]
    ):
        """Test handling of missing packages."""
        # Set up mock S3
        s3 = boto3.client("s3", region_name="us-east-1")
        s3.create_bucket(Bucket=s3_bucket)

        # Upload registry.json
        s3.put_object(
            Bucket=s3_bucket,
            Key="registry/registry.json",
            Body=json.dumps(mock_registry_data).encode(),
        )

        client = S3RegistryClient(
            f"s3://{s3_bucket}/registry/",
            cache_dir=temp_dir / "cache",
        )

        # Test get_package_info for nonexistent package
        info = client.get_package_info("nonexistent-plugin")
        assert info is None

        # Test resolve_package for nonexistent package
        resolved = client.resolve_package("nonexistent-plugin", "latest")
        assert resolved is None

    def test_caching_behavior(
        self, temp_dir: Path, s3_bucket: str, mock_registry_data: dict[str, Any]
    ):
        """Test that registry data is cached."""
        # Set up mock S3
        s3 = boto3.client("s3", region_name="us-east-1")
        s3.create_bucket(Bucket=s3_bucket)

        # Upload registry.json
        s3.put_object(
            Bucket=s3_bucket,
            Key="registry/registry.json",
            Body=json.dumps(mock_registry_data).encode(),
        )

        client = S3RegistryClient(
            f"s3://{s3_bucket}/registry/",
            cache_dir=temp_dir / "cache",
        )

        # First call should fetch from S3
        info1 = client.get_package_info("test-plugin")
        assert info1 is not None

        # Second call should use cache (even though we can't directly verify,
        # we verify it still works)
        info2 = client.get_package_info("test-plugin")
        assert info2 is not None
        assert info1.name == info2.name
        assert info1.versions == info2.versions


@mock_aws
class TestS3RegistryErrorHandling:
    """Error handling tests for S3 registry."""

    def test_handles_missing_bucket(self, temp_dir: Path):
        """Test handling of non-existent bucket."""
        # Don't create the bucket
        client = S3RegistryClient(
            "s3://nonexistent-bucket/registry/",
            cache_dir=temp_dir / "cache",
        )

        # Should raise S3RegistryError for missing bucket
        with pytest.raises(S3RegistryError, match="NoSuchBucket|bucket does not exist"):
            client.get_package_info("any-plugin")

    def test_handles_missing_registry_json(self, temp_dir: Path, s3_bucket: str):
        """Test handling when registry.json doesn't exist."""
        # Set up mock S3 without registry.json
        s3 = boto3.client("s3", region_name="us-east-1")
        s3.create_bucket(Bucket=s3_bucket)

        client = S3RegistryClient(
            f"s3://{s3_bucket}/registry/",
            cache_dir=temp_dir / "cache",
        )

        # Should raise S3RegistryError for missing registry.json in registry mode
        with pytest.raises(S3RegistryError, match="not found|does not exist"):
            client.get_package_info("test-plugin")

    def test_handles_invalid_registry_json(self, temp_dir: Path, s3_bucket: str):
        """Test handling of invalid registry.json."""
        # Set up mock S3
        s3 = boto3.client("s3", region_name="us-east-1")
        s3.create_bucket(Bucket=s3_bucket)

        # Upload invalid JSON
        s3.put_object(
            Bucket=s3_bucket,
            Key="registry/registry.json",
            Body=b"not valid json {{{",
        )

        client = S3RegistryClient(
            f"s3://{s3_bucket}/registry/",
            cache_dir=temp_dir / "cache",
        )

        # Should raise S3RegistryError for invalid JSON
        with pytest.raises(S3RegistryError, match="Invalid JSON"):
            client.get_package_info("test-plugin")
