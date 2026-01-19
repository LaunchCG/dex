"""Tests for dex.registry.git module."""

from pathlib import Path
from unittest.mock import MagicMock, patch

import pytest

from dex.registry.git import GitRef, GitRegistryClient, GitRegistryError


class TestGitRegistryClientParseUrl:
    """Tests for Git URL parsing."""

    def test_parses_https_url(self):
        """Parses git+https:// URL."""
        repo_url, ref = GitRegistryClient._parse_url("git+https://github.com/user/repo.git")

        assert repo_url == "https://github.com/user/repo.git"
        assert ref.ref_type == "default"
        assert ref.value is None

    def test_parses_ssh_url(self):
        """Parses git+ssh:// URL."""
        repo_url, ref = GitRegistryClient._parse_url("git+ssh://git@github.com/user/repo.git")

        assert repo_url == "ssh://git@github.com/user/repo.git"
        assert ref.ref_type == "default"

    def test_parses_git_at_url(self):
        """Parses git+git@... URL."""
        repo_url, ref = GitRegistryClient._parse_url("git+git@github.com:user/repo.git")

        assert repo_url == "git@github.com:user/repo.git"
        assert ref.ref_type == "default"

    def test_parses_url_with_tag_fragment(self):
        """Parses URL with tag fragment."""
        repo_url, ref = GitRegistryClient._parse_url("git+https://github.com/user/repo.git#v1.0.0")

        assert repo_url == "https://github.com/user/repo.git"
        assert ref.ref_type == "tag"
        assert ref.value == "v1.0.0"

    def test_parses_url_with_explicit_tag(self):
        """Parses URL with explicit tag=... fragment."""
        repo_url, ref = GitRegistryClient._parse_url(
            "git+https://github.com/user/repo.git#tag=release-1.0"
        )

        assert repo_url == "https://github.com/user/repo.git"
        assert ref.ref_type == "tag"
        assert ref.value == "release-1.0"

    def test_parses_url_with_explicit_branch(self):
        """Parses URL with explicit branch=... fragment."""
        repo_url, ref = GitRegistryClient._parse_url(
            "git+https://github.com/user/repo.git#branch=develop"
        )

        assert repo_url == "https://github.com/user/repo.git"
        assert ref.ref_type == "branch"
        assert ref.value == "develop"

    def test_parses_url_with_explicit_commit(self):
        """Parses URL with explicit commit=... fragment."""
        repo_url, ref = GitRegistryClient._parse_url(
            "git+https://github.com/user/repo.git#commit=abc123"
        )

        assert repo_url == "https://github.com/user/repo.git"
        assert ref.ref_type == "commit"
        assert ref.value == "abc123"

    def test_raises_for_missing_git_prefix(self):
        """Raises error for URL without git+ prefix."""
        with pytest.raises(GitRegistryError, match="must start with 'git\\+'"):
            GitRegistryClient._parse_url("https://github.com/user/repo.git")

    def test_raises_for_invalid_scheme(self):
        """Raises error for invalid URL scheme."""
        with pytest.raises(GitRegistryError, match="Invalid Git URL scheme"):
            GitRegistryClient._parse_url("git+ftp://example.com/repo.git")

    def test_raises_for_invalid_ref_type(self):
        """Raises error for invalid ref type."""
        with pytest.raises(GitRegistryError, match="Invalid ref type"):
            GitRegistryClient._parse_url("git+https://github.com/user/repo.git#invalid=value")


