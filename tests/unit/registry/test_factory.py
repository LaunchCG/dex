"""Tests for dex.registry.factory module."""

from pathlib import Path

import pytest

from dex.registry.factory import (
    UnsupportedProtocolError,
    create_registry_client,
    is_git_source,
    is_https_source,
    is_local_source,
    is_s3_source,
    normalize_source,
)
from dex.registry.git import GitRegistryClient
from dex.registry.https import HttpsRegistryClient
from dex.registry.local import LocalRegistryClient
from dex.registry.s3 import S3RegistryClient


class TestCreateRegistryClient:
    """Tests for create_registry_client function."""

    def test_creates_local_client_for_file_url(self, temp_dir: Path):
        """Creates LocalRegistryClient for file:// URL."""
        client = create_registry_client(f"file://{temp_dir}")
        assert isinstance(client, LocalRegistryClient)

    def test_creates_local_client_for_file_path(self, temp_dir: Path):
        """Creates LocalRegistryClient for file: relative path."""
        client = create_registry_client("file:./path")
        assert isinstance(client, LocalRegistryClient)

    def test_creates_local_client_for_plain_path(self, temp_dir: Path):
        """Creates LocalRegistryClient for plain path."""
        client = create_registry_client(str(temp_dir))
        assert isinstance(client, LocalRegistryClient)

    def test_creates_https_client_for_https_url(self, temp_dir: Path):
        """Creates HttpsRegistryClient for https:// URL."""
        client = create_registry_client(
            "https://example.com/registry/", cache_dir=temp_dir / "cache"
        )
        assert isinstance(client, HttpsRegistryClient)

    def test_raises_for_http(self):
        """Raises UnsupportedProtocolError for HTTP."""
        with pytest.raises(UnsupportedProtocolError):
            create_registry_client("http://example.com/registry")

    def test_creates_s3_client_for_s3_url(self, temp_dir: Path):
        """Creates S3RegistryClient for s3:// URL."""
        client = create_registry_client("s3://bucket/registry", cache_dir=temp_dir / "cache")
        assert isinstance(client, S3RegistryClient)

    def test_creates_git_client_for_git_url(self, temp_dir: Path):
        """Creates GitRegistryClient for git+ URL."""
        client = create_registry_client(
            "git+https://github.com/user/repo.git", cache_dir=temp_dir / "cache"
        )
        assert isinstance(client, GitRegistryClient)

    def test_raises_for_plain_git(self):
        """Raises UnsupportedProtocolError for plain git:// (not git+)."""
        with pytest.raises(UnsupportedProtocolError):
            create_registry_client("git://github.com/repo")

    def test_raises_for_unknown_protocol(self):
        """Raises UnsupportedProtocolError for unknown protocol."""
        with pytest.raises(UnsupportedProtocolError):
            create_registry_client("custom://some/path")


class TestIsLocalSource:
    """Tests for is_local_source function."""

    def test_file_url_is_local(self):
        """file:// URL is local."""
        assert is_local_source("file:///path/to/registry") is True

    def test_file_relative_is_local(self):
        """file: relative path is local."""
        assert is_local_source("file:./path") is True
        assert is_local_source("file:../path") is True

    def test_relative_path_is_local(self):
        """Relative path is local."""
        assert is_local_source("./path/to/plugin") is True
        assert is_local_source("../sibling/plugin") is True

    def test_absolute_path_is_local(self):
        """Absolute path is local."""
        assert is_local_source("/absolute/path") is True

    def test_windows_path_is_local(self):
        """Windows absolute path is local."""
        assert is_local_source("C:/path/to/registry") is True
        assert is_local_source("D:\\path\\to\\registry") is True

    def test_https_is_not_local(self):
        """HTTPS URL is not local."""
        assert is_local_source("https://example.com/registry") is False

    def test_s3_is_not_local(self):
        """S3 URL is not local."""
        assert is_local_source("s3://bucket/registry") is False


