"""Platform adapters for Dex.

This module provides the adapter registration system and discovery mechanism.
All platform-specific logic is encapsulated within adapters that implement
the PlatformAdapter interface.
"""

from __future__ import annotations

import importlib
from collections.abc import Callable
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from dex.adapters.base import PlatformAdapter

_ADAPTERS: dict[str, type[PlatformAdapter]] = {}
_LOADED = False

# Known adapter modules - add new adapters here
_ADAPTER_MODULES = [
    "dex.adapters.antigravity",
    "dex.adapters.claude_code",
    "dex.adapters.codex",
    "dex.adapters.cursor",
    "dex.adapters.github_copilot",
]


def register_adapter(
    name: str,
) -> Callable[[type[PlatformAdapter]], type[PlatformAdapter]]:
    """Decorator for adapter registration.

    Usage:
        @register_adapter("claude-code")
        class ClaudeCodeAdapter(PlatformAdapter):
            ...
    """

    def decorator(cls: type[PlatformAdapter]) -> type[PlatformAdapter]:
        _ADAPTERS[name] = cls
        return cls

    return decorator


def _load_adapters() -> None:
    """Load all adapter modules to trigger registration."""
    global _LOADED
    if _LOADED:
        return

    for module_name in _ADAPTER_MODULES:
        importlib.import_module(module_name)

    _LOADED = True


def get_adapter(name: str) -> PlatformAdapter:
    """Get an instantiated adapter by name.

    Args:
        name: The adapter name (e.g., "claude-code", "cursor")

    Returns:
        An instantiated adapter

    Raises:
        ValueError: If the adapter is not registered
    """
    _load_adapters()

    if name not in _ADAPTERS:
        available = ", ".join(_ADAPTERS.keys()) or "none"
        raise ValueError(f"Unknown adapter: {name}. Available adapters: {available}")
    return _ADAPTERS[name]()


def list_adapters() -> list[str]:
    """List all registered adapter names."""
    _load_adapters()
    return list(_ADAPTERS.keys())