class TestGitRegistryClientInit:
    """Tests for GitRegistryClient initialization."""

    def test_parses_url_on_init(self, temp_dir: Path):
        """Parses URL on initialization."""
        client = GitRegistryClient(
            "git+https://github.com/user/repo.git#v1.0.0",
            cache_dir=temp_dir / "cache",
        )

        assert client.repo_url == "https://github.com/user/repo.git"
        assert client.ref.ref_type == "tag"
        assert client.ref.value == "v1.0.0"
        assert client.protocol == "git"

    def test_default_mode_is_package(self, temp_dir: Path):
        """Default mode is 'package' for Git (repo is the package)."""
        client = GitRegistryClient(
            "git+https://github.com/user/repo.git",
            cache_dir=temp_dir / "cache",
        )

        assert client.mode == "package"

    def test_explicit_registry_mode(self, temp_dir: Path):
        """Can set mode to 'registry' explicitly."""
        client = GitRegistryClient(
            "git+https://github.com/user/repo.git",
            mode="registry",
            cache_dir=temp_dir / "cache",
        )

        assert client.mode == "registry"


class TestGitRegistryClientGetTags:
    """Tests for GitRegistryClient._get_tags()."""

    def test_parses_ls_remote_output(self, temp_dir: Path):
        """Parses git ls-remote --tags output."""
        client = GitRegistryClient(
            "git+https://github.com/user/repo.git",
            cache_dir=temp_dir / "cache",
        )

        mock_result = MagicMock()
        mock_result.stdout = """abc123\trefs/tags/v1.0.0
def456\trefs/tags/v2.0.0
ghi789\trefs/tags/v3.0.0-beta.1"""

        with patch.object(client, "_run_git", return_value=mock_result):
            tags = client._get_tags()

        assert tags == ["v1.0.0", "v2.0.0", "v3.0.0-beta.1"]

    def test_raises_on_git_error(self, temp_dir: Path):
        """Raises GitRegistryError when git fails."""
        client = GitRegistryClient(
            "git+https://github.com/user/repo.git",
            cache_dir=temp_dir / "cache",
        )

        with (
            patch.object(client, "_run_git", side_effect=GitRegistryError("failed")),
            pytest.raises(GitRegistryError),
        ):
            client._get_tags()


class TestGitRegistryClientGetPackageInfo:
    """Tests for GitRegistryClient.get_package_info()."""

    def test_gets_package_info_from_clone(self, temp_dir: Path):
        """Gets package info by cloning repository."""
        client = GitRegistryClient(
            "git+https://github.com/user/repo.git",
            cache_dir=temp_dir / "cache",
        )

        # Set up cached manifest
        client._cached_manifest = {
            "name": "test-plugin",
            "version": "1.0.0",
            "description": "Test",
        }

        # Mock tags
        with patch.object(client, "_get_tags", return_value=["v1.0.0", "v2.0.0"]):
            info = client.get_package_info("test-plugin")

        assert info is not None
        assert info.name == "test-plugin"
        assert "1.0.0" in info.versions
        assert "2.0.0" in info.versions
        assert info.latest == "2.0.0"

    def test_returns_none_for_wrong_name(self, temp_dir: Path):
        """Returns None when name doesn't match manifest."""
        client = GitRegistryClient(
            "git+https://github.com/user/repo.git",
            cache_dir=temp_dir / "cache",
        )

        client._cached_manifest = {"name": "actual-name", "version": "1.0.0"}

        with patch.object(client, "_get_tags", return_value=[]):
            info = client.get_package_info("wrong-name")

        assert info is None

    def test_uses_manifest_version_when_no_tags(self, temp_dir: Path):
        """Uses manifest version when no version tags exist."""
        client = GitRegistryClient(
            "git+https://github.com/user/repo.git",
            cache_dir=temp_dir / "cache",
        )

        client._cached_manifest = {"name": "test-plugin", "version": "0.1.0"}

        with patch.object(client, "_get_tags", return_value=[]):
            info = client.get_package_info("test-plugin")

        assert info is not None
        assert info.versions == ["0.1.0"]
        assert info.latest == "0.1.0"