class TestNormalizeSource:
    """Tests for normalize_source function."""

    def test_file_url_unchanged(self):
        """file:// URL is unchanged."""
        assert normalize_source("file:///path/to/registry") == "file:///path/to/registry"

    def test_file_relative_unchanged(self):
        """file: relative path is unchanged."""
        assert normalize_source("file:./path") == "file:./path"

    def test_normalizes_relative_dot_path(self):
        """Normalizes ./ relative path."""
        assert normalize_source("./path/to/plugin") == "file:./path/to/plugin"

    def test_normalizes_relative_dotdot_path(self):
        """Normalizes ../ relative path."""
        assert normalize_source("../sibling/plugin") == "file:../sibling/plugin"

    def test_normalizes_absolute_unix_path(self):
        """Normalizes absolute Unix path."""
        assert normalize_source("/absolute/path") == "file:///absolute/path"

    def test_normalizes_windows_path(self):
        """Normalizes Windows absolute path."""
        assert normalize_source("C:/path/to/registry") == "file:///C:/path/to/registry"
        assert normalize_source("D:\\path\\to\\registry") == "file:///D:\\path\\to\\registry"

    def test_other_urls_unchanged(self):
        """Other URLs are unchanged."""
        assert normalize_source("https://example.com") == "https://example.com"
        assert normalize_source("s3://bucket/path") == "s3://bucket/path"


class TestIsGitSource:
    """Tests for is_git_source function."""

    def test_git_https_is_git(self):
        """git+https:// URL is git source."""
        assert is_git_source("git+https://github.com/user/repo.git") is True

    def test_git_ssh_is_git(self):
        """git+ssh:// URL is git source."""
        assert is_git_source("git+ssh://git@github.com/user/repo.git") is True

    def test_plain_git_is_not_git(self):
        """Plain git:// URL is not a git source (needs git+ prefix)."""
        assert is_git_source("git://github.com/repo") is False

    def test_https_is_not_git(self):
        """Plain https:// URL is not git source."""
        assert is_git_source("https://github.com/user/repo.git") is False

    def test_file_is_not_git(self):
        """File URL is not git source."""
        assert is_git_source("file:///path/to/repo") is False


class TestIsS3Source:
    """Tests for is_s3_source function."""

    def test_s3_url_is_s3(self):
        """s3:// URL is S3 source."""
        assert is_s3_source("s3://bucket/path") is True

    def test_s3_bucket_only_is_s3(self):
        """s3://bucket is S3 source."""
        assert is_s3_source("s3://bucket") is True

    def test_https_is_not_s3(self):
        """https:// URL is not S3 source."""
        assert is_s3_source("https://bucket.s3.amazonaws.com") is False

    def test_file_is_not_s3(self):
        """File URL is not S3 source."""
        assert is_s3_source("file:///path/to/dir") is False


class TestIsHttpsSource:
    """Tests for is_https_source function."""

    def test_https_url_is_https(self):
        """https:// URL is HTTPS source."""
        assert is_https_source("https://example.com/registry") is True

    def test_https_tarball_is_https(self):
        """https:// tarball URL is HTTPS source."""
        assert is_https_source("https://example.com/plugin-1.0.0.tar.gz") is True

    def test_http_is_not_https(self):
        """http:// URL is not HTTPS source."""
        assert is_https_source("http://example.com/registry") is False

    def test_s3_is_not_https(self):
        """s3:// URL is not HTTPS source."""
        assert is_https_source("s3://bucket/path") is False

    def test_file_is_not_https(self):
        """File URL is not HTTPS source."""
        assert is_https_source("file:///path/to/dir") is False


class TestUnsupportedProtocolError:
    """Tests for UnsupportedProtocolError exception."""

    def test_error_attributes(self):
        """Error has protocol and url attributes."""
        error = UnsupportedProtocolError("https", "https://example.com")
        assert error.protocol == "https"
        assert error.url == "https://example.com"

    def test_error_message(self):
        """Error has descriptive message."""
        error = UnsupportedProtocolError("https", "https://example.com")
        assert "https" in str(error)
        assert "https://example.com" in str(error)
