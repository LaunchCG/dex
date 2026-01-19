"""Marker-based content management for agent files.

This module provides utilities for managing content within shared files
(like CLAUDE.md or AGENTS.md) using HTML-style comment markers. Each plugin's
content is wrapped in markers that allow for safe insertion, update, and removal
without affecting other plugins or user content.

Marker Format:
    <!-- dex:plugin:{plugin-name}:start -->
    ... managed content ...
    <!-- dex:plugin:{plugin-name}:end -->
"""

from __future__ import annotations

import re
from dataclasses import dataclass


def make_start_marker(plugin_name: str) -> str:
    """Create the start marker for a plugin's content block.

    Args:
        plugin_name: Name of the plugin

    Returns:
        HTML comment start marker
    """
    return f"<!-- dex:plugin:{plugin_name}:start -->"


def make_end_marker(plugin_name: str) -> str:
    """Create the end marker for a plugin's content block.

    Args:
        plugin_name: Name of the plugin

    Returns:
        HTML comment end marker
    """
    return f"<!-- dex:plugin:{plugin_name}:end -->"


def wrap_content(plugin_name: str, content: str) -> str:
    """Wrap content in plugin markers.

    Args:
        plugin_name: Name of the plugin
        content: Content to wrap

    Returns:
        Content wrapped with start and end markers
    """
    start = make_start_marker(plugin_name)
    end = make_end_marker(plugin_name)
    # Ensure content has proper newlines
    content = content.strip()
    return f"{start}\n{content}\n{end}"


@dataclass
class PluginSection:
    """Represents a plugin's section within a file."""

    plugin_name: str
    content: str
    start_pos: int
    end_pos: int


def find_plugin_section(file_content: str, plugin_name: str) -> PluginSection | None:
    """Find a plugin's section in file content.

    Args:
        file_content: The full file content
        plugin_name: Name of the plugin to find

    Returns:
        PluginSection if found, None otherwise
    """
    start_marker = make_start_marker(plugin_name)
    end_marker = make_end_marker(plugin_name)

    start_pos = file_content.find(start_marker)
    if start_pos == -1:
        return None

    end_pos = file_content.find(end_marker, start_pos)
    if end_pos == -1:
        return None

    # Include the end marker in the section
    end_pos += len(end_marker)

    # Extract the content between markers (excluding markers themselves)
    content_start = start_pos + len(start_marker)
    inner_content = file_content[content_start : end_pos - len(end_marker)].strip()

    return PluginSection(
        plugin_name=plugin_name,
        content=inner_content,
        start_pos=start_pos,
        end_pos=end_pos,
    )


def list_plugin_sections(file_content: str) -> list[str]:
    """List all plugin names that have sections in the file.

    Args:
        file_content: The full file content

    Returns:
        List of plugin names with sections in the file
    """
    pattern = r"<!-- dex:plugin:([^:]+):start -->"
    matches = re.findall(pattern, file_content)
    return matches


def insert_plugin_section(file_content: str, plugin_name: str, content: str) -> str:
    """Insert or update a plugin's section in the file.

    If the plugin already has a section, it will be replaced.
    If not, the section will be appended to the end of the file.

    Args:
        file_content: The full file content
        plugin_name: Name of the plugin
        content: Content to insert (will be wrapped in markers)

    Returns:
        Updated file content
    """
    wrapped = wrap_content(plugin_name, content)
    existing = find_plugin_section(file_content, plugin_name)

    if existing:
        # Replace existing section
        return file_content[: existing.start_pos] + wrapped + file_content[existing.end_pos :]
    else:
        # Append to end (with blank line separator if file has content)
        if file_content.strip():
            return file_content.rstrip() + "\n\n" + wrapped + "\n"
        else:
            return wrapped + "\n"


def remove_plugin_section(file_content: str, plugin_name: str) -> str:
    """Remove a plugin's section from the file.

    Args:
        file_content: The full file content
        plugin_name: Name of the plugin to remove

    Returns:
        Updated file content with the section removed
    """
    section = find_plugin_section(file_content, plugin_name)
    if not section:
        return file_content

    # Remove the section
    before = file_content[: section.start_pos]
    after = file_content[section.end_pos :]

    # Clean up extra blank lines that might result from removal
    result = before.rstrip() + after.lstrip("\n")

    # If the file is now empty or just whitespace, return empty string
    if not result.strip():
        return ""

    # Ensure proper ending
    if not result.endswith("\n"):
        result += "\n"

    return result
