"""Centralized utilities for registry clients.

This module provides shared functionality used across all registry clients:
- Tarball filename parsing
- Registry data extraction
- Package manifest handling
- Source mode type definitions
"""

from __future__ import annotations

import re
from typing import TYPE_CHECKING, Any, Literal

if TYPE_CHECKING:
    from dex.registry.base import PackageInfo

# Source mode: determines whether to expect registry.json or package.json
SourceMode = Literal["registry", "package"]


def parse_tarball_info(filename: str) -> dict[str, str]:
    """Parse package name and version from a tarball filename.

    Supports common naming patterns:
    - package-1.0.0.tar.gz
    - package-v1.0.0.tar.gz
    - package_1.0.0.tar.gz
    - package-1.0.0.tgz

    Args:
        filename: The tarball filename (not full path)

    Returns:
        Dict with 'name' and 'version' keys. If parsing fails,
        returns the basename as name with version "0.0.0".
    """
    # Remove .tar.gz or .tgz extension
    if filename.endswith(".tar.gz"):
        basename = filename[:-7]
    elif filename.endswith(".tgz"):
        basename = filename[:-4]
    else:
        return {"name": filename, "version": "0.0.0"}

    # Try to match common patterns: name-v?version or name_v?version
    patterns = [
        r"^(.+?)[_-]v?(\d+\.\d+\.\d+.*)$",  # name-1.0.0 or name-v1.0.0
        r"^(.+?)[_-]v?(\d+\.\d+)$",  # name-1.0
    ]

    for pattern in patterns:
        match = re.match(pattern, basename)
        if match:
            return {"name": match.group(1), "version": match.group(2)}

    # Couldn't parse version - return basename as name with unknown version
    return {"name": basename, "version": "0.0.0"}


def extract_package_from_registry_data(
    registry_data: dict[str, Any],
    name: str,
) -> PackageInfo | None:
    """Extract package info from parsed registry.json data.

    Args:
        registry_data: Parsed registry.json contents
        name: Package name to look up

    Returns:
        PackageInfo if package exists in registry, None otherwise
    """
    from dex.registry.base import PackageInfo

    packages = registry_data.get("packages", {})
    if name not in packages:
        return None

    pkg_data = packages[name]
    versions = pkg_data.get("versions", [])
    latest = pkg_data.get("latest", versions[-1] if versions else "0.0.0")

    return PackageInfo(
        name=name,
        versions=versions,
        latest=latest,
    )


def extract_package_from_manifest_data(
    manifest_data: dict[str, Any],
    expected_name: str,
) -> PackageInfo | None:
    """Extract package info from parsed package.json data.

    Args:
        manifest_data: Parsed package.json contents
        expected_name: The name we're looking for (must match)

    Returns:
        PackageInfo if name matches, None otherwise
    """
    from dex.registry.base import PackageInfo

    manifest_name = manifest_data.get("name", "")
    if manifest_name != expected_name:
        return None

    manifest_version = manifest_data.get("version", "0.0.0")

    return PackageInfo(
        name=manifest_name,
        versions=[manifest_version],
        latest=manifest_version,
    )


def names_match(name1: str, name2: str) -> bool:
    """Check if two package names match (handles - vs _ normalization).

    Args:
        name1: First package name
        name2: Second package name

    Returns:
        True if names match (exact or normalized)
    """
    if name1 == name2:
        return True
    # Normalize dashes and underscores
    return name1.replace("-", "_") == name2.replace("-", "_")
