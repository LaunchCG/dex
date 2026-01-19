"""Platform-specific context file resolution.

This module provides utilities for resolving context files based on the target
platform, supporting platform-specific overrides using file naming conventions.

Convention:
    - `context.md` - default for all platforms
    - `context.claude_code.md` - Claude Code override
    - `context.{claude_code,cursor}.md` - shared override for multiple platforms

Platform identifiers (underscores, not hyphens):
    - claude_code
    - cursor
    - codex
    - github_copilot
    - antigravity
"""

from __future__ import annotations

import re
from pathlib import Path
from typing import Any

# Valid platform identifiers (using underscores)
PLATFORM_IDENTIFIERS = frozenset(
    {
        "claude_code",
        "cursor",
        "codex",
        "github_copilot",
        "antigravity",
    }
)

# Regex to match brace expansion syntax: {platform1,platform2,...}
BRACE_EXPANSION_PATTERN = re.compile(r"\{([^}]+)\}")


def normalize_adapter_name(adapter_name: str) -> str:
    """Normalize adapter name to platform identifier.

    Converts adapter names like 'claude-code' to 'claude_code'.

    Args:
        adapter_name: The adapter name (e.g., 'claude-code', 'github-copilot')

    Returns:
        Normalized platform identifier (e.g., 'claude_code', 'github_copilot')
    """
    return adapter_name.replace("-", "_")


def parse_brace_expansion(filename: str) -> set[str]:
    """Parse brace expansion syntax from filename.

    Args:
        filename: Filename potentially containing brace expansion
                  e.g., 'context.{claude_code,cursor}.md'

    Returns:
        Set of platform identifiers from brace expansion, or empty set if none
    """
    match = BRACE_EXPANSION_PATTERN.search(filename)
    if not match:
        return set()

    # Split on comma and strip whitespace
    platforms = {p.strip() for p in match.group(1).split(",")}
    # Only return valid platform identifiers
    return platforms & PLATFORM_IDENTIFIERS


def extract_platform_from_filename(filename: str, base_name: str) -> str | None:
    """Extract single platform identifier from filename.

    Args:
        filename: The filename to extract from (e.g., 'context.claude_code.md')
        base_name: The base name without extension (e.g., 'context')

    Returns:
        Platform identifier if found and valid, None otherwise
    """
    # Pattern: {base_name}.{platform}.md
    # First, remove the base_name prefix and .md suffix
    stem = Path(filename).stem  # e.g., 'context.claude_code'

    if not stem.startswith(f"{base_name}."):
        return None

    # Extract the platform part
    platform = stem[len(base_name) + 1 :]  # +1 for the dot

    # Skip if it contains braces (brace expansion case)
    if "{" in platform or "}" in platform:
        return None

    # Validate it's a known platform
    if platform in PLATFORM_IDENTIFIERS:
        return platform

    return None


def find_platform_specific_file(
    source_dir: Path,
    context_path: str,
    platform: str,
) -> str:
    """Find the best context file for the given platform.

    Resolution order:
    1. Platform-specific override (e.g., `context.claude_code.md`)
    2. Multi-platform override (e.g., `context.{claude_code,cursor}.md`)
    3. Default file (e.g., `context.md`)

    Args:
        source_dir: The plugin source directory
        context_path: The context path from config (e.g., './context.md' or 'context.md')
        platform: The target platform identifier (e.g., 'claude_code')

    Returns:
        The resolved context path (relative to source_dir)
    """
    # Normalize the context path
    if context_path.startswith("./"):
        context_path = context_path[2:]

    # Parse the path
    context_file = Path(context_path)
    context_dir = context_file.parent
    base_name = context_file.stem  # e.g., 'context'
    extension = context_file.suffix  # e.g., '.md'

    # Normalize platform name
    normalized_platform = normalize_adapter_name(platform)

    # Build search directory
    search_dir = source_dir / context_dir if context_dir != Path(".") else source_dir

    if not search_dir.exists():
        # Directory doesn't exist, return original path
        return context_path

    # 1. Check for platform-specific override
    platform_specific = f"{base_name}.{normalized_platform}{extension}"
    platform_specific_path = search_dir / platform_specific
    if platform_specific_path.exists():
        if context_dir != Path("."):
            return str(context_dir / platform_specific)
        return platform_specific

    # 2. Check for multi-platform override (brace expansion)
    for file in search_dir.iterdir():
        if not file.is_file():
            continue
        if not file.name.startswith(f"{base_name}."):
            continue
        if not file.name.endswith(extension):
            continue

        # Check for brace expansion
        platforms = parse_brace_expansion(file.name)
        if normalized_platform in platforms:
            if context_dir != Path("."):
                return str(context_dir / file.name)
            return file.name

    # 3. Fall back to default file
    return context_path


def resolve_context_spec(
    context_spec: str | list[Any],
    source_dir: Path,
    adapter_name: str,
) -> str | list[Any]:
    """Resolve context spec with platform-specific overrides.

    Handles both single file paths and lists of files/conditional includes.

    Args:
        context_spec: The context specification from config
        source_dir: The plugin source directory
        adapter_name: The adapter name (e.g., 'claude-code')

    Returns:
        Resolved context spec with platform-specific paths where applicable
    """
    if isinstance(context_spec, str):
        return find_platform_specific_file(source_dir, context_spec, adapter_name)

    elif isinstance(context_spec, list):
        resolved: list[Any] = []
        for item in context_spec:
            if isinstance(item, str):
                resolved.append(find_platform_specific_file(source_dir, item, adapter_name))
            elif isinstance(item, dict):
                # Conditional include - resolve the path within
                path = item.get("path", "")
                if path:
                    resolved_path = find_platform_specific_file(source_dir, path, adapter_name)
                    resolved.append({**item, "path": resolved_path})
                else:
                    resolved.append(item)
            else:
                resolved.append(item)
        return resolved

    return context_spec
