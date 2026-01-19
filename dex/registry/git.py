"""Git registry client for Git-hosted packages."""

from __future__ import annotations

import json
import logging
import re
import shutil
import subprocess
import tempfile
from dataclasses import dataclass
from pathlib import Path
from typing import Any

from dex.registry.base import PackageInfo, RegistryClient, ResolvedPackage
from dex.registry.cache import RegistryCache
from dex.registry.common import (
    SourceMode,
    extract_package_from_manifest_data,
    extract_package_from_registry_data,
)
from dex.utils.version import find_best_version

logger = logging.getLogger(__name__)


class GitRegistryError(Exception):
    """Error interacting with Git repository."""

    def __init__(self, message: str, url: str | None = None, ref: str | None = None):
        self.url = url
        self.ref = ref
        super().__init__(message)


@dataclass
class GitRef:
    """Parsed Git reference."""

    ref_type: str  # "tag", "branch", "commit", or "default"
    value: str | None


class GitRegistryClient(RegistryClient):
    """Registry client for Git-hosted packages.

    Supports two modes (specified explicitly, not auto-detected):
    1. Registry mode: Repository contains registry.json with multiple packages
    2. Package mode: Repository IS the plugin - package.json at repo root

    URL format: git+https://github.com/user/repo.git#ref
    - git+https://github.com/user/repo.git (default branch)
    - git+https://github.com/user/repo.git#v1.0.0 (tag or branch)
    - git+https://github.com/user/repo.git#tag=v1.0.0 (explicit tag)
    - git+https://github.com/user/repo.git#branch=main (explicit branch)
    - git+ssh://git@github.com/user/repo.git#v1.0.0 (SSH URL)

    Uses system `git` command for all operations (no gitpython dependency).
    Performs shallow clones (--depth 1) for efficiency.
    """

    def __init__(
        self,
        url: str,
        mode: SourceMode = "package",
        cache_dir: Path | None = None,
    ):
        """Initialize the Git registry client.

        Args:
            url: Git URL (git+https://... or git+ssh://...)
            mode: Source mode - "registry" expects registry.json,
                  "package" expects package.json
            cache_dir: Optional directory for caching clones
        """
        self._url = url
        self._mode = mode
        self._repo_url, self._ref = self._parse_url(url)

        logger.info("Initializing Git registry client for %s (mode=%s)", self._repo_url, mode)
        logger.debug("Git ref: %s=%s", self._ref.ref_type, self._ref.value)

        # Set up caching
        if cache_dir is None:
            cache_dir = Path(tempfile.gettempdir()) / "dex-cache" / "git"
        self._cache = RegistryCache(cache_dir)
        logger.debug("Using cache directory: %s", cache_dir)

        # Cached data
        self._cached_manifest: dict[str, Any] | None = None
        self._cached_registry: dict[str, Any] | None = None

    @staticmethod
    def _parse_url(url: str) -> tuple[str, GitRef]:
        """Parse Git URL into repository URL and ref.

        Args:
            url: Git URL (git+https://... or git+ssh://...)

        Returns:
            Tuple of (repo_url, ref)

        Raises:
            GitRegistryError: If URL is invalid
        """
        if not url.startswith("git+"):
            raise GitRegistryError(f"Invalid Git URL: must start with 'git+': {url}", url=url)

        # Remove git+ prefix
        git_url = url[4:]

        # Split URL and fragment (ref)
        ref = GitRef(ref_type="default", value=None)

        if "#" in git_url:
            repo_url, fragment = git_url.split("#", 1)

            if "=" in fragment:
                # Explicit ref type: tag=v1.0.0 or branch=main
                ref_type, value = fragment.split("=", 1)
                if ref_type in ("tag", "branch", "commit"):
                    ref = GitRef(ref_type=ref_type, value=value)
                else:
                    raise GitRegistryError(
                        f"Invalid ref type: {ref_type} (must be tag, branch, or commit)",
                        url=url,
                    )
            else:
                # Implicit ref (could be tag or branch)
                ref = GitRef(ref_type="tag", value=fragment)
        else:
            repo_url = git_url

        # Validate URL scheme
        if not repo_url.startswith(("https://", "ssh://", "git@")):
            raise GitRegistryError(
                f"Invalid Git URL scheme: must be https://, ssh://, or git@: {repo_url}",
                url=url,
            )

        return repo_url, ref

    @property
    def protocol(self) -> str:
        return "git"

    @property
    def mode(self) -> SourceMode:
        """Get the source mode."""
        return self._mode

    @property
    def repo_url(self) -> str:
        """Get the Git repository URL."""
        return self._repo_url

    @property
    def ref(self) -> GitRef:
        """Get the Git ref."""
        return self._ref

    def _run_git(
        self, args: list[str], cwd: Path | None = None, check: bool = True
    ) -> subprocess.CompletedProcess[str]:
        """Run a git command.

        Args:
            args: Git command arguments (without 'git')
            cwd: Working directory
            check: Whether to raise on non-zero exit

        Returns:
            Completed process

        Raises:
            GitRegistryError: If command fails and check=True
        """
        cmd = ["git"] + args
        logger.debug("Running git command: %s", " ".join(cmd))
        try:
            result = subprocess.run(
                cmd,
                cwd=cwd,
                capture_output=True,
                text=True,
                check=check,
            )
            return result
        except subprocess.CalledProcessError as e:
            logger.error("Git command failed: %s - %s", " ".join(cmd), e.stderr.strip())
            raise GitRegistryError(
                f"Git command failed: {' '.join(cmd)}\n{e.stderr}",
                url=self._repo_url,
            ) from e
        except FileNotFoundError as e:
            logger.error("Git is not installed or not in PATH")
            raise GitRegistryError(
                "Git is not installed or not in PATH",
                url=self._repo_url,
            ) from e

    def _get_tags(self) -> list[str]:
        """Get all tags from the remote repository.

        Returns:
            List of tag names (sorted by version)

        Raises:
            GitRegistryError: If unable to fetch tags
        """
        result = self._run_git(["ls-remote", "--tags", "--refs", self._repo_url])
        tags = []
        for line in result.stdout.strip().split("\n"):
            if line:
                # Format: <sha>\trefs/tags/<tagname>
                parts = line.split("\t")
                if len(parts) == 2:
                    tag = parts[1].replace("refs/tags/", "")
                    tags.append(tag)
        return tags

    def _clone_repo(self, dest: Path, ref: str | None = None) -> Path:
        """Clone the repository to a destination.

        Args:
            dest: Destination directory
            ref: Optional ref to checkout

        Returns:
            Path to cloned repository

        Raises:
            GitRegistryError: If clone fails
        """
        logger.info("Cloning repository %s to %s", self._repo_url, dest)
        if ref:
            logger.debug("Using ref: %s", ref)

        # Build clone command
        args = ["clone", "--depth", "1"]

        if ref:
            args.extend(["--branch", ref])

        args.extend([self._repo_url, str(dest)])

        self._run_git(args)

        # Remove .git directory
        git_dir = dest / ".git"
        if git_dir.exists():
            logger.debug("Removing .git directory from clone")
            shutil.rmtree(git_dir)

        logger.debug("Clone completed successfully")
        return dest

    def _get_clone_path(self) -> Path | None:
        """Get path to cloned repository, using cache or cloning if needed.

        Returns:
            Path to cloned repository, or None if clone fails

        Raises:
            GitRegistryError: If unable to clone and error is not recoverable
        """
        # Build cache key
        cache_key = f"{self._repo_url}#{self._ref.ref_type}={self._ref.value or 'HEAD'}"

        # Check cache
        cached_path = self._cache.get(cache_key)
        if cached_path and cached_path.exists():
            return cached_path

        # Clone to temp directory
        with tempfile.TemporaryDirectory() as temp_dir:
            clone_dest = Path(temp_dir) / "repo"

            ref_value = self._ref.value if self._ref.ref_type != "default" else None
            self._clone_repo(clone_dest, ref_value)

            # Cache the clone
            final_path = self._cache.put(cache_key, clone_dest)
            return final_path

    def _get_registry_data(self) -> dict[str, Any]:
        """Load registry.json from cloned repository.

        Returns:
            Parsed registry.json contents

        Raises:
            GitRegistryError: If registry.json not found or invalid
        """
        if self._cached_registry is not None:
            return self._cached_registry

        clone_path = self._get_clone_path()
        if clone_path is None:
            raise GitRegistryError(
                f"Failed to clone repository: {self._repo_url}",
                url=self._repo_url,
            )

        registry_file = clone_path / "registry.json"
        if not registry_file.exists():
            raise GitRegistryError(
                f"Registry not found: {self._repo_url} does not contain registry.json",
                url=self._repo_url,
            )

        try:
            with open(registry_file, encoding="utf-8") as f:
                self._cached_registry = json.load(f)
                return self._cached_registry
        except json.JSONDecodeError as e:
            raise GitRegistryError(
                f"Invalid JSON in registry.json: {e}",
                url=self._repo_url,
            ) from e

    def _get_manifest_data(self) -> dict[str, Any]:
        """Load package.json from cloned repository.

        Returns:
            Parsed package.json contents

        Raises:
            GitRegistryError: If package.json not found or invalid
        """
        if self._cached_manifest is not None:
            return self._cached_manifest

        clone_path = self._get_clone_path()
        if clone_path is None:
            raise GitRegistryError(
                f"Failed to clone repository: {self._repo_url}",
                url=self._repo_url,
            )

        package_file = clone_path / "package.json"
        if not package_file.exists():
            raise GitRegistryError(
                f"Package not found: {self._repo_url} does not contain package.json",
                url=self._repo_url,
            )

        try:
            with open(package_file, encoding="utf-8") as f:
                self._cached_manifest = json.load(f)
                return self._cached_manifest
        except json.JSONDecodeError as e:
            raise GitRegistryError(
                f"Invalid JSON in package.json: {e}",
                url=self._repo_url,
            ) from e

    def get_package_info(self, name: str) -> PackageInfo | None:
        """Get package information from Git repository.

        Args:
            name: Package name to look up

        Returns:
            PackageInfo if found, None if package doesn't exist

        Raises:
            GitRegistryError: If there's an error cloning or reading files
        """
        logger.debug("Getting package info for '%s' from Git (mode=%s)", name, self._mode)

        if self._mode == "registry":
            return self._get_package_from_registry(name)
        else:
            return self._get_package_from_manifest(name)

    def _get_package_from_registry(self, name: str) -> PackageInfo | None:
        """Get package info from registry.json.

        Returns:
            PackageInfo if found, None if package not in registry

        Raises:
            GitRegistryError: If registry.json cannot be read or parsed
        """
        registry_data = self._get_registry_data()
        return extract_package_from_registry_data(registry_data, name)

    def _get_package_from_manifest(self, name: str) -> PackageInfo | None:
        """Get package info from package.json (package mode).

        Returns:
            PackageInfo if found and name matches, None if name doesn't match

        Raises:
            GitRegistryError: If package.json cannot be read or parsed
        """
        manifest_data = self._get_manifest_data()
        info = extract_package_from_manifest_data(manifest_data, name)
        if info is None:
            return None

        # For Git, enhance with version tags
        tags = self._get_tags()
        logger.debug("Found %d tags in repository", len(tags))

        # Filter to semver-like tags (v1.0.0 or 1.0.0)
        version_pattern = re.compile(r"^v?(\d+\.\d+\.\d+.*)$")
        versions = []
        for tag in tags:
            match = version_pattern.match(tag)
            if match:
                # Normalize to version without 'v' prefix
                versions.append(match.group(1))

        # If no version tags, use manifest version
        if not versions:
            versions = info.versions

        # Sort versions
        versions.sort()
        logger.debug("Package '%s' has versions: %s", name, versions)

        return PackageInfo(
            name=info.name,
            versions=versions,
            latest=versions[-1] if versions else info.latest,
        )

    def resolve_package(self, name: str, version: str) -> ResolvedPackage | None:
        """Resolve a package to a Git ref.

        Args:
            name: Package name
            version: Version specifier ('latest', exact, or semver range)

        Returns:
            ResolvedPackage with Git URL, or None if package/version not found

        Raises:
            GitRegistryError: If there's an error reading files
        """
        logger.info("Resolving package '%s' version '%s' from Git", name, version)
        info = self.get_package_info(name)
        if info is None:
            logger.warning("Package '%s' not found in Git repository", name)
            return None

        # Determine resolved version
        resolved_version: str | None
        if version == "latest":
            resolved_version = info.latest
        else:
            resolved_version = find_best_version(version, info.versions)

        if resolved_version is None:
            logger.warning("Version '%s' not found for package '%s'", version, name)
            return None

        # Build resolved URL with appropriate ref
        # Try with 'v' prefix first (common convention)
        tags = self._get_tags()
        ref: str | None
        if f"v{resolved_version}" in tags:
            ref = f"v{resolved_version}"
        elif resolved_version in tags:
            ref = resolved_version
        else:
            # Fall back to the current ref or default branch
            ref = self._ref.value

        resolved_url = f"git+{self._repo_url}#tag={ref}" if ref else f"git+{self._repo_url}"
        logger.info(
            "Resolved package '%s' to version %s at %s", name, resolved_version, resolved_url
        )

        return ResolvedPackage(
            name=name,
            version=resolved_version,
            resolved_url=resolved_url,
            local_path=None,
        )

    def fetch_package(self, resolved: ResolvedPackage, dest_dir: Path) -> Path:
        """Fetch a package from Git to a local directory.

        Args:
            resolved: Resolved package information
            dest_dir: Destination directory

        Returns:
            Path to the fetched package

        Raises:
            GitRegistryError: If fetch fails
        """
        logger.info(
            "Fetching package '%s' v%s from Git to %s", resolved.name, resolved.version, dest_dir
        )
        dest_dir.mkdir(parents=True, exist_ok=True)

        # Parse the resolved URL to get the ref
        _, ref = self._parse_url(resolved.resolved_url)

        # Build cache key
        cache_key = f"{self._repo_url}#{ref.ref_type}={ref.value or 'HEAD'}"

        # Check cache
        cached_path = self._cache.get(cache_key)
        if cached_path and cached_path.exists():
            logger.debug("Using cached clone from %s", cached_path)
            # Copy from cache
            plugin_dir = dest_dir / resolved.name
            if plugin_dir.exists():
                shutil.rmtree(plugin_dir)
            shutil.copytree(cached_path, plugin_dir)
            logger.info("Package '%s' fetched from cache", resolved.name)
            return plugin_dir

        # Clone directly to destination
        plugin_dir = dest_dir / resolved.name
        if plugin_dir.exists():
            shutil.rmtree(plugin_dir)

        ref_value = ref.value if ref.ref_type != "default" else None
        self._clone_repo(plugin_dir, ref_value)

        # Verify expected file exists based on mode
        if self._mode == "package" and not (plugin_dir / "package.json").exists():
            logger.error(
                "Cloned repository does not contain package.json: %s", resolved.resolved_url
            )
            raise GitRegistryError(
                f"Cloned repository does not contain package.json: {resolved.resolved_url}",
                url=resolved.resolved_url,
            )
        # For registry mode, individual packages are subdirectories - handled elsewhere

        # Cache for future use
        self._cache.put(cache_key, plugin_dir)
        logger.info("Package '%s' fetched and cached", resolved.name)

        return plugin_dir

    def list_packages(self) -> list[str]:
        """List packages in the Git repository.

        Returns:
            List of package names

        Raises:
            GitRegistryError: If there's an error reading files
        """
        if self._mode == "registry":
            registry_data = self._get_registry_data()
            return list(registry_data.get("packages", {}).keys())
        else:
            manifest_data = self._get_manifest_data()
            name = manifest_data.get("name")
            return [name] if name else []