class TestGitRegistryClientResolvePackage:
    """Tests for GitRegistryClient.resolve_package()."""

    def test_resolves_latest_to_highest_version(self, temp_dir: Path):
        """Resolves 'latest' to highest version tag."""
        client = GitRegistryClient(
            "git+https://github.com/user/repo.git",
            cache_dir=temp_dir / "cache",
        )

        client._cached_manifest = {"name": "test-plugin", "version": "1.0.0"}

        with patch.object(client, "_get_tags", return_value=["v1.0.0", "v2.0.0"]):
            resolved = client.resolve_package("test-plugin", "latest")

        assert resolved is not None
        assert resolved.name == "test-plugin"
        assert resolved.version == "2.0.0"
        assert "tag=v2.0.0" in resolved.resolved_url

    def test_resolves_specific_version(self, temp_dir: Path):
        """Resolves specific version."""
        client = GitRegistryClient(
            "git+https://github.com/user/repo.git",
            cache_dir=temp_dir / "cache",
        )

        client._cached_manifest = {"name": "test-plugin", "version": "1.0.0"}

        with patch.object(client, "_get_tags", return_value=["v1.0.0", "v2.0.0"]):
            resolved = client.resolve_package("test-plugin", "1.0.0")

        assert resolved is not None
        assert resolved.version == "1.0.0"
        assert "v1.0.0" in resolved.resolved_url

    def test_resolves_version_range(self, temp_dir: Path):
        """Resolves version range to best match."""
        client = GitRegistryClient(
            "git+https://github.com/user/repo.git",
            cache_dir=temp_dir / "cache",
        )

        client._cached_manifest = {"name": "test-plugin", "version": "1.0.0"}

        with patch.object(client, "_get_tags", return_value=["v1.0.0", "v1.5.0", "v2.0.0"]):
            resolved = client.resolve_package("test-plugin", "^1.0.0")

        assert resolved is not None
        assert resolved.version == "1.5.0"

    def test_returns_none_for_unknown_package(self, temp_dir: Path):
        """Returns None for unknown package."""
        client = GitRegistryClient(
            "git+https://github.com/user/repo.git",
            cache_dir=temp_dir / "cache",
        )

        client._cached_manifest = {"name": "other-name", "version": "1.0.0"}

        with patch.object(client, "_get_tags", return_value=[]):
            resolved = client.resolve_package("nonexistent", "latest")

        assert resolved is None


class TestGitRegistryClientListPackages:
    """Tests for GitRegistryClient.list_packages()."""

    def test_lists_single_package(self, temp_dir: Path):
        """Lists single package from repository."""
        client = GitRegistryClient(
            "git+https://github.com/user/repo.git",
            cache_dir=temp_dir / "cache",
        )

        client._cached_manifest = {"name": "my-plugin", "version": "1.0.0"}

        packages = client.list_packages()

        assert packages == ["my-plugin"]

    def test_returns_empty_when_no_name_in_manifest(self, temp_dir: Path):
        """Returns empty list when manifest has no name."""
        client = GitRegistryClient(
            "git+https://github.com/user/repo.git",
            cache_dir=temp_dir / "cache",
        )

        client._cached_manifest = {"version": "1.0.0"}

        packages = client.list_packages()

        assert packages == []


class TestGitRef:
    """Tests for GitRef dataclass."""

    def test_default_ref(self):
        """Creates default ref."""
        ref = GitRef(ref_type="default", value=None)

        assert ref.ref_type == "default"
        assert ref.value is None

    def test_tag_ref(self):
        """Creates tag ref."""
        ref = GitRef(ref_type="tag", value="v1.0.0")

        assert ref.ref_type == "tag"
        assert ref.value == "v1.0.0"


class TestGitRegistryError:
    """Tests for GitRegistryError."""

    def test_stores_url_and_ref(self):
        """Error stores URL and ref."""
        error = GitRegistryError(
            "Test error",
            url="https://github.com/user/repo.git",
            ref="v1.0.0",
        )

        assert error.url == "https://github.com/user/repo.git"
        assert error.ref == "v1.0.0"
        assert "Test error" in str(error)

    def test_handles_none_url_and_ref(self):
        """Error handles None URL and ref."""
        error = GitRegistryError("Test error")

        assert error.url is None
        assert error.ref is None
